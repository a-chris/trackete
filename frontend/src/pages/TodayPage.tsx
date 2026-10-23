import { useEffect, useState } from 'react'
import { CalendarDays, Flame, Plus, Target } from 'lucide-react'
import { EmptyState } from '../components/ui/EmptyState'
import { Spinner } from '../components/ui/Spinner'
import { TodayTrackerCard } from '../components/trackers/TodayTrackerCard'
import { TrackerFormModal } from '../components/trackers/TrackerFormModal'
import { formatLongDate, todayKey } from '../lib/date'
import { useOverview } from '../lib/queries'

export function TodayPage() {
  const today = todayKey()
  const { data, isLoading } = useOverview(today)
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

  return (
    <div className="space-y-6">
      <header className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="flex items-center gap-1.5 text-xs font-medium uppercase tracking-wider text-base-content/50">
            <CalendarDays className="size-3.5" />
            {formatLongDate(today)}
          </p>
          <h1 className="mt-1 text-2xl font-semibold tracking-tight sm:text-3xl">Today</h1>
          <p className="mt-1 text-sm text-base-content/60">
            Log what happened, keep the streak alive.
          </p>
        </div>
        <button type="button" className="btn btn-primary" onClick={() => setCreateOpen(true)}>
          <Plus className="size-4" />
          New tracker
        </button>
      </header>

      {items.length > 0 && (
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
          <div className="surface flex items-center gap-3 p-4">
            <span className="grid size-9 place-items-center rounded-xl bg-primary/10 text-primary">
              <Target className="size-4" />
            </span>
            <div>
              <p className="text-lg font-semibold leading-none">
                {doneToday}
                <span className="text-base-content/40">/{items.length}</span>
              </p>
              <p className="mt-1 text-xs text-base-content/50">logged today</p>
            </div>
          </div>
          <div className="surface flex items-center gap-3 p-4">
            <span className="grid size-9 place-items-center rounded-xl bg-success/10 text-success">
              <Flame className="size-4" />
            </span>
            <div>
              <p className="text-lg font-semibold leading-none">{activeStreaks}</p>
              <p className="mt-1 text-xs text-base-content/50">active streaks</p>
            </div>
          </div>
          <div className="surface col-span-2 flex items-center gap-3 p-4 sm:col-span-1">
            <span className="grid size-9 place-items-center rounded-xl bg-warning/10 text-warning">
              <Flame className="size-4" />
            </span>
            <div>
              <p className="text-lg font-semibold leading-none">{bestStreak}</p>
              <p className="mt-1 text-xs text-base-content/50">best current streak (days)</p>
            </div>
          </div>
        </div>
      )}

      {isLoading ? (
        <Spinner label="Loading your trackers…" />
      ) : items.length === 0 ? (
        <EmptyState
          icon={Target}
          title="Nothing tracked yet"
          description="Start with one thing you want to keep an eye on — a habit, a number, or anything else. You can build a dashboard around it later."
          action={
            <button type="button" className="btn btn-primary" onClick={() => setCreateOpen(true)}>
              <Plus className="size-4" />
              Create your first tracker
            </button>
          }
        />
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          {items.map((item) => (
            <TodayTrackerCard key={item.tracker.id} item={item} />
          ))}
        </div>
      )}

      {createOpen && <TrackerFormModal onClose={() => setCreateOpen(false)} />}
    </div>
  )
}
