import { useEffect, useState, type FormEvent } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { Logo } from '../components/Logo'
import { ThemeToggle } from '../components/ThemeToggle'
import { Spinner } from '../components/ui/Spinner'
import { api, errorMessage } from '../lib/api'
import { dayKey, formatLongDate } from '../lib/date'
import { useInvitation } from '../lib/queries'

const field = 'input h-10 w-full rounded-[3px] border-base-300'
const label = 'mb-1.5 block text-[13px] font-medium text-base-content/75'

// InvitePage is public: the signed token in the URL is the credential, so the
// page has to work before the invitee has an account or a session.
export function InvitePage() {
  const { token = '' } = useParams()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const invitation = useInvitation(token)

  const [name, setName] = useState('')
  const [password, setPassword] = useState('')
  const [declined, setDeclined] = useState(false)

  useEffect(() => {
    document.title = 'Invitation · Trackete'
  }, [])

  const accept = useMutation({
    mutationFn: () =>
      api.acceptInvitation(token, invitation.data?.needsAccount ? { name, password } : undefined),
    onSuccess: async (result) => {
      queryClient.setQueryData(['me'], result.user)
      await queryClient.invalidateQueries({ queryKey: ['trackers'] })
      navigate(`/trackers/${result.tracker.id}`, { replace: true })
    },
  })

  const decline = useMutation({
    mutationFn: () => api.rejectInvitation(token),
    onSuccess: () => setDeclined(true),
  })

  const onSubmit = (event: FormEvent) => {
    event.preventDefault()
    accept.mutate()
  }

  const details = invitation.data
  // The API refuses an expired link, so the page only needs to state the
  // deadline rather than compare clocks.
  const expiresOn = details?.expiresAt ? dayKey(new Date(details.expiresAt)) : ''

  return (
    <div className="flex min-h-screen flex-col bg-base-100 px-6 py-10">
      <div className="mx-auto flex w-full max-w-md flex-1 flex-col justify-center">
        <Link to="/" aria-label="Trackete home" className="mb-10 inline-block">
          <Logo />
        </Link>

        <h1 className="display text-2xl">Tracker invitation</h1>
        <p className="mt-2 text-sm text-base-content/65">
          Someone shared a tracker with you. Accepting lets you log your own entries and see the
          combined charts.
        </p>

        <div className="mt-8">
          {invitation.isLoading ? (
            <Spinner label="Checking invitation…" />
          ) : invitation.isError ? (
            <div className="alert alert-error rounded-[3px] text-sm">
              {errorMessage(invitation.error)}
            </div>
          ) : declined ? (
            <div className="surface p-5">
              <p className="text-sm">You declined this invitation. Nothing else happens.</p>
              <Link to="/" className="btn btn-sm mt-4 border-base-300">
                Go to Trackete
              </Link>
            </div>
          ) : details ? (
            <form className="space-y-4" onSubmit={onSubmit}>
              <div className="border-y border-base-300 py-4">
                <p className="eyebrow">Tracker</p>
                <p className="mt-1 text-lg leading-tight">{details.trackerName}</p>
                <p className="mt-2 text-xs text-base-content/65">Invitation for {details.email}</p>
                {expiresOn && (
                  <p className="mt-2 text-xs text-base-content/60">
                    The link stops working after {formatLongDate(expiresOn)}.
                  </p>
                )}
              </div>

              {(accept.isError || decline.isError) && (
                <div className="alert alert-error rounded-[3px] text-sm">
                  {errorMessage(accept.error ?? decline.error)}
                </div>
              )}

              {details.needsAccount && (
                <>
                  <p className="text-sm text-base-content/70">
                    New here? Pick a name and a password to create your account.
                  </p>
                  <label className="block">
                    <span className={label}>Name</span>
                    <input
                      className={field}
                      value={name}
                      onChange={(event) => setName(event.target.value)}
                      placeholder="Alex"
                      maxLength={80}
                      required
                    />
                  </label>
                  <label className="block">
                    <span className={label}>Password</span>
                    <input
                      className={field}
                      type="password"
                      value={password}
                      onChange={(event) => setPassword(event.target.value)}
                      autoComplete="new-password"
                      minLength={8}
                      required
                    />
                  </label>
                </>
              )}

              {!details.needsAccount && (
                <p className="text-sm text-base-content/70">
                  You already have an account for {details.email}. Accepting signs you in.
                </p>
              )}

              <div className="flex flex-wrap gap-2">
                <button
                  type="submit"
                  className="btn btn-primary"
                  disabled={accept.isPending}
                >
                  {accept.isPending && <span className="loading loading-spinner loading-sm" />}
                  Accept invitation
                </button>
                <button
                  type="button"
                  className="btn btn-ghost"
                  onClick={() => decline.mutate()}
                  disabled={decline.isPending}
                >
                  {decline.isPending && <span className="loading loading-spinner loading-sm" />}
                  Decline
                </button>
              </div>
            </form>
          ) : null}
        </div>

        <div className="mt-10 border-t border-base-300 pt-4">
          <ThemeToggle compact />
        </div>
      </div>
    </div>
  )
}
