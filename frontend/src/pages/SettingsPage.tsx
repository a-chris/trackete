import { useEffect, useState, type FormEvent } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { CircleUserRound, ExternalLink, LockKeyhole, LogOut, Palette } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { ThemeToggle } from '../components/ThemeToggle'
import { useToast } from '../components/ui/Toast'
import { api, errorMessage } from '../lib/api'
import { useMe, useMeta } from '../lib/queries'

export function SettingsPage() {
  const toast = useToast()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { data: user } = useMe()
  const { data: meta } = useMeta()

  const [nameDraft, setNameDraft] = useState<string | null>(null)
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')

  useEffect(() => {
    document.title = 'Settings · Trackete'
  }, [])

  const name = nameDraft ?? user?.name ?? ''

  const updateName = useMutation({
    mutationFn: (value: string) => api.updateMe(value),
    onSuccess: (updated) => {
      queryClient.setQueryData(['me'], updated)
      setNameDraft(null)
      toast.push('success', 'Profile updated.')
    },
    onError: (error) => toast.push('error', errorMessage(error)),
  })

  const changePassword = useMutation({
    mutationFn: () => api.changePassword(currentPassword, newPassword),
    onSuccess: () => {
      setCurrentPassword('')
      setNewPassword('')
      setConfirmPassword('')
      toast.push('success', 'Password changed.')
    },
    onError: (error) => toast.push('error', errorMessage(error)),
  })

  const logout = useMutation({
    mutationFn: api.logout,
    onSuccess: () => {
      queryClient.clear()
      navigate('/login', { replace: true })
    },
  })

  const onProfileSubmit = (event: FormEvent) => {
    event.preventDefault()
    if (!name.trim()) {
      toast.push('error', 'Name cannot be empty.')
      return
    }
    updateName.mutate(name.trim())
  }

  const onPasswordSubmit = (event: FormEvent) => {
    event.preventDefault()
    if (newPassword.length < 8) {
      toast.push('error', 'New password must be at least 8 characters.')
      return
    }
    if (newPassword !== confirmPassword) {
      toast.push('error', 'New passwords do not match.')
      return
    }
    changePassword.mutate()
  }

  return (
    <div className="space-y-6">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight sm:text-3xl">Settings</h1>
        <p className="mt-1 text-sm text-base-content/60">Your account, appearance and instance details.</p>
      </header>

      <div className="grid gap-6 lg:grid-cols-2">
        <section className="surface p-5">
          <div className="flex items-center gap-2">
            <CircleUserRound className="size-4 text-primary" />
            <h2 className="font-semibold">Profile</h2>
          </div>
          <p className="mt-1 text-sm text-base-content/55">How you show up inside Trackete.</p>

          <form className="mt-4 space-y-3" onSubmit={onProfileSubmit}>
            <label className="form-control">
              <span className="mb-1.5 block text-sm font-medium">Name</span>
              <input
                className="input input-bordered w-full"
                value={name}
                onChange={(event) => setNameDraft(event.target.value)}
                maxLength={80}
              />
            </label>
            <label className="form-control">
              <span className="mb-1.5 block text-sm font-medium">Email</span>
              <input className="input input-bordered w-full" value={user?.email ?? ''} disabled />
            </label>
            <button type="submit" className="btn btn-primary btn-sm" disabled={updateName.isPending}>
              {updateName.isPending && <span className="loading loading-spinner loading-xs" />}
              Save profile
            </button>
          </form>
        </section>

        <section className="surface p-5">
          <div className="flex items-center gap-2">
            <LockKeyhole className="size-4 text-primary" />
            <h2 className="font-semibold">Password</h2>
          </div>
          <p className="mt-1 text-sm text-base-content/55">Changing it signs out other browsers.</p>

          <form className="mt-4 space-y-3" onSubmit={onPasswordSubmit}>
            <label className="form-control">
              <span className="mb-1.5 block text-sm font-medium">Current password</span>
              <input
                type="password"
                className="input input-bordered w-full"
                value={currentPassword}
                onChange={(event) => setCurrentPassword(event.target.value)}
                autoComplete="current-password"
                required
              />
            </label>
            <div className="grid gap-3 sm:grid-cols-2">
              <label className="form-control">
                <span className="mb-1.5 block text-sm font-medium">New password</span>
                <input
                  type="password"
                  className="input input-bordered w-full"
                  value={newPassword}
                  onChange={(event) => setNewPassword(event.target.value)}
                  autoComplete="new-password"
                  minLength={8}
                  required
                />
              </label>
              <label className="form-control">
                <span className="mb-1.5 block text-sm font-medium">Confirm</span>
                <input
                  type="password"
                  className="input input-bordered w-full"
                  value={confirmPassword}
                  onChange={(event) => setConfirmPassword(event.target.value)}
                  autoComplete="new-password"
                  minLength={8}
                  required
                />
              </label>
            </div>
            <button type="submit" className="btn btn-sm" disabled={changePassword.isPending}>
              {changePassword.isPending && <span className="loading loading-spinner loading-xs" />}
              Change password
            </button>
          </form>
        </section>

        <section className="surface p-5">
          <div className="flex items-center gap-2">
            <Palette className="size-4 text-primary" />
            <h2 className="font-semibold">Appearance</h2>
          </div>
          <p className="mt-1 text-sm text-base-content/55">Follow your system, or pick a side.</p>
          <div className="mt-4">
            <ThemeToggle />
          </div>
        </section>

        <section className="surface p-5">
          <h2 className="font-semibold">About</h2>
          <p className="mt-1 text-sm text-base-content/55">
            Trackete is open source. Track anything, then build your own dashboards.
          </p>
          <dl className="mt-4 grid grid-cols-2 gap-2 text-sm">
            <dt className="text-base-content/50">Version</dt>
            <dd className="text-right font-medium">{meta?.version ?? '—'}</dd>
            <dt className="text-base-content/50">Registration</dt>
            <dd className="text-right font-medium">
              {meta?.allowRegistration === false ? 'Closed' : 'Open'}
            </dd>
            <dt className="text-base-content/50">Storage</dt>
            <dd className="text-right font-medium">SQLite</dd>
          </dl>
          <a
            className="btn btn-outline btn-sm mt-4"
            href="https://github.com/a-chris/trackete"
            target="_blank"
            rel="noreferrer noopener"
          >
            <ExternalLink className="size-3.5" />
            View on GitHub
          </a>
        </section>
      </div>

      <section className="surface flex flex-col gap-3 p-5 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 className="font-semibold">Sign out</h2>
          <p className="text-sm text-base-content/55">Ends this session on this device.</p>
        </div>
        <button type="button" className="btn btn-outline btn-sm" onClick={() => logout.mutate()}>
          <LogOut className="size-3.5" />
          Sign out
        </button>
      </section>
    </div>
  )
}
