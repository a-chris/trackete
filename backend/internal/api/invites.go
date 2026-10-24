package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"github.com/a-chris/trackete/backend/internal/auth"
	"github.com/a-chris/trackete/backend/internal/mailer"
	"github.com/a-chris/trackete/backend/internal/model"
	"github.com/a-chris/trackete/backend/internal/store"
)

var (
	errInviteTokenInvalid = errors.New("invalid invitation link")
	errInviteTokenExpired = errors.New("this invitation link has expired")
	// errHandled marks a request whose error response has already been written.
	errHandled = errors.New("response already written")
)

type inviteRequest struct {
	Email string `json:"email"`
}

type acceptInvitationRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

// inviteToken signs the invitation ID, the normalized email and the expiry with
// the application secret. Nothing is stored: the link can be regenerated for as
// long as the invitation stays active, and it stops validating the moment the
// invitation is accepted, rejected or removed.
func (a *API) inviteToken(inv *model.Invitation) string {
	payload := strings.Join([]string{inv.ID, inv.Email, strconv.FormatInt(inv.ExpiresAt.Unix(), 10)}, "\n")
	encoded := base64.RawURLEncoding.EncodeToString([]byte(payload))
	return encoded + "." + a.signInvitePayload(encoded)
}

func (a *API) signInvitePayload(encoded string) string {
	mac := hmac.New(sha256.New, []byte(a.cfg.JWTSecret))
	mac.Write([]byte("invite:" + encoded))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (a *API) parseInviteToken(token string) (id, email string, err error) {
	encoded, signature, ok := strings.Cut(token, ".")
	if !ok || encoded == "" || signature == "" {
		return "", "", errInviteTokenInvalid
	}
	if !hmac.Equal([]byte(a.signInvitePayload(encoded)), []byte(signature)) {
		return "", "", errInviteTokenInvalid
	}
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", "", errInviteTokenInvalid
	}
	parts := strings.Split(string(raw), "\n")
	if len(parts) != 3 {
		return "", "", errInviteTokenInvalid
	}
	expires, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return "", "", errInviteTokenInvalid
	}
	if time.Now().After(time.Unix(expires, 0)) {
		return "", "", errInviteTokenExpired
	}
	return parts[0], parts[1], nil
}

// invitationFromToken resolves a link to a live invitation. The address in the
// token must still match the stored invitation, so editing the link cannot
// redirect an invitation to another inbox.
func (a *API) invitationFromToken(ctx context.Context, token string) (*model.Invitation, error) {
	id, email, err := a.parseInviteToken(token)
	if err != nil {
		return nil, err
	}
	inv, err := a.store.InvitationByID(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, errInviteTokenInvalid
		}
		return nil, err
	}
	if inv.Email != email {
		return nil, errInviteTokenInvalid
	}
	if inv.Status != model.InvitePending && inv.Status != model.InviteDelivered {
		return nil, fmt.Errorf("this invitation is %s", inv.Status)
	}
	return inv, nil
}

func failInvitation(c *gin.Context, err error) {
	switch {
	case errors.Is(err, errInviteTokenExpired):
		fail(c, http.StatusGone, err.Error())
	case errors.Is(err, errInviteTokenInvalid):
		fail(c, http.StatusNotFound, err.Error())
	default:
		// Lifecycle errors ("this invitation is accepted") are not transient.
		fail(c, http.StatusConflict, err.Error())
	}
}

func (a *API) inviteURL(inv *model.Invitation) string {
	return fmt.Sprintf("%s/invite/%s", a.cfg.AppURL, a.inviteToken(inv))
}

// deliverInvitation sends the acceptance mail and records the outcome. A failed
// send leaves the invitation pending so the owner can retry; it never fails the
// caller, which may be in the middle of creating a tracker.
func (a *API) deliverInvitation(ctx context.Context, inv *model.Invitation, trackerName, inviterName string) {
	message := mailer.Invitation(inv.Email, inviterName, trackerName, a.inviteURL(inv), inv.ExpiresAt)
	if err := a.mail.Send(ctx, message); err != nil {
		log.Printf("invitation %s to %s could not be delivered: %v", inv.ID, inv.Email, err)
		return
	}
	if err := a.store.SetInvitationStatus(ctx, inv.ID, model.InviteDelivered); err != nil {
		log.Printf("invitation %s delivered but not marked delivered: %v", inv.ID, err)
	}
}

// inviteByEmail records an invitation and tries to deliver it. Errors are
// returned only for things the caller can act on (already a member, bad email);
// delivery problems stay visible as a pending invitation.
func (a *API) inviteByEmail(ctx context.Context, inviterID string, tracker *model.Tracker, email string) (*model.Invitation, error) {
	inv, _, err := a.store.CreateInvitation(ctx, inviterID, tracker.ID, email)
	if err != nil {
		return nil, err
	}
	inviterName := ""
	if inviter, err := a.store.UserByID(ctx, inviterID); err == nil {
		inviterName = inviter.Name
	}
	a.deliverInvitation(ctx, inv, tracker.Name, inviterName)
	return a.store.InvitationByID(ctx, inv.ID)
}

// ownerTracker loads the tracker in the request path and insists the caller is
// its owner. Members keep read/log access; metadata, invitations and removal
// stay with the owner.
func (a *API) ownerTracker(c *gin.Context) (*model.Tracker, bool) {
	tracker, err := a.store.GetTracker(c.Request.Context(), userID(c), c.Param("id"))
	if err != nil {
		failStore(c, err)
		return nil, false
	}
	if tracker.Role != model.RoleOwner {
		fail(c, http.StatusForbidden, "only the tracker owner can do that")
		return nil, false
	}
	return tracker, true
}

func normalizeInviteEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if !validEmail(email) {
		return "", fmt.Errorf("please enter a valid email address")
	}
	return email, nil
}

func (a *API) createInvitation(c *gin.Context) {
	tracker, ok := a.ownerTracker(c)
	if !ok {
		return
	}

	var req inviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid request body")
		return
	}
	email, err := normalizeInviteEmail(req.Email)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	inv, err := a.inviteByEmail(c.Request.Context(), userID(c), tracker, email)
	if err != nil {
		if errors.Is(err, store.ErrAlreadyMember) {
			fail(c, http.StatusConflict, "that address already has access to this tracker")
			return
		}
		failStore(c, err)
		return
	}
	c.JSON(http.StatusCreated, inv)
}

func (a *API) resendInvitation(c *gin.Context) {
	tracker, ok := a.ownerTracker(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	inv, err := a.store.InvitationByID(ctx, c.Param("inviteId"))
	if err != nil || inv.TrackerID != tracker.ID {
		fail(c, http.StatusNotFound, "not found")
		return
	}
	if inv.Status != model.InvitePending && inv.Status != model.InviteDelivered {
		fail(c, http.StatusConflict, "this invitation is "+inv.Status)
		return
	}
	// Extending the expiry first, so a lapsed invitation still emails a usable link.
	if err := a.store.RefreshInvitationExpiry(ctx, inv.ID); err != nil {
		failStore(c, err)
		return
	}
	inv, err = a.store.InvitationByID(ctx, inv.ID)
	if err != nil {
		failStore(c, err)
		return
	}
	inviterName := ""
	if inviter, err := a.store.UserByID(ctx, userID(c)); err == nil {
		inviterName = inviter.Name
	}
	a.deliverInvitation(ctx, inv, tracker.Name, inviterName)
	refreshed, err := a.store.InvitationByID(ctx, inv.ID)
	if err != nil {
		failStore(c, err)
		return
	}
	c.JSON(http.StatusOK, refreshed)
}

// trackerSharing lists who can see the tracker and what was invited. Members may
// read it; only the owner gets the action endpoints that change it.
func (a *API) trackerSharing(c *gin.Context) {
	tracker, err := a.store.GetTracker(c.Request.Context(), userID(c), c.Param("id"))
	if err != nil {
		failStore(c, err)
		return
	}
	members, err := a.store.ListMembers(c.Request.Context(), tracker.ID)
	if err != nil {
		failStore(c, err)
		return
	}
	invitations, err := a.store.ListInvitations(c.Request.Context(), tracker.ID)
	if err != nil {
		failStore(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"trackerId":   tracker.ID,
		"role":        tracker.Role,
		"members":     members,
		"invitations": invitations,
	})
}

func (a *API) removeMember(c *gin.Context) {
	tracker, ok := a.ownerTracker(c)
	if !ok {
		return
	}
	memberID := c.Param("memberId")
	if memberID == userID(c) {
		fail(c, http.StatusBadRequest, "the owner cannot be removed")
		return
	}
	if err := a.store.RemoveMember(c.Request.Context(), tracker.ID, memberID); err != nil {
		failStore(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (a *API) inviteSuggestions(c *gin.Context) {
	suggestions, err := a.store.InviteSuggestions(c.Request.Context(), userID(c), 20)
	if err != nil {
		failStore(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"suggestions": suggestions})
}

// invitationDetails is what the acceptance page needs: enough to name the
// tracker and decide between "sign in and join" and "create your account".
type invitationDetails struct {
	model.Invitation
	NeedsAccount  bool   `json:"needsAccount"`
	SignedInEmail string `json:"signedInEmail,omitempty"`
}

func (a *API) getInvitation(c *gin.Context) {
	ctx := c.Request.Context()
	inv, err := a.invitationFromToken(ctx, c.Param("token"))
	if err != nil {
		failInvitation(c, err)
		return
	}

	details := invitationDetails{Invitation: *inv}
	_, _, lookupErr := a.store.UserByEmail(ctx, inv.Email)
	details.NeedsAccount = errors.Is(lookupErr, store.ErrNotFound)
	if user := a.optionalUser(c); user != nil {
		details.SignedInEmail = user.Email
	}
	c.JSON(http.StatusOK, details)
}

// acceptInvitation joins the tracker and signs the invitee in. An existing
// account is recognised by the invited address; a new invitee sets a name and
// password here, which is why invitation-based signup keeps working even when
// open registration is disabled.
func (a *API) acceptInvitation(c *gin.Context) {
	ctx := c.Request.Context()
	inv, err := a.invitationFromToken(ctx, c.Param("token"))
	if err != nil {
		failInvitation(c, err)
		return
	}

	var req acceptInvitationRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			fail(c, http.StatusBadRequest, "invalid request body")
			return
		}
	}

	user, _, err := a.store.UserByEmail(ctx, inv.Email)
	switch {
	case err == nil:
		// Password is not used: holding the link already proves control of the
		// invited inbox.
	case errors.Is(err, store.ErrNotFound):
		user, err = a.createInvitedUser(c, inv.Email, req)
		if err != nil {
			return
		}
	default:
		failStore(c, err)
		return
	}

	if err := a.store.AddMember(ctx, inv.TrackerID, user.ID, model.RoleMember); err != nil {
		failStore(c, err)
		return
	}
	if err := a.store.SetInvitationStatus(ctx, inv.ID, model.InviteAccepted); err != nil {
		failStore(c, err)
		return
	}

	token := a.issueSession(c, user.ID, user.TokenVersion)
	c.JSON(http.StatusOK, gin.H{
		"user":    user,
		"tracker": gin.H{"id": inv.TrackerID, "name": inv.TrackerName},
		"token":   token,
	})
}

func (a *API) createInvitedUser(c *gin.Context, email string, req acceptInvitationRequest) (*model.User, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name, _, _ = strings.Cut(email, "@")
	}
	if utf8.RuneCountInString(name) > 80 {
		fail(c, http.StatusBadRequest, "name must be 80 characters or fewer")
		return nil, errHandled
	}
	if len(req.Password) < 8 {
		fail(c, http.StatusBadRequest, "password must be at least 8 characters")
		return nil, errHandled
	}
	if len([]byte(req.Password)) > auth.MaxPasswordBytes {
		fail(c, http.StatusBadRequest, "password must be 72 bytes or fewer")
		return nil, errHandled
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		failStore(c, err)
		return nil, errHandled
	}
	user, err := a.store.CreateUser(c.Request.Context(), email, name, hash)
	if err != nil {
		failStore(c, err)
		return nil, errHandled
	}
	if err := a.store.EnsureDefaultDashboard(c.Request.Context(), user.ID); err != nil {
		failStore(c, err)
		return nil, errHandled
	}
	return user, nil
}

func (a *API) rejectInvitation(c *gin.Context) {
	ctx := c.Request.Context()
	inv, err := a.invitationFromToken(ctx, c.Param("token"))
	if err != nil {
		failInvitation(c, err)
		return
	}
	if err := a.store.SetInvitationStatus(ctx, inv.ID, model.InviteRejected); err != nil {
		failStore(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
