import { GripVertical, Pencil, Trash } from 'lucide-react'
import { ChartRenderer } from '../charts/ChartRenderer'
import { rangeDays } from '../../lib/date'
import type { ChartData, Tracker, Widget } from '../../lib/types'

interface WidgetCardProps {
  widget: Widget
  tracker: Tracker | undefined
  data: ChartData | undefined
  loading: boolean
  editing: boolean
  onEdit: () => void
  onDelete: () => void
}

export function WidgetCard({
  widget,
  tracker,
  data,
  loading,
  editing,
  onEdit,
  onDelete,
}: WidgetCardProps) {
  const title = widget.title || tracker?.name || 'Chart'

  return (
    <div className="surface flex h-full flex-col overflow-hidden">
      <div
        className={`flex items-start justify-between gap-2 px-4 pb-1 pt-3 ${
          editing ? 'widget-drag' : ''
        }`}
      >
        <div className="min-w-0">
          <p className="truncate text-sm font-semibold" title={title}>
            {title}
          </p>
          <p className="truncate text-[11px] text-base-content/45">
            {tracker ? `${tracker.icon} ${tracker.name}` : 'Tracker missing'} ·{' '}
            {rangeDays(widget.config.range)}
          </p>
        </div>
        {editing && (
          <div className="flex shrink-0 items-center gap-0.5">
            <GripVertical className="size-4 opacity-30" aria-hidden="true" />
            <button
              type="button"
              className="btn btn-ghost btn-xs btn-square"
              onClick={onEdit}
              aria-label="Edit chart"
              title="Edit chart"
            >
              <Pencil className="size-3.5" />
            </button>
            <button
              type="button"
              className="btn btn-ghost btn-xs btn-square text-error"
              onClick={onDelete}
              aria-label="Delete chart"
              title="Delete chart"
            >
              <Trash className="size-3.5" />
            </button>
          </div>
        )}
      </div>

      <div className="min-h-0 flex-1 px-2 pb-2 pt-1">
        {loading ? (
          <div className="flex h-full items-center justify-center">
            <span className="loading loading-spinner loading-sm text-base-content/30" />
          </div>
        ) : tracker ? (
          <ChartRenderer data={data} kind={tracker.kind} unit={tracker.unit} color={tracker.color} />
        ) : (
          <div className="flex h-full items-center justify-center px-4 text-center text-xs text-base-content/50">
            This chart points at a tracker that no longer exists.
          </div>
        )}
      </div>
    </div>
  )
}
