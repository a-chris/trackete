import { useEffect, useState } from 'react'
import { Archive, ArchiveRestore, ListChecks, Pencil, Plus, Trash } from 'lucide-react'
import { Link } from 'react-router-dom'
import { EmptyState } from '../components/ui/EmptyState'
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

export function TrackersPage() {
  const toast = useToast()
  const { data: trackers = [], isLoading } = useTrackers(true)
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

  const renderCard = (tracker: Tracker) => (
    <div key={tracker.id} className="surface flex flex-col gap-3 p-4">
      <div className="flex items-start gap-3">
        <div
          className="grid size-10 shrink-0 place-items-center rounded-xl text-lg"
          style={{ backgroundColor: `${tracker.color}22` }}
        >
          {tracker.icon}
        </div>
        <div className="min-w-0 flex-1">
          <Link
            to={`/trackers/${tracker.id}`}
            className="block truncate font-medium hover:underline"
            title={tracker.name}
          >
            {tracker.name}
          </Link>
          <p className="truncate text-xs text-base-content/50">
            {goalLabel(tracker.goal, tracker.kind, tracker.unit) ??
              (tracker.kind === 'boolean' ? 'Yes / No' : tracker.unit || 'Number')}
          </p>
        </div>
        <span className={`badge badge-sm ${tracker.kind === 'boolean' ? 'badge-primary' : 'badge-secondary'}`}>
          {tracker.kind === 'boolean' ? 'Yes/No' : 'Number'}
        </span>
      </div>

      {tracker.description && (
        <p className="line-clamp-2 text-sm text-base-content/60">{tracker.description}</p>
      )}

      <div className="mt-auto flex items-center justify-between gap-2 border-t border-base-300 pt-3">
        <span className="text-[11px] text-base-content/40">
          created {formatDate(tracker.createdAt.slice(0, 10), { month: 'short', day: 'numeric', year: 'numeric' })}
        </span>
        <div className="flex items-center gap-0.5">
          <button
            type="button"
            className="btn btn-ghost btn-xs"
            onClick={() => {
              setEditing(tracker)
              setModalOpen(true)
            }}
            title="Edit tracker"
          >
            <Pencil className="size-3.5" />
            <span className="hidden sm:inline">Edit</span>
          </button>
          <button
            type="button"
            className="btn btn-ghost btn-xs"
            onClick={() => toggleArchive(tracker)}
            title={tracker.archived ? 'Restore tracker' : 'Archive tracker'}
          >
            {tracker.archived ? (
              <ArchiveRestore className="size-3.5" />
            ) : (
              <Archive className="size-3.5" />
            )}
          </button>
          <button
            type="button"
            className="btn btn-ghost btn-xs text-error"
            onClick={() => setDeleting(tracker)}
            title="Delete tracker"
          >
            <Trash className="size-3.5" />
          </button>
        </div>
      </div>
    </div>
  )

  return (
    <div className="space-y-6">
      <header className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight sm:text-3xl">Trackers</h1>
          <p className="mt-1 text-sm text-base-content/60">
            Everything you track. Each tracker feeds the charts on your dashboards.
          </p>
        </div>
        <button
          type="button"
          className="btn btn-primary"
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
      ) : active.length === 0 && archived.length === 0 ? (
        <EmptyState
          icon={ListChecks}
          title="No trackers yet"
          description="A tracker is one thing you log: a yes/no habit or a number with a unit. Create one to get started."
          action={
            <button
              type="button"
              className="btn btn-primary"
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
        <>
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">{active.map(renderCard)}</div>

          {archived.length > 0 && (
            <section className="space-y-3">
              <h2 className="text-sm font-semibold uppercase tracking-wider text-base-content/50">
                Archived
              </h2>
              <div className="grid gap-4 opacity-70 sm:grid-cols-2 xl:grid-cols-3">
                {archived.map(renderCard)}
              </div>
            </section>
          )}
        </>
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
