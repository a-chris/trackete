import { useState, type FormEvent } from 'react'
import { Goal, Hash, ToggleLeft } from 'lucide-react'
import { Modal } from '../ui/Modal'
import { useToast } from '../ui/Toast'
import { errorMessage } from '../../lib/api'
import { useTrackerMutations } from '../../lib/queries'
import { TRACKER_COLORS, TRACKER_ICONS } from '../../lib/palette'
import type { GoalPeriod, GoalType, Tracker, TrackerInput, TrackerKind } from '../../lib/types'

interface TrackerFormModalProps {
  tracker?: Tracker | null
  onClose: () => void
}

const periodOptions: { value: GoalPeriod; label: string }[] = [
  { value: 'day', label: 'per day' },
  { value: 'week', label: 'per week' },
  { value: 'month', label: 'per month' },
]

export function TrackerFormModal({ tracker, onClose }: TrackerFormModalProps) {
  const toast = useToast()
  const { create, update } = useTrackerMutations()

  // Mount this component only while the modal is open: the state below is the
  // initial state of one editing session.
  const [name, setName] = useState(tracker?.name ?? '')
  const [description, setDescription] = useState(tracker?.description ?? '')
  const [icon, setIcon] = useState(tracker?.icon || '🎯')
  const [color, setColor] = useState(tracker?.color || TRACKER_COLORS[0].value)
  const [kind, setKind] = useState<TrackerKind>(tracker?.kind ?? 'boolean')
  const [unit, setUnit] = useState(tracker?.unit ?? '')
  const [hasGoal, setHasGoal] = useState(Boolean(tracker?.goal))
  const [goalType, setGoalType] = useState<GoalType>(tracker?.goal?.type ?? 'at_least')
  const [goalValue, setGoalValue] = useState(tracker?.goal ? String(tracker.goal.value) : '1')
  const [goalPeriod, setGoalPeriod] = useState<GoalPeriod>(tracker?.goal?.period ?? 'day')
  const [error, setError] = useState('')

  const pending = create.isPending || update.isPending

  const onSubmit = (event: FormEvent) => {
    event.preventDefault()
    setError('')

    if (!name.trim()) {
      setError('Give your tracker a name.')
      return
    }

    const numericGoal = Number(goalValue)
    if (hasGoal && (!Number.isFinite(numericGoal) || numericGoal < 0)) {
      setError('Goal value must be a positive number.')
      return
    }

    const input: TrackerInput = {
      name: name.trim(),
      description: description.trim(),
      icon,
      color,
      kind,
      unit: kind === 'numeric' ? unit.trim() : '',
      goal: hasGoal
        ? {
            type: goalType,
            value: kind === 'boolean' ? Math.max(1, numericGoal || 1) : numericGoal,
            period: goalPeriod,
          }
        : null,
      archived: tracker?.archived ?? false,
    }

    const onSuccess = () => {
      toast.push('success', tracker ? 'Tracker updated.' : 'Tracker created.')
      onClose()
    }
    const onError = (err: unknown) => setError(errorMessage(err))

    if (tracker) {
      update.mutate({ id: tracker.id, input }, { onSuccess, onError })
    } else {
      create.mutate(input, { onSuccess, onError })
    }
  }

  return (
    <Modal
      open
      title={tracker ? 'Edit tracker' : 'New tracker'}
      onClose={onClose}
      wide
      footer={
        <>
          <button type="button" className="btn btn-ghost" onClick={onClose} disabled={pending}>
            Cancel
          </button>
          <button type="submit" form="tracker-form" className="btn btn-primary" disabled={pending}>
            {pending && <span className="loading loading-spinner loading-sm" />}
            {tracker ? 'Save changes' : 'Create tracker'}
          </button>
        </>
      }
    >
      <form id="tracker-form" className="space-y-5" onSubmit={onSubmit}>
        {error && <div className="alert alert-error text-sm">{error}</div>}

        <div className="grid gap-4 sm:grid-cols-[1fr_auto]">
          <label className="form-control">
            <span className="mb-1.5 block text-sm font-medium">Name</span>
            <input
              className="input input-bordered w-full"
              value={name}
              onChange={(event) => setName(event.target.value)}
              placeholder="Beers, Weight, Meditation…"
              maxLength={80}
              autoFocus
              required
            />
          </label>
          <div className="form-control">
            <span className="mb-1.5 block text-sm font-medium">Icon</span>
            <div className="flex flex-wrap gap-1.5">
              {TRACKER_ICONS.slice(0, 8).map((emoji) => (
                <button
                  key={emoji}
                  type="button"
                  onClick={() => setIcon(emoji)}
                  className={`grid size-9 place-items-center rounded-lg border text-lg transition ${
                    icon === emoji ? 'border-primary bg-primary/10' : 'border-base-300 hover:bg-base-200'
                  }`}
                  aria-label={`Use ${emoji}`}
                  aria-pressed={icon === emoji}
                >
                  {emoji}
                </button>
              ))}
            </div>
          </div>
        </div>

        <label className="form-control">
          <span className="mb-1.5 block text-sm font-medium">
            Description <span className="font-normal text-base-content/40">(optional)</span>
          </span>
          <input
            className="input input-bordered w-full"
            value={description}
            onChange={(event) => setDescription(event.target.value)}
            placeholder="What exactly are you tracking?"
            maxLength={500}
          />
        </label>

        <div>
          <span className="mb-2 block text-sm font-medium">What kind of data?</span>
          <div className="grid gap-3 sm:grid-cols-2">
            <button
              type="button"
              onClick={() => {
                setKind('boolean')
                setUnit('')
                setGoalPeriod('day')
                setGoalValue('1')
              }}
              className={`rounded-box border p-4 text-left transition ${
                kind === 'boolean'
                  ? 'border-primary bg-primary/5 ring-1 ring-primary'
                  : 'border-base-300 hover:border-base-content/30'
              }`}
              aria-pressed={kind === 'boolean'}
            >
              <ToggleLeft className="size-5 text-primary" />
              <p className="mt-2 font-medium">Yes / No</p>
              <p className="mt-0.5 text-xs text-base-content/60">
                A habit you did or did not do today.
              </p>
            </button>
            <button
              type="button"
              onClick={() => setKind('numeric')}
              className={`rounded-box border p-4 text-left transition ${
                kind === 'numeric'
                  ? 'border-primary bg-primary/5 ring-1 ring-primary'
                  : 'border-base-300 hover:border-base-content/30'
              }`}
              aria-pressed={kind === 'numeric'}
            >
              <Hash className="size-5 text-primary" />
              <p className="mt-2 font-medium">Number</p>
              <p className="mt-0.5 text-xs text-base-content/60">
                A value per day, like 3 beers, 72.5 kg or 30 minutes.
              </p>
            </button>
          </div>
        </div>

        {kind === 'numeric' && (
          <label className="form-control">
            <span className="mb-1.5 block text-sm font-medium">
              Unit <span className="font-normal text-base-content/40">(optional)</span>
            </span>
            <input
              className="input input-bordered w-full sm:max-w-xs"
              value={unit}
              onChange={(event) => setUnit(event.target.value)}
              placeholder="beers, kg, minutes…"
              maxLength={16}
            />
          </label>
        )}

        <div className="rounded-box border border-base-300 bg-base-200/40 p-4">
          <label className="flex cursor-pointer items-center justify-between gap-4">
            <span className="flex items-center gap-2 text-sm font-medium">
              <Goal className="size-4 text-primary" />
              Add a goal
            </span>
            <input
              type="checkbox"
              className="toggle toggle-primary toggle-sm"
              checked={hasGoal}
              onChange={(event) => setHasGoal(event.target.checked)}
            />
          </label>

          {hasGoal && (
            <div className="mt-4 grid gap-3 sm:grid-cols-[auto_1fr_1fr]">
              <label className="form-control">
                <span className="mb-1 block text-xs font-medium text-base-content/60">Target</span>
                <select
                  className="select select-bordered select-sm"
                  value={goalType}
                  onChange={(event) => setGoalType(event.target.value as GoalType)}
                >
                  <option value="at_least">at least</option>
                  <option value="at_most">at most</option>
                </select>
              </label>
              <label className="form-control">
                <span className="mb-1 block text-xs font-medium text-base-content/60">
                  Value {unit && <span className="text-base-content/40">({unit})</span>}
                </span>
                <input
                  type="number"
                  className="input input-bordered input-sm w-full"
                  value={goalValue}
                  onChange={(event) => setGoalValue(event.target.value)}
                  min={0}
                  step="any"
                />
              </label>
              <label className="form-control">
                <span className="mb-1 block text-xs font-medium text-base-content/60">Period</span>
                <select
                  className="select select-bordered select-sm"
                  value={goalPeriod}
                  onChange={(event) => setGoalPeriod(event.target.value as GoalPeriod)}
                >
                  {periodOptions.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
              </label>
              <p className="text-xs text-base-content/50 sm:col-span-3">
                Example: “at most 5 per week” for beers, or “at least 1 per day” for meditation.
              </p>
            </div>
          )}
        </div>

        <div>
          <span className="mb-2 block text-sm font-medium">Colour</span>
          <div className="flex flex-wrap gap-2">
            {TRACKER_COLORS.map((option) => (
              <button
                key={option.value}
                type="button"
                onClick={() => setColor(option.value)}
                className={`size-8 rounded-full border-2 transition ${
                  color === option.value ? 'border-base-content scale-110' : 'border-transparent'
                }`}
                style={{ backgroundColor: option.value }}
                aria-label={option.name}
                aria-pressed={color === option.value}
              />
            ))}
          </div>
        </div>
      </form>
    </Modal>
  )
}
