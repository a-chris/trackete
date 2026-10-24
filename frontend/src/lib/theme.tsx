import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import type { ThemeMode } from './types'

const STORAGE_KEY = 'trackete.theme'

interface ThemeContextValue {
  mode: ThemeMode
  resolved: 'light' | 'dark'
  setMode: (mode: ThemeMode) => void
}

const ThemeContext = createContext<ThemeContextValue | null>(null)

// Storage and matchMedia can both throw (private mode, blocked cookies,
// sandboxed frames). A theme preference is never worth crashing the app over.
function prefersDark(): boolean {
  if (typeof window === 'undefined') return false
  try {
    return window.matchMedia('(prefers-color-scheme: dark)').matches
  } catch {
    return false
  }
}

function storedMode(): ThemeMode {
  if (typeof window === 'undefined') return 'system'
  let stored: string | null
  try {
    stored = window.localStorage.getItem(STORAGE_KEY)
  } catch {
    return 'system'
  }
  return stored === 'light' || stored === 'dark' || stored === 'system' ? stored : 'system'
}

function persistMode(mode: ThemeMode) {
  try {
    window.localStorage.setItem(STORAGE_KEY, mode)
  } catch {
    // Not persisted; the choice still applies for this session.
  }
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [mode, setModeState] = useState<ThemeMode>(storedMode)
  const [systemDark, setSystemDark] = useState(prefersDark)

  useEffect(() => {
    let media: MediaQueryList
    try {
      media = window.matchMedia('(prefers-color-scheme: dark)')
    } catch {
      return
    }
    const listener = (event: MediaQueryListEvent) => setSystemDark(event.matches)
    media.addEventListener('change', listener)
    return () => media.removeEventListener('change', listener)
  }, [])

  const resolved: 'light' | 'dark' = mode === 'system' ? (systemDark ? 'dark' : 'light') : mode

  useEffect(() => {
    document.documentElement.dataset.theme = `trackete-${resolved}`
    document.documentElement.style.colorScheme = resolved
    document
      .querySelector('meta[name="theme-color"]')
      ?.setAttribute('content', resolved === 'dark' ? '#171512' : '#fbfaf7')
  }, [resolved])

  const setMode = useCallback((next: ThemeMode) => {
    setModeState(next)
    persistMode(next)
  }, [])

  const value = useMemo(() => ({ mode, resolved, setMode }), [mode, resolved, setMode])
  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
}

// eslint-disable-next-line react-refresh/only-export-components
export function useTheme(): ThemeContextValue {
  const context = useContext(ThemeContext)
  if (!context) throw new Error('useTheme must be used inside ThemeProvider')
  return context
}
