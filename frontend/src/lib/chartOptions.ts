import type { ApexOptions } from 'apexcharts'
import type { ChartData, Granularity, TrackerKind } from './types'
import type { ChartTheme } from './chartTheme'
import { bucketLabel } from './date'
import { formatValue } from './format'
import { CHART_COLORS } from './palette'

type SeriesChartType = 'line' | 'bar' | 'area'

function baseOptions(theme: ChartTheme): ApexOptions {
  return {
    chart: {
      background: 'transparent',
      fontFamily: 'inherit',
      foreColor: theme.muted,
      toolbar: { show: false },
      zoom: { enabled: false },
      parentHeightOffset: 0,
      animations: { enabled: true, speed: 350 },
    },
    theme: { mode: theme.mode },
    dataLabels: { enabled: false },
    grid: {
      borderColor: theme.grid,
      // Solid hairlines: a dashed grid is the visual signature of a generic
      // dashboard template. One weight, one colour, no rhythm.
      strokeDashArray: 0,
      xaxis: { lines: { show: false } },
      padding: { left: 6, right: 12, top: 0, bottom: 0 },
    },
    tooltip: { theme: theme.mode },
    legend: { labels: { colors: theme.muted } },
  }
}

export function buildSeriesOptions(
  data: ChartData,
  kind: TrackerKind,
  unit: string,
  color: string,
  chartType: SeriesChartType,
  granularity: Granularity,
  theme: ChartTheme,
  compact = false,
): ApexOptions {
  const categories = data.labels.map((label) => bucketLabel(label, granularity))
  const valueFormatter = (value: number) => formatValue(Number(value), unit, kind)

  return {
    ...baseOptions(theme),
    chart: {
      ...baseOptions(theme).chart,
      type: chartType,
    },
    colors: [color],
    series: data.series.map((series) => ({ name: series.name, data: series.data })),
    stroke: {
      curve: kind === 'boolean' ? 'stepline' : 'smooth',
      width: chartType === 'bar' ? 0 : 2,
      lineCap: 'round',
    },
    fill:
      chartType === 'area'
        ? {
            type: 'gradient',
            gradient: {
              shadeIntensity: 1,
              opacityFrom: 0.22,
              opacityTo: 0.01,
              stops: [0, 95],
            },
          }
        : { opacity: 1 },
    markers: { size: 0, strokeWidth: 2, hover: { size: 4 } },
    plotOptions: {
      bar: {
        borderRadius: 2,
        columnWidth: kind === 'boolean' ? '44%' : '58%',
        borderRadiusApplication: 'end',
      },
    },
    xaxis: {
      categories,
      tickAmount: Math.min(compact ? 2 : 4, Math.max(2, categories.length)),
      axisBorder: { show: false },
      axisTicks: { show: false },
      labels: {
        rotate: 0,
        rotateAlways: false,
        trim: false,
        style: { colors: theme.muted, fontSize: '10px' },
        hideOverlappingLabels: true,
      },
      tooltip: { enabled: false },
    },
    yaxis: {
      labels: {
        style: { colors: theme.muted, fontSize: '11px' },
        formatter: (value: number) =>
          kind === 'boolean' ? (value >= 0.5 ? 'Yes' : 'No') : formatValue(value),
      },
      min: 0,
      max: kind === 'boolean' ? 1 : undefined,
      tickAmount: kind === 'boolean' ? 1 : undefined,
      forceNiceScale: kind !== 'boolean',
    },
    tooltip: {
      theme: theme.mode,
      y: { formatter: valueFormatter },
    },
  }
}

export function buildDonutOptions(
  data: ChartData,
  kind: TrackerKind,
  unit: string,
  color: string,
  theme: ChartTheme,
): ApexOptions {
  const series = data.series[0]?.data ?? []
  const total = series.reduce((sum, value) => sum + value, 0)

  return {
    ...baseOptions(theme),
    chart: { ...baseOptions(theme).chart, type: 'donut' },
    labels: data.labels,
    series,
    colors:
      kind === 'boolean'
        ? [color, theme.emptyCell]
        : CHART_COLORS.slice(0, data.labels.length),
    stroke: { width: 2, colors: [theme.surface] },
    dataLabels: { enabled: false },
    legend: {
      position: 'bottom',
      labels: { colors: theme.muted },
      markers: { size: 5 },
      itemMargin: { horizontal: 8, vertical: 2 },
    },
    plotOptions: {
      pie: {
        donut: {
          size: '72%',
          labels: {
            show: true,
            name: { color: theme.muted, fontSize: '10px' },
            value: { color: theme.text, fontSize: '22px', fontWeight: 500 },
            total: {
              show: true,
              label: kind === 'boolean' ? 'Days' : 'Total',
              color: theme.muted,
              fontSize: '10px',
              formatter: () => (kind === 'boolean' ? formatValue(total) : formatValue(total, unit)),
            },
          },
        },
      },
    },
    tooltip: {
      theme: theme.mode,
      y: { formatter: (value: number) => formatValue(Number(value), unit, kind) },
    },
  }
}

export function buildHeatmapOptions(
  data: ChartData,
  kind: TrackerKind,
  unit: string,
  color: string,
  theme: ChartTheme,
): ApexOptions {
  const heat = data.heatmap
  if (!heat) return baseOptions(theme)

  const all = heat.rows.flatMap((row) => row.data)
  const max = Math.max(1, ...all)

  const ranges: { from: number; to: number; color: string; name?: string }[] =
    kind === 'boolean'
      ? [
          { from: 0, to: 0.5, color: theme.emptyCell, name: 'No' },
          { from: 0.5, to: 1.5, color, name: 'Yes' },
        ]
      : [
          { from: 0, to: 0.001, color: theme.emptyCell, name: 'No data' },
          { from: 0.001, to: max / 3, color: `${color}55`, name: 'Low' },
          { from: max / 3, to: (max * 2) / 3, color: `${color}99`, name: 'Medium' },
          { from: (max * 2) / 3, to: max + 0.001, color, name: 'High' },
        ]

  return {
    ...baseOptions(theme),
    chart: { ...baseOptions(theme).chart, type: 'heatmap' },
    series: [...heat.rows].reverse().map((row) => ({
      name: row.name.slice(0, 3),
      data: row.data.map((value, index) => ({
        x: bucketLabel(heat.columns[index] ?? '', 'week'),
        y: value,
      })),
    })),
    colors: [color],
    dataLabels: { enabled: false },
    stroke: { width: 2, colors: [theme.surface] },
    plotOptions: {
      heatmap: {
        radius: 2,
        enableShades: false,
        colorScale: { ranges },
      },
    },
    xaxis: {
      type: 'category',
      axisBorder: { show: false },
      axisTicks: { show: false },
      labels: { style: { colors: theme.muted, fontSize: '10px' }, hideOverlappingLabels: true },
    },
    yaxis: {
      labels: { style: { colors: theme.muted, fontSize: '10px' } },
    },
    tooltip: {
      theme: theme.mode,
      y: { formatter: (value: number) => formatValue(Number(value), unit, kind) },
    },
  }
}

export function buildSparklineOptions(color: string, theme: ChartTheme): ApexOptions {
  return {
    chart: {
      type: 'area',
      sparkline: { enabled: true },
      animations: { enabled: false },
      background: 'transparent',
    },
    colors: [color],
    stroke: { curve: 'smooth', width: 2 },
    fill: {
      type: 'gradient',
      gradient: { opacityFrom: 0.35, opacityTo: 0, stops: [0, 100] },
    },
    tooltip: { enabled: false },
    theme: { mode: theme.mode },
    grid: { show: false },
  }
}
