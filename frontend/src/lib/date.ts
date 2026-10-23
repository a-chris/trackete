import type { Granularity, RangePreset } from './types'

const MS_PER_DAY = 86_400_000

export function dayKey(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

export function parseDay(key: string): Date {
  const [year, month, day] = key.split('-').map(Number)
  return new Date(year ?? 1970, (month ?? 1) - 1, day ?? 1)
}

export function todayKey(): string {
  return dayKey(new Date())
}

export function addDays(date: Date, amount: number): Date {
  const next = new Date(date.getFullYear(), date.getMonth(), date.getDate())
  next.setDate(next.getDate() + amount)
  return next
}

export function startOfWeek(date: Date): Date {
  const offset = (date.getDay() + 6) % 7
  return addDays(date, -offset)
}

export function startOfMonth(date: Date): Date {
  return new Date(date.getFullYear(), date.getMonth(), 1)
}

export function rangeStart(preset: RangePreset, to: Date): string {
  switch (preset) {
    case '7d':
      return dayKey(addDays(to, -6))
    case '30d':
      return dayKey(addDays(to, -29))
    case '90d':
      return dayKey(addDays(to, -89))
    case '365d':
      return dayKey(addDays(to, -364))
    case 'all':
      return dayKey(new Date(2000, 0, 1))
  }
}

export function rangeDays(preset: RangePreset): string {
  switch (preset) {
    case '7d':
      return 'Last 7 days'
    case '30d':
      return 'Last 30 days'
    case '90d':
      return 'Last 90 days'
    case '365d':
      return 'Last 12 months'
    case 'all':
      return 'All time'
  }
}

export function formatDate(key: string, options?: Intl.DateTimeFormatOptions): string {
  if (!key) return '—'
  return parseDay(key).toLocaleDateString(undefined, options ?? { month: 'short', day: 'numeric' })
}

export function formatLongDate(key: string): string {
  return formatDate(key, { weekday: 'long', month: 'long', day: 'numeric', year: 'numeric' })
}

export function formatMonth(key: string, options?: Intl.DateTimeFormatOptions): string {
  return formatDate(key, options ?? { month: 'short', year: 'numeric' })
}

export function formatBucket(key: string, granularity: Granularity): string {
  switch (granularity) {
    case 'month':
      return formatMonth(key)
    case 'week':
      return `Week of ${formatDate(key)}`
    default:
      return formatDate(key)
  }
}

export function bucketLabel(key: string, granularity: Granularity): string {
  switch (granularity) {
    case 'month':
      return formatMonth(key, { month: 'short' })
    case 'week':
      return formatDate(key)
    default:
      return formatDate(key, { month: 'short', day: 'numeric' })
  }
}

export function relativeDay(key: string): string {
  const today = todayKey()
  if (key === today) return 'Today'
  if (key === dayKey(addDays(new Date(), -1))) return 'Yesterday'
  return formatDate(key)
}

export function daysBetween(from: string, to: string): number {
  const a = parseDay(from).getTime()
  const b = parseDay(to).getTime()
  return Math.round((b - a) / MS_PER_DAY) + 1
}

export function isToday(key: string): boolean {
  return key === todayKey()
}
