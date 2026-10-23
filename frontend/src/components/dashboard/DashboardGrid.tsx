import { useEffect, useRef, useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { GridLayout, useContainerWidth, type Layout } from 'react-grid-layout'
import { WidgetCard } from './WidgetCard'
import { api, errorMessage } from '../../lib/api'
import { useToast } from '../ui/Toast'
import type { ChartData, Dashboard, Tracker, Widget, WidgetLayout } from '../../lib/types'

interface DashboardGridProps {
  dashboard: Dashboard
  data: Record<string, ChartData> | undefined
  loading: boolean
  editing: boolean
  mobile: boolean
  trackers: Tracker[]
  onEditWidget: (widget: Widget) => void
  onDeleteWidget: (widget: Widget) => void
}

const GRID_MARGIN: [number, number] = [16, 16]

function toLayout(widgets: Widget[]): Layout {
  return widgets.map((widget) => ({
    i: widget.id,
    x: widget.layout.x,
    y: widget.layout.y,
    w: widget.layout.w,
    h: widget.layout.h,
    minW: 2,
    minH: 2,
  }))
}

function toPayload(layout: Layout): WidgetLayout[] {
  return layout.map((item) => ({ id: item.i, x: item.x, y: item.y, w: item.w, h: item.h }))
}

function gridHeight(widget: Widget): number {
  const rows = Math.max(widget.layout.h, 3)
  return rows * 56 + (rows - 1) * GRID_MARGIN[1]
}

export function DashboardGrid({
  dashboard,
  data,
  loading,
  editing,
  mobile,
  trackers,
  onEditWidget,
  onDeleteWidget,
}: DashboardGridProps) {
  const toast = useToast()
  const queryClient = useQueryClient()
  const { width, containerRef, mounted } = useContainerWidth()

  const signature = dashboard.widgets
    .map((w) => `${w.id}:${w.layout.x}:${w.layout.y}:${w.layout.w}:${w.layout.h}`)
    .join('|')

  const [layout, setLayout] = useState<Layout>(() => toLayout(dashboard.widgets))
  const signatureRef = useRef(signature)

  useEffect(() => {
    if (signatureRef.current === signature) return
    signatureRef.current = signature
    setLayout(toLayout(dashboard.widgets))
  }, [signature, dashboard.widgets])

  const persist = useMutation({
    mutationFn: (items: WidgetLayout[]) => api.updateLayouts(dashboard.id, items),
    onError: (error) => {
      toast.push('error', `Could not save layout: ${errorMessage(error)}`)
      void queryClient.invalidateQueries({ queryKey: ['dashboards'] })
    },
  })

  const handleStop = (next: Layout) => {
    setLayout(next)
    persist.mutate(toPayload(next))
  }

  const byId = new Map(trackers.map((tracker) => [tracker.id, tracker]))
  const widgetById = new Map(dashboard.widgets.map((widget) => [widget.id, widget]))

  const card = (widget: Widget) => (
    <WidgetCard
      widget={widget}
      tracker={byId.get(widget.trackerId)}
      data={data?.[widget.id]}
      loading={loading}
      editing={editing}
      onEdit={() => onEditWidget(widget)}
      onDelete={() => onDeleteWidget(widget)}
    />
  )

  if (mobile) {
    return (
      <div className="space-y-4">
        {[...dashboard.widgets]
          .sort((a, b) => a.layout.y - b.layout.y || a.layout.x - b.layout.x)
          .map((widget) => (
            <div key={widget.id} style={{ height: gridHeight(widget) }}>
              {card(widget)}
            </div>
          ))}
      </div>
    )
  }

  return (
    <div ref={containerRef} className="w-full">
      {!mounted ? (
        <div className="grid gap-4 sm:grid-cols-2">
          {dashboard.widgets.map((widget) => (
            <div key={widget.id} className="surface h-72 animate-pulse opacity-60" />
          ))}
        </div>
      ) : (
        <GridLayout
          width={width}
          layout={layout}
          gridConfig={{ cols: 12, rowHeight: 56, margin: GRID_MARGIN, containerPadding: [0, 0] }}
          dragConfig={{ enabled: editing, handle: '.widget-drag', cancel: 'button, a, input, select' }}
          resizeConfig={{ enabled: editing, handles: ['se'] as const }}
          onDragStop={handleStop}
          onResizeStop={handleStop}
        >
          {layout.map((item) => {
            const widget = widgetById.get(item.i)
            if (!widget) return <div key={item.i} />
            return (
              <div key={item.i} className="h-full">
                {card(widget)}
              </div>
            )
          })}
        </GridLayout>
      )}
    </div>
  )
}
