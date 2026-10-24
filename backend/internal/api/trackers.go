package api

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"github.com/a-chris/trackete/backend/internal/analytics"
	"github.com/a-chris/trackete/backend/internal/model"
)

var colorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

type trackerRequest struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Icon        string      `json:"icon"`
	Color       string      `json:"color"`
	Kind        string      `json:"kind"`
	Unit        string      `json:"unit"`
	Goal        *model.Goal `json:"goal"`
	Archived    bool        `json:"archived"`
	// ShareWith invites these addresses to the tracker. It is optional: an
	// omitted or empty list leaves the request and its response untouched.
	ShareWith []string `json:"shareWith"`
}

// maxInvitesPerRequest bounds how many emails one tracker create/update may
// invite, so a single request cannot spray invitations.
const maxInvitesPerRequest = 20

// trackerResponse is the tracker payload plus the invitation outcome. The
// embedded tracker keeps the existing JSON shape; invitations only appear when
// the request asked to share.
type trackerResponse struct {
	model.Tracker
	Invitations []model.Invitation `json:"invitations,omitempty"`
}

type entryRequest struct {
	Value float64 `json:"value"`
	Note  string  `json:"note"`
}

func (r trackerRequest) normalize() (*model.Tracker, error) {
	name := strings.TrimSpace(r.Name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if utf8.RuneCountInString(name) > 80 {
		return nil, fmt.Errorf("name must be 80 characters or fewer")
	}
	description := strings.TrimSpace(r.Description)
	if utf8.RuneCountInString(description) > 500 {
		return nil, fmt.Errorf("description must be 500 characters or fewer")
	}

	kind := strings.TrimSpace(r.Kind)
	if kind != model.KindBoolean && kind != model.KindNumeric {
		return nil, fmt.Errorf("kind must be boolean or numeric")
	}

	unit := strings.TrimSpace(r.Unit)
	if utf8.RuneCountInString(unit) > 16 {
		return nil, fmt.Errorf("unit must be 16 characters or fewer")
	}
	if kind == model.KindBoolean {
		unit = ""
	}

	icon := strings.TrimSpace(r.Icon)
	if utf8.RuneCountInString(icon) > 16 {
		return nil, fmt.Errorf("icon must be short (an emoji works best)")
	}
	if icon == "" {
		icon = "📌"
	}

	color := strings.TrimSpace(r.Color)
	if color == "" {
		color = "#6366f1"
	}
	if !colorPattern.MatchString(color) {
		return nil, fmt.Errorf("color must be a hex value like #6366f1")
	}

	var goal *model.Goal
	if r.Goal != nil {
		if r.Goal.Type != model.GoalAtLeast && r.Goal.Type != model.GoalAtMost {
			return nil, fmt.Errorf("goal type must be at_least or at_most")
		}
		switch r.Goal.Period {
		case model.PeriodDay, model.PeriodWeek, model.PeriodMonth:
		default:
			return nil, fmt.Errorf("goal period must be day, week or month")
		}
		if r.Goal.Value < 0 || math.IsNaN(r.Goal.Value) || math.IsInf(r.Goal.Value, 0) || r.Goal.Value > 1_000_000 {
			return nil, fmt.Errorf("goal value must be a positive number")
		}
		goal = &model.Goal{Type: r.Goal.Type, Value: r.Goal.Value, Period: r.Goal.Period}
	}

	return &model.Tracker{
		Name:        name,
		Description: description,
		Icon:        icon,
		Color:       color,
		Kind:        kind,
		Unit:        unit,
		Goal:        goal,
		Archived:    r.Archived,
	}, nil
}

// normalizeShareWith trims, lowercases and de-duplicates invitation addresses.
func (r trackerRequest) normalizeShareWith() ([]string, error) {
	if len(r.ShareWith) > maxInvitesPerRequest {
		return nil, fmt.Errorf("share with at most %d addresses at a time", maxInvitesPerRequest)
	}
	seen := make(map[string]bool, len(r.ShareWith))
	out := make([]string, 0, len(r.ShareWith))
	for _, raw := range r.ShareWith {
		email := strings.ToLower(strings.TrimSpace(raw))
		if email == "" {
			continue
		}
		if !validEmail(email) {
			return nil, fmt.Errorf("%q is not a valid email address", raw)
		}
		if seen[email] {
			continue
		}
		seen[email] = true
		out = append(out, email)
	}
	return out, nil
}

// shareTracker invites every address and reports the ones it could record. A
// failed invitation is logged, never surfaced as a failure of the tracker
// itself: the tracker already exists and the owner can retry from its page.
func (a *API) shareTracker(c *gin.Context, tracker *model.Tracker, emails []string) []model.Invitation {
	if len(emails) == 0 {
		return nil
	}
	out := make([]model.Invitation, 0, len(emails))
	for _, email := range emails {
		inv, err := a.inviteByEmail(c.Request.Context(), userID(c), tracker, email)
		if err != nil {
			log.Printf("share tracker %s with %s: %v", tracker.ID, email, err)
			continue
		}
		out = append(out, *inv)
	}
	return out
}

func (a *API) listTrackers(c *gin.Context) {
	includeArchived := c.Query("includeArchived") == "true" || c.Query("includeArchived") == "1"
	trackers, err := a.store.ListTrackers(c.Request.Context(), userID(c), includeArchived)
	if err != nil {
		failStore(c, err)
		return
	}
	ids := make([]string, len(trackers))
	for i, tracker := range trackers {
		ids[i] = tracker.ID
	}
	counts, err := a.store.OpenInvitationCounts(c.Request.Context(), ids)
	if err != nil {
		failStore(c, err)
		return
	}
	for i := range trackers {
		trackers[i].OpenInvitations = counts[trackers[i].ID]
	}
	c.JSON(http.StatusOK, gin.H{"trackers": trackers})
}

func (a *API) createTracker(c *gin.Context) {
	var req trackerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid request body")
		return
	}
	tracker, err := req.normalize()
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	shareWith, err := req.normalizeShareWith()
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	created, err := a.store.CreateTracker(c.Request.Context(), userID(c), tracker)
	if err != nil {
		failStore(c, err)
		return
	}
	c.JSON(http.StatusCreated, trackerResponse{
		Tracker:     *created,
		Invitations: a.shareTracker(c, created, shareWith),
	})
}

func (a *API) getTracker(c *gin.Context) {
	tracker, err := a.store.GetTracker(c.Request.Context(), userID(c), c.Param("id"))
	if err != nil {
		failStore(c, err)
		return
	}
	c.JSON(http.StatusOK, tracker)
}

func (a *API) updateTracker(c *gin.Context) {
	var req trackerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid request body")
		return
	}
	tracker, err := req.normalize()
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	shareWith, err := req.normalizeShareWith()
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	tracker.ID = c.Param("id")
	updated, err := a.store.UpdateTracker(c.Request.Context(), userID(c), tracker)
	if err != nil {
		failStore(c, err)
		return
	}
	c.JSON(http.StatusOK, trackerResponse{
		Tracker:     *updated,
		Invitations: a.shareTracker(c, updated, shareWith),
	})
}

func (a *API) deleteTracker(c *gin.Context) {
	if err := a.store.DeleteTracker(c.Request.Context(), userID(c), c.Param("id")); err != nil {
		failStore(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (a *API) listEntries(c *gin.Context) {
	tracker, err := a.store.GetTracker(c.Request.Context(), userID(c), c.Param("id"))
	if err != nil {
		failStore(c, err)
		return
	}
	from, to, err := resolveRange(c)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	entries, err := a.store.ListEntries(c.Request.Context(), tracker.ID, from.Format(analytics.DayLayout), to.Format(analytics.DayLayout))
	if err != nil {
		failStore(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"entries": entries, "from": from.Format(analytics.DayLayout), "to": to.Format(analytics.DayLayout)})
}

func (a *API) upsertEntry(c *gin.Context) {
	tracker, err := a.store.GetTracker(c.Request.Context(), userID(c), c.Param("id"))
	if err != nil {
		failStore(c, err)
		return
	}

	day := c.Param("day")
	if _, err := time.Parse(analytics.DayLayout, day); err != nil {
		fail(c, http.StatusBadRequest, "day must be a date in YYYY-MM-DD format")
		return
	}

	var req entryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if math.IsNaN(req.Value) || math.IsInf(req.Value, 0) || math.Abs(req.Value) > 1e9 {
		fail(c, http.StatusBadRequest, "value must be a finite number")
		return
	}
	note := strings.TrimSpace(req.Note)
	if utf8.RuneCountInString(note) > 500 {
		fail(c, http.StatusBadRequest, "note must be 500 characters or fewer")
		return
	}

	value := req.Value
	if tracker.Kind == model.KindBoolean {
		if value >= 0.5 {
			value = 1
		} else {
			value = 0
		}
	} else if value < 0 {
		value = 0
	}

	entry, err := a.store.UpsertEntry(c.Request.Context(), userID(c), tracker.ID, day, value, note)
	if err != nil {
		failStore(c, err)
		return
	}
	c.JSON(http.StatusOK, entry)
}

func (a *API) deleteEntry(c *gin.Context) {
	tracker, err := a.store.GetTracker(c.Request.Context(), userID(c), c.Param("id"))
	if err != nil {
		failStore(c, err)
		return
	}
	day := c.Param("day")
	if _, err := time.Parse(analytics.DayLayout, day); err != nil {
		fail(c, http.StatusBadRequest, "day must be a date in YYYY-MM-DD format")
		return
	}
	if err := a.store.DeleteEntry(c.Request.Context(), userID(c), tracker.ID, day); err != nil {
		failStore(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (a *API) trackerData(c *gin.Context) {
	tracker, err := a.store.GetTracker(c.Request.Context(), userID(c), c.Param("id"))
	if err != nil {
		failStore(c, err)
		return
	}

	from, to, err := resolveRange(c)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	chartType := c.DefaultQuery("chart", model.ChartLine)
	if !model.ValidChartType(chartType) {
		fail(c, http.StatusBadRequest, "unknown chart type")
		return
	}

	earliest, err := a.store.EarliestEntryDay(c.Request.Context(), tracker.ID)
	if err != nil {
		failStore(c, err)
		return
	}
	historyFrom := streakHistoryFrom(to, earliest)
	fetchFrom := from
	if historyFrom.Before(fetchFrom) {
		fetchFrom = historyFrom
	}

	entries, err := a.store.ListEntries(c.Request.Context(), tracker.ID, fetchFrom.Format(analytics.DayLayout), to.Format(analytics.DayLayout))
	if err != nil {
		failStore(c, err)
		return
	}

	data := analytics.Build(toPoints(tracker.ID, entries), analytics.Options{
		Name:        tracker.Name,
		Kind:        tracker.Kind,
		Unit:        tracker.Unit,
		Goal:        tracker.Goal,
		ChartType:   chartType,
		From:        from,
		To:          to,
		HistoryFrom: historyFrom,
		Granularity: c.DefaultQuery("granularity", "auto"),
		Aggregate:   c.DefaultQuery("aggregate", "sum"),
		Metric:      c.DefaultQuery("metric", "total"),
		Today:       today(c),
	})
	c.JSON(http.StatusOK, data)
}

func toPoints(trackerID string, entries []model.Entry) []model.Entry {
	out := make([]model.Entry, 0, len(entries))
	for _, e := range entries {
		e.TrackerID = trackerID
		out = append(out, e)
	}
	return out
}
