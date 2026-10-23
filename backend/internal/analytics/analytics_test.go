package analytics

import (
	"testing"
	"time"

	"github.com/a-chris/trackete/backend/internal/model"
)

func day(s string) time.Time {
	t, err := time.Parse(DayLayout, s)
	if err != nil {
		panic(err)
	}
	return t
}

func entries(pairs map[string]float64) []model.Entry {
	out := make([]model.Entry, 0, len(pairs))
	for d, v := range pairs {
		out = append(out, model.Entry{Day: d, Value: v})
	}
	return out
}

func TestBooleanWeeklyBucketsCountYesDays(t *testing.T) {
	input := entries(map[string]float64{
		"2026-01-01": 1, // Thu
		"2026-01-02": 1, // Fri
		"2026-01-05": 1, // Mon
	})

	data := Build(input, Options{
		Kind:      model.KindBoolean,
		ChartType: model.ChartLine,
		From:      day("2026-01-01"),
		To:        day("2026-01-11"),
		Today:     day("2026-01-11"),
		// auto granularity: 11 day span -> day buckets
	})

	if len(data.Labels) != 11 {
		t.Fatalf("expected 11 day buckets, got %d", len(data.Labels))
	}
	if data.Summary.Total != 3 {
		t.Fatalf("expected total 3, got %v", data.Summary.Total)
	}
	if data.Summary.BestStreak != 2 {
		t.Fatalf("expected best streak 2, got %d", data.Summary.BestStreak)
	}

	weekly := Build(input, Options{
		Kind:        model.KindBoolean,
		ChartType:   model.ChartLine,
		From:        day("2026-01-01"),
		To:          day("2026-01-18"),
		Today:       day("2026-01-18"),
		Granularity: model.PeriodWeek,
	})
	if weekly.Granularity != model.PeriodWeek {
		t.Fatalf("expected week granularity, got %s", weekly.Granularity)
	}
	if len(weekly.Labels) != 3 {
		t.Fatalf("expected 3 week buckets, got %d (%v)", len(weekly.Labels), weekly.Labels)
	}
	// Week starting 2025-12-29 has 2 yes days, week 2026-01-05 has 1.
	if weekly.Series[0].Data[0] != 2 || weekly.Series[0].Data[1] != 1 {
		t.Fatalf("unexpected weekly counts: %v", weekly.Series[0].Data)
	}
}

func TestNumericAverageAggregation(t *testing.T) {
	input := entries(map[string]float64{
		"2026-02-02": 70,
		"2026-02-04": 74,
	})
	data := Build(input, Options{
		Kind:        model.KindNumeric,
		ChartType:   model.ChartLine,
		From:        day("2026-02-02"),
		To:          day("2026-02-08"),
		Today:       day("2026-02-08"),
		Granularity: model.PeriodWeek,
		Aggregate:   "avg",
	})
	if got := data.Series[0].Data[0]; got != 72 {
		t.Fatalf("expected weekly average 72, got %v", got)
	}
	if data.Summary.Average != 72 {
		t.Fatalf("expected summary average 72, got %v", data.Summary.Average)
	}
	if data.Summary.LoggedDays != 2 {
		t.Fatalf("expected 2 logged days, got %d", data.Summary.LoggedDays)
	}
}

func TestWeeklyAtMostGoalCompletion(t *testing.T) {
	goal := &model.Goal{Type: model.GoalAtMost, Value: 5, Period: model.PeriodWeek}
	input := entries(map[string]float64{
		// Week of 2026-01-05: 4 drinks -> on target
		"2026-01-05": 2,
		"2026-01-09": 2,
		// Week of 2026-01-12: 8 drinks -> over target
		"2026-01-12": 5,
		"2026-01-13": 3,
	})
	data := Build(input, Options{
		Kind:      model.KindNumeric,
		Goal:      goal,
		ChartType: model.ChartStat,
		From:      day("2026-01-05"),
		To:        day("2026-01-18"),
		Today:     day("2026-01-18"),
	})
	if data.Summary.PeriodsTotal != 2 || data.Summary.PeriodsCompleted != 1 {
		t.Fatalf("expected 1/2 periods completed, got %d/%d",
			data.Summary.PeriodsCompleted, data.Summary.PeriodsTotal)
	}
	if data.Summary.CompletionRate != 0.5 {
		t.Fatalf("expected completion rate 0.5, got %v", data.Summary.CompletionRate)
	}
}

func TestDonutAndHeatmapShapes(t *testing.T) {
	input := entries(map[string]float64{
		"2026-01-05": 1, // Mon
		"2026-01-06": 0, // Tue
		"2026-01-07": 1, // Wed
	})
	donut := Build(input, Options{Kind: model.KindBoolean, ChartType: model.ChartDonut,
		From: day("2026-01-05"), To: day("2026-01-11"), Today: day("2026-01-11")})
	if len(donut.Series) != 1 || len(donut.Series[0].Data) != 2 {
		t.Fatalf("expected Yes/No donut, got %+v", donut.Series)
	}
	if donut.Series[0].Data[0] != 2 || donut.Series[0].Data[1] != 5 {
		t.Fatalf("unexpected donut values: %v", donut.Series[0].Data)
	}

	heat := Build(input, Options{Kind: model.KindBoolean, ChartType: model.ChartHeatmap,
		From: day("2026-01-05"), To: day("2026-01-11"), Today: day("2026-01-11")})
	if heat.Heatmap == nil || len(heat.Heatmap.Rows) != 7 || len(heat.Heatmap.Columns) != 1 {
		t.Fatalf("unexpected heatmap shape: %+v", heat.Heatmap)
	}
	if heat.Heatmap.Rows[0].Data[0] != 1 || heat.Heatmap.Rows[2].Data[0] != 1 || heat.Heatmap.Rows[1].Data[0] != 0 {
		t.Fatalf("unexpected heatmap values: %+v", heat.Heatmap.Rows)
	}
}

func TestCurrentStreakIgnoresUnloggedToday(t *testing.T) {
	input := entries(map[string]float64{
		"2026-03-10": 1,
		"2026-03-11": 1,
		"2026-03-12": 1,
	})
	data := Build(input, Options{Kind: model.KindBoolean, ChartType: model.ChartStat,
		From: day("2026-03-01"), To: day("2026-03-13"), Today: day("2026-03-13")})
	if data.Summary.CurrentStreak != 3 {
		t.Fatalf("expected current streak 3, got %d", data.Summary.CurrentStreak)
	}
}
