package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/a-chris/trackete/backend/internal/analytics"
	"github.com/a-chris/trackete/backend/internal/model"
)

// today resolves the "as of" day. The SPA sends `today`; `to` and `date` are
// kept for API clients. Values that are not YYYY-MM-DD are ignored.
func today(c *gin.Context) time.Time {
	for _, key := range []string{"today", "to", "date"} {
		if raw := c.Query(key); raw != "" {
			if t, err := time.Parse(analytics.DayLayout, raw); err == nil {
				return analytics.DateOnly(t)
			}
		}
	}
	return analytics.DateOnly(time.Now().UTC())
}

func resolveRange(c *gin.Context) (time.Time, time.Time, error) {
	to := today(c)
	if raw := c.Query("from"); raw != "" {
		t, err := time.Parse(analytics.DayLayout, raw)
		if err != nil {
			return to, to, fmt.Errorf("from must be a date in YYYY-MM-DD format")
		}
		from := analytics.DateOnly(t)
		if from.After(to) {
			from, to = to, from
		}
		return from, to, nil
	}
	from := analytics.RangeStart(to, c.DefaultQuery("range", "30d"))
	return from, to, nil
}

// streakHistoryFrom returns the earliest day loaded for streak computation: the
// tracker's first entry, bounded to AllRangeDays before the chart end. Charts
// keep rendering only [from, to], the extra days exist so a long run is not
// truncated by the selected range.
func streakHistoryFrom(to time.Time, earliest string) time.Time {
	from := to.AddDate(0, 0, -(analytics.AllRangeDays - 1))
	if earliest != "" {
		if d, err := time.Parse(analytics.DayLayout, earliest); err == nil {
			if d = analytics.DateOnly(d); d.After(from) {
				from = d
			}
		}
	}
	return from
}

func (a *API) overview(c *gin.Context) {
	uid := userID(c)
	now := today(c)

	weekStart := startOfWeek(now)
	if raw := c.Query("weekStart"); raw != "" {
		if t, err := time.Parse(analytics.DayLayout, raw); err == nil {
			weekStart = analytics.DateOnly(t)
		}
	}
	monthStart := firstOfMonth(now)
	if raw := c.Query("monthStart"); raw != "" {
		if t, err := time.Parse(analytics.DayLayout, raw); err == nil {
			monthStart = analytics.DateOnly(t)
		}
	}

	from := now.AddDate(0, 0, -400)
	if weekStart.Before(from) {
		from = weekStart
	}
	if monthStart.Before(from) {
		from = monthStart
	}
	// Period totals need the week/month start; streaks need the long history.
	streakFrom := streakHistoryFrom(now, "")
	fetchFrom := from
	if streakFrom.Before(fetchFrom) {
		fetchFrom = streakFrom
	}

	trackers, err := a.store.ListTrackers(c.Request.Context(), uid, false)
	if err != nil {
		failStore(c, err)
		return
	}
	byTracker, err := a.store.EntriesBetween(c.Request.Context(), uid, fetchFrom.Format(analytics.DayLayout), now.Format(analytics.DayLayout))
	if err != nil {
		failStore(c, err)
		return
	}

	todayKey := now.Format(analytics.DayLayout)
	items := make([]model.OverviewItem, 0, len(trackers))

	for _, tracker := range trackers {
		entries := byTracker[tracker.ID]
		item := model.OverviewItem{Tracker: tracker}

		for i := range entries {
			if entries[i].Day == todayKey {
				entry := entries[i]
				item.Entry = &entry
				break
			}
		}

		periodStart := weekStart
		if tracker.Goal != nil {
			switch tracker.Goal.Period {
			case model.PeriodDay:
				periodStart = now
			case model.PeriodMonth:
				periodStart = monthStart
			}
		}
		for _, e := range entries {
			day, err := time.Parse(analytics.DayLayout, e.Day)
			if err != nil {
				continue
			}
			day = analytics.DateOnly(day)
			if day.Before(periodStart) || day.After(now) {
				continue
			}
			if tracker.Kind == model.KindBoolean {
				if e.Value >= 0.5 {
					item.PeriodTotal++
				}
			} else {
				item.PeriodTotal += e.Value
			}
			item.PeriodLogged++
		}

		if tracker.Goal != nil {
			target := tracker.Goal.Value
			item.PeriodTarget = &target
			met := item.PeriodTotal >= target
			if tracker.Goal.Type == model.GoalAtMost {
				met = item.PeriodTotal <= target
			}
			item.GoalMet = &met
		}

		summary := analytics.Build(entries, analytics.Options{
			Name:        tracker.Name,
			Kind:        tracker.Kind,
			Unit:        tracker.Unit,
			Goal:        tracker.Goal,
			ChartType:   model.ChartStat,
			From:        now.AddDate(0, 0, -400),
			To:          now,
			HistoryFrom: streakFrom,
			Today:       now,
		}).Summary
		item.Streak = summary.CurrentStreak

		if len(entries) > 0 {
			last := entries[len(entries)-1]
			value := last.Value
			item.LastValue = &value
			item.LastLogged = last.Day
		}

		items = append(items, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"items":      items,
		"today":      todayKey,
		"weekStart":  weekStart.Format(analytics.DayLayout),
		"monthStart": monthStart.Format(analytics.DayLayout),
	})
}

func startOfWeek(t time.Time) time.Time {
	offset := (int(t.Weekday()) + 6) % 7
	return t.AddDate(0, 0, -offset)
}

func firstOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}
