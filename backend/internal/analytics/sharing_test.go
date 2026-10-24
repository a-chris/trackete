package analytics

import (
	"reflect"
	"testing"

	"github.com/a-chris/trackete/backend/internal/model"
)

// memberEntries flattens per-member day/value maps into attributed entries.
func memberEntries(values map[string]map[string]float64) []model.Entry {
	out := make([]model.Entry, 0, len(values))
	for name, days := range values {
		for day, value := range days {
			out = append(out, model.Entry{Day: day, UserID: name, UserName: name, Value: value})
		}
	}
	return out
}

// A tracker with one author keeps the payload it always had: a single series
// named after the tracker.
func TestSingleMemberKeepsOneSeries(t *testing.T) {
	input := []model.Entry{
		{Day: "2026-01-05", UserID: "u1", UserName: "Alice", Value: 2},
		{Day: "2026-01-06", UserID: "u1", UserName: "Alice", Value: 3},
	}

	data := Build(input, Options{
		Name:      "Beers",
		Kind:      model.KindNumeric,
		ChartType: model.ChartLine,
		From:      day("2026-01-05"),
		To:        day("2026-01-06"),
		Today:     day("2026-01-06"),
	})

	if len(data.Series) != 1 {
		t.Fatalf("expected one series for one author, got %+v", data.Series)
	}
	if data.Series[0].Name != "Beers" {
		t.Fatalf("single-author series must keep the tracker name, got %q", data.Series[0].Name)
	}
	if !reflect.DeepEqual(data.Series[0].Data, []float64{2, 3}) {
		t.Fatalf("unexpected single-author data: %v", data.Series[0].Data)
	}
	if data.Summary.Total != 5 || data.Summary.LoggedDays != 2 {
		t.Fatalf("unexpected single-author summary: %+v", data.Summary)
	}
}

// Entries without attribution (legacy rows, internal callers) must keep working
// as one series.
func TestUnattributedEntriesStaySingleSeries(t *testing.T) {
	data := Build([]model.Entry{{Day: "2026-01-05", Value: 1}}, Options{
		Kind:      model.KindNumeric,
		ChartType: model.ChartLine,
		From:      day("2026-01-05"),
		To:        day("2026-01-05"),
		Today:     day("2026-01-05"),
	})
	if len(data.Series) != 1 {
		t.Fatalf("expected one series without attribution, got %+v", data.Series)
	}
}

// Two members logging the same day produce one series each plus a combined
// summary.
func TestMultiMemberSeriesAndCombinedSummary(t *testing.T) {
	input := []model.Entry{
		{Day: "2026-01-05", UserID: "u1", UserName: "Alice", Value: 2},
		{Day: "2026-01-06", UserID: "u1", UserName: "Alice", Value: 1},
		{Day: "2026-01-05", UserID: "u2", UserName: "Bob", Value: 4},
	}

	data := Build(input, Options{
		Name:      "Beers",
		Kind:      model.KindNumeric,
		ChartType: model.ChartBar,
		From:      day("2026-01-05"),
		To:        day("2026-01-06"),
		Today:     day("2026-01-06"),
	})

	if len(data.Series) != 2 {
		t.Fatalf("expected one series per member, got %+v", data.Series)
	}
	if data.Series[0].Name != "Alice" || data.Series[1].Name != "Bob" {
		t.Fatalf("series must be named after the members, got %q and %q", data.Series[0].Name, data.Series[1].Name)
	}
	if !reflect.DeepEqual(data.Series[0].Data, []float64{2, 1}) {
		t.Fatalf("unexpected Alice series: %v", data.Series[0].Data)
	}
	if !reflect.DeepEqual(data.Series[1].Data, []float64{4, 0}) {
		t.Fatalf("unexpected Bob series: %v", data.Series[1].Data)
	}
	// The combined summary still adds both members up: 2 + 4 + 1.
	if data.Summary.Total != 7 {
		t.Fatalf("expected combined total 7, got %v", data.Summary.Total)
	}
	if data.Summary.LoggedDays != 2 {
		t.Fatalf("expected two logged days, got %d", data.Summary.LoggedDays)
	}
}

// The heatmap names its rows after the members once more than one person logs
// the tracker, so it stays readable instead of merging everyone.
func TestMultiMemberHeatmapRowsPerMember(t *testing.T) {
	input := memberEntries(map[string]map[string]float64{
		"Alice": {"2026-01-05": 1},
		"Bob":   {"2026-01-06": 1},
	})

	data := Build(input, Options{
		Kind:      model.KindBoolean,
		ChartType: model.ChartHeatmap,
		From:      day("2026-01-05"),
		To:        day("2026-01-11"),
		Today:     day("2026-01-11"),
	})

	if data.Heatmap == nil {
		t.Fatal("expected a heatmap payload")
	}
	if len(data.Heatmap.Rows) != 2 {
		t.Fatalf("expected one row per member, got %+v", data.Heatmap.Rows)
	}
	if data.Heatmap.Rows[0].Name != "Alice" || data.Heatmap.Rows[1].Name != "Bob" {
		t.Fatalf("heatmap rows must be named after members, got %+v", data.Heatmap.Rows)
	}
	if data.Heatmap.Rows[0].Data[0] != 1 || data.Heatmap.Rows[1].Data[0] != 1 {
		t.Fatalf("expected each member's yes day in the same week column, got %+v", data.Heatmap.Rows)
	}
}

// The donut shows each member's contribution; the combined numbers stay in the
// summary.
func TestMultiMemberDonutShowsContributions(t *testing.T) {
	input := []model.Entry{
		{Day: "2026-01-05", UserID: "u1", UserName: "Alice", Value: 2},
		{Day: "2026-01-06", UserID: "u1", UserName: "Alice", Value: 3},
		{Day: "2026-01-05", UserID: "u2", UserName: "Bob", Value: 4},
	}

	data := Build(input, Options{
		Kind:      model.KindNumeric,
		ChartType: model.ChartDonut,
		From:      day("2026-01-05"),
		To:        day("2026-01-06"),
		Today:     day("2026-01-06"),
	})

	if !reflect.DeepEqual(data.Labels, []string{"Alice", "Bob"}) {
		t.Fatalf("expected member labels, got %v", data.Labels)
	}
	if len(data.Series) != 1 || !reflect.DeepEqual(data.Series[0].Data, []float64{5, 4}) {
		t.Fatalf("unexpected member contributions: %+v", data.Series)
	}
	if data.Summary.Total != 9 {
		t.Fatalf("expected combined total 9, got %v", data.Summary.Total)
	}
}
