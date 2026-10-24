import { useEffect, useState, type FormEvent } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { ExternalLink } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { ThemeToggle } from '../components/ThemeToggle'
import { useToast } from '../components/ui/Toast'
import { api, errorMessage } from '../lib/api'
import { useMe, useMeta } from '../lib/queries'

const fieldLabel = 'mb-1.5 block text-[13px] font-medium text-base-content/75'
const field = 'input h-10 w-full rounded-[3px] border-base-300'

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
    <div className="space-y-10">
      <header className="border-b border-base-300 pb-6">
        <p className="eyebrow">Account</p>
        <h1 className="display mt-2 text-[40px] leading-none sm:text-5xl">Settings</h1>
      </header>

      <div className="grid gap-x-14 gap-y-10 lg:grid-cols-2">
        <section>
          <h2 className="eyebrow border-b border-base-300 pb-2">Profile</h2>
          <p className="mt-3 text-sm text-base-content/65">How you show up inside Trackete.</p>

          <form className="mt-5 space-y-4" onSubmit={onProfileSubmit}>
            <label className="block">
              <span className={fieldLabel}>Name</span>
              <input
                className={field}
                value={name}
                onChange={(event) => setNameDraft(event.target.value)}
                maxLength={80}
              />
            </label>
            <label className="block">
              <span className={fieldLabel}>Email</span>
              <input className={`${field} num text-[13px]`} value={user?.email ?? ''} disabled />
            </label>
            <button type="submit" className="btn btn-sm btn-primary" disabled={updateName.isPending}>
              {updateName.isPending && <span className="loading loading-spinner loading-xs" />}
              Save profile
            </button>
          </form>
        </section>

        <section>
          <h2 className="eyebrow border-b border-base-300 pb-2">Password</h2>
          <p className="mt-3 text-sm text-base-content/65">Changing it signs out other browsers.</p>

          <form className="mt-5 space-y-4" onSubmit={onPasswordSubmit}>
            <label className="block">
              <span className={fieldLabel}>Current password</span>
              <input
                type="password"
                className={field}
                value={currentPassword}
                onChange={(event) => setCurrentPassword(event.target.value)}
                autoComplete="current-password"
                required
              />
            </label>
            <div className="grid gap-4 sm:grid-cols-2">
              <label className="block">
                <span className={fieldLabel}>New password</span>
                <input
                  type="password"
                  className={field}
                  value={newPassword}
                  onChange={(event) => setNewPassword(event.target.value)}
                  autoComplete="new-password"
                  minLength={8}
                  required
                />
              </label>
              <label className="block">
                <span className={fieldLabel}>Confirm</span>
                <input
                  type="password"
                  className={field}
                  value={confirmPassword}
                  onChange={(event) => setConfirmPassword(event.target.value)}
                  autoComplete="new-password"
                  minLength={8}
                  required
                />
              </label>
            </div>
            <button type="submit" className="btn btn-sm border-base-300" disabled={changePassword.isPending}>
              {changePassword.isPending && <span className="loading loading-spinner loading-xs" />}
              Change password
            </button>
          </form>
        </section>

        <section>
          <h2 className="eyebrow border-b border-base-300 pb-2">Appearance</h2>
          <p className="mt-3 text-sm text-base-content/65">Follow your system, or pick a side.</p>
          <div className="mt-5">
            <ThemeToggle />
          </div>
        </section>

        <section>
          <h2 className="eyebrow border-b border-base-300 pb-2">About</h2>
          <p className="mt-3 text-sm text-base-content/65">
            Trackete is open source. Track anything, then build your own dashboards.
          </p>
          <dl className="num mt-5 space-y-2 text-[13px]">
            <div className="flex items-baseline justify-between gap-4">
              <dt className="text-base-content/65">Version</dt>
              <dd>{meta?.version ?? '—'}</dd>
            </div>
            <div className="flex items-baseline justify-between gap-4">
              <dt className="text-base-content/65">Registration</dt>
              <dd>{meta?.allowRegistration === false ? 'closed' : 'open'}</dd>
            </div>
            <div className="flex items-baseline justify-between gap-4">
              <dt className="text-base-content/65">Storage</dt>
              <dd>sqlite</dd>
            </div>
          </dl>
          <a
            className="btn btn-sm btn-ghost mt-5 gap-2 border-base-300"
            href="https://github.com/a-chris/trackete"
            target="_blank"
            rel="noreferrer noopener"
          >
            <ExternalLink className="size-3.5" />
            View on GitHub
          </a>
        </section>
      </div>

      <section className="flex flex-col gap-4 border-t border-base-300 pt-6 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 className="text-sm font-medium">Sign out</h2>
          <p className="mt-0.5 text-sm text-base-content/65">Ends this session on this device.</p>
        </div>
        <button
          type="button"
          className="btn btn-sm btn-ghost gap-2 border-base-300"
          onClick={() => logout.mutate()}
        >
          Sign out
        </button>
      </section>
    </div>
  )
}
