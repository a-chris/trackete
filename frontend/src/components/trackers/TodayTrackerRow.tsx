import { useState } from 'react'
import { Check, Minus, Plus, X } from 'lucide-react'
import { Link } from 'react-router-dom'
import { formatNumber, formatValue, goalLabel } from '../../lib/format'
import { todayKey } from '../../lib/date'
import { useEntryMutations } from '../../lib/queries'
import type { OverviewItem, Tracker } from '../../lib/types'

/**
 * One line of the day's ledger: what it is on the left, the control on the
 * right, and the objective for the period spelled out in between. No card —
 * a rule between rows is enough to hold the list together.
 */
export function TodayTrackerRow({ item }: { item: OverviewItem }) {
  const { tracker } = item
  const day = todayKey()
  const { save, remove } = useEntryMutations()

  const entryValue = item.entry?.value
  const isDone = (entryValue ?? 0) >= 0.5

  const target = item.periodTarget
  const percent = target && target > 0 ? Math.min(100, (item.periodTotal / target) * 100) : 0
  const onTarget = item.goalMet === true
  const overTarget = tracker.goal?.type === 'at_most' && item.goalMet === false
  const meta =
    goalLabel(tracker.goal, tracker.kind, tracker.unit) ??
    (tracker.kind === 'boolean' ? 'Yes / No' : tracker.unit ? `Daily ${tracker.unit}` : 'Number per day')
  // A yes/no habit with a daily target is either done or not: a 0–100% meter
  // would always read 100%. Only show the period count when it says something.
  const showPeriod = target !== null && tracker.goal && !(tracker.kind === 'boolean' && tracker.goal.period === 'day')

  return (
    <div className="flex items-center gap-3 py-3 sm:gap-4 sm:py-3.5">
      <span
        className="grid size-8 shrink-0 place-items-center rounded-[3px] text-[15px] leading-none"
        style={{ backgroundColor: `${tracker.color}1f` }}
        aria-hidden="true"
      >
        {tracker.icon}
      </span>

      <div className="min-w-0 flex-1">
        <div className="flex items-baseline gap-2">
          <Link
            to={`/trackers/${tracker.id}`}
            className="truncate text-[15px] font-medium leading-tight hover:underline hover:decoration-base-content/30 hover:underline-offset-4"
            title={tracker.name}
          >
            {tracker.name}
          </Link>
          {item.streak > 0 && (
            <span className="num shrink-0 text-[11px] text-secondary" title={`${item.streak} day streak`}>
              {item.streak}d
            </span>
          )}
        </div>

        <div className="mt-1 flex items-center gap-2">
          <p className="eyebrow truncate text-[10px] text-base-content/65">{meta}</p>
          {showPeriod && (
            <span className="ml-auto flex shrink-0 items-center gap-1.5">
              <span className="hidden h-[2px] w-8 bg-base-300 sm:block" aria-hidden="true">
                <span
                  className={`block h-full ${
                    overTarget ? 'bg-error' : onTarget ? 'bg-success' : 'bg-base-content'
                  }`}
                  style={{ width: `${percent}%` }}
                />
              </span>
              <span
                className={`num text-[10px] ${
                  overTarget ? 'text-error' : onTarget ? 'text-success' : 'text-base-content/65'
                }`}
                title={`${formatValue(item.periodTotal, tracker.unit, tracker.kind)} of ${formatValue(
                  target ?? 0,
                  tracker.unit,
                  tracker.kind,
                )} this ${tracker.goal?.period}`}
              >
                {formatNumber(item.periodTotal)}/{formatNumber(target ?? 0)}
              </span>
            </span>
          )}
        </div>
      </div>

      <div className="flex shrink-0 items-center gap-1.5">
        {tracker.kind === 'boolean' ? (
          <button
            type="button"
            className={`grid size-9 place-items-center rounded-[3px] border transition-colors ${
              isDone
                ? 'border-base-content bg-base-content text-base-100'
                : 'border-base-300 text-base-content/55 hover:border-base-content/50 hover:text-base-content/70'
            }`}
            onClick={() => save.mutate({ trackerId: tracker.id, day, value: isDone ? 0 : 1 })}
            disabled={save.isPending}
            aria-pressed={isDone}
            aria-label={isDone ? `Mark ${tracker.name} undone` : `Mark ${tracker.name} done today`}
            title={isDone ? 'Done today' : 'Mark as done'}
          >
            {save.isPending ? (
              <span className="loading loading-spinner loading-xs" />
            ) : (
              <Check className="size-4" strokeWidth={2.5} />
            )}
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
      </div>
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
    <div className="flex items-center gap-1">
      <button
        type="button"
        className="grid size-8 place-items-center rounded-[3px] border border-base-300 text-base-content/70 transition-colors hover:border-base-content/40 hover:text-base-content disabled:opacity-40"
        onClick={() => saveValue(currentValue - 1)}
        disabled={pending}
        aria-label="Decrease"
      >
        <Minus className="size-3.5" />
      </button>
      <div className="relative">
        <input
          type="number"
          inputMode="decimal"
          className="input input-sm num h-8 w-14 rounded-[3px] border-base-300 px-1 text-center text-[13px] focus:border-base-content sm:w-20"
          value={draft}
          placeholder="—"
          step="any"
          min={0}
          onChange={(event) => setDraft(event.target.value)}
          onBlur={commitDraft}
          onKeyDown={(event) => {
            if (event.key === 'Enter') event.currentTarget.blur()
          }}
          aria-label={`${tracker.name} today`}
          title={tracker.unit ? `In ${tracker.unit}` : undefined}
        />
      </div>
      <button
        type="button"
        className="grid size-8 place-items-center rounded-[3px] border border-base-300 text-base-content/70 transition-colors hover:border-base-content/40 hover:text-base-content disabled:opacity-40"
        onClick={() => saveValue(currentValue + 1)}
        disabled={pending}
        aria-label="Increase"
      >
        <Plus className="size-3.5" />
      </button>
      {entryValue !== undefined && (
        <button
          type="button"
          className="hidden size-7 place-items-center rounded-[3px] text-base-content/55 transition-colors hover:text-error sm:grid"
          onClick={onClear}
          disabled={pending}
          aria-label="Clear today's entry"
          title="Clear today's entry"
        >
          <X className="size-3" />
        </button>
      )}
    </div>
  )
}
