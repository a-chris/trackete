import { useState, type FormEvent } from 'react'
import { Pencil, RotateCw, Save, Trash } from 'lucide-react'
import { Spinner } from '../ui/Spinner'
import { useToast } from '../ui/Toast'
import { errorMessage } from '../../lib/api'
import { formatDate, todayKey } from '../../lib/date'
import { formatValue } from '../../lib/format'
import { useEntryMutations } from '../../lib/queries'
import type { Entry, Tracker } from '../../lib/types'

const entryLabel = 'mb-1 block font-mono text-[10px] uppercase tracking-[0.14em] text-base-content/40'

interface EntryEditorProps {
  tracker: Tracker
  entries: Entry[]
  loading: boolean
  error?: boolean
  errorDetail?: string
  onRetry?: () => void
  retrying?: boolean
}

export function EntryEditor({
  tracker,
  entries,
  loading,
  error = false,
  errorDetail,
  onRetry,
  retrying = false,
}: EntryEditorProps) {
  const toast = useToast()
  const { save, remove } = useEntryMutations()
  const today = todayKey()

  const [day, setDay] = useState(today)
  const [value, setValue] = useState(tracker.kind === 'boolean' ? '1' : '')
  const [note, setNote] = useState('')

  const upsert = (event: FormEvent) => {
    event.preventDefault()
    const parsed = tracker.kind === 'boolean' ? (Number(value) > 0 ? 1 : 0) : Number(value)
    if (!Number.isFinite(parsed)) {
      toast.push('error', 'Enter a valid number.')
      return
    }
    save.mutate(
      { trackerId: tracker.id, day, value: parsed, note },
      {
        onSuccess: () => {
          toast.push('success', `Saved ${formatDate(day)}.`)
          setNote('')
          if (tracker.kind === 'numeric') setValue(String(parsed))
        },
        onError: (error) => toast.push('error', errorMessage(error)),
      },
    )
  }

  return (
    <section className="surface overflow-hidden">
      <div className="flex flex-col gap-3 border-b border-base-300 px-0 py-4 sm:py-5">
        <div>
          <h2 className="eyebrow">Add or update a day</h2>
          <p className="mt-1 text-xs text-base-content/40">
            One value per day — saving the same date replaces the previous value.
          </p>
        </div>
        <form className="flex flex-wrap items-end gap-2" onSubmit={upsert}>
          <label className="form-control">
            <span className={entryLabel}>Day</span>
            <input
              type="date"
              className="input input-sm num rounded-[3px] border-base-300"
              value={day}
              max={today}
              onChange={(event) => setDay(event.target.value)}
              required
            />
          </label>
          <label className="form-control">
            <span className={entryLabel}>
              {tracker.kind === 'boolean' ? 'Done?' : 'Value'}
            </span>
            {tracker.kind === 'boolean' ? (
              <select
                className="select select-sm rounded-[3px] border-base-300"
                value={value || '1'}
                onChange={(event) => setValue(event.target.value)}
              >
                <option value="1">Yes</option>
                <option value="0">No</option>
              </select>
            ) : (
              <input
                type="number"
                className="input input-sm num w-28 rounded-[3px] border-base-300"
                value={value}
                onChange={(event) => setValue(event.target.value)}
                placeholder="0"
                step="any"
                min={0}
                required
              />
            )}
          </label>
          <label className="form-control min-w-40 flex-1">
            <span className={entryLabel}>Note</span>
            <input
              type="text"
              className="input input-sm w-full rounded-[3px] border-base-300"
              value={note}
              onChange={(event) => setNote(event.target.value)}
              placeholder="Optional"
              maxLength={500}
            />
          </label>
          <button type="submit" className="btn btn-primary btn-sm gap-2" disabled={save.isPending}>
            {save.isPending ? (
              <span className="loading loading-spinner loading-xs" />
            ) : (
              <Save className="size-3.5" />
            )}
            Save
          </button>
        </form>
      </div>

      {loading ? (
        <Spinner />
      ) : error ? (
        <div
          role="alert"
          className="flex flex-wrap items-center justify-center gap-3 p-6 text-center text-sm text-base-content/60"
        >
          <span>Couldn't load entries.{errorDetail ? ` ${errorDetail}` : ''}</span>
          {onRetry && (
            <button type="button" className="btn btn-xs btn-outline" onClick={onRetry} disabled={retrying}>
              {retrying ? <span className="loading loading-spinner loading-xs" /> : <RotateCw className="size-3.5" />}
              Retry
            </button>
          )}
        </div>
      ) : entries.length === 0 ? (
        <p className="p-6 text-center text-sm text-base-content/50">No entries in this range yet.</p>
      ) : (
        <div className="max-h-96 overflow-auto">
          <table className="table table-sm">
            <thead className="sticky top-0 z-10 bg-base-100">
              <tr className="border-base-300">
                <th className="eyebrow font-normal">Day</th>
                <th className="eyebrow text-right font-normal">Value</th>
                <th className="eyebrow font-normal">Note</th>
                <th className="w-12" />
              </tr>
            </thead>
            <tbody>
              {[...entries].reverse().map((entry) => (
                <tr key={entry.day} className="border-base-300">
                  <td className="num whitespace-nowrap text-[12px] text-base-content/70">
                    {formatDate(entry.day, { weekday: 'short', month: 'short', day: 'numeric' })}
                  </td>
                  <td className="num text-right text-[12px]">
                    {formatValue(entry.value, tracker.unit, tracker.kind)}
                  </td>
                  <td className="max-w-72 truncate text-xs text-base-content/50">{entry.note || '—'}</td>
                  <td>
                    <div className="flex justify-end gap-0.5">
                      <button
                        type="button"
                        className="grid size-7 place-items-center rounded-[3px] text-base-content/40 transition-colors hover:bg-base-content/5 hover:text-base-content"
                        onClick={() => {
                          setDay(entry.day)
                          setValue(String(entry.value))
                          setNote(entry.note)
                        }}
                        aria-label={`Edit ${entry.day}`}
                      >
                        <Pencil className="size-3" />
                      </button>
                      <button
                        type="button"
                        className="grid size-7 place-items-center rounded-[3px] text-base-content/40 transition-colors hover:bg-error/10 hover:text-error"
                        onClick={() =>
                          remove.mutate(
                            { trackerId: tracker.id, day: entry.day },
                            {
                              onSuccess: () => toast.push('success', 'Entry deleted.'),
                              onError: (error) => toast.push('error', errorMessage(error)),
                            },
                          )
                        }
                        aria-label={`Delete ${entry.day}`}
                      >
                        <Trash className="size-3" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  )
}
