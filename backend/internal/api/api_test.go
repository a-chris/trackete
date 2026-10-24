package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/a-chris/trackete/backend/internal/config"
	"github.com/a-chris/trackete/backend/internal/store"
)

type testClient struct {
	t        *testing.T
	router   http.Handler
	cookie   *http.Cookie
	st       *store.Store
	mailPath string
}

func newTestClient(t *testing.T) *testClient {
	t.Helper()
	return newTestClientWithConfig(t, config.Config{
		AllowRegistration: true,
		Version:           "test",
	})
}

// newTestClientWithConfig wires the router like production does: a real store in
// a temp dir and invitation mail written to a temp outbox the tests can read.
func newTestClientWithConfig(t *testing.T, cfg config.Config) *testClient {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	if cfg.JWTSecret == "" {
		cfg.JWTSecret = "test-secret"
	}
	if cfg.TokenTTL == 0 {
		cfg.TokenTTL = time.Hour
	}
	if cfg.AppURL == "" {
		cfg.AppURL = "http://trackete.test"
	}
	mailPath := filepath.Join(t.TempDir(), "outbox.log")
	cfg.MailTransport = "file"
	cfg.MailFile = mailPath

	router := NewRouter(cfg, st)
	return &testClient{t: t, router: router, st: st, mailPath: mailPath}
}

func (c *testClient) do(method, path, body string) *httptest.ResponseRecorder {
	c.t.Helper()
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if c.cookie != nil {
		req.AddCookie(c.cookie)
	}
	rec := httptest.NewRecorder()
	c.router.ServeHTTP(rec, req)
	return rec
}

func (c *testClient) doJSON(method, path, body string, out any) *httptest.ResponseRecorder {
	c.t.Helper()
	rec := c.do(method, path, body)
	if out != nil && rec.Body.Len() > 0 && rec.Code < 400 {
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			c.t.Fatalf("%s %s: decode response: %v (%s)", method, path, err, rec.Body.String())
		}
	}
	return rec
}

func TestAuthenticationAndTrackingFlow(t *testing.T) {
	c := newTestClient(t)

	rec := c.do("POST", "/api/auth/register", `{"email":"alice@example.com","password":"supersecret","name":"Alice"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("register: expected a session cookie")
	}
	c.cookie = cookies[0]

	var me struct {
		User struct {
			Email string `json:"email"`
		} `json:"user"`
	}
	if rec := c.doJSON("GET", "/api/auth/me", "", &me); rec.Code != http.StatusOK || me.User.Email != "alice@example.com" {
		t.Fatalf("me: unexpected response %d %+v", rec.Code, me)
	}

	if rec := c.do("GET", "/api/trackers", ""); rec.Code != http.StatusOK {
		t.Fatalf("list trackers: expected 200, got %d", rec.Code)
	}

	var tracker struct {
		ID   string `json:"id"`
		Kind string `json:"kind"`
	}
	rec = c.doJSON("POST", "/api/trackers", `{"name":"Beer","kind":"numeric","unit":"beers","icon":"🍺","color":"#f59e0b","goal":{"type":"at_most","value":5,"period":"week"}}`, &tracker)
	if rec.Code != http.StatusCreated || tracker.ID == "" {
		t.Fatalf("create tracker: %d %s", rec.Code, rec.Body.String())
	}

	// One value per day: logging twice overwrites.
	today := time.Now().UTC().Format("2006-01-02")
	if rec := c.do("PUT", "/api/trackers/"+tracker.ID+"/entries/"+today, `{"value":2}`); rec.Code != http.StatusOK {
		t.Fatalf("upsert entry: %d %s", rec.Code, rec.Body.String())
	}
	if rec := c.do("PUT", "/api/trackers/"+tracker.ID+"/entries/"+today, `{"value":3}`); rec.Code != http.StatusOK {
		t.Fatalf("upsert entry (2): %d %s", rec.Code, rec.Body.String())
	}

	var entries struct {
		Entries []struct {
			Value float64 `json:"value"`
		} `json:"entries"`
	}
	if rec := c.doJSON("GET", "/api/trackers/"+tracker.ID+"/entries?range=7d", "", &entries); rec.Code != http.StatusOK {
		t.Fatalf("list entries: %d", rec.Code)
	}
	if len(entries.Entries) != 1 || entries.Entries[0].Value != 3 {
		t.Fatalf("expected one entry with value 3, got %+v", entries.Entries)
	}

	var overview struct {
		Items []struct {
			PeriodTotal float64 `json:"periodTotal"`
			Streak      int     `json:"currentStreak"`
		} `json:"items"`
	}
	if rec := c.doJSON("GET", "/api/overview?today="+today, "", &overview); rec.Code != http.StatusOK {
		t.Fatalf("overview: %d %s", rec.Code, rec.Body.String())
	}
	if len(overview.Items) != 1 {
		t.Fatalf("overview: expected 1 item, got %d", len(overview.Items))
	}
	if overview.Items[0].PeriodTotal != 3 {
		t.Fatalf("overview: expected week total 3, got %v", overview.Items[0].PeriodTotal)
	}

	var dashboards struct {
		Dashboards []struct {
			ID string `json:"id"`
		} `json:"dashboards"`
	}
	if rec := c.doJSON("GET", "/api/dashboards", "", &dashboards); rec.Code != http.StatusOK {
		t.Fatalf("dashboards: %d", rec.Code)
	}
	if len(dashboards.Dashboards) != 1 {
		t.Fatalf("expected one default dashboard, got %d", len(dashboards.Dashboards))
	}
	dashboardID := dashboards.Dashboards[0].ID

	var widget struct {
		ID string `json:"id"`
	}
	rec = c.doJSON("POST", "/api/dashboards/"+dashboardID+"/widgets",
		`{"trackerId":"`+tracker.ID+`","title":"Beers","chartType":"line","config":{"range":"30d"}}`, &widget)
	if rec.Code != http.StatusCreated || widget.ID == "" {
		t.Fatalf("create widget: %d %s", rec.Code, rec.Body.String())
	}

	var data struct {
		Widgets map[string]struct {
			ChartType string   `json:"chartType"`
			Labels    []string `json:"labels"`
			Summary   struct {
				Total float64 `json:"total"`
			} `json:"summary"`
		} `json:"widgets"`
	}
	rec = c.doJSON("GET", "/api/dashboards/"+dashboardID+"/data?to="+today, "", &data)
	if rec.Code != http.StatusOK {
		t.Fatalf("dashboard data: %d %s", rec.Code, rec.Body.String())
	}
	chart, ok := data.Widgets[widget.ID]
	if !ok {
		t.Fatalf("dashboard data: missing widget %s", widget.ID)
	}
	if chart.Summary.Total != 3 {
		t.Fatalf("dashboard data: expected total 3, got %v", chart.Summary.Total)
	}

	var secondWidget struct {
		ID     string `json:"id"`
		Layout struct {
			X int `json:"x"`
			Y int `json:"y"`
			W int `json:"w"`
			H int `json:"h"`
		} `json:"layout"`
	}
	rec = c.doJSON("POST", "/api/dashboards/"+dashboardID+"/widgets",
		`{"trackerId":"`+tracker.ID+`","title":"Second","chartType":"bar","config":{"range":"7d"}}`, &secondWidget)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create second widget: %d %s", rec.Code, rec.Body.String())
	}
	if secondWidget.Layout.X == 0 && secondWidget.Layout.Y == 0 {
		t.Fatalf("second widget should be auto-placed, got %+v", secondWidget.Layout)
	}

	// Layout updates must persist.
	rec = c.do("PUT", "/api/dashboards/"+dashboardID+"/layout",
		`{"items":[{"id":"`+widget.ID+`","x":1,"y":2,"w":6,"h":5}]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("update layout: %d %s", rec.Code, rec.Body.String())
	}

	// A second user must not see the first user's tracker.
	other := &testClient{t: t, router: c.router}
	rec = other.do("POST", "/api/auth/register", `{"email":"bob@example.com","password":"supersecret","name":"Bob"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register second user: %d %s", rec.Code, rec.Body.String())
	}
	other.cookie = rec.Result().Cookies()[0]
	if rec := other.do("GET", "/api/trackers/"+tracker.ID, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("cross-user access: expected 404, got %d", rec.Code)
	}
}

func TestRegistrationCanBeDisabled(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	router := NewRouter(config.Config{
		JWTSecret:         "test-secret",
		TokenTTL:          time.Hour,
		AllowRegistration: false,
	}, st)

	req := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(`{"email":"a@b.co","password":"supersecret"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when registration is disabled, got %d", rec.Code)
	}
}

func (c *testClient) session(rec *httptest.ResponseRecorder) *http.Cookie {
	c.t.Helper()
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		c.t.Fatalf("expected a session cookie, got none")
	}
	return cookies[0]
}

// The SPA sends ?today=..., which today() used to ignore in favour of the
// server clock, so the "Today" screen showed the wrong day.
func TestOverviewHonoursTodayParam(t *testing.T) {
	c := newTestClient(t)
	c.cookie = c.session(c.do("POST", "/api/auth/register",
		`{"email":"today@example.com","password":"supersecret","name":"Today"}`))

	var tracker struct {
		ID string `json:"id"`
	}
	rec := c.doJSON("POST", "/api/trackers",
		`{"name":"Water","kind":"numeric","unit":"glasses","icon":"💧","color":"#0ea5e9"}`, &tracker)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create tracker: %d %s", rec.Code, rec.Body.String())
	}

	// 2026-03-05 is a Thursday: its week starts 2026-03-02. The extra 2026-03-20
	// entry is in the same month but a later week, so a scope bug shows up.
	for day, value := range map[string]float64{
		"2026-03-02": 2,
		"2026-03-05": 3,
		"2026-03-20": 9,
	} {
		body := `{"value":` + strconv.FormatFloat(value, 'f', -1, 64) + `}`
		if rec := c.do("PUT", "/api/trackers/"+tracker.ID+"/entries/"+day, body); rec.Code != http.StatusOK {
			t.Fatalf("upsert entry %s: %d %s", day, rec.Code, rec.Body.String())
		}
	}

	var overview struct {
		Today     string `json:"today"`
		WeekStart string `json:"weekStart"`
		Items     []struct {
			Entry *struct {
				Day   string  `json:"day"`
				Value float64 `json:"value"`
			} `json:"entry"`
			PeriodTotal  float64 `json:"periodTotal"`
			PeriodLogged int     `json:"periodLogged"`
		} `json:"items"`
	}
	rec = c.doJSON("GET", "/api/overview?today=2026-03-05", "", &overview)
	if rec.Code != http.StatusOK {
		t.Fatalf("overview: %d %s", rec.Code, rec.Body.String())
	}
	if overview.Today != "2026-03-05" {
		t.Fatalf("overview: expected today 2026-03-05, got %q (server date used?)", overview.Today)
	}
	if overview.WeekStart != "2026-03-02" {
		t.Fatalf("overview: expected weekStart 2026-03-02, got %q", overview.WeekStart)
	}
	if len(overview.Items) != 1 {
		t.Fatalf("overview: expected 1 item, got %d", len(overview.Items))
	}
	item := overview.Items[0]
	if item.Entry == nil || item.Entry.Day != "2026-03-05" || item.Entry.Value != 3 {
		t.Fatalf("overview: expected todayEntry 2026-03-05/3, got %+v", item.Entry)
	}
	if item.PeriodTotal != 5 || item.PeriodLogged != 2 {
		t.Fatalf("overview: expected week total 5 over 2 days, got %v over %d", item.PeriodTotal, item.PeriodLogged)
	}
}

func TestPasswordChangeInvalidatesOldSession(t *testing.T) {
	c := newTestClient(t)
	c.cookie = c.session(c.do("POST", "/api/auth/register",
		`{"email":"rotate@example.com","password":"supersecret","name":"Rotate"}`))
	oldCookie := c.cookie

	rec := c.do("POST", "/api/auth/password",
		`{"currentPassword":"supersecret","newPassword":"an-even-better-secret"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("change password: %d %s", rec.Code, rec.Body.String())
	}
	freshCookie := c.session(rec)

	stale := &testClient{t: t, router: c.router, cookie: oldCookie}
	if rec := stale.do("GET", "/api/auth/me", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("old cookie after password change: expected 401, got %d %s", rec.Code, rec.Body.String())
	}

	fresh := &testClient{t: t, router: c.router, cookie: freshCookie}
	if rec := fresh.do("GET", "/api/auth/me", ""); rec.Code != http.StatusOK {
		t.Fatalf("cookie issued by the password change: expected 200, got %d %s", rec.Code, rec.Body.String())
	}

	// Logging in with the new password works and yields a usable session.
	rec = c.do("POST", "/api/auth/login",
		`{"email":"rotate@example.com","password":"an-even-better-secret"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("login with new password: %d %s", rec.Code, rec.Body.String())
	}
	fresh.cookie = c.session(rec)
	if rec := fresh.do("GET", "/api/auth/me", ""); rec.Code != http.StatusOK {
		t.Fatalf("new login session: expected 200, got %d %s", rec.Code, rec.Body.String())
	}
}

// bcrypt rejects inputs above 72 bytes, which used to surface as a 500.
func TestPasswordLongerThanBcryptLimitIsRejected(t *testing.T) {
	c := newTestClient(t)
	long := strings.Repeat("a", 100)

	rec := c.do("POST", "/api/auth/register",
		`{"email":"long@example.com","password":"`+long+`","name":"Long"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("register with 100-byte password: expected 400, got %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "72") {
		t.Fatalf("register error should mention the byte limit, got %s", rec.Body.String())
	}

	c.cookie = c.session(c.do("POST", "/api/auth/register",
		`{"email":"long@example.com","password":"supersecret","name":"Long"}`))
	rec = c.do("POST", "/api/auth/password",
		`{"currentPassword":"supersecret","newPassword":"`+long+`"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("password change with 100-byte password: expected 400, got %d %s", rec.Code, rec.Body.String())
	}
}

// Every user keeps exactly one default dashboard across create and delete.
func TestDashboardDefaultInvariant(t *testing.T) {
	c := newTestClient(t)
	c.cookie = c.session(c.do("POST", "/api/auth/register",
		`{"email":"dash@example.com","password":"supersecret","name":"Dash"}`))

	type dashboard struct {
		ID        string `json:"id"`
		IsDefault bool   `json:"isDefault"`
	}
	list := func() []dashboard {
		t.Helper()
		var out struct {
			Dashboards []dashboard `json:"dashboards"`
		}
		if rec := c.doJSON("GET", "/api/dashboards", "", &out); rec.Code != http.StatusOK {
			t.Fatalf("list dashboards: %d %s", rec.Code, rec.Body.String())
		}
		return out.Dashboards
	}

	dashboards := list()
	if len(dashboards) != 1 || !dashboards[0].IsDefault {
		t.Fatalf("expected one default dashboard after registration, got %+v", dashboards)
	}
	defaultID := dashboards[0].ID

	var second dashboard
	if rec := c.doJSON("POST", "/api/dashboards", `{"name":"Second"}`, &second); rec.Code != http.StatusCreated {
		t.Fatalf("create dashboard: %d %s", rec.Code, rec.Body.String())
	}
	if second.IsDefault {
		t.Fatalf("second dashboard must not become the default")
	}

	if rec := c.do("DELETE", "/api/dashboards/"+defaultID, ""); rec.Code != http.StatusOK {
		t.Fatalf("delete default dashboard: %d %s", rec.Code, rec.Body.String())
	}
	dashboards = list()
	if len(dashboards) != 1 || dashboards[0].ID != second.ID || !dashboards[0].IsDefault {
		t.Fatalf("expected the remaining dashboard promoted to default, got %+v", dashboards)
	}

	// Deleting the last dashboard leaves the user with none; the next list
	// recreates a default one.
	if rec := c.do("DELETE", "/api/dashboards/"+second.ID, ""); rec.Code != http.StatusOK {
		t.Fatalf("delete last dashboard: %d %s", rec.Code, rec.Body.String())
	}
	dashboards = list()
	if len(dashboards) != 1 || !dashboards[0].IsDefault {
		t.Fatalf("expected a recreated default dashboard, got %+v", dashboards)
	}
}

// A streak longer than the visible range must survive: the chart covers 30 days,
// but the summary streak is computed over the tracker's history.
func TestTrackerDataStreakOutlivesVisibleRange(t *testing.T) {
	c := newTestClient(t)
	c.cookie = c.session(c.do("POST", "/api/auth/register",
		`{"email":"streak@example.com","password":"supersecret","name":"Streak"}`))

	var tracker struct {
		ID string `json:"id"`
	}
	if rec := c.doJSON("POST", "/api/trackers",
		`{"name":"Run","kind":"numeric","unit":"km","icon":"🏃","color":"#22c55e"}`, &tracker); rec.Code != http.StatusCreated {
		t.Fatalf("create tracker: %d %s", rec.Code, rec.Body.String())
	}

	const days = 40
	to := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	for i := 0; i < days; i++ {
		day := to.AddDate(0, 0, -i).Format("2006-01-02")
		if rec := c.do("PUT", "/api/trackers/"+tracker.ID+"/entries/"+day, `{"value":5}`); rec.Code != http.StatusOK {
			t.Fatalf("upsert entry %s: %d %s", day, rec.Code, rec.Body.String())
		}
	}

	var data struct {
		Labels  []string `json:"labels"`
		Summary struct {
			CurrentStreak int `json:"currentStreak"`
			BestStreak    int `json:"bestStreak"`
		} `json:"summary"`
	}
	rec := c.doJSON("GET", "/api/trackers/"+tracker.ID+"/data?to=2026-06-30&range=30d&chart=stat", "", &data)
	if rec.Code != http.StatusOK {
		t.Fatalf("tracker data: %d %s", rec.Code, rec.Body.String())
	}
	if data.Summary.CurrentStreak != days {
		t.Fatalf("expected current streak %d across history, got %d", days, data.Summary.CurrentStreak)
	}
	if data.Summary.BestStreak != 30 {
		t.Fatalf("expected best streak 30 within the 30-day range, got %d", data.Summary.BestStreak)
	}
	if len(data.Labels) != 30 {
		t.Fatalf("expected 30 buckets, got %d", len(data.Labels))
	}
}
