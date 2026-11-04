package api

import (
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/a-chris/trackete/backend/internal/auth"
	"github.com/a-chris/trackete/backend/internal/config"
	"github.com/a-chris/trackete/backend/internal/model"
)

// invitationLink reads the newest acceptance link out of the mail outbox, which
// is also the end-to-end check that the file transport rendered a usable mail.
func (c *testClient) invitationLink() string {
	c.t.Helper()
	body, err := os.ReadFile(c.mailPath)
	if err != nil {
		c.t.Fatalf("read outbox: %v", err)
	}
	text := string(body)
	index := strings.LastIndex(text, "/invite/")
	if index < 0 {
		c.t.Fatalf("no invitation link in the outbox:\n%s", text)
	}
	link := text[index:]
	for _, stop := range []string{"\r", "\n", " "} {
		if cut := strings.Index(link, stop); cut >= 0 {
			link = link[:cut]
		}
	}
	return link
}

func (c *testClient) invitationToken() string {
	c.t.Helper()
	link := c.invitationLink()
	return link[strings.LastIndex(link, "/")+1:]
}

type invitationPayload struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	Status      string `json:"status"`
	TrackerName string `json:"trackerName"`
}

type sharingPayload struct {
	Role        string                `json:"role"`
	Members     []model.TrackerMember `json:"members"`
	Invitations []invitationPayload   `json:"invitations"`
}

func TestTrackerSharingLifecycle(t *testing.T) {
	c := newTestClient(t)
	c.cookie = c.session(c.do("POST", "/api/auth/register",
		`{"email":"owner@example.com","password":"supersecret","name":"Owner"}`))

	var created struct {
		ID          string              `json:"id"`
		Role        string              `json:"role"`
		Invitations []invitationPayload `json:"invitations"`
	}
	rec := c.doJSON("POST", "/api/trackers",
		`{"name":"Beers","kind":"numeric","unit":"beers","icon":"🍺","color":"#f59e0b","shareWith":["Friend@Example.com"]}`,
		&created)
	if rec.Code != http.StatusCreated || created.ID == "" {
		t.Fatalf("create shared tracker: %d %s", rec.Code, rec.Body.String())
	}
	if created.Role != model.RoleOwner {
		t.Fatalf("creator must be the owner, got %q", created.Role)
	}
	if len(created.Invitations) != 1 || created.Invitations[0].Status != model.InviteDelivered {
		t.Fatalf("expected one delivered invitation, got %+v", created.Invitations)
	}
	if created.Invitations[0].Email != "friend@example.com" {
		t.Fatalf("invitation address must be normalized, got %q", created.Invitations[0].Email)
	}

	// The list tells the owner who the tracker is shared with: an address with
	// an open invitation counts, so a share request is visible right away.
	var listed struct {
		Trackers []struct {
			ID         string   `json:"id"`
			SharedWith []string `json:"sharedWith"`
		} `json:"trackers"`
	}
	if rec := c.doJSON("GET", "/api/trackers", "", &listed); rec.Code != http.StatusOK {
		t.Fatalf("list trackers: %d %s", rec.Code, rec.Body.String())
	}
	if len(listed.Trackers) != 1 || listed.Trackers[0].ID != created.ID {
		t.Fatalf("unexpected tracker list: %+v", listed.Trackers)
	}
	if got := listed.Trackers[0].SharedWith; len(got) != 1 || got[0] != "friend@example.com" {
		t.Fatalf("expected the invited address on the list, got %+v", got)
	}

	var sharing sharingPayload
	if rec := c.doJSON("GET", "/api/trackers/"+created.ID+"/sharing", "", &sharing); rec.Code != http.StatusOK {
		t.Fatalf("sharing overview: %d %s", rec.Code, rec.Body.String())
	}
	if sharing.Role != model.RoleOwner || len(sharing.Members) != 1 || sharing.Members[0].Role != model.RoleOwner {
		t.Fatalf("unexpected sharing payload: %+v", sharing)
	}
	if len(sharing.Invitations) != 1 {
		t.Fatalf("expected the invitation listed, got %+v", sharing.Invitations)
	}

	// A stranger still cannot see the tracker, shared or not.
	stranger := &testClient{t: t, router: c.router}
	stranger.cookie = stranger.session(stranger.do("POST", "/api/auth/register",
		`{"email":"stranger@example.com","password":"supersecret","name":"Stranger"}`))
	if rec := stranger.do("GET", "/api/trackers/"+created.ID, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("stranger access: expected 404, got %d", rec.Code)
	}

	// The invitation link creates the invitee's account, joins the tracker and
	// signs them in.
	token := c.invitationToken()
	var inviteInfo struct {
		TrackerName  string `json:"trackerName"`
		Email        string `json:"email"`
		Status       string `json:"status"`
		NeedsAccount bool   `json:"needsAccount"`
	}
	if rec := stranger.doJSON("GET", "/api/invitations/"+token, "", &inviteInfo); rec.Code != http.StatusOK {
		t.Fatalf("invitation details: %d %s", rec.Code, rec.Body.String())
	}
	if !inviteInfo.NeedsAccount || inviteInfo.TrackerName != "Beers" || inviteInfo.Email != "friend@example.com" {
		t.Fatalf("unexpected invitation details: %+v", inviteInfo)
	}

	friend := &testClient{t: t, router: c.router}
	rec = friend.do("POST", "/api/invitations/"+token+"/accept",
		`{"name":"Friend","password":"supersecret"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("accept invitation: %d %s", rec.Code, rec.Body.String())
	}
	friend.cookie = friend.session(rec)

	var me struct {
		User struct {
			Email string `json:"email"`
		} `json:"user"`
	}
	if rec := friend.doJSON("GET", "/api/auth/me", "", &me); rec.Code != http.StatusOK || me.User.Email != "friend@example.com" {
		t.Fatalf("acceptance should sign the invitee in: %d %+v", rec.Code, me)
	}

	var asMember struct {
		ID   string `json:"id"`
		Role string `json:"role"`
	}
	if rec := friend.doJSON("GET", "/api/trackers/"+created.ID, "", &asMember); rec.Code != http.StatusOK || asMember.Role != model.RoleMember {
		t.Fatalf("member read: %d %+v", rec.Code, asMember)
	}
	if rec := friend.do("GET", "/api/trackers", ""); rec.Code != http.StatusOK {
		t.Fatalf("member tracker list: %d", rec.Code)
	}
	// Once the invitation is accepted the address stays on the list, now as a
	// member. A fresh decode target is required: encoding/json reuses slice
	// elements and does not zero fields the response omits.
	var listedAfterJoin struct {
		Trackers []struct {
			ID         string   `json:"id"`
			SharedWith []string `json:"sharedWith"`
		} `json:"trackers"`
	}
	if rec := c.doJSON("GET", "/api/trackers", "", &listedAfterJoin); rec.Code != http.StatusOK {
		t.Fatalf("list after join: %d %s", rec.Code, rec.Body.String())
	}
	if got := listedAfterJoin.Trackers[0].SharedWith; len(got) != 1 || got[0] != "friend@example.com" {
		t.Fatalf("accepted invitation should stay listed as a member, got %+v", got)
	}

	// Members log data but never change the tracker, invite, or remove people.
	today := time.Now().UTC().Format("2006-01-02")
	if rec := friend.do("PUT", "/api/trackers/"+created.ID+"/entries/"+today, `{"value":2}`); rec.Code != http.StatusOK {
		t.Fatalf("member logs an entry: %d %s", rec.Code, rec.Body.String())
	}
	if rec := c.do("PUT", "/api/trackers/"+created.ID+"/entries/"+today, `{"value":3}`); rec.Code != http.StatusOK {
		t.Fatalf("owner logs the same day: %d %s", rec.Code, rec.Body.String())
	}
	// Metadata writes stay keyed to the owner row, so a member just sees 404.
	if rec := friend.do("PUT", "/api/trackers/"+created.ID, `{"name":"Hijacked","kind":"numeric"}`); rec.Code != http.StatusNotFound {
		t.Fatalf("member tracker update: expected 404, got %d %s", rec.Code, rec.Body.String())
	}
	if rec := friend.do("DELETE", "/api/trackers/"+created.ID, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("member tracker delete: expected 404, got %d %s", rec.Code, rec.Body.String())
	}
	if rec := friend.do("POST", "/api/trackers/"+created.ID+"/invitations", `{"email":"third@example.com"}`); rec.Code != http.StatusForbidden {
		t.Fatalf("member invite: expected 403, got %d", rec.Code)
	}
	if rec := friend.do("DELETE", "/api/trackers/"+created.ID+"/members/"+sharing.Members[0].UserID, ""); rec.Code != http.StatusForbidden {
		t.Fatalf("member removal by member: expected 403, got %d", rec.Code)
	}

	// Both members' values are visible and attributed.
	var entries struct {
		Entries []struct {
			Day      string  `json:"day"`
			Value    float64 `json:"value"`
			UserName string  `json:"userName"`
		} `json:"entries"`
	}
	if rec := c.doJSON("GET", "/api/trackers/"+created.ID+"/entries", "", &entries); rec.Code != http.StatusOK {
		t.Fatalf("list entries: %d %s", rec.Code, rec.Body.String())
	}
	if len(entries.Entries) != 2 {
		t.Fatalf("expected one entry per member for the same day, got %+v", entries.Entries)
	}
	names := map[string]bool{}
	for _, entry := range entries.Entries {
		names[entry.UserName] = true
	}
	if !names["Owner"] || !names["Friend"] {
		t.Fatalf("entries must be attributed to their authors, got %+v", entries.Entries)
	}

	// One chart series per member, combined summary.
	var chart struct {
		Series []struct {
			Name string    `json:"name"`
			Data []float64 `json:"data"`
		} `json:"series"`
		Summary struct {
			Total float64 `json:"total"`
		} `json:"summary"`
	}
	if rec := c.doJSON("GET", "/api/trackers/"+created.ID+"/data?chart=bar&range=7d", "", &chart); rec.Code != http.StatusOK {
		t.Fatalf("tracker data: %d %s", rec.Code, rec.Body.String())
	}
	if len(chart.Series) != 2 || chart.Summary.Total != 5 {
		t.Fatalf("expected two member series totalling 5, got %+v (total %v)", chart.Series, chart.Summary.Total)
	}

	// Dashboards stay user-owned, but a member can chart a shared tracker on
	// their own dashboard and gets the same per-member series there.
	var dashboards struct {
		Dashboards []struct {
			ID string `json:"id"`
		} `json:"dashboards"`
	}
	if rec := friend.doJSON("GET", "/api/dashboards", "", &dashboards); rec.Code != http.StatusOK || len(dashboards.Dashboards) == 0 {
		t.Fatalf("member dashboards: %d %+v", rec.Code, dashboards)
	}
	var widget struct {
		ID string `json:"id"`
	}
	if rec := friend.doJSON("POST", "/api/dashboards/"+dashboards.Dashboards[0].ID+"/widgets",
		`{"trackerId":"`+created.ID+`","title":"Beers","chartType":"line","config":{"range":"7d"}}`, &widget); rec.Code != http.StatusCreated {
		t.Fatalf("member widget on a shared tracker: %d %s", rec.Code, rec.Body.String())
	}
	var dashboardData struct {
		Widgets map[string]struct {
			Series []struct {
				Name string `json:"name"`
			} `json:"series"`
		} `json:"widgets"`
	}
	if rec := friend.doJSON("GET", "/api/dashboards/"+dashboards.Dashboards[0].ID+"/data", "", &dashboardData); rec.Code != http.StatusOK {
		t.Fatalf("member dashboard data: %d %s", rec.Code, rec.Body.String())
	}
	if got := dashboardData.Widgets[widget.ID].Series; len(got) != 2 {
		t.Fatalf("expected a member series per author in the widget, got %+v", got)
	}

	// The overview lists the shared tracker with the member's own values only.
	var overview struct {
		Items []struct {
			Tracker struct {
				ID string `json:"id"`
			} `json:"tracker"`
			PeriodLogged int `json:"periodLogged"`
		} `json:"items"`
	}
	if rec := friend.doJSON("GET", "/api/overview", "", &overview); rec.Code != http.StatusOK {
		t.Fatalf("member overview: %d %s", rec.Code, rec.Body.String())
	}
	found := false
	for _, item := range overview.Items {
		if item.Tracker.ID == created.ID {
			found = true
			if item.PeriodLogged != 1 {
				t.Fatalf("overview must count only the member's own day, got %d", item.PeriodLogged)
			}
		}
	}
	if !found {
		t.Fatal("the overview should list the shared tracker for its member")
	}

	// A member only deletes their own entry.
	if rec := friend.do("DELETE", "/api/trackers/"+created.ID+"/entries/"+today, ""); rec.Code != http.StatusOK {
		t.Fatalf("member deletes own entry: %d %s", rec.Code, rec.Body.String())
	}
	if rec := friend.do("DELETE", "/api/trackers/"+created.ID+"/entries/"+today, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("member must not delete the owner's entry: expected 404, got %d", rec.Code)
	}

	// The link is single use.
	if rec := friend.do("POST", "/api/invitations/"+token+"/accept", `{}`); rec.Code != http.StatusConflict {
		t.Fatalf("replayed invitation: expected 409, got %d %s", rec.Code, rec.Body.String())
	}

	// Removed members lose access; the invitation stays as history and powers
	// the owner's suggestions.
	var afterJoin sharingPayload
	if rec := c.doJSON("GET", "/api/trackers/"+created.ID+"/sharing", "", &afterJoin); rec.Code != http.StatusOK {
		t.Fatalf("sharing after join: %d %s", rec.Code, rec.Body.String())
	}
	memberID := ""
	for _, member := range afterJoin.Members {
		if member.Role == model.RoleMember {
			memberID = member.UserID
		}
	}
	if memberID == "" {
		t.Fatalf("expected the invitee among the members: %+v", afterJoin.Members)
	}
	if rec := c.do("DELETE", "/api/trackers/"+created.ID+"/members/"+memberID, ""); rec.Code != http.StatusOK {
		t.Fatalf("remove member: %d %s", rec.Code, rec.Body.String())
	}
	if rec := friend.do("GET", "/api/trackers/"+created.ID, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("removed member access: expected 404, got %d", rec.Code)
	}
	var suggestions struct {
		Suggestions []struct {
			Email string `json:"email"`
			Name  string `json:"name"`
		} `json:"suggestions"`
	}
	if rec := c.doJSON("GET", "/api/sharing/suggestions", "", &suggestions); rec.Code != http.StatusOK {
		t.Fatalf("suggestions: %d %s", rec.Code, rec.Body.String())
	}
	if len(suggestions.Suggestions) != 1 || suggestions.Suggestions[0].Email != "friend@example.com" {
		t.Fatalf("expected the invited address as a suggestion, got %+v", suggestions.Suggestions)
	}
}

func TestInvitationAcceptanceSignsInExistingAccount(t *testing.T) {
	c := newTestClient(t)
	c.cookie = c.session(c.do("POST", "/api/auth/register",
		`{"email":"owner@example.com","password":"supersecret","name":"Owner"}`))
	if rec := c.do("POST", "/api/auth/register", `{"email":"friend@example.com","password":"supersecret","name":"Friend"}`); rec.Code != http.StatusCreated {
		t.Fatalf("register friend: %d %s", rec.Code, rec.Body.String())
	}

	var tracker struct {
		ID string `json:"id"`
	}
	if rec := c.doJSON("POST", "/api/trackers", `{"name":"Run","kind":"boolean"}`, &tracker); rec.Code != http.StatusCreated {
		t.Fatalf("create tracker: %d %s", rec.Code, rec.Body.String())
	}
	if rec := c.do("POST", "/api/trackers/"+tracker.ID+"/invitations", `{"email":"friend@example.com"}`); rec.Code != http.StatusCreated {
		t.Fatalf("invite: %d %s", rec.Code, rec.Body.String())
	}
	// Inviting an address that already has access is refused.
	if rec := c.do("POST", "/api/trackers/"+tracker.ID+"/invitations", `{"email":"owner@example.com"}`); rec.Code != http.StatusConflict {
		t.Fatalf("invite the owner: expected 409, got %d %s", rec.Code, rec.Body.String())
	}

	token := c.invitationToken()
	friend := &testClient{t: t, router: c.router}
	rec := friend.do("POST", "/api/invitations/"+token+"/accept", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("accept as an existing account: %d %s", rec.Code, rec.Body.String())
	}
	friend.cookie = friend.session(rec)

	var asMember struct {
		Role string `json:"role"`
	}
	if rec := friend.doJSON("GET", "/api/trackers/"+tracker.ID, "", &asMember); rec.Code != http.StatusOK || asMember.Role != model.RoleMember {
		t.Fatalf("existing account should join as a member: %d %+v", rec.Code, asMember)
	}
}

func TestInvitationRejectionAndSingleUse(t *testing.T) {
	c := newTestClient(t)
	c.cookie = c.session(c.do("POST", "/api/auth/register",
		`{"email":"owner2@example.com","password":"supersecret","name":"Owner"}`))

	var tracker struct {
		ID string `json:"id"`
	}
	if rec := c.doJSON("POST", "/api/trackers", `{"name":"Books","kind":"boolean","shareWith":["reader@example.com"]}`, &tracker); rec.Code != http.StatusCreated {
		t.Fatalf("create tracker: %d %s", rec.Code, rec.Body.String())
	}

	token := c.invitationToken()
	guest := &testClient{t: t, router: c.router}
	if rec := guest.do("POST", "/api/invitations/"+token+"/reject", ""); rec.Code != http.StatusOK {
		t.Fatalf("reject: %d %s", rec.Code, rec.Body.String())
	}
	if rec := guest.do("POST", "/api/invitations/"+token+"/accept", `{"password":"supersecret"}`); rec.Code != http.StatusConflict {
		t.Fatalf("accept after reject: expected 409, got %d %s", rec.Code, rec.Body.String())
	}
	if rec := guest.do("GET", "/api/invitations/"+token, ""); rec.Code != http.StatusConflict {
		t.Fatalf("details after reject: expected 409, got %d", rec.Code)
	}
}

// Invitation acceptance is the only way into an invite-only instance, so it has
// to create accounts even when open registration is off.
func TestInvitationSignupWorksWithRegistrationDisabled(t *testing.T) {
	c := newTestClientWithConfig(t, config.Config{AllowRegistration: false})

	hash, err := auth.HashPassword("supersecret")
	if err != nil {
		t.Fatalf("hash seed password: %v", err)
	}
	if _, err := c.st.CreateUser(t.Context(), "owner3@example.com", "Owner", hash); err != nil {
		t.Fatalf("seed owner: %v", err)
	}
	if rec := c.do("POST", "/api/auth/register", `{"email":"nope@example.com","password":"supersecret"}`); rec.Code != http.StatusForbidden {
		t.Fatalf("registration should stay disabled: %d", rec.Code)
	}

	c.cookie = c.session(c.do("POST", "/api/auth/login", `{"email":"owner3@example.com","password":"supersecret"}`))

	var tracker struct {
		ID string `json:"id"`
	}
	if rec := c.doJSON("POST", "/api/trackers", `{"name":"Water","kind":"boolean","shareWith":["newbie@example.com"]}`, &tracker); rec.Code != http.StatusCreated {
		t.Fatalf("create tracker: %d %s", rec.Code, rec.Body.String())
	}

	guest := &testClient{t: t, router: c.router}
	rec := guest.do("POST", "/api/invitations/"+c.invitationToken()+"/accept",
		`{"name":"Newbie","password":"supersecret"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("accept on an invite-only instance: %d %s", rec.Code, rec.Body.String())
	}
	guest.cookie = guest.session(rec)
	var asMember struct {
		Role string `json:"role"`
	}
	if rec := guest.doJSON("GET", "/api/trackers/"+tracker.ID, "", &asMember); rec.Code != http.StatusOK || asMember.Role != model.RoleMember {
		t.Fatalf("invited signup should join the tracker: %d %+v", rec.Code, asMember)
	}
}

// The signed link is the credential: tampering, an expired expiry or a link for
// a different address must all be refused.
func TestInvitationTokenValidation(t *testing.T) {
	a := &API{cfg: config.Config{JWTSecret: "test-secret"}}

	live := &model.Invitation{ID: "inv-1", Email: "friend@example.com", ExpiresAt: time.Now().Add(time.Hour)}
	token := a.inviteToken(live)
	id, email, err := a.parseInviteToken(token)
	if err != nil || id != live.ID || email != live.Email {
		t.Fatalf("valid token rejected: id=%q email=%q err=%v", id, email, err)
	}

	expired := &model.Invitation{ID: "inv-2", Email: "friend@example.com", ExpiresAt: time.Now().Add(-time.Minute)}
	if _, _, err := a.parseInviteToken(a.inviteToken(expired)); err == nil {
		t.Fatal("expected an expired token to be refused")
	}

	tampered := token[:len(token)-2] + "xy"
	if _, _, err := a.parseInviteToken(tampered); err == nil {
		t.Fatal("expected a tampered token to be refused")
	}
	if _, _, err := a.parseInviteToken("not-a-token"); err == nil {
		t.Fatal("expected a malformed token to be refused")
	}

	// A token signed by another secret is worthless.
	other := &API{cfg: config.Config{JWTSecret: "another-secret"}}
	if _, _, err := a.parseInviteToken(other.inviteToken(live)); err == nil {
		t.Fatal("expected a foreign signature to be refused")
	}
}

// Resending a lapsed invitation extends its expiry, so the re-emailed link is
// still usable.
// Withdrawing an invitation removes the address from everything that says who
// has access, keeps the row as history and stops the mailed link from working.
func TestCancelInvitation(t *testing.T) {
	c := newTestClient(t)
	c.cookie = c.session(c.do("POST", "/api/auth/register",
		`{"email":"owner@example.com","password":"supersecret","name":"Owner"}`))

	var tracker struct {
		ID string `json:"id"`
	}
	if rec := c.doJSON("POST", "/api/trackers", `{"name":"Run","kind":"boolean","shareWith":["friend@example.com"]}`, &tracker); rec.Code != http.StatusCreated {
		t.Fatalf("create tracker: %d %s", rec.Code, rec.Body.String())
	}

	var sharing sharingPayload
	if rec := c.doJSON("GET", "/api/trackers/"+tracker.ID+"/sharing", "", &sharing); rec.Code != http.StatusOK {
		t.Fatalf("sharing overview: %d %s", rec.Code, rec.Body.String())
	}
	if len(sharing.Invitations) != 1 {
		t.Fatalf("expected one invitation, got %+v", sharing.Invitations)
	}
	invite := sharing.Invitations[0]
	token := c.invitationToken()

	// A member has no say over who else may join.
	stranger := &testClient{t: t, router: c.router}
	stranger.cookie = stranger.session(stranger.do("POST", "/api/auth/register",
		`{"email":"stranger@example.com","password":"supersecret","name":"Stranger"}`))
	if rec := stranger.do("DELETE", "/api/trackers/"+tracker.ID+"/invitations/"+invite.ID, ""); rec.Code != http.StatusForbidden && rec.Code != http.StatusNotFound {
		t.Fatalf("cancel by a stranger: expected 403 or 404, got %d %s", rec.Code, rec.Body.String())
	}

	var cancelled invitationPayload
	if rec := c.doJSON("DELETE", "/api/trackers/"+tracker.ID+"/invitations/"+invite.ID, "", &cancelled); rec.Code != http.StatusOK {
		t.Fatalf("cancel invitation: %d %s", rec.Code, rec.Body.String())
	}
	if cancelled.Status != model.InviteRemoved {
		t.Fatalf("a cancelled invitation is retired, got %q", cancelled.Status)
	}

	// The row stays as history, so the owner can see the address was withdrawn.
	var after sharingPayload
	if rec := c.doJSON("GET", "/api/trackers/"+tracker.ID+"/sharing", "", &after); rec.Code != http.StatusOK {
		t.Fatalf("sharing after cancel: %d %s", rec.Code, rec.Body.String())
	}
	if len(after.Invitations) != 1 || after.Invitations[0].Status != model.InviteRemoved {
		t.Fatalf("expected the withdrawn invitation to remain as history, got %+v", after.Invitations)
	}

	// Nobody is shared with anymore, so the tracker list keeps quiet about it.
	var listed struct {
		Trackers []struct {
			SharedWith []string `json:"sharedWith"`
		} `json:"trackers"`
	}
	if rec := c.doJSON("GET", "/api/trackers", "", &listed); rec.Code != http.StatusOK {
		t.Fatalf("list trackers: %d %s", rec.Code, rec.Body.String())
	}
	if len(listed.Trackers) != 1 || len(listed.Trackers[0].SharedWith) != 0 {
		t.Fatalf("a withdrawn invitation must not survive on the list, got %+v", listed.Trackers)
	}

	// The link that was mailed is dead, and so is a second cancel.
	if rec := c.do("GET", "/api/invitations/"+token, ""); rec.Code != http.StatusConflict {
		t.Fatalf("withdrawn invitation link: expected 409, got %d %s", rec.Code, rec.Body.String())
	}
	if rec := c.do("DELETE", "/api/trackers/"+tracker.ID+"/invitations/"+invite.ID, ""); rec.Code != http.StatusConflict {
		t.Fatalf("cancelling twice: expected 409, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestResendRefreshesLapsedInvitation(t *testing.T) {
	c := newTestClient(t)
	c.cookie = c.session(c.do("POST", "/api/auth/register",
		`{"email":"owner@example.com","password":"supersecret","name":"Owner"}`))

	var tracker struct {
		ID string `json:"id"`
	}
	if rec := c.doJSON("POST", "/api/trackers", `{"name":"Run","kind":"boolean","shareWith":["friend@example.com"]}`, &tracker); rec.Code != http.StatusCreated {
		t.Fatalf("create tracker: %d %s", rec.Code, rec.Body.String())
	}

	type inviteWithExpiry struct {
		ID        string    `json:"id"`
		Status    string    `json:"status"`
		ExpiresAt time.Time `json:"expiresAt"`
	}
	var sharing struct {
		Invitations []inviteWithExpiry `json:"invitations"`
	}
	if rec := c.doJSON("GET", "/api/trackers/"+tracker.ID+"/sharing", "", &sharing); rec.Code != http.StatusOK {
		t.Fatalf("sharing overview: %d %s", rec.Code, rec.Body.String())
	}
	if len(sharing.Invitations) != 1 {
		t.Fatalf("expected one invitation, got %+v", sharing.Invitations)
	}
	original := sharing.Invitations[0]

	if rec := c.do("POST", "/api/trackers/"+tracker.ID+"/invitations/"+original.ID+"/resend", ""); rec.Code != http.StatusOK {
		t.Fatalf("resend: %d %s", rec.Code, rec.Body.String())
	}
	if rec := c.doJSON("GET", "/api/trackers/"+tracker.ID+"/sharing", "", &sharing); rec.Code != http.StatusOK {
		t.Fatalf("sharing after resend: %d %s", rec.Code, rec.Body.String())
	}
	resent := sharing.Invitations[0]
	if resent.ExpiresAt.Before(original.ExpiresAt) {
		t.Fatalf("resend must not shorten the expiry: before=%v after=%v", original.ExpiresAt, resent.ExpiresAt)
	}

	// The newest mail carries a link that is still accepted.
	var info struct {
		Status string `json:"status"`
	}
	if rec := c.doJSON("GET", "/api/invitations/"+c.invitationToken(), "", &info); rec.Code != http.StatusOK {
		t.Fatalf("re-emailed link should be usable: %d", rec.Code)
	}
}
