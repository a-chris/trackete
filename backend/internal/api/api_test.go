package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/a-chris/trackete/backend/internal/config"
	"github.com/a-chris/trackete/backend/internal/store"
)

type testClient struct {
	t      *testing.T
	router http.Handler
	cookie *http.Cookie
}

func newTestClient(t *testing.T) *testClient {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	router := NewRouter(config.Config{
		JWTSecret:         "test-secret",
		TokenTTL:          time.Hour,
		AllowRegistration: true,
		Version:           "test",
	}, st)

	return &testClient{t: t, router: router}
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
