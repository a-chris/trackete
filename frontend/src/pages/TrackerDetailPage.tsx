import { useEffect, useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  ArrowLeft,
  ChartArea,
  ChartColumn,
  ChartLine,
  ChartPie,
  Grid3x3,
  Hash,
  Pencil,
} from 'lucide-react'
import { Link, useParams } from 'react-router-dom'
import { ChartRenderer } from '../components/charts/ChartRenderer'
import { TrackerFormModal } from '../components/trackers/TrackerFormModal'
import { EntryEditor } from '../components/trackers/EntryEditor'
import { ErrorState } from '../components/ui/ErrorState'
import { Spinner } from '../components/ui/Spinner'
import { api, errorMessage } from '../lib/api'
import { rangeStart, todayKey } from '../lib/date'
import { formatPercent, formatValue, goalLabel } from '../lib/format'
import { useTrackerData, useTrackers } from '../lib/queries'
import type { ChartType, Granularity, RangePreset } from '../lib/types'

const chartOptions: { value: ChartType; label: string; icon: typeof ChartLine }[] = [
  { value: 'line', label: 'Line', icon: ChartLine },
  { value: 'bar', label: 'Bar', icon: ChartColumn },
  { value: 'area', label: 'Area', icon: ChartArea },
  { value: 'donut', label: 'Donut', icon: ChartPie },
  { value: 'heatmap', label: 'Heatmap', icon: Grid3x3 },
  { value: 'stat', label: 'Stat', icon: Hash },
]

const ranges: { value: RangePreset; label: string }[] = [
  { value: '7d', label: '7 days' },
  { value: '30d', label: '30 days' },
  { value: '90d', label: '90 days' },
  { value: '365d', label: '12 months' },
  { value: 'all', label: 'All time' },
]

export function TrackerDetailPage() {
  const { trackerId = '' } = useParams()
  const { data: trackers = [], isLoading, isError, error, refetch, isFetching } = useTrackers(true)
  const tracker = trackers.find((candidate) => candidate.id === trackerId)

  const [range, setRange] = useState<RangePreset>('30d')
  const [chart, setChart] = useState<ChartType>('line')
  const [granularity, setGranularity] = useState<Granularity>('auto')
  const [aggregate, setAggregate] = useState<'sum' | 'avg'>('sum')
  const [editOpen, setEditOpen] = useState(false)

  const to = todayKey()
  const from = useMemo(() => rangeStart(range, new Date()), [range])

  const {
    data: chartData,
    isLoading: chartLoading,
    isError: chartIsError,
    refetch: refetchChart,
    isFetching: chartFetching,
  } = useTrackerData({
    trackerId,
    from,
    to,
    chart,
    granularity,
    aggregate,
    enabled: Boolean(tracker),
  })

  const {
    data: entries = [],
    isLoading: entriesLoading,
    isError: entriesError,
    error: entriesErrorValue,
    refetch: refetchEntries,
    isFetching: entriesFetching,
  } = useQuery({
    queryKey: ['entries', trackerId, from, to],
    queryFn: () => api.entries(trackerId, from, to),
    enabled: Boolean(tracker),
  })

  useEffect(() => {
    document.title = tracker ? `${tracker.name} · Trackete` : 'Tracker · Trackete'
  }, [tracker])

  if (isLoading) return <Spinner label="Loading tracker…" />
  if (isError) {
    return (
      <ErrorState
        title="Couldn't load your trackers"
        description={errorMessage(error)}
        onRetry={() => void refetch()}
        retrying={isFetching}
      />
    )
  }
  if (!tracker) {
    return (
      <div className="surface p-8 text-center">
        <p className="font-medium">Tracker not found</p>
        <p className="mt-1 text-sm text-base-content/60">It may have been deleted.</p>
        <Link to="/trackers" className="btn btn-primary btn-sm mt-4">
          Back to trackers
        </Link>
      </div>
    )
  }

  const summary = chartData?.summary
  const showAverageNote = tracker.kind === 'numeric' && summary && summary.loggedDays > 0

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-2 text-sm text-base-content/50">
        <Link to="/trackers" className="inline-flex items-center gap-1 hover:text-base-content">
          <ArrowLeft className="size-3.5" />
          Trackers
        </Link>
        <span>/</span>
        <span className="truncate">{tracker.name}</span>
      </div>

      <header className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div className="flex items-start gap-3">
          <div
            className="grid size-12 shrink-0 place-items-center rounded-2xl text-2xl"
            style={{ backgroundColor: `${tracker.color}22` }}
          >
            {tracker.icon}
          </div>
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">{tracker.name}</h1>
            <p className="mt-0.5 text-sm text-base-content/60">
              {goalLabel(tracker.goal, tracker.kind, tracker.unit) ??
                (tracker.kind === 'boolean' ? 'Yes / No habit' : `Number${tracker.unit ? ` in ${tracker.unit}` : ''}`)}
            </p>
            {tracker.description && (
              <p className="mt-2 max-w-2xl text-sm text-base-content/60">{tracker.description}</p>
            )}
          </div>
        </div>
        <div className="flex items-center gap-2">
          {tracker.archived && <span className="badge badge-ghost">Archived</span>}
          <button type="button" className="btn btn-sm btn-outline" onClick={() => setEditOpen(true)}>
            <Pencil className="size-3.5" />
            Edit
          </button>
        </div>
      </header>

      {summary && summary.days > 0 && (
        <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
          <StatBox
            label={tracker.kind === 'boolean' ? 'Days done' : 'Total'}
            value={formatValue(summary.total, tracker.unit, tracker.kind)}
            hint={`${summary.loggedDays} logged day${summary.loggedDays === 1 ? '' : 's'}`}
          />
          <StatBox
            label={tracker.kind === 'boolean' ? 'Completion' : 'Average / logged day'}
            value={
              tracker.kind === 'boolean'
                ? formatPercent(summary.completionRate)
                : formatValue(summary.average, tracker.unit)
            }
            hint={
              tracker.kind === 'boolean'
                ? `${summary.days} days in range`
                : showAverageNote
                  ? `min ${formatValue(summary.min)} · max ${formatValue(summary.max)}`
                  : undefined
            }
          />
          <StatBox
            label="Current streak"
            value={`${summary.currentStreak} d`}
            hint={`best ${summary.bestStreak} d`}
          />
          <StatBox
            label="Strongest day"
            value={summary.bestWeekday || '—'}
            hint={
              summary.bestWeekday
                ? `${formatValue(summary.bestWeekdayValue, tracker.unit, tracker.kind)} total`
                : undefined
            }
          />
        </div>
      )}

      <section className="surface p-4 sm:p-5">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
          <div className="flex flex-wrap gap-1">
            {chartOptions.map(({ value, label, icon: Icon }) => (
              <button
                key={value}
                type="button"
                className={`btn btn-sm ${chart === value ? 'btn-primary' : 'btn-ghost'}`}
                onClick={() => setChart(value)}
                aria-pressed={chart === value}
              >
                <Icon className="size-4" />
                <span className="hidden sm:inline">{label}</span>
              </button>
            ))}
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <select
              className="select select-sm"
              value={range}
              onChange={(event) => setRange(event.target.value as RangePreset)}
              aria-label="Range"
            >
              {ranges.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
            <select
              className="select select-sm"
              value={granularity}
              onChange={(event) => setGranularity(event.target.value as Granularity)}
              aria-label="Granularity"
            >
              <option value="auto">Auto grouping</option>
              <option value="day">Daily</option>
              <option value="week">Weekly</option>
              <option value="month">Monthly</option>
            </select>
            {tracker.kind === 'numeric' && (
              <select
                className="select select-sm"
                value={aggregate}
                onChange={(event) => setAggregate(event.target.value as 'sum' | 'avg')}
                aria-label="Aggregate"
              >
                <option value="sum">Sum</option>
                <option value="avg">Average</option>
              </select>
            )}
          </div>
        </div>

        <div className="mt-4 h-72 w-full sm:h-80">
          {chartLoading ? (
            <Spinner label="Loading chart…" />
          ) : (
            <ChartRenderer
              data={chartIsError ? undefined : chartData}
              kind={tracker.kind}
              unit={tracker.unit}
              color={tracker.color}
              onRetry={chartIsError ? () => void refetchChart() : undefined}
              retrying={chartFetching}
            />
          )}
        </div>

        {summary && tracker.goal && summary.periodsTotal > 0 && (
          <p className="mt-2 text-xs text-base-content/50">
            On target in {summary.periodsCompleted} of {summary.periodsTotal} {tracker.goal.period}s.
          </p>
        )}
      </section>

      <EntryEditor
        key={tracker.id}
        tracker={tracker}
        entries={entries}
        loading={entriesLoading}
        error={entriesError}
        errorDetail={entriesError ? errorMessage(entriesErrorValue) : undefined}
        onRetry={() => void refetchEntries()}
        retrying={entriesFetching}
      />

      {editOpen && <TrackerFormModal tracker={tracker} onClose={() => setEditOpen(false)} />}
    </div>
  )
}

function StatBox({ label, value, hint }: { label: string; value: string; hint?: string }) {
  return (
    <div className="surface p-4">
      <p className="stat-label">{label}</p>
      <p className="mt-1 truncate text-xl font-semibold tabular-nums">{value}</p>
      {hint && <p className="mt-0.5 truncate text-[11px] text-base-content/40">{hint}</p>}
    </div>
  )
}
