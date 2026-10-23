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
    <div className="flex min-h-screen flex-col items-center justify-center bg-base-200 px-4 py-10">
      <div className="w-full max-w-md">
        <div className="mb-6 flex flex-col items-center gap-2 text-center">
          <Logo />
          <p className="max-w-xs text-sm text-base-content/50">
            Track anything, then build the dashboard you actually want.
          </p>
        </div>
        <div className="surface p-6 shadow-sm sm:p-8">
          <h1 className="text-xl font-semibold tracking-tight">{title}</h1>
          <p className="mt-1 text-sm text-base-content/60">{subtitle}</p>
          <div className="mt-6">{children}</div>
        </div>
        <div className="mt-5 text-center text-sm text-base-content/60">{footer}</div>
        <div className="mt-8 flex justify-center">
          <ThemeToggle compact />
        </div>
      </div>
    </div>
  )
}

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
        {login.isError && <div className="alert alert-error text-sm">{errorMessage(login.error)}</div>}
        <label className="form-control w-full">
          <span className="mb-1.5 block text-sm font-medium">Email</span>
          <input
            type="email"
            className="input w-full"
            value={email}
            onChange={(event) => setEmail(event.target.value)}
            autoComplete="email"
            required
            autoFocus
          />
        </label>
        <label className="form-control w-full">
          <span className="mb-1.5 block text-sm font-medium">Password</span>
          <input
            type="password"
            className="input w-full"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            autoComplete="current-password"
            required
          />
        </label>
        <button type="submit" className="btn btn-primary w-full" disabled={login.isPending}>
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
        <p className="text-sm text-base-content/70">
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
          <div className="alert alert-error text-sm">{errorMessage(register.error)}</div>
        )}
        <label className="form-control w-full">
          <span className="mb-1.5 block text-sm font-medium">Name</span>
          <input
            type="text"
            className="input w-full"
            value={name}
            onChange={(event) => setName(event.target.value)}
            autoComplete="name"
            placeholder="Alex"
            maxLength={80}
          />
        </label>
        <label className="form-control w-full">
          <span className="mb-1.5 block text-sm font-medium">Email</span>
          <input
            type="email"
            className="input w-full"
            value={email}
            onChange={(event) => setEmail(event.target.value)}
            autoComplete="email"
            required
            autoFocus
          />
        </label>
        <label className="form-control w-full">
          <span className="mb-1.5 block text-sm font-medium">Password</span>
          <input
            type="password"
            className="input w-full"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            autoComplete="new-password"
            minLength={8}
            required
          />
          <span className="mt-1.5 block text-xs text-base-content/50">At least 8 characters.</span>
        </label>
        <button type="submit" className="btn btn-primary w-full" disabled={register.isPending}>
          {register.isPending && <span className="loading loading-spinner loading-sm" />}
          Create account
        </button>
      </form>
    </AuthFrame>
  )
}
