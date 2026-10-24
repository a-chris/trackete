import { useMutation, useQueryClient } from '@tanstack/react-query'
import { LayoutDashboard, ListChecks, LogOut, Moon, Settings, Sun } from 'lucide-react'
import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { api } from '../lib/api'
import { useMe } from '../lib/queries'
import { useTheme } from '../lib/theme'
import { Logo } from './Logo'

const navItems = [
  { to: '/', label: 'Today', icon: Sun, end: true },
  { to: '/dashboard', label: 'Board', icon: LayoutDashboard, end: false },
  { to: '/trackers', label: 'Trackers', icon: ListChecks, end: false },
  { to: '/settings', label: 'Settings', icon: Settings, end: false },
]

export function AppShell() {
  const { data: user } = useMe()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { mode, setMode } = useTheme()

  const logout = useMutation({
    mutationFn: api.logout,
    onSuccess: () => {
      queryClient.clear()
      navigate('/login', { replace: true })
    },
  })

  const initial = (user?.name || user?.email || '?').trim().charAt(0).toUpperCase()
  const nextTheme = mode === 'dark' ? 'light' : 'dark'

  return (
    <div className="min-h-screen bg-base-100">
      {/* A rail, not a sidebar: the app is four places wide. */}
      <aside className="fixed inset-y-0 left-0 z-40 hidden w-[76px] flex-col border-r border-base-300 lg:flex">
        <div className="flex h-20 items-center justify-center">
          <NavLink to="/" aria-label="Trackete home" className="rounded-sm">
            <Logo withText={false} />
          </NavLink>
        </div>

        <nav className="flex flex-1 flex-col gap-0.5 px-2 py-3">
          {navItems.map(({ to, label, icon: Icon, end }) => (
            <NavLink
              key={to}
              to={to}
              end={end}
              title={label}
              className={({ isActive }) =>
                [
                  'group relative flex flex-col items-center gap-1.5 rounded-[2px] py-2.5',
                  'text-[9px] font-medium uppercase tracking-[0.14em] transition-colors',
                  isActive
                    ? 'bg-base-content/[0.055] text-base-content'
                    : 'text-base-content/35 hover:text-base-content/70',
                ].join(' ')
              }
            >
              {({ isActive }) => (
                <>
                  <span
                    aria-hidden="true"
                    className={`absolute left-0 top-1/2 h-4 w-[2px] -translate-y-1/2 bg-base-content transition-opacity ${
                      isActive ? 'opacity-100' : 'opacity-0'
                    }`}
                  />
                  <Icon className="size-[18px]" strokeWidth={isActive ? 2.2 : 1.7} />
                  {label}
                </>
              )}
            </NavLink>
          ))}
        </nav>

        <div className="flex flex-col items-center gap-1 border-t border-base-300 px-2 py-3">
          <button
            type="button"
            className="grid size-9 place-items-center rounded-sm border border-base-300 font-mono text-[11px] font-semibold text-base-content transition-colors hover:border-base-content/40"
            onClick={() => logout.mutate()}
            title={user?.email ? `Sign out ${user.email}` : 'Sign out'}
            aria-label="Sign out"
          >
            {initial}
            <span className="sr-only">Sign out</span>
          </button>
        </div>
      </aside>

      <div className="lg:pl-[76px]">
        <header className="sticky top-0 z-30 flex h-14 items-center justify-between border-b border-base-300 bg-base-100/90 px-4 backdrop-blur lg:hidden">
          <NavLink to="/" aria-label="Trackete home">
            <Logo />
          </NavLink>
          <div className="flex items-center gap-1">
            <button
              type="button"
              className="btn btn-ghost btn-sm btn-square"
              onClick={() => setMode(nextTheme)}
              aria-label={`Switch to ${nextTheme} theme`}
            >
              {mode === 'dark' ? <Moon className="size-4" /> : <Sun className="size-4" />}
            </button>
            <button
              type="button"
              className="btn btn-ghost btn-sm btn-square"
              onClick={() => logout.mutate()}
              aria-label="Sign out"
              title="Sign out"
            >
              <LogOut className="size-4" />
            </button>
          </div>
        </header>

        <main className="mx-auto w-full max-w-6xl px-5 pb-24 pt-8 sm:px-8 lg:pb-16 lg:pt-12">
          <Outlet />
        </main>
      </div>

      <nav className="fixed inset-x-0 bottom-0 z-40 flex border-t border-base-300 bg-base-100/95 backdrop-blur lg:hidden">
        {navItems.map(({ to, label, icon: Icon, end }) => (
          <NavLink
            key={to}
            to={to}
            end={end}
            className={({ isActive }) =>
              `flex flex-1 flex-col items-center gap-1 py-2.5 text-[9px] font-medium uppercase tracking-[0.14em] ${
                isActive ? 'text-base-content' : 'text-base-content/35'
              }`
            }
          >
            {({ isActive }) => (
              <>
                <Icon className="size-5" strokeWidth={isActive ? 2.2 : 1.7} />
                {label}
              </>
            )}
          </NavLink>
        ))}
      </nav>
    </div>
  )
}
