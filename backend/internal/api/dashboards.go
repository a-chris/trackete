package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"github.com/a-chris/trackete/backend/internal/analytics"
	"github.com/a-chris/trackete/backend/internal/model"
)

type dashboardRequest struct {
	Name string `json:"name"`
}

type widgetRequest struct {
	TrackerID string             `json:"trackerId"`
	Title     string             `json:"title"`
	ChartType string             `json:"chartType"`
	Config    model.WidgetConfig `json:"config"`
	Layout    model.LayoutItem   `json:"layout"`
}

type layoutRequest struct {
	Items []model.LayoutItem `json:"items"`
}

var validRanges = map[string]bool{"7d": true, "30d": true, "90d": true, "365d": true, "all": true}

func normalizeWidgetConfig(cfg model.WidgetConfig) model.WidgetConfig {
	def := model.DefaultWidgetConfig()
	out := cfg
	if out.Range == "" {
		out.Range = def.Range
	}
	if !validRanges[out.Range] {
		out.Range = def.Range
	}
	switch out.Granularity {
	case "", "auto", model.PeriodDay, model.PeriodWeek, model.PeriodMonth:
	default:
		out.Granularity = "auto"
	}
	if out.Granularity == "" {
		out.Granularity = "auto"
	}
	if out.Aggregate != "avg" {
		out.Aggregate = "sum"
	}
	switch out.Metric {
	case "average", "count", "streak":
	default:
		out.Metric = "total"
	}
	return out
}

func clampLayout(l model.LayoutItem, chartType string) model.LayoutItem {
	def := defaultLayout(chartType)
	if l.W <= 0 {
		l.W = def.W
	}
	if l.H <= 0 {
		l.H = def.H
	}
	if l.W < 2 {
		l.W = 2
	}
	if l.W > 12 {
		l.W = 12
	}
	if l.H < 2 {
		l.H = 2
	}
	if l.H > 12 {
		l.H = 12
	}
	if l.X < 0 {
		l.X = 0
	}
	if l.X > 12-l.W {
		l.X = 12 - l.W
	}
	if l.Y < 0 {
		l.Y = 0
	}
	return l
}

func defaultLayout(chartType string) model.LayoutItem {
	switch chartType {
	case model.ChartStat:
		return model.LayoutItem{W: 3, H: 3}
	case model.ChartHeatmap:
		return model.LayoutItem{W: 12, H: 5}
	case model.ChartDonut:
		return model.LayoutItem{W: 4, H: 5}
	default:
		return model.LayoutItem{W: 6, H: 5}
	}
}

func (a *API) listDashboards(c *gin.Context) {
	dashboards, err := a.store.ListDashboards(c.Request.Context(), userID(c))
	if err != nil {
		failStore(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"dashboards": dashboards})
}

func (a *API) createDashboard(c *gin.Context) {
	var req dashboardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid request body")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "New dashboard"
	}
	if utf8.RuneCountInString(name) > 60 {
		fail(c, http.StatusBadRequest, "name must be 60 characters or fewer")
		return
	}
	dashboard, err := a.store.CreateDashboard(c.Request.Context(), userID(c), name)
	if err != nil {
		failStore(c, err)
		return
	}
	c.JSON(http.StatusCreated, dashboard)
}

func (a *API) getDashboard(c *gin.Context) {
	dashboard, err := a.store.GetDashboard(c.Request.Context(), userID(c), c.Param("id"))
	if err != nil {
		failStore(c, err)
		return
	}
	c.JSON(http.StatusOK, dashboard)
}

func (a *API) renameDashboard(c *gin.Context) {
	var req dashboardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid request body")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" || utf8.RuneCountInString(name) > 60 {
		fail(c, http.StatusBadRequest, "name must be between 1 and 60 characters")
		return
	}
	dashboard, err := a.store.RenameDashboard(c.Request.Context(), userID(c), c.Param("id"), name)
	if err != nil {
		failStore(c, err)
		return
	}
	c.JSON(http.StatusOK, dashboard)
}

func (a *API) deleteDashboard(c *gin.Context) {
	if err := a.store.DeleteDashboard(c.Request.Context(), userID(c), c.Param("id")); err != nil {
		failStore(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (a *API) buildWidget(c *gin.Context, req widgetRequest) (*model.Widget, error) {
	title := strings.TrimSpace(req.Title)
	if utf8.RuneCountInString(title) > 100 {
		return nil, fmt.Errorf("title must be 100 characters or fewer")
	}
	if !model.ValidChartType(req.ChartType) {
		return nil, fmt.Errorf("unknown chart type")
	}
	if req.TrackerID == "" {
		return nil, fmt.Errorf("a tracker is required")
	}
	if _, err := a.store.GetTracker(c.Request.Context(), userID(c), req.TrackerID); err != nil {
		return nil, fmt.Errorf("tracker not found")
	}
	return &model.Widget{
		TrackerID: req.TrackerID,
		Title:     title,
		ChartType: req.ChartType,
		Config:    normalizeWidgetConfig(req.Config),
		Layout:    clampLayout(req.Layout, req.ChartType),
	}, nil
}

func (a *API) createWidget(c *gin.Context) {
	var req widgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid request body")
		return
	}
	widget, err := a.buildWidget(c, req)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	widget.DashboardID = c.Param("id")
	if req.Layout.W <= 0 || req.Layout.H <= 0 {
		widget.Layout.X, widget.Layout.Y = a.autoPlace(c, widget.DashboardID, widget.Layout.W, widget.Layout.H)
	}
	created, err := a.store.CreateWidget(c.Request.Context(), userID(c), widget)
	if err != nil {
		failStore(c, err)
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (a *API) updateWidget(c *gin.Context) {
	var req widgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid request body")
		return
	}
	widget, err := a.buildWidget(c, req)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	widget.ID = c.Param("id")
	updated, err := a.store.UpdateWidget(c.Request.Context(), userID(c), widget)
	if err != nil {
		failStore(c, err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (a *API) deleteWidget(c *gin.Context) {
	if err := a.store.DeleteWidget(c.Request.Context(), userID(c), c.Param("id")); err != nil {
		failStore(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (a *API) updateLayouts(c *gin.Context) {
	var req layoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "invalid request body")
		return
	}
	for i := range req.Items {
		req.Items[i] = clampLayout(req.Items[i], model.ChartLine)
	}
	if err := a.store.UpdateLayouts(c.Request.Context(), userID(c), c.Param("id"), req.Items); err != nil {
		failStore(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// autoPlace finds the first free slot in the 12-column grid so new charts do
// not all stack in the top-left corner.
func (a *API) autoPlace(c *gin.Context, dashboardID string, w, h int) (int, int) {
	dashboard, err := a.store.GetDashboard(c.Request.Context(), userID(c), dashboardID)
	if err != nil {
		return 0, 0
	}
	for y := 0; y < 200; y++ {
		for x := 0; x+w <= 12; x++ {
			if !collides(x, y, w, h, dashboard.Widgets) {
				return x, y
			}
		}
	}
	return 0, 0
}

func collides(x, y, w, h int, widgets []model.Widget) bool {
	for _, other := range widgets {
		if x < other.Layout.X+other.Layout.W && x+w > other.Layout.X &&
			y < other.Layout.Y+other.Layout.H && y+h > other.Layout.Y {
			return true
		}
	}
	return false
}

func (a *API) dashboardData(c *gin.Context) {
	uid := userID(c)
	dashboard, err := a.store.GetDashboard(c.Request.Context(), uid, c.Param("id"))
	if err != nil {
		failStore(c, err)
		return
	}

	to := today(c)
	toKey := to.Format(analytics.DayLayout)
	out := make(map[string]model.ChartData, len(dashboard.Widgets))

	for _, widget := range dashboard.Widgets {
		tracker, err := a.store.GetTracker(c.Request.Context(), uid, widget.TrackerID)
		if err != nil {
			continue
		}

		from := analytics.RangeStart(to, widget.Config.Range)
		if widget.Config.Range == "all" {
			earliest, err := a.store.EarliestEntryDay(c.Request.Context(), uid, tracker.ID)
			if err == nil && earliest != "" {
				if d, perr := time.Parse(analytics.DayLayout, earliest); perr == nil {
					from = analytics.DateOnly(d)
				}
			} else {
				from = analytics.RangeStart(to, "365d")
			}
		}

		entries, err := a.store.ListEntries(c.Request.Context(), tracker.ID, from.Format(analytics.DayLayout), toKey)
		if err != nil {
			failStore(c, err)
			return
		}

		out[widget.ID] = analytics.Build(toPoints(tracker.ID, entries), analytics.Options{
			Name:        tracker.Name,
			Kind:        tracker.Kind,
			Unit:        tracker.Unit,
			Goal:        tracker.Goal,
			ChartType:   widget.ChartType,
			From:        from,
			To:          to,
			Granularity: widget.Config.Granularity,
			Aggregate:   widget.Config.Aggregate,
			Metric:      widget.Config.Metric,
			Today:       to,
		})
	}

	c.JSON(http.StatusOK, gin.H{"to": toKey, "widgets": out})
}
