export type TrackerKind = 'boolean' | 'numeric'
export type GoalType = 'at_least' | 'at_most'
export type GoalPeriod = 'day' | 'week' | 'month'
export type ChartType = 'line' | 'bar' | 'area' | 'donut' | 'heatmap' | 'stat'
export type RangePreset = '7d' | '30d' | '90d' | '365d' | 'all'
export type Granularity = 'auto' | 'day' | 'week' | 'month'
export type Aggregate = 'sum' | 'avg'
export type StatMetric = 'total' | 'average' | 'count' | 'streak'
export type ThemeMode = 'system' | 'light' | 'dark'
export type TrackerRole = 'owner' | 'member'
export type InvitationStatus = 'pending' | 'delivered' | 'accepted' | 'rejected' | 'removed'

export interface User {
  id: string
  email: string
  name: string
  createdAt: string
}

export interface Goal {
  type: GoalType
  value: number
  period: GoalPeriod
}

export interface Tracker {
  id: string
  name: string
  description: string
  icon: string
  color: string
  kind: TrackerKind
  unit: string
  goal: Goal | null
  archived: boolean
  /** How the signed-in user relates to this tracker; the owner manages it. */
  role?: TrackerRole
  /** Addresses the tracker is shared with, owner excluded; list endpoint only. */
  sharedWith?: string[]
  createdAt: string
  updatedAt: string
}

export interface Entry {
  trackerId?: string
  /** Who logged the value. Absent when the entry is already the caller's own. */
  userId?: string
  userName?: string
  day: string
  value: number
  note: string
  updatedAt: string
}

export interface TrackerInput {
  name: string
  description: string
  icon: string
  color: string
  kind: TrackerKind
  unit: string
  goal: Goal | null
  archived: boolean
}

export interface TrackerMember {
  userId: string
  email: string
  name: string
  role: TrackerRole
  joinedAt: string
}

export interface Invitation {
  id: string
  trackerId: string
  trackerName?: string
  email: string
  status: InvitationStatus
  createdAt: string
  expiresAt: string
}

export interface TrackerSharing {
  trackerId: string
  role: TrackerRole
  members: TrackerMember[]
  invitations: Invitation[]
}

export interface InviteSuggestion {
  email: string
  name?: string
}

export interface InvitationDetails {
  id: string
  trackerId: string
  trackerName: string
  email: string
  status: InvitationStatus
  expiresAt: string
  needsAccount: boolean
  signedInEmail?: string
}

export interface WidgetConfig {
  range: RangePreset
  granularity: Granularity
  aggregate: Aggregate
  metric: StatMetric
  showGoal: boolean
}

export interface WidgetLayout {
  id: string
  x: number
  y: number
  w: number
  h: number
}

export interface Widget {
  id: string
  dashboardId: string
  trackerId: string
  title: string
  chartType: ChartType
  config: WidgetConfig
  layout: WidgetLayout
  createdAt: string
  updatedAt: string
}

export interface Dashboard {
  id: string
  name: string
  isDefault: boolean
  widgets: Widget[]
  createdAt: string
  updatedAt: string
}

export interface Series {
  name: string
  data: number[]
}

export interface HeatRow {
  name: string
  data: number[]
}

export interface Heatmap {
  columns: string[]
  rows: HeatRow[]
}

export interface Stat {
  value: number
  label: string
  spark: number[]
  sparkLabels: string[]
}

export interface Summary {
  days: number
  loggedDays: number
  total: number
  average: number
  min: number
  max: number
  completionRate: number
  currentStreak: number
  bestStreak: number
  bestWeekday: string
  bestWeekdayValue: number
  periodsCompleted: number
  periodsTotal: number
  unit: string
  kind: TrackerKind
  goal: Goal | null
  firstDay: string
  lastDay: string
}

export interface ChartData {
  chartType: ChartType
  granularity: Granularity
  labels: string[]
  series: Series[]
  heatmap?: Heatmap
  stat?: Stat
  summary: Summary
  empty: boolean
}

export interface OverviewItem {
  tracker: Tracker
  entry: Entry | null
  periodTotal: number
  periodLogged: number
  periodTarget: number | null
  goalMet: boolean | null
  streak: number
  lastValue: number | null
  lastLoggedDay: string
}

export interface OverviewResponse {
  items: OverviewItem[]
  today: string
  weekStart: string
  monthStart: string
}

export interface Meta {
  version: string
  allowRegistration: boolean
}

export interface WidgetInput {
  trackerId: string
  title: string
  chartType: ChartType
  config: WidgetConfig
  layout: WidgetLayout
}
