import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useEffect, useState, type FormEvent, type ReactNode } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { Logo } from '../components/Logo'
import { ThemeToggle } from '../components/ThemeToggle'
import { api, errorMessage } from '../lib/api'
import { useMeta } from '../lib/queries'

function AuthFrame({
  title,
  subtitle,
  children,
  footer,
}: {
  title: string
  subtitle: string
  children: ReactNode
  footer: ReactNode
}) {
  return (
    <div className="grid min-h-screen bg-base-100 lg:grid-cols-[1.05fr_1fr]">
      {/* The pitch, not a card: a page you land on rather than a form on a field. */}
      <div className="hidden flex-col justify-between border-r border-base-300 px-12 py-10 lg:flex">
        <Link to="/" aria-label="Trackete home">
          <Logo />
        </Link>

        <div className="max-w-md">
          <p className="eyebrow">One entry at a time</p>
          <p className="display mt-5 text-[44px] leading-[0.95] xl:text-5xl">
            Small numbers,
            <br />
            kept honestly,
            <br />
            become a life.
          </p>
          <div className="mt-10 h-px w-14 bg-base-content/20" />
          <p className="mt-6 max-w-sm text-sm leading-relaxed text-base-content/65">
            Log a habit or a number in seconds. Trackete turns the entries into the charts you
            actually want to look at.
          </p>
        </div>

        <p className="eyebrow">Self-hosted · SQLite · Open source</p>
      </div>

      <div className="flex flex-col justify-center px-6 py-12 sm:px-10 lg:px-16">
        <div className="w-full max-w-sm">
          <div className="mb-10 lg:hidden">
            <Logo />
          </div>
          <h1 className="display text-2xl">{title}</h1>
          <p className="mt-2 text-sm text-base-content/65">{subtitle}</p>
          <div className="mt-8">{children}</div>
          <div className="mt-6 text-sm text-base-content/70">{footer}</div>
          <div className="mt-10 border-t border-base-300 pt-4">
            <ThemeToggle compact />
          </div>
        </div>
      </div>
    </div>
  )
}

const authField = 'input h-10 w-full rounded-[3px] border-base-300'
const authLabel = 'mb-1.5 block text-[13px] font-medium text-base-content/75'

function useRedirectTarget(): string {
  const location = useLocation()
  const state = location.state as { from?: string } | null
  return state?.from && state.from.startsWith('/') ? state.from : '/'
}

export function LoginPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const redirectTo = useRedirectTarget()
  const meta = useMeta()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')

  const login = useMutation({
    mutationFn: () => api.login(email.trim(), password),
    onSuccess: (user) => {
      queryClient.setQueryData(['me'], user)
      navigate(redirectTo, { replace: true })
    },
  })

  useEffect(() => {
    document.title = 'Sign in · Trackete'
  }, [])

  const onSubmit = (event: FormEvent) => {
    event.preventDefault()
    login.mutate()
  }

  return (
    <AuthFrame
      title="Welcome back"
      subtitle="Sign in to keep your streaks going."
      footer={
        meta.data?.allowRegistration === false ? (
          <span>Registration is disabled on this instance.</span>
        ) : (
          <span>
            New here?{' '}
            <Link to="/register" className="link link-primary">
              Create an account
            </Link>
          </span>
        )
      }
    >
      <form className="space-y-4" onSubmit={onSubmit}>
        {login.isError && <div className="alert alert-error rounded-[3px] text-sm">{errorMessage(login.error)}</div>}
        <label className="block">
          <span className={authLabel}>Email</span>
          <input
            type="email"
            className={authField}
            value={email}
            onChange={(event) => setEmail(event.target.value)}
            autoComplete="email"
            required
            autoFocus
          />
        </label>
        <label className="block">
          <span className={authLabel}>Password</span>
          <input
            type="password"
            className={authField}
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            autoComplete="current-password"
            required
          />
        </label>
        <button type="submit" className="btn btn-primary h-10 w-full" disabled={login.isPending}>
          {login.isPending && <span className="loading loading-spinner loading-sm" />}
          Sign in
        </button>
      </form>
    </AuthFrame>
  )
}

export function RegisterPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const meta = useMeta()
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')

  const register = useMutation({
    mutationFn: () => api.register(email.trim(), password, name.trim()),
    onSuccess: (user) => {
      queryClient.setQueryData(['me'], user)
      navigate('/', { replace: true })
    },
  })

  useEffect(() => {
    document.title = 'Create account · Trackete'
  }, [])

  const onSubmit = (event: FormEvent) => {
    event.preventDefault()
    register.mutate()
  }

  if (meta.data?.allowRegistration === false) {
    return (
      <AuthFrame
        title="Registration is closed"
        subtitle="This instance does not accept new accounts."
        footer={
          <Link to="/login" className="link link-primary">
            Back to sign in
          </Link>
        }
      >
        <p className="text-sm text-base-content/75">
          Ask the person running this Trackete instance to create an account for you, or deploy your
          own instance.
        </p>
      </AuthFrame>
    )
  }

  return (
    <AuthFrame
      title="Create your account"
      subtitle="Track anything, then build the dashboard that fits."
      footer={
        <span>
          Already have an account?{' '}
          <Link to="/login" className="link link-primary">
            Sign in
          </Link>
        </span>
      }
    >
      <form className="space-y-4" onSubmit={onSubmit}>
        {register.isError && (
          <div className="alert alert-error rounded-[3px] text-sm">{errorMessage(register.error)}</div>
        )}
        <label className="block">
          <span className={authLabel}>Name</span>
          <input
            type="text"
            className={authField}
            value={name}
            onChange={(event) => setName(event.target.value)}
            autoComplete="name"
            placeholder="Alex"
            maxLength={80}
          />
        </label>
        <label className="block">
          <span className={authLabel}>Email</span>
          <input
            type="email"
            className={authField}
            value={email}
            onChange={(event) => setEmail(event.target.value)}
            autoComplete="email"
            required
            autoFocus
          />
        </label>
        <label className="block">
          <span className={authLabel}>Password</span>
          <input
            type="password"
            className={authField}
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            autoComplete="new-password"
            minLength={8}
            required
          />
          <span className="mt-1.5 block text-xs text-base-content/65">At least 8 characters.</span>
        </label>
        <button type="submit" className="btn btn-primary h-10 w-full" disabled={register.isPending}>
          {register.isPending && <span className="loading loading-spinner loading-sm" />}
          Create account
        </button>
      </form>
    </AuthFrame>
  )
}
