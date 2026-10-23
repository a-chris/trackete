// Package analytics turns raw daily entries into the chart payloads the UI
// renders. It is pure (no database, no clock) so it can be unit tested.
package analytics

import (
	"math"
	"time"

	"github.com/a-chris/trackete/backend/internal/model"
)

const DayLayout = "2006-01-02"

var weekdayNames = []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}

type Options struct {
	Name        string
	Kind        string
	Unit        string
	Goal        *model.Goal
	ChartType   string
	From        time.Time
	To          time.Time
	Granularity string // auto | day | week | month
	Aggregate   string // sum | avg
	Metric      string // total | average | count | streak
	Today       time.Time
}

func (o Options) kind() string {
	if o.Kind == model.KindBoolean {
		return model.KindBoolean
	}
	return model.KindNumeric
}

// RangeStart converts a preset ("7d", "30d", ...) into an inclusive start date.
func RangeStart(to time.Time, preset string) time.Time {
	to = DateOnly(to)
	switch preset {
	case "7d":
		return to.AddDate(0, 0, -6)
	case "30d":
		return to.AddDate(0, 0, -29)
	case "90d":
		return to.AddDate(0, 0, -89)
	case "365d":
		return to.AddDate(0, 0, -364)
	default:
		return to.AddDate(0, 0, -1825)
	}
}

func DateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func (o Options) resolved() (from, to time.Time, span int, gran string) {
	from, to = DateOnly(o.From), DateOnly(o.To)
	if to.Before(from) {
		from, to = to, from
	}
	span = int(to.Sub(from).Hours()/24) + 1
	if span > 3660 {
		from = to.AddDate(0, 0, -3659)
		span = 3660
	}

	gran = o.Granularity
	if gran != model.PeriodDay && gran != model.PeriodWeek && gran != model.PeriodMonth {
		switch {
		case span <= 62:
			gran = model.PeriodDay
		case span <= 370:
			gran = model.PeriodWeek
		default:
			gran = model.PeriodMonth
		}
	}
	return from, to, span, gran
}

func Build(entries []model.Entry, o Options) model.ChartData {
	from, to, span, gran := o.resolved()
	if o.Today.IsZero() {
		o.Today = to
	}
	kind := o.kind()

	daily := make(map[string]float64, len(entries))
	logged := make(map[string]bool, len(entries))
	first, last := "", ""
	for _, e := range entries {
		d, err := time.Parse(DayLayout, e.Day)
		if err != nil {
			continue
		}
		d = DateOnly(d)
		if first == "" || d.Format(DayLayout) < first {
			first = d.Format(DayLayout)
		}
		if last == "" || d.Format(DayLayout) > last {
			last = d.Format(DayLayout)
		}
		if d.Before(from) || d.After(to) {
			continue
		}
		key := d.Format(DayLayout)
		value := e.Value
		if kind == model.KindBoolean {
			value = boolValue(value)
		}
		daily[key] = value
		logged[key] = true
	}

	keys, values, present := buildBuckets(from, to, gran, daily, logged, kind, o.Aggregate)

	seriesName := o.Name
	if seriesName == "" {
		seriesName = "Value"
	}

	data := model.ChartData{
		ChartType:   o.ChartType,
		Granularity: gran,
		Labels:      keys,
		Series:      []model.Series{{Name: seriesName, Data: values}},
	}
	if o.ChartType == "" {
		data.ChartType = model.ChartLine
	}

	summary := buildSummary(from, to, span, kind, o, daily, logged, first, last)
	data.Summary = summary
	data.Empty = len(daily) == 0

	switch o.ChartType {
	case model.ChartHeatmap:
		data.Heatmap = buildHeatmap(from, to, daily, logged)
		data.Series = nil
	case model.ChartDonut:
		data.Labels, data.Series = buildDonut(span, kind, daily, logged)
	case model.ChartStat:
		data.Stat = buildStat(summary, keys, values, present, o, kind)
		data.Series = nil
	}

	return data
}

func boolValue(v float64) float64 {
	if v >= 0.5 {
		return 1
	}
	return 0
}

type bucket struct {
	key    string
	start  time.Time
	end    time.Time
	values []float64
}

func buildBuckets(from, to time.Time, gran string, daily map[string]float64, logged map[string]bool, kind, aggregate string) ([]string, []float64, []bool) {
	buckets := make([]bucket, 0, 64)

	switch gran {
	case model.PeriodWeek:
		for cursor := startOfWeek(from); !cursor.After(to); cursor = cursor.AddDate(0, 0, 7) {
			buckets = append(buckets, bucket{key: cursor.Format(DayLayout), start: cursor, end: cursor.AddDate(0, 0, 6)})
		}
	case model.PeriodMonth:
		for cursor := firstOfMonth(from); !cursor.After(to); cursor = cursor.AddDate(0, 1, 0) {
			buckets = append(buckets, bucket{key: cursor.Format(DayLayout), start: cursor, end: cursor.AddDate(0, 1, -1)})
		}
	default:
		for cursor := from; !cursor.After(to); cursor = cursor.AddDate(0, 0, 1) {
			buckets = append(buckets, bucket{key: cursor.Format(DayLayout), start: cursor, end: cursor})
		}
	}

	keys := make([]string, 0, len(buckets))
	values := make([]float64, 0, len(buckets))
	present := make([]bool, 0, len(buckets))

	for _, b := range buckets {
		start := b.start
		if start.Before(from) {
			start = from
		}
		end := b.end
		if end.After(to) {
			end = to
		}

		total, count := 0.0, 0
		for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
			key := d.Format(DayLayout)
			if !logged[key] {
				continue
			}
			total += daily[key]
			count++
		}

		keys = append(keys, b.key)
		present = append(present, count > 0)

		switch {
		case kind == model.KindBoolean && gran == model.PeriodDay:
			values = append(values, total)
		case gran == model.PeriodDay:
			values = append(values, total)
		case kind == model.KindBoolean:
			values = append(values, total) // count of "yes" days in the period
		case aggregate == "avg":
			if count == 0 {
				values = append(values, 0)
			} else {
				values = append(values, round2(total/float64(count)))
			}
		default:
			values = append(values, round2(total))
		}
	}

	return keys, values, present
}

func buildSummary(from, to time.Time, span int, kind string, o Options, daily map[string]float64, logged map[string]bool, first, last string) model.Summary {
	s := model.Summary{
		Days:     span,
		Unit:     o.Unit,
		Kind:     kind,
		Goal:     o.Goal,
		FirstDay: first,
		LastDay:  last,
	}

	var sum float64
	minV, maxV := math.Inf(1), math.Inf(-1)
	for key, v := range daily {
		if !logged[key] {
			continue
		}
		s.LoggedDays++
		sum += v
		minV = math.Min(minV, v)
		maxV = math.Max(maxV, v)
	}
	s.Total = round2(sum)
	if s.LoggedDays > 0 {
		s.Average = round2(sum / float64(s.LoggedDays))
		s.Min = round2(minV)
		s.Max = round2(maxV)
	}

	// Completion + streaks.
	hitDays := 0
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		key := d.Format(DayLayout)
		if dayHit(kind, o.Goal, daily[key], logged[key]) {
			hitDays++
		}
	}
	if o.Goal != nil && o.Goal.Period != model.PeriodDay {
		periodsTotal, periodsCompleted := evaluatePeriods(from, to, o.Goal, daily, logged)
		s.PeriodsTotal = periodsTotal
		s.PeriodsCompleted = periodsCompleted
		if periodsTotal > 0 {
			s.CompletionRate = round2(float64(periodsCompleted) / float64(periodsTotal))
		}
	} else if o.Goal != nil {
		s.CompletionRate = round2(float64(hitDays) / float64(span))
	} else if kind == model.KindBoolean {
		s.CompletionRate = round2(float64(hitDays) / float64(span))
	} else if span > 0 {
		s.CompletionRate = round2(float64(s.LoggedDays) / float64(span))
	}

	s.CurrentStreak, s.BestStreak = streaks(from, to, DateOnly(o.Today), kind, o.Goal, daily, logged)
	s.BestWeekday, s.BestWeekdayValue = bestWeekday(kind, daily, logged)

	return s
}

// dayHit reports whether a single day satisfies the tracker's target. A day
// without a record counts as 0, which satisfies "at most" goals.
func dayHit(kind string, goal *model.Goal, value float64, logged bool) bool {
	if !logged {
		value = 0
	}
	if kind == model.KindBoolean {
		value = boolValue(value)
	}
	if goal != nil && goal.Period == model.PeriodDay {
		switch goal.Type {
		case model.GoalAtLeast:
			return value >= goal.Value
		case model.GoalAtMost:
			return value <= goal.Value
		}
	}
	return value > 0
}

func evaluatePeriods(from, to time.Time, goal *model.Goal, daily map[string]float64, logged map[string]bool) (total, completed int) {
	totals := map[string]float64{}
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		key := d.Format(DayLayout)
		if !logged[key] {
			continue
		}
		var periodKey string
		if goal.Period == model.PeriodMonth {
			periodKey = firstOfMonth(d).Format(DayLayout)
		} else {
			periodKey = startOfWeek(d).Format(DayLayout)
		}
		totals[periodKey] += daily[key]
	}

	// Count every period that intersects the range, even with no records.
	seen := map[string]bool{}
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		var periodKey string
		if goal.Period == model.PeriodMonth {
			periodKey = firstOfMonth(d).Format(DayLayout)
		} else {
			periodKey = startOfWeek(d).Format(DayLayout)
		}
		if seen[periodKey] {
			continue
		}
		seen[periodKey] = true
		total++
		v := totals[periodKey]
		switch goal.Type {
		case model.GoalAtLeast:
			if v >= goal.Value {
				completed++
			}
		case model.GoalAtMost:
			if v <= goal.Value {
				completed++
			}
		}
	}
	return total, completed
}

func streaks(from, to, today time.Time, kind string, goal *model.Goal, daily map[string]float64, logged map[string]bool) (current, best int) {
	// Best streak across the whole requested range.
	run := 0
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		key := d.Format(DayLayout)
		if dayHit(kind, goal, daily[key], logged[key]) {
			run++
			if run > best {
				best = run
			}
		} else {
			run = 0
		}
	}

	// Current streak, anchored on today (or the end of the range for past ranges).
	anchor := to
	if today.Before(anchor) {
		anchor = today
	}
	if anchor.Before(from) {
		return 0, best
	}
	if !logged[anchor.Format(DayLayout)] && anchor.Equal(today) {
		anchor = anchor.AddDate(0, 0, -1)
	}
	for d := anchor; !d.Before(from); d = d.AddDate(0, 0, -1) {
		key := d.Format(DayLayout)
		if dayHit(kind, goal, daily[key], logged[key]) {
			current++
		} else {
			break
		}
	}
	return current, best
}

func bestWeekday(kind string, daily map[string]float64, logged map[string]bool) (string, float64) {
	totals := make([]float64, 7)
	counts := make([]int, 7)
	for key, v := range daily {
		if !logged[key] {
			continue
		}
		d, err := time.Parse(DayLayout, key)
		if err != nil {
			continue
		}
		idx := int(d.Weekday()+6) % 7 // Monday = 0
		totals[idx] += v
		counts[idx]++
	}

	bestIdx, bestValue := -1, 0.0
	for i := range weekdayNames {
		var score float64
		if kind == model.KindBoolean {
			score = totals[i]
		} else if counts[i] > 0 {
			score = totals[i]
		}
		if score > bestValue {
			bestIdx, bestValue = i, score
		}
	}
	if bestIdx == -1 {
		return "", 0
	}
	return weekdayNames[bestIdx], round2(bestValue)
}

func buildHeatmap(from, to time.Time, daily map[string]float64, logged map[string]bool) *model.Heatmap {
	columns := make([]string, 0, 64)
	for cursor := startOfWeek(from); !startOfWeek(to).Before(cursor); cursor = cursor.AddDate(0, 0, 7) {
		columns = append(columns, cursor.Format(DayLayout))
	}

	rows := make([]model.HeatRow, 0, 7)
	for idx, name := range weekdayNames {
		values := make([]float64, len(columns))
		for col, colStart := range columns {
			colDate, err := time.Parse(DayLayout, colStart)
			if err != nil {
				continue
			}
			day := colDate.AddDate(0, 0, idx)
			if day.Before(from) || day.After(to) {
				continue
			}
			key := day.Format(DayLayout)
			if logged[key] {
				values[col] = round2(daily[key])
			}
		}
		rows = append(rows, model.HeatRow{Name: name, Data: values})
	}
	return &model.Heatmap{Columns: columns, Rows: rows}
}

func buildDonut(span int, kind string, daily map[string]float64, logged map[string]bool) ([]string, []model.Series) {
	if kind == model.KindBoolean {
		yes := 0.0
		for key, v := range daily {
			if logged[key] && v >= 0.5 {
				yes++
			}
		}
		missed := float64(span) - yes
		if missed < 0 {
			missed = 0
		}
		return []string{"Done", "Missed"}, []model.Series{
			{Name: "Days", Data: []float64{yes, missed}},
		}
	}

	totals := make([]float64, 7)
	for key, v := range daily {
		if !logged[key] {
			continue
		}
		d, err := time.Parse(DayLayout, key)
		if err != nil {
			continue
		}
		totals[int(d.Weekday()+6)%7] += v
	}
	for i := range totals {
		totals[i] = round2(totals[i])
	}
	return weekdayNames, []model.Series{{Name: "Total", Data: totals}}
}

func buildStat(summary model.Summary, keys []string, values []float64, present []bool, o Options, kind string) *model.Stat {
	stat := &model.Stat{}
	switch o.Metric {
	case "average":
		stat.Value = summary.Average
		stat.Label = "Average per logged day"
		if kind == model.KindBoolean {
			stat.Value = round2(summary.CompletionRate * 100)
			stat.Label = "Completion rate"
		}
	case "count":
		stat.Value = float64(summary.LoggedDays)
		stat.Label = "Days logged"
	case "streak":
		stat.Value = float64(summary.CurrentStreak)
		stat.Label = "Current streak"
	default:
		stat.Value = summary.Total
		stat.Label = "Total"
		if kind == model.KindBoolean {
			stat.Label = "Days completed"
		}
	}

	start := len(values) - 14
	if start < 0 {
		start = 0
	}
	stat.Spark = append([]float64{}, values[start:]...)
	stat.SparkLabels = append([]string{}, keys[start:]...)
	return stat
}

func startOfWeek(t time.Time) time.Time {
	offset := (int(t.Weekday()) + 6) % 7 // Monday = 0
	return DateOnly(t).AddDate(0, 0, -offset)
}

func firstOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func round2(v float64) float64 {
	if math.IsInf(v, 0) || math.IsNaN(v) {
		return 0
	}
	return math.Round(v*100) / 100
}
