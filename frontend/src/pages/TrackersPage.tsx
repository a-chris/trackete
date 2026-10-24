import { useEffect, useState } from 'react'
import { Archive, ArchiveRestore, ListChecks, Pencil, Plus, Trash } from 'lucide-react'
import { Link } from 'react-router-dom'
import { EmptyState } from '../components/ui/EmptyState'
import { ErrorState } from '../components/ui/ErrorState'
import { Spinner } from '../components/ui/Spinner'
import { ConfirmDialog } from '../components/ui/ConfirmDialog'
import { useToast } from '../components/ui/Toast'
import { TrackerFormModal } from '../components/trackers/TrackerFormModal'
import { errorMessage } from '../lib/api'
import { useTrackerMutations, useTrackers } from '../lib/queries'
import { formatDate } from '../lib/date'
import { goalLabel } from '../lib/format'
import type { Tracker } from '../lib/types'
import { trackerToInput } from '../lib/trackers'

// The row says who else logs here and what the tracker asks of you. Segments
// that do not apply are left out instead of announced as missing.
function trackerLines(tracker: Tracker): { shared: string | null; meta: string[] } {
  const shared = tracker.sharedWith ?? []
  return {
    shared: shared.length > 0 ? `Shared with ${shared.join(', ')}` : null,
    meta: [
      goalLabel(tracker.goal, tracker.kind, tracker.unit) ??
        (tracker.kind === 'boolean' ? null : tracker.unit || null),
      `since ${formatDate(tracker.createdAt.slice(0, 10), { month: 'short', day: 'numeric', year: 'numeric' })}`,
    ].filter((part): part is string => Boolean(part)),
  }
}

export function TrackersPage() {
  const toast = useToast()
  const { data: trackers = [], isLoading, isError, error, refetch, isFetching } = useTrackers(true)
  const { update, remove } = useTrackerMutations()
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<Tracker | null>(null)
  const [deleting, setDeleting] = useState<Tracker | null>(null)

  useEffect(() => {
    document.title = 'Trackers · Trackete'
  }, [])

  const active = trackers.filter((tracker) => !tracker.archived)
  const archived = trackers.filter((tracker) => tracker.archived)

  const toggleArchive = (tracker: Tracker) => {
    update.mutate(
      { id: tracker.id, input: { ...trackerToInput(tracker), archived: !tracker.archived } },
      {
        onSuccess: () =>
          toast.push('success', tracker.archived ? 'Tracker restored.' : 'Tracker archived.'),
        onError: (error) => toast.push('error', errorMessage(error)),
      },
    )
  }

  const confirmDelete = () => {
    if (!deleting) return
    remove.mutate(deleting.id, {
      onSuccess: () => {
        toast.push('success', `Deleted “${deleting.name}” and all its entries.`)
        setDeleting(null)
      },
      onError: (error) => toast.push('error', errorMessage(error)),
    })
  }

  const renderRow = (tracker: Tracker) => {
    const { shared, meta } = trackerLines(tracker)
    return (
      <div
        key={tracker.id}
        className={`group flex items-center gap-3 py-3 sm:gap-4 sm:py-3.5 ${
          tracker.archived ? 'opacity-70' : ''
        }`}
      >
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
            <span className="eyebrow shrink-0 font-semibold text-base-content/80">
              {tracker.kind === 'boolean' ? 'yes/no' : 'number'}
            </span>
            {tracker.role === 'member' && (
              <span className="eyebrow shrink-0 font-semibold text-base-content/80">shared</span>
            )}
          </div>
          {shared && (
            <p className="eyebrow mt-1 truncate font-semibold text-base-content/80">{shared}</p>
          )}
          <p
            className={`eyebrow ${shared ? 'mt-0.5' : 'mt-1'} truncate font-semibold text-base-content/80`}
          >
            {meta.map((part, index) => (
              <span key={part}>
                {index > 0 && <span className="mx-1.5 text-base-content/40">/</span>}
                {part}
              </span>
            ))}
          </p>
          {tracker.description && (
            <p className="mt-1 line-clamp-1 text-xs text-base-content/65">{tracker.description}</p>
          )}
        </div>

        {/* Metadata, archiving and deletion belong to the owner; a member only
            sees their own entries and the charts. */}
        {tracker.role !== 'member' && (
        <div className="flex shrink-0 items-center gap-0.5">
          <button
            type="button"
            className="grid size-8 place-items-center rounded-[3px] text-base-content/65 transition-colors hover:bg-base-content/5 hover:text-base-content"
            onClick={() => {
              setEditing(tracker)
              setModalOpen(true)
            }}
            title="Edit tracker"
            aria-label={`Edit ${tracker.name}`}
          >
            <Pencil className="size-3.5" />
          </button>
          <button
            type="button"
            className="grid size-8 place-items-center rounded-[3px] text-base-content/65 transition-colors hover:bg-base-content/5 hover:text-base-content"
            onClick={() => toggleArchive(tracker)}
            title={tracker.archived ? 'Restore tracker' : 'Archive tracker'}
            aria-label={tracker.archived ? `Restore ${tracker.name}` : `Archive ${tracker.name}`}
          >
            {tracker.archived ? (
              <ArchiveRestore className="size-3.5" />
            ) : (
              <Archive className="size-3.5" />
            )}
          </button>
          <button
            type="button"
            className="grid size-8 place-items-center rounded-[3px] text-base-content/65 transition-colors hover:bg-error/10 hover:text-error"
            onClick={() => setDeleting(tracker)}
            title="Delete tracker"
            aria-label={`Delete ${tracker.name}`}
          >
            <Trash className="size-3.5" />
          </button>
        </div>
        )}
      </div>
    )
  }

  return (
    <div className="space-y-8">
      <header className="flex flex-wrap items-end justify-between gap-x-8 gap-y-5 border-b border-base-300 pb-6">
        <div>
          <p className="eyebrow">Everything you log</p>
          <h1 className="display mt-2 text-[40px] leading-none sm:text-5xl">Trackers</h1>
        </div>
        <div className="flex items-end gap-6 sm:gap-8">
          <div>
            <p className="eyebrow">active</p>
            <p className="num mt-1.5 text-2xl leading-none">{active.length}</p>
          </div>
          <div>
            <p className="eyebrow">archived</p>
            <p className="num mt-1.5 text-2xl leading-none text-base-content/65">{archived.length}</p>
          </div>
        </div>
        <button
          type="button"
          className="btn btn-sm btn-primary gap-2"
          onClick={() => {
            setEditing(null)
            setModalOpen(true)
          }}
        >
          <Plus className="size-4" />
          New tracker
        </button>
      </header>

      {isLoading ? (
        <Spinner label="Loading trackers…" />
      ) : isError ? (
        <ErrorState
          title="Couldn't load your trackers"
          description={errorMessage(error)}
          onRetry={() => void refetch()}
          retrying={isFetching}
        />
      ) : active.length === 0 && archived.length === 0 ? (
        <EmptyState
          icon={ListChecks}
          title="No trackers yet"
          description="A tracker is one thing you log: a yes/no habit or a number with a unit. Create one to get started."
          action={
            <button
              type="button"
              className="btn btn-primary gap-2"
              onClick={() => {
                setEditing(null)
                setModalOpen(true)
              }}
            >
              <Plus className="size-4" />
              New tracker
            </button>
          }
        />
      ) : (
        <div className="space-y-10">
          <div className="divide-y divide-base-300">
            {active.map(renderRow)}
          </div>

          {archived.length > 0 && (
            <section>
              <h2 className="eyebrow border-b border-base-300 pb-2">Archived</h2>
              <div className="divide-y divide-base-300">{archived.map(renderRow)}</div>
            </section>
          )}
        </div>
      )}

      {modalOpen && (
        <TrackerFormModal
          tracker={editing}
          onClose={() => {
            setModalOpen(false)
            setEditing(null)
          }}
        />
      )}

      <ConfirmDialog
        open={Boolean(deleting)}
        title={`Delete “${deleting?.name ?? ''}”?`}
        message="This permanently removes the tracker, every entry it recorded, and the dashboard charts that use it. Archiving keeps the data instead."
        busy={remove.isPending}
        onConfirm={confirmDelete}
        onClose={() => setDeleting(null)}
      />
    </div>
  )
}
