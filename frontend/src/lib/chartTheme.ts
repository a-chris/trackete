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
// design tokens DaisyUI exposes as CSS variables. These track the paper/ink
// palette in index.css: warm paper, near-black ink, hairline gridlines, and no
// chrome colour — only the series itself is coloured.
const palettes: Record<'light' | 'dark', ChartTheme> = {
  light: {
    mode: 'light',
    text: '#1c1a17',
    muted: '#6b675f',
    grid: '#e7e4dc',
    surface: '#fbfaf7',
    primary: '#1c1a17',
    emptyCell: '#efede6',
  },
  dark: {
    mode: 'dark',
    text: '#eeeae1',
    muted: '#9a948b',
    grid: '#2b2723',
    surface: '#171512',
    primary: '#eeeae1',
    emptyCell: '#26221e',
  },
}

export function useChartTheme(): ChartTheme {
  const { resolved } = useTheme()
  return palettes[resolved]
}
