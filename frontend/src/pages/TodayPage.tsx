import { useEffect, useState } from 'react'
import { Plus, Target } from 'lucide-react'
import { EmptyState } from '../components/ui/EmptyState'
import { ErrorState } from '../components/ui/ErrorState'
import { Spinner } from '../components/ui/Spinner'
import { TodayTrackerRow } from '../components/trackers/TodayTrackerRow'
import { TrackerFormModal } from '../components/trackers/TrackerFormModal'
import { errorMessage } from '../lib/api'
import { formatLongDate, todayKey } from '../lib/date'
import { useOverview } from '../lib/queries'

export function TodayPage() {
  const today = todayKey()
  const { data, isLoading, isError, error, refetch, isFetching } = useOverview(today)
  const [createOpen, setCreateOpen] = useState(false)

  useEffect(() => {
    document.title = 'Today · Trackete'
  }, [])

  const items = data?.items ?? []
  const doneToday = items.filter((item) => {
    if (item.tracker.kind === 'boolean') return (item.entry?.value ?? 0) >= 0.5
    return Boolean(item.entry)
  }).length
  const activeStreaks = items.filter((item) => item.streak > 0).length
  const bestStreak = items.reduce((max, item) => Math.max(max, item.streak), 0)
  const remaining = items.length - doneToday

  return (
    <div>
      <header className="border-b border-base-300 pb-6">
        <div className="flex flex-col gap-5 sm:flex-row sm:items-end sm:justify-between sm:gap-x-8">
          <div className="min-w-0">
            <p className="eyebrow">{formatLongDate(today)}</p>
            <h1 className="display mt-2 text-[40px] leading-none sm:text-5xl">Today</h1>
          </div>

          <div className="flex items-end justify-between gap-6 sm:justify-end sm:gap-8">
            {items.length > 0 && (
              <div className="flex items-end gap-6 sm:gap-8">
                <Readout label="logged" value={`${doneToday}/${items.length}`} />
                <Readout label="streaks" value={String(activeStreaks)} />
                <Readout label="best" value={`${bestStreak}d`} muted />
              </div>
            )}

            <button
              type="button"
              className="btn btn-sm btn-primary shrink-0 gap-2"
              onClick={() => setCreateOpen(true)}
            >
              <Plus className="size-4" />
              New tracker
            </button>
          </div>
        </div>

        {items.length > 0 && (
          <p className="mt-5 text-sm text-base-content/70">
            {remaining === 0
              ? 'Everything is logged. That is the whole day closed out.'
              : `${remaining} still open${remaining === 1 ? '' : 's'} — log it and keep the run going.`}
          </p>
        )}
      </header>

      {isLoading ? (
        <div className="pt-16">
          <Spinner label="Loading your trackers…" />
        </div>
      ) : isError ? (
        <div className="pt-8">
          <ErrorState
            title="Couldn't load your day"
            description={errorMessage(error)}
            onRetry={() => void refetch()}
            retrying={isFetching}
          />
        </div>
      ) : items.length === 0 ? (
        <div className="pt-4">
          <EmptyState
            icon={Target}
            title="Nothing tracked yet"
            description="Start with one thing you want to keep an eye on — a habit, a number, or anything else. You can build a dashboard around it later."
            action={
              <button type="button" className="btn btn-primary gap-2" onClick={() => setCreateOpen(true)}>
                <Plus className="size-4" />
                Create your first tracker
              </button>
            }
          />
        </div>
      ) : (
        <div className="divide-y divide-base-300">
          {items.map((item) => (
            <TodayTrackerRow key={item.tracker.id} item={item} />
          ))}
        </div>
      )}

      {createOpen && <TrackerFormModal onClose={() => setCreateOpen(false)} />}
    </div>
  )
}

function Readout({ label, value, muted = false }: { label: string; value: string; muted?: boolean }) {
  return (
    <div>
      <p className="eyebrow">{label}</p>
      <p
        className={`num mt-1.5 text-2xl leading-none ${muted ? 'text-base-content/65' : 'text-base-content'}`}
      >
        {value}
      </p>
    </div>
  )
}
