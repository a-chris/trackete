package api

import (
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"github.com/a-chris/trackete/backend/internal/auth"
	"github.com/a-chris/trackete/backend/internal/store"
)

// dummyHash is compared against when an email is unknown, so login timing does
// not reveal whether an account exists.
const dummyHash = "$2a$12$KN4jMmNUXVJQFiMRs7VIVOF5aR7Nwr10/S0hq2HkAxbSFpk0PFOEK"

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type updateMeRequest struct {
	Name string `json:"name"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func (a *API) register(c *gin.Context) {
	if !a.cfg.AllowRegistration {
		fail(c, http.StatusForbidden, "registration is disabled on this instance")
		return
	}

	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid request body")
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	name := strings.TrimSpace(req.Name)
	if !validEmail(email) {
		fail(c, http.StatusBadRequest, "please enter a valid email address")
		return
	}
	if len(req.Password) < 8 {
		fail(c, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}
	// bcrypt refuses inputs above 72 bytes; validate before hashing so an
	// over-long password is a 400 rather than a 500.
	if len([]byte(req.Password)) > auth.MaxPasswordBytes {
		fail(c, http.StatusBadRequest, "password must be 72 bytes or fewer")
		return
	}
	if name == "" {
		name, _, _ = strings.Cut(email, "@")
	}
	if utf8.RuneCountInString(name) > 80 {
		fail(c, http.StatusBadRequest, "name must be 80 characters or fewer")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		failStore(c, err)
		return
	}

	user, err := a.store.CreateUser(c.Request.Context(), email, name, hash)
	if err != nil {
		failStore(c, err)
		return
	}
	if err := a.store.EnsureDefaultDashboard(c.Request.Context(), user.ID); err != nil {
		failStore(c, err)
		return
	}

	token := a.issueSession(c, user.ID, user.TokenVersion)
	a.limiter.reset(c.ClientIP())
	c.JSON(http.StatusCreated, gin.H{"user": user, "token": token})
}

func (a *API) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid request body")
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	user, hash, err := a.store.UserByEmail(c.Request.Context(), email)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			auth.CheckPassword(dummyHash, req.Password)
			fail(c, http.StatusUnauthorized, "invalid email or password")
			return
		}
		failStore(c, err)
		return
	}
	if !auth.CheckPassword(hash, req.Password) {
		fail(c, http.StatusUnauthorized, "invalid email or password")
		return
	}

	token := a.issueSession(c, user.ID, user.TokenVersion)
	a.limiter.reset(c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"user": user, "token": token})
}

func (a *API) logout(c *gin.Context) {
	a.clearSession(c)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (a *API) me(c *gin.Context) {
	user, err := a.store.UserByID(c.Request.Context(), userID(c))
	if err != nil {
		failStore(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (a *API) updateMe(c *gin.Context) {
	var req updateMeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid request body")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" || utf8.RuneCountInString(name) > 80 {
		fail(c, http.StatusBadRequest, "name must be between 1 and 80 characters")
		return
	}
	uid := userID(c)
	if err := a.store.UpdateUserName(c.Request.Context(), uid, name); err != nil {
		failStore(c, err)
		return
	}
	user, err := a.store.UserByID(c.Request.Context(), uid)
	if err != nil {
		failStore(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (a *API) changePassword(c *gin.Context) {
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.NewPassword) < 8 {
		fail(c, http.StatusBadRequest, "new password must be at least 8 characters")
		return
	}
	if len([]byte(req.NewPassword)) > auth.MaxPasswordBytes {
		fail(c, http.StatusBadRequest, "new password must be 72 bytes or fewer")
		return
	}

	uid := userID(c)
	me, err := a.store.UserByID(c.Request.Context(), uid)
	if err != nil {
		failStore(c, err)
		return
	}
	_, hash, err := a.store.UserByEmail(c.Request.Context(), me.Email)
	if err != nil {
		failStore(c, err)
		return
	}
	if !auth.CheckPassword(hash, req.CurrentPassword) {
		fail(c, http.StatusForbidden, "current password is incorrect")
		return
	}

	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		failStore(c, err)
		return
	}
	version, err := a.store.UpdateUserPassword(c.Request.Context(), uid, newHash)
	if err != nil {
		failStore(c, err)
		return
	}
	// The version bump invalidates the old cookie; hand the caller a fresh one.
	a.issueSession(c, uid, version)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (a *API) issueSession(c *gin.Context, uid string, version int) string {
	token, err := auth.NewToken(a.cfg.JWTSecret, uid, version, a.cfg.TokenTTL)
	if err != nil {
		failStore(c, err)
		return ""
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   a.secureCookie(c),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(a.cfg.TokenTTL.Seconds()),
	})
	return token
}

func (a *API) clearSession(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   a.secureCookie(c),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func (a *API) secureCookie(c *gin.Context) bool {
	return c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
}

func validEmail(email string) bool {
	if email == "" || len(email) > 254 || !strings.Contains(email, "@") {
		return false
	}
	addr, err := mail.ParseAddress(email)
	return err == nil && strings.EqualFold(addr.Address, email)
}
