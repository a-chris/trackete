import { useTheme } from './theme'

export interface ChartTheme {
  mode: 'light' | 'dark'
  text: string
  muted: string
  grid: string
  surface: string
  primary: string
  emptyCell: string
}

// Apex paints on canvas, so it needs plain sRGB values rather than the oklch
// design tokens DaisyUI exposes as CSS variables.
const palettes: Record<'light' | 'dark', ChartTheme> = {
  light: {
    mode: 'light',
    text: '#272932',
    muted: '#6b7280',
    grid: '#e5e7eb',
    surface: '#ffffff',
    primary: '#6366f1',
    emptyCell: '#ececf1',
  },
  dark: {
    mode: 'dark',
    text: '#e7e8ee',
    muted: '#9aa1b1',
    grid: '#2f3441',
    surface: '#191b23',
    primary: '#8b93f8',
    emptyCell: '#2b2f3a',
  },
}

export function useChartTheme(): ChartTheme {
  const { resolved } = useTheme()
  return palettes[resolved]
}
