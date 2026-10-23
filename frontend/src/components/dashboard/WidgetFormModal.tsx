import { useState, type FormEvent } from 'react'
import {
  ChartArea,
  ChartColumn,
  ChartLine,
  ChartPie,
  Grid3x3,
  Hash,
} from 'lucide-react'
import { Modal } from '../ui/Modal'
import { useToast } from '../ui/Toast'
import { useQueryClient } from '@tanstack/react-query'
import { api, errorMessage } from '../../lib/api'
import type {
  Aggregate,
  ChartType,
  Granularity,
  RangePreset,
  StatMetric,
  Tracker,
  Widget,
  WidgetInput,
} from '../../lib/types'

interface WidgetFormModalProps {
  dashboardId: string
  widget?: Widget | null
  trackers: Tracker[]
  onClose: () => void
}

const chartChoices: { value: ChartType; label: string; hint: string; icon: typeof ChartLine }[] = [
  { value: 'line', label: 'Line', hint: 'Trend over time', icon: ChartLine },
  { value: 'bar', label: 'Bar', hint: 'Compare periods', icon: ChartColumn },
  { value: 'area', label: 'Area', hint: 'Trend with volume', icon: ChartArea },
  { value: 'donut', label: 'Donut', hint: 'Share by weekday', icon: ChartPie },
  { value: 'heatmap', label: 'Heatmap', hint: 'Calendar density', icon: Grid3x3 },
  { value: 'stat', label: 'Stat', hint: 'One headline number', icon: Hash },
]

const ranges: { value: RangePreset; label: string }[] = [
  { value: '7d', label: 'Last 7 days' },
  { value: '30d', label: 'Last 30 days' },
  { value: '90d', label: 'Last 90 days' },
  { value: '365d', label: 'Last 12 months' },
  { value: 'all', label: 'All time' },
]

export function WidgetFormModal({
  dashboardId,
  widget,
  trackers,
  onClose,
}: WidgetFormModalProps) {
  const toast = useToast()
  const queryClient = useQueryClient()

  // Mounted only while open, so this is one editing session.
  const [trackerId, setTrackerId] = useState(widget?.trackerId ?? trackers[0]?.id ?? '')
  const [chartType, setChartType] = useState<ChartType>(widget?.chartType ?? 'line')
  const [title, setTitle] = useState(widget?.title ?? '')
  const [range, setRange] = useState<RangePreset>(widget?.config?.range ?? '30d')
  const [granularity, setGranularity] = useState<Granularity>(widget?.config?.granularity ?? 'auto')
  const [aggregate, setAggregate] = useState<Aggregate>(widget?.config?.aggregate ?? 'sum')
  const [metric, setMetric] = useState<StatMetric>(widget?.config?.metric ?? 'total')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  const tracker = trackers.find((candidate) => candidate.id === trackerId)

  const onSubmit = async (event: FormEvent) => {
    event.preventDefault()
    setError('')
    if (!trackerId) {
      setError('Pick a tracker to chart.')
      return
    }

    const input: WidgetInput = {
      trackerId,
      title: title.trim(),
      chartType,
      config: { range, granularity, aggregate, metric, showGoal: true },
      layout: widget?.layout ?? { id: '', x: 0, y: 0, w: 0, h: 0 },
    }

    setSaving(true)
    try {
      if (widget) {
        await api.updateWidget(widget.id, input)
      } else {
        await api.createWidget(dashboardId, input)
      }
      await queryClient.invalidateQueries({ queryKey: ['dashboards'] })
      await queryClient.invalidateQueries({ queryKey: ['dashboard-data'] })
      toast.push('success', widget ? 'Chart updated.' : 'Chart added to the dashboard.')
      onClose()
    } catch (err) {
      setError(errorMessage(err))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal
      open
      title={widget ? 'Edit chart' : 'Add chart'}
      onClose={onClose}
      wide
      footer={
        <>
          <button type="button" className="btn btn-ghost" onClick={onClose} disabled={saving}>
            Cancel
          </button>
          <button type="submit" form="widget-form" className="btn btn-primary" disabled={saving}>
            {saving && <span className="loading loading-spinner loading-sm" />}
            {widget ? 'Save chart' : 'Add to dashboard'}
          </button>
        </>
      }
    >
      {trackers.length === 0 ? (
        <p className="text-sm text-base-content/60">
          Create a tracker first — charts are always built on top of one tracker.
        </p>
      ) : (
        <form id="widget-form" className="space-y-5" onSubmit={onSubmit}>
          {error && <div className="alert alert-error text-sm">{error}</div>}

          <div className="grid gap-4 sm:grid-cols-2">
            <label className="form-control">
              <span className="mb-1.5 block text-sm font-medium">Tracker</span>
              <select
                className="select select-bordered w-full"
                value={trackerId}
                onChange={(event) => setTrackerId(event.target.value)}
                required
              >
                {trackers.map((option) => (
                  <option key={option.id} value={option.id}>
                    {option.icon} {option.name}
                  </option>
                ))}
              </select>
            </label>
            <label className="form-control">
              <span className="mb-1.5 block text-sm font-medium">
                Title <span className="font-normal text-base-content/40">(optional)</span>
              </span>
              <input
                className="input input-bordered w-full"
                value={title}
                onChange={(event) => setTitle(event.target.value)}
                placeholder={tracker?.name ?? 'Chart title'}
                maxLength={100}
              />
            </label>
          </div>

          <div>
            <span className="mb-2 block text-sm font-medium">Chart type</span>
            <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
              {chartChoices.map(({ value, label, hint, icon: Icon }) => (
                <button
                  key={value}
                  type="button"
                  onClick={() => setChartType(value)}
                  className={`rounded-box border p-3 text-left transition ${
                    chartType === value
                      ? 'border-primary bg-primary/5 ring-1 ring-primary'
                      : 'border-base-300 hover:border-base-content/30'
                  }`}
                  aria-pressed={chartType === value}
                >
                  <Icon className="size-4 text-primary" />
                  <p className="mt-1.5 text-sm font-medium">{label}</p>
                  <p className="text-[11px] leading-tight text-base-content/50">{hint}</p>
                </button>
              ))}
            </div>
          </div>

          <div className="grid gap-4 sm:grid-cols-3">
            <label className="form-control">
              <span className="mb-1.5 block text-sm font-medium">Range</span>
              <select
                className="select select-bordered w-full"
                value={range}
                onChange={(event) => setRange(event.target.value as RangePreset)}
              >
                {ranges.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>
            </label>

            {chartType === 'stat' ? (
              <label className="form-control">
                <span className="mb-1.5 block text-sm font-medium">Metric</span>
                <select
                  className="select select-bordered w-full"
                  value={metric}
                  onChange={(event) => setMetric(event.target.value as StatMetric)}
                >
                  <option value="total">Total</option>
                  <option value="average">Average per day</option>
                  <option value="count">Days logged</option>
                  <option value="streak">Current streak</option>
                </select>
              </label>
            ) : (
              <label className="form-control">
                <span className="mb-1.5 block text-sm font-medium">Grouping</span>
                <select
                  className="select select-bordered w-full"
                  value={granularity}
                  onChange={(event) => setGranularity(event.target.value as Granularity)}
                >
                  <option value="auto">Auto</option>
                  <option value="day">Daily</option>
                  <option value="week">Weekly</option>
                  <option value="month">Monthly</option>
                </select>
              </label>
            )}

            {tracker?.kind === 'numeric' && chartType !== 'stat' ? (
              <label className="form-control">
                <span className="mb-1.5 block text-sm font-medium">Aggregate</span>
                <select
                  className="select select-bordered w-full"
                  value={aggregate}
                  onChange={(event) => setAggregate(event.target.value as Aggregate)}
                >
                  <option value="sum">Sum</option>
                  <option value="avg">Average</option>
                </select>
              </label>
            ) : (
              <div />
            )}
          </div>

          <p className="text-xs text-base-content/50">
            Charts update automatically. You can drag and resize them on the dashboard while edit mode is on.
          </p>
        </form>
      )}
    </Modal>
  )
}
