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
        className="btn btn-ghost btn-sm btn-square"
        onClick={() => setMode(next)}
        aria-label={`Switch to ${next} theme`}
        title={`Switch to ${next} theme`}
      >
        <Icon className="size-4" />
      </button>
    )
  }

  // Three choices, one control: the active mode is the only ink, the rest are
  // quiet labels. No pill, no shadow, no gradient.
  return (
    <div role="group" className="inline-flex border border-base-300" aria-label="Theme">
      {options.map(({ mode: value, label, icon: Icon }) => (
        <button
          key={value}
          type="button"
          className={`flex h-8 items-center gap-1.5 border-l border-base-300 px-3 font-mono text-[10px] uppercase tracking-[0.1em] transition-colors first:border-l-0 ${
            mode === value
              ? 'bg-base-content text-base-100'
              : 'text-base-content/65 hover:text-base-content'
          }`}
          onClick={() => setMode(value)}
          aria-pressed={mode === value}
          aria-label={label}
          title={label}
        >
          <Icon className="size-3.5" />
          {label}
        </button>
      ))}
    </div>
  )
}
