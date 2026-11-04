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
  Users,
} from 'lucide-react'
import { Link, useParams } from 'react-router-dom'
import { ChartRenderer } from '../components/charts/ChartRenderer'
import { TrackerFormModal } from '../components/trackers/TrackerFormModal'
import { TrackerSharing } from '../components/trackers/TrackerSharing'
import { EntryEditor } from '../components/trackers/EntryEditor'
import { ErrorState } from '../components/ui/ErrorState'
import { Modal } from '../components/ui/Modal'
import { Spinner } from '../components/ui/Spinner'
import { api, errorMessage } from '../lib/api'
import { rangeStart, todayKey } from '../lib/date'
import { formatPercent, formatValue, goalLabel } from '../lib/format'
import { useTrackerData, useTrackerSharing, useTrackers } from '../lib/queries'
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
  const [shareOpen, setShareOpen] = useState(false)
  const sharing = useTrackerSharing(trackerId)
  const shared = (sharing.data?.members.length ?? 1) > 1
  const openInvitations = (sharing.data?.invitations ?? []).filter(
    (invitation) => invitation.status === 'pending' || invitation.status === 'delivered',
  ).length

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
        <p className="mt-1 text-sm text-base-content/70">It may have been deleted.</p>
        <Link to="/trackers" className="btn btn-primary btn-sm mt-4">
          Back to trackers
        </Link>
      </div>
    )
  }

  const summary = chartData?.summary
  const showAverageNote = tracker.kind === 'numeric' && summary && summary.loggedDays > 0

  return (
    <div className="space-y-8">
      <div className="eyebrow flex items-center gap-2 text-base-content/65">
        <Link to="/trackers" className="inline-flex items-center gap-1.5 hover:text-base-content">
          <ArrowLeft className="size-3" />
          Trackers
        </Link>
        <span className="text-base-content/40">/</span>
        <span className="truncate text-base-content/70">{tracker.name}</span>
      </div>

      <header className="flex flex-col gap-5 border-b border-base-300 pb-6 sm:flex-row sm:items-start sm:justify-between">
        <div className="flex items-start gap-4">
          <span
            className="grid size-11 shrink-0 place-items-center rounded-[3px] text-2xl leading-none"
            style={{ backgroundColor: `${tracker.color}1f` }}
            aria-hidden="true"
          >
            {tracker.icon}
          </span>
          <div>
            <h1 className="display text-[34px] leading-none sm:text-4xl">{tracker.name}</h1>
            <p className="eyebrow mt-3">
              {goalLabel(tracker.goal, tracker.kind, tracker.unit) ??
                (tracker.kind === 'boolean'
                  ? 'Yes / No habit'
                  : `Number${tracker.unit ? ` in ${tracker.unit}` : ''}`)}
            </p>
            {tracker.description && (
              <p className="mt-3 max-w-2xl text-sm leading-relaxed text-base-content/70">
                {tracker.description}
              </p>
            )}
          </div>
        </div>
        <div className="flex items-center gap-2">
          {tracker.archived && (
            <span className="eyebrow border border-base-300 px-1.5 py-0.5">Archived</span>
          )}
          {shared && (
            <span className="eyebrow inline-flex items-center gap-1.5 border border-base-300 px-1.5 py-0.5">
              <Users className="size-3" />
              Shared
            </span>
          )}
          {/* The owner always gets the sharing panel so invitations stay
              trackable; members see it once the tracker has another member. */}
          {(tracker.role === 'owner' || shared) && (
            <button
              type="button"
              className="btn btn-sm btn-ghost gap-2 border-base-300"
              onClick={() => setShareOpen(true)}
            >
              <Users className="size-3.5" />
              {tracker.role === 'member' ? 'Members' : 'Share'}
              {tracker.role === 'owner' && openInvitations > 0 && (
                <span className="num text-[11px] text-base-content/60">{openInvitations}</span>
              )}
            </button>
          )}
          {tracker.role !== 'member' && (
            <button
              type="button"
              className="btn btn-sm btn-ghost gap-2 border-base-300"
              onClick={() => setEditOpen(true)}
            >
              <Pencil className="size-3.5" />
              Edit
            </button>
          )}
        </div>
      </header>

      {summary && summary.days > 0 && (
        <dl className="flex flex-col divide-y divide-base-300 border-b border-base-300 sm:flex-row sm:divide-x sm:divide-y-0">
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
            value={`${summary.currentStreak}d`}
            hint={`best ${summary.bestStreak}d`}
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
        </dl>
      )}

      <section>
        <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
          <div className="flex flex-wrap gap-1">
            {chartOptions.map(({ value, label, icon: Icon }) => (
              <button
                key={value}
                type="button"
                className={`flex h-8 items-center gap-1.5 rounded-[3px] px-2.5 font-mono text-[10px] uppercase tracking-[0.1em] transition-colors ${
                  chart === value
                    ? 'bg-base-content text-base-100'
                    : 'text-base-content/65 hover:bg-base-content/5 hover:text-base-content'
                }`}
                onClick={() => setChart(value)}
                aria-pressed={chart === value}
              >
                <Icon className="size-3.5" />
                <span className="hidden sm:inline">{label}</span>
              </button>
            ))}
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <select
              className="select select-sm w-auto rounded-[3px] border-base-300 font-mono text-[11px] uppercase tracking-[0.06em]"
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
              className="select select-sm w-auto rounded-[3px] border-base-300 font-mono text-[11px] uppercase tracking-[0.06em]"
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
                className="select select-sm w-auto rounded-[3px] border-base-300 font-mono text-[11px] uppercase tracking-[0.06em]"
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

        <div className="mt-4 h-72 w-full border-t border-base-300 pt-4 sm:h-80">
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
          <p className="num mt-2 text-[11px] uppercase tracking-[0.08em] text-base-content/65">
            on target in {summary.periodsCompleted} of {summary.periodsTotal} {tracker.goal.period}s
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

      <Modal
        open={shareOpen}
        title={tracker.role === 'member' ? 'Members' : 'Share this tracker'}
        onClose={() => setShareOpen(false)}
      >
        <TrackerSharing trackerId={tracker.id} readOnly={tracker.role === 'member'} />
      </Modal>
    </div>
  )
}

function StatBox({ label, value, hint }: { label: string; value: string; hint?: string }) {
  return (
    <div className="flex-1 py-3.5 sm:px-5 sm:first:pl-0 sm:last:pr-0">
      <dt className="eyebrow">{label}</dt>
      <dd className="num mt-1.5 truncate text-xl leading-none">{value}</dd>
      {hint && <p className="mt-1.5 truncate text-[11px] text-base-content/60">{hint}</p>}
    </div>
  )
}
