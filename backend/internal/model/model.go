package model

import "time"

const (
	KindBoolean = "boolean"
	KindNumeric = "numeric"

	GoalAtLeast = "at_least"
	GoalAtMost  = "at_most"

	PeriodDay   = "day"
	PeriodWeek  = "week"
	PeriodMonth = "month"

	ChartLine    = "line"
	ChartBar     = "bar"
	ChartArea    = "area"
	ChartDonut   = "donut"
	ChartHeatmap = "heatmap"
	ChartStat    = "stat"
)

// ChartTypes lists the widget/chart types the UI can render.
var ChartTypes = []string{ChartLine, ChartBar, ChartArea, ChartDonut, ChartHeatmap, ChartStat}

func ValidChartType(v string) bool {
	for _, t := range ChartTypes {
		if t == v {
			return true
		}
	}
	return false
}

// Roles a user can hold on a tracker. Only the owner may change tracker
// metadata, invite people or remove members.
const (
	RoleOwner  = "owner"
	RoleMember = "member"
)

// Invitation lifecycle. pending means the address was recorded but delivery
// failed; delivered means the message reached the transport. accepted, rejected
// and removed are terminal and retained as history.
const (
	InvitePending   = "pending"
	InviteDelivered = "delivered"
	InviteAccepted  = "accepted"
	InviteRejected  = "rejected"
	InviteRemoved   = "removed"
)

// Goal is an optional target attached to a tracker.
type Goal struct {
	Type   string  `json:"type"`   // at_least | at_most
	Value  float64 `json:"value"`  // target amount per period
	Period string  `json:"period"` // day | week | month
}

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	// TokenVersion invalidates sessions issued before a password change. It is
	// never exposed over the API.
	TokenVersion int `json:"-"`
}

type Tracker struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Color       string `json:"color"`
	Kind        string `json:"kind"`
	Unit        string `json:"unit"`
	Goal        *Goal  `json:"goal"`
	Archived    bool   `json:"archived"`
	// Role is the caller's membership on this tracker (owner or member). It is
	// empty only for a tracker handed back without an access check.
	Role string `json:"role,omitempty"`
	// SharedWith lists the addresses this tracker is shared with, outside the
	// owner. It is populated only by the list endpoint, where the owner wants
	// to see who has access without opening the tracker.
	SharedWith []string `json:"sharedWith,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// TrackerMember is one accepted member of a shared tracker.
type TrackerMember struct {
	UserID   string    `json:"userId"`
	Email    string    `json:"email"`
	Name     string    `json:"name"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joinedAt"`
}

// Invitation is one address invited to a tracker. It survives acceptance,
// rejection and removal so the owner's suggestions can offer it again.
type Invitation struct {
	ID          string    `json:"id"`
	TrackerID   string    `json:"trackerId"`
	TrackerName string    `json:"trackerName,omitempty"`
	InviterName string    `json:"inviterName,omitempty"`
	Email       string    `json:"email"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

// InviteSuggestion is an address the current user invited before.
type InviteSuggestion struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

// Entry is one logged day for a tracker. Days are local calendar dates in the
// user's timezone, stored as YYYY-MM-DD.
type Entry struct {
	TrackerID string `json:"trackerId,omitempty"`
	// UserID/UserName attribute the value to the member who logged it. Both are
	// omitted when reading a user's own entries (the caller already knows who).
	UserID    string    `json:"userId,omitempty"`
	UserName  string    `json:"userName,omitempty"`
	Day       string    `json:"day"`
	Value     float64   `json:"value"`
	Note      string    `json:"note"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// WidgetConfig controls how a widget aggregates and renders its tracker.
type WidgetConfig struct {
	Range       string `json:"range"`       // 7d | 30d | 90d | 365d | all
	Granularity string `json:"granularity"` // auto | day | week | month
	Aggregate   string `json:"aggregate"`   // sum | avg
	Metric      string `json:"metric"`      // total | average | count | streak
	ShowGoal    bool   `json:"showGoal"`
}

func DefaultWidgetConfig() WidgetConfig {
	return WidgetConfig{
		Range:       "30d",
		Granularity: "auto",
		Aggregate:   "sum",
		Metric:      "total",
		ShowGoal:    true,
	}
}

type LayoutItem struct {
	ID string `json:"id"`
	X  int    `json:"x"`
	Y  int    `json:"y"`
	W  int    `json:"w"`
	H  int    `json:"h"`
}

type Widget struct {
	ID          string       `json:"id"`
	DashboardID string       `json:"dashboardId"`
	TrackerID   string       `json:"trackerId"`
	Title       string       `json:"title"`
	ChartType   string       `json:"chartType"`
	Config      WidgetConfig `json:"config"`
	Layout      LayoutItem   `json:"layout"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
}

type Dashboard struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	IsDefault bool      `json:"isDefault"`
	Widgets   []Widget  `json:"widgets"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Series is one chart series (ApexCharts-shaped).
type Series struct {
	Name string    `json:"name"`
	Data []float64 `json:"data"`
}

type HeatRow struct {
	Name string    `json:"name"`
	Data []float64 `json:"data"`
}

type Heatmap struct {
	Columns []string  `json:"columns"`
	Rows    []HeatRow `json:"rows"`
}

type Stat struct {
	Value       float64   `json:"value"`
	Label       string    `json:"label"`
	Spark       []float64 `json:"spark"`
	SparkLabels []string  `json:"sparkLabels"`
}

type Summary struct {
	Days             int     `json:"days"`
	LoggedDays       int     `json:"loggedDays"`
	Total            float64 `json:"total"`
	Average          float64 `json:"average"`
	Min              float64 `json:"min"`
	Max              float64 `json:"max"`
	CompletionRate   float64 `json:"completionRate"`
	CurrentStreak    int     `json:"currentStreak"`
	BestStreak       int     `json:"bestStreak"`
	BestWeekday      string  `json:"bestWeekday"`
	BestWeekdayValue float64 `json:"bestWeekdayValue"`
	PeriodsCompleted int     `json:"periodsCompleted"`
	PeriodsTotal     int     `json:"periodsTotal"`
	Unit             string  `json:"unit"`
	Kind             string  `json:"kind"`
	Goal             *Goal   `json:"goal"`
	FirstDay         string  `json:"firstDay"`
	LastDay          string  `json:"lastDay"`
}

// ChartData is the single payload every chart widget understands.
type ChartData struct {
	ChartType   string   `json:"chartType"`
	Granularity string   `json:"granularity"`
	Labels      []string `json:"labels"`
	Series      []Series `json:"series"`
	Heatmap     *Heatmap `json:"heatmap,omitempty"`
	Stat        *Stat    `json:"stat,omitempty"`
	Summary     Summary  `json:"summary"`
	Empty       bool     `json:"empty"`
}

// OverviewItem feeds the "Today" screen.
type OverviewItem struct {
	Tracker      Tracker  `json:"tracker"`
	Entry        *Entry   `json:"entry"`
	PeriodTotal  float64  `json:"periodTotal"`
	PeriodLogged int      `json:"periodLogged"`
	PeriodTarget *float64 `json:"periodTarget"`
	GoalMet      *bool    `json:"goalMet"`
	Streak       int      `json:"streak"`
	LastValue    *float64 `json:"lastValue"`
	LastLogged   string   `json:"lastLoggedDay"`
}
