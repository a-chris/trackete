import ReactApexChart from 'react-apexcharts'
import type { ChartData, TrackerKind } from '../../lib/types'
import { useChartTheme } from '../../lib/chartTheme'
import {
  buildDonutOptions,
  buildHeatmapOptions,
  buildSeriesOptions,
  buildSparklineOptions,
} from '../../lib/chartOptions'
import { formatValue } from '../../lib/format'
import { ChartColumn, RotateCw, TriangleAlert } from 'lucide-react'

interface ChartRendererProps {
  data: ChartData | undefined
  kind: TrackerKind
  unit?: string
  color?: string
  /** Rendered as a Retry button when the chart data could not be loaded. */
  onRetry?: () => void
  retrying?: boolean
}

export function ChartRenderer({
  data,
  kind,
  unit = '',
  color = '#6366f1',
  onRetry,
  retrying = false,
}: ChartRendererProps) {
  const theme = useChartTheme()

  if (data === undefined) {
    return (
      <div
        role="alert"
        className="flex h-full min-h-32 flex-col items-center justify-center gap-2 text-center text-base-content/40"
      >
        <TriangleAlert className="size-6" />
        <p className="text-xs">Couldn't load chart data.</p>
        {onRetry && (
          <button type="button" className="btn btn-xs btn-outline" onClick={onRetry} disabled={retrying}>
            {retrying ? <span className="loading loading-spinner loading-xs" /> : <RotateCw className="size-3.5" />}
            Retry
          </button>
        )}
      </div>
    )
  }

  if (data.empty) {
    return (
      <div className="flex h-full min-h-32 flex-col items-center justify-center gap-2 text-center text-base-content/40">
        <ChartColumn className="size-6" />
        <p className="text-xs">No entries in this range yet.</p>
      </div>
    )
  }

  if (data.chartType === 'stat' && data.stat) {
    return <StatCard data={data} kind={kind} unit={unit} color={color} theme={theme} />
  }

  if (data.chartType === 'heatmap' && data.heatmap) {
    return (
      <div className="h-full w-full">
        <ReactApexChart
          type="heatmap"
          height="100%"
          options={buildHeatmapOptions(data, kind, unit, color, theme)}
          series={buildHeatmapOptions(data, kind, unit, color, theme).series}
        />
      </div>
    )
  }

  if (data.chartType === 'donut') {
    return (
      <div className="h-full w-full">
        <ReactApexChart
          type="donut"
          height="100%"
          options={buildDonutOptions(data, kind, unit, color, theme)}
          series={data.series[0]?.data ?? []}
        />
      </div>
    )
  }

  const chartType = data.chartType === 'bar' || data.chartType === 'area' ? data.chartType : 'line'
  const options = buildSeriesOptions(data, kind, unit, color, chartType, data.granularity, theme)

  return (
    <div className="h-full w-full">
      <ReactApexChart type={chartType} height="100%" options={options} series={options.series} />
    </div>
  )
}

function StatCard({
  data,
  kind,
  unit,
  color,
  theme,
}: {
  data: ChartData
  kind: TrackerKind
  unit: string
  color: string
  theme: ReturnType<typeof useChartTheme>
}) {
  const stat = data.stat!
  const display =
    stat.label.toLowerCase().startsWith('completion')
      ? `${Math.round(stat.value)}%`
      : formatValue(stat.value, unit, kind === 'boolean' && stat.label.includes('streak') ? 'numeric' : kind)

  return (
    <div className="flex h-full min-h-32 flex-col justify-between gap-2 pb-2 pr-1">
      <div>
        <p className="text-xs font-medium uppercase tracking-wider text-base-content/50">{stat.label}</p>
        <p className="mt-1 text-3xl font-semibold tabular-nums">{display}</p>
      </div>
      {stat.spark.length > 1 && (
        <div className="h-14 w-full pb-1">
          <ReactApexChart
            type="area"
            height="100%"
            options={{
              ...buildSparklineOptions(color, theme),
              xaxis: { categories: stat.sparkLabels, labels: { show: false } },
            }}
            series={[{ name: stat.label, data: stat.spark }]}
          />
        </div>
      )}
    </div>
  )
}
