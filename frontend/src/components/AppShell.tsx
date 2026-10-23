import { useMutation, useQueryClient } from '@tanstack/react-query'
import { LayoutDashboard, ListChecks, LogOut, Settings, Sun } from 'lucide-react'
import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { api } from '../lib/api'
import { useMe } from '../lib/queries'
import { Logo } from './Logo'
import { ThemeToggle } from './ThemeToggle'

const navItems = [
  { to: '/', label: 'Today', icon: Sun, end: true },
  { to: '/dashboard', label: 'Dashboard', icon: LayoutDashboard, end: false },
  { to: '/trackers', label: 'Trackers', icon: ListChecks, end: false },
  { to: '/settings', label: 'Settings', icon: Settings, end: false },
]

export function AppShell() {
  const { data: user } = useMe()
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const logout = useMutation({
    mutationFn: api.logout,
    onSuccess: () => {
      queryClient.clear()
      navigate('/login', { replace: true })
    },
  })

  const initial = (user?.name || user?.email || '?').trim().charAt(0).toUpperCase()

  const navLinkClass = ({ isActive }: { isActive: boolean }) =>
    [
      'flex items-center gap-3 rounded-xl px-3 py-2 text-sm font-medium transition-colors',
      isActive
        ? 'bg-primary/10 text-primary'
        : 'text-base-content/70 hover:bg-base-200 hover:text-base-content',
    ].join(' ')

  return (
    <div className="min-h-screen bg-base-200">
      <aside className="fixed inset-y-0 left-0 z-40 hidden w-64 flex-col border-r border-base-300 bg-base-100 lg:flex">
        <div className="flex h-16 items-center px-5">
          <NavLink to="/" aria-label="Trackete home">
            <Logo />
          </NavLink>
        </div>

        <nav className="flex-1 space-y-1 px-3 py-4">
          {navItems.map(({ to, label, icon: Icon, end }) => (
            <NavLink key={to} to={to} end={end} className={navLinkClass}>
              <Icon className="size-4" />
              {label}
            </NavLink>
          ))}
        </nav>

        <div className="border-t border-base-300 p-3">
          <div className="rounded-xl bg-base-200/60 p-3">
            <div className="flex items-center gap-3">
              <div className="grid size-9 shrink-0 place-items-center rounded-full bg-primary/15 text-sm font-semibold text-primary">
                {initial}
              </div>
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium">{user?.name || 'You'}</p>
                <p className="truncate text-xs text-base-content/50">{user?.email}</p>
              </div>
              <button
                type="button"
                className="btn btn-ghost btn-sm btn-circle"
                onClick={() => logout.mutate()}
                aria-label="Sign out"
                title="Sign out"
              >
                <LogOut className="size-4" />
              </button>
            </div>
          </div>
        </div>
      </aside>

      <div className="lg:pl-64">
        <header className="sticky top-0 z-30 flex h-16 items-center justify-between border-b border-base-300 bg-base-100/90 px-4 backdrop-blur lg:hidden">
          <NavLink to="/" aria-label="Trackete home">
            <Logo />
          </NavLink>
          <div className="flex items-center gap-1">
            <ThemeToggle compact />
            <div className="dropdown dropdown-end">
              <button type="button" tabIndex={0} className="btn btn-ghost btn-sm btn-circle">
                <span className="grid size-7 place-items-center rounded-full bg-primary/15 text-xs font-semibold text-primary">
                  {initial}
                </span>
              </button>
              <ul
                tabIndex={0}
                className="menu dropdown-content z-40 mt-2 w-52 rounded-box border border-base-300 bg-base-100 p-2 shadow-lg"
              >
                <li className="menu-title truncate">{user?.email}</li>
                <li>
                  <button type="button" onClick={() => logout.mutate()}>
                    <LogOut className="size-4" />
                    Sign out
                  </button>
                </li>
              </ul>
            </div>
          </div>
        </header>

        <main className="mx-auto w-full max-w-7xl px-4 pb-24 pt-6 sm:px-6 lg:pb-12 lg:pt-8">
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
              `flex flex-1 flex-col items-center gap-0.5 py-2.5 text-[11px] font-medium ${
                isActive ? 'text-primary' : 'text-base-content/60'
              }`
            }
          >
            <Icon className="size-5" />
            {label}
          </NavLink>
        ))}
      </nav>
    </div>
  )
}
