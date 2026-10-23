import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import 'react-grid-layout/css/styles.css'
import './index.css'
import { App } from './App'
import { ThemeProvider } from './lib/theme'
import { ToastProvider } from './components/ui/Toast'

// Resolve the theme before the first paint so the app does not flash. Storage
// and matchMedia can throw (private mode, blocked cookies, sandboxed frames),
// so both reads are guarded and fall back to the system preference.
function readStoredTheme(): string | null {
  try {
    return window.localStorage.getItem('trackete.theme')
  } catch {
    return null
  }
}

function systemPrefersDark(): boolean {
  try {
    return window.matchMedia('(prefers-color-scheme: dark)').matches
  } catch {
    return false
  }
}

const storedTheme = readStoredTheme()
const prefersDark = systemPrefersDark()
const dark = storedTheme === 'dark' || ((storedTheme === 'system' || storedTheme === null) && prefersDark)
document.documentElement.dataset.theme = dark ? 'trackete-dark' : 'trackete-light'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
      staleTime: 10_000,
    },
  },
})

const container = document.getElementById('root')
if (!container) throw new Error('Root element not found')

createRoot(container).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <ToastProvider>
          <BrowserRouter>
            <App />
          </BrowserRouter>
        </ToastProvider>
      </ThemeProvider>
    </QueryClientProvider>
  </StrictMode>,
)
