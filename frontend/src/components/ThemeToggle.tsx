import { Monitor, Moon, Sun } from 'lucide-react'
import { useTheme } from '../lib/theme'
import type { ThemeMode } from '../lib/types'

const options: { mode: ThemeMode; label: string; icon: typeof Sun }[] = [
  { mode: 'light', label: 'Light', icon: Sun },
  { mode: 'dark', label: 'Dark', icon: Moon },
  { mode: 'system', label: 'System', icon: Monitor },
]

export function ThemeToggle({ compact = false }: { compact?: boolean }) {
  const { mode, setMode } = useTheme()

  if (compact) {
    const next: ThemeMode = mode === 'dark' ? 'light' : 'dark'
    const Icon = mode === 'dark' ? Sun : Moon
    return (
      <button
        type="button"
        className="btn btn-ghost btn-sm btn-circle"
        onClick={() => setMode(next)}
        aria-label={`Switch to ${next} theme`}
        title={`Switch to ${next} theme`}
      >
        <Icon className="size-4" />
      </button>
    )
  }

  return (
    <div role="group" className="join" aria-label="Theme">
      {options.map(({ mode: value, label, icon: Icon }) => (
        <button
          key={value}
          type="button"
          className={`btn btn-sm join-item ${mode === value ? 'btn-active' : ''}`}
          onClick={() => setMode(value)}
          aria-pressed={mode === value}
          aria-label={label}
          title={label}
        >
          <Icon className="size-4" />
          <span>{label}</span>
        </button>
      ))}
    </div>
  )
}
