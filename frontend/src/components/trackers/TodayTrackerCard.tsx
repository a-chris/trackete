import { useState } from 'react'
import { Check, ChevronRight, Flame, Minus, Plus, X } from 'lucide-react'
import { Link } from 'react-router-dom'
import { formatValue, goalLabel } from '../../lib/format'
import { todayKey } from '../../lib/date'
import { useEntryMutations } from '../../lib/queries'
import type { OverviewItem, Tracker } from '../../lib/types'

export function TodayTrackerCard({ item }: { item: OverviewItem }) {
  const { tracker } = item
  const day = todayKey()
  const { save, remove } = useEntryMutations()

  const entryValue = item.entry?.value
  const isDone = (entryValue ?? 0) >= 0.5

  const target = item.periodTarget
  const percent = target && target > 0 ? Math.min(100, (item.periodTotal / target) * 100) : 0
  const onTarget = item.goalMet === true
  const overTarget = tracker.goal?.type === 'at_most' && item.goalMet === false

  return (
    <div className="surface flex flex-col gap-3 p-4">
      <div className="flex items-start gap-3">
        <div
          className="grid size-10 shrink-0 place-items-center rounded-xl text-lg"
          style={{ backgroundColor: `${tracker.color}22` }}
          aria-hidden="true"
        >
          {tracker.icon}
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5">
            <Link
              to={`/trackers/${tracker.id}`}
              className="truncate font-medium hover:underline"
              title={tracker.name}
            >
              {tracker.name}
            </Link>
            {item.streak > 0 && (
              <span className="badge badge-ghost badge-sm gap-1 text-[10px]">
                <Flame className="size-3 text-warning" />
                {item.streak}
              </span>
            )}
          </div>
          <p className="truncate text-xs text-base-content/50">
            {goalLabel(tracker.goal, tracker.kind, tracker.unit) ??
              (tracker.kind === 'boolean'
                ? 'Yes / No'
                : tracker.unit
                  ? `Daily ${tracker.unit}`
                  : 'Number per day')}
          </p>
        </div>
        <Link
          to={`/trackers/${tracker.id}`}
          className="btn btn-ghost btn-xs btn-circle shrink-0"
          aria-label={`Open ${tracker.name}`}
        >
          <ChevronRight className="size-4" />
        </Link>
      </div>

      {tracker.kind === 'boolean' ? (
        <button
          type="button"
          className={`btn w-full ${isDone ? 'btn-success' : 'btn-outline'}`}
          onClick={() => save.mutate({ trackerId: tracker.id, day, value: isDone ? 0 : 1 })}
          disabled={save.isPending}
          aria-pressed={isDone}
        >
          {save.isPending ? (
            <span className="loading loading-spinner loading-xs" />
          ) : isDone ? (
            <Check className="size-4" />
          ) : (
            <Plus className="size-4" />
          )}
          {isDone ? 'Done today' : 'Mark as done'}
        </button>
      ) : (
        <NumericControl
          key={`${tracker.id}:${entryValue ?? 'empty'}`}
          tracker={tracker}
          entryValue={entryValue}
          pending={save.isPending || remove.isPending}
          onSave={(value) => save.mutate({ trackerId: tracker.id, day, value })}
          onClear={() => remove.mutate({ trackerId: tracker.id, day })}
        />
      )}

      {target !== null && tracker.goal && (
        <div>
          <div className="flex items-baseline justify-between gap-2 text-[11px] text-base-content/60">
            <span>
              {formatValue(item.periodTotal, tracker.unit, tracker.kind)} this {tracker.goal.period}
            </span>
            <span
              className={
                overTarget ? 'font-medium text-error' : onTarget ? 'font-medium text-success' : ''
              }
            >
              {tracker.goal.type === 'at_most' ? 'limit' : 'goal'}{' '}
              {formatValue(target, tracker.unit, tracker.kind)}
            </span>
          </div>
          <progress
            className={`progress mt-1.5 h-1.5 ${
              overTarget ? 'progress-error' : onTarget ? 'progress-success' : 'progress-primary'
            }`}
            value={percent}
            max={100}
          />
        </div>
      )}

      {item.entry?.note && (
        <p className="truncate border-t border-base-300 pt-2 text-xs text-base-content/50">
          {item.entry.note}
        </p>
      )}
    </div>
  )
}

interface NumericControlProps {
  tracker: Tracker
  entryValue: number | undefined
  pending: boolean
  onSave: (value: number) => void
  onClear: () => void
}

function NumericControl({ tracker, entryValue, pending, onSave, onClear }: NumericControlProps) {
  const [draft, setDraft] = useState(entryValue !== undefined ? String(entryValue) : '')

  const currentValue =
    draft.trim() !== '' && Number.isFinite(Number(draft)) ? Number(draft) : (entryValue ?? 0)

  const saveValue = (next: number) => {
    const value = Math.max(0, next)
    setDraft(String(value))
    onSave(value)
  }

  const commitDraft = () => {
    if (draft.trim() === '') {
      if (entryValue !== undefined) onClear()
      return
    }
    const parsed = Number(draft)
    if (!Number.isFinite(parsed)) {
      setDraft(entryValue !== undefined ? String(entryValue) : '')
      return
    }
    if (parsed !== entryValue) saveValue(parsed)
  }

  return (
    <div className="flex items-center gap-2">
      <button
        type="button"
        className="btn btn-square btn-sm btn-outline"
        onClick={() => saveValue(currentValue - 1)}
        disabled={pending}
        aria-label="Decrease"
      >
        <Minus className="size-3.5" />
      </button>
      <div className="relative flex-1">
        <input
          type="number"
          inputMode="decimal"
          className="input input-sm w-full pr-14 text-center tabular-nums"
          value={draft}
          placeholder="0"
          step="any"
          min={0}
          onChange={(event) => setDraft(event.target.value)}
          onBlur={commitDraft}
          onKeyDown={(event) => {
            if (event.key === 'Enter') event.currentTarget.blur()
          }}
          aria-label={`${tracker.name} today`}
        />
        {tracker.unit && (
          <span className="pointer-events-none absolute right-2.5 top-1/2 -translate-y-1/2 truncate text-[11px] text-base-content/40">
            {tracker.unit}
          </span>
        )}
      </div>
      <button
        type="button"
        className="btn btn-square btn-sm btn-outline"
        onClick={() => saveValue(currentValue + 1)}
        disabled={pending}
        aria-label="Increase"
      >
        <Plus className="size-3.5" />
      </button>
      {entryValue !== undefined && (
        <button
          type="button"
          className="btn btn-ghost btn-square btn-sm"
          onClick={onClear}
          disabled={pending}
          aria-label="Clear today's entry"
          title="Clear today's entry"
        >
          <X className="size-3.5" />
        </button>
      )}
    </div>
  )
}
