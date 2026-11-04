import { useEffect } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Inbox } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { EmptyState } from '../components/ui/EmptyState'
import { ErrorState } from '../components/ui/ErrorState'
import { Spinner } from '../components/ui/Spinner'
import { useToast } from '../components/ui/Toast'
import { api, errorMessage } from '../lib/api'
import { formatDate } from '../lib/date'
import { usePendingInvitations } from '../lib/queries'
import type { PendingInvitation } from '../lib/types'

// The invitee's side of sharing. Until now an invitation only existed as an
// email, so anybody who closed it had no way back in; this page is that way
// back, and the nav dot is how they learn it is waiting.
export function InvitationsPage() {
  const toast = useToast()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const {
    data: invitations = [],
    isLoading,
    isError,
    error,
    refetch,
    isFetching,
  } = usePendingInvitations()

  useEffect(() => {
    document.title = 'Invitations · Trackete'
  }, [])

  const accept = useMutation({
    mutationFn: (token: string) => api.acceptInvitation(token),
    onSuccess: async (result) => {
      // Accepting signs the invitee in, so the cached session becomes theirs.
      queryClient.setQueryData(['me'], result.user)
      await queryClient.invalidateQueries()
      navigate(`/trackers/${result.tracker.id}`)
    },
    onError: (error) => toast.push('error', errorMessage(error)),
  })

  const decline = useMutation({
    mutationFn: (token: string) => api.rejectInvitation(token),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['pending-invitations'] })
      toast.push('success', 'Invitation declined.')
    },
    onError: (error) => toast.push('error', errorMessage(error)),
  })

  const renderRow = (invitation: PendingInvitation) => {
    const accepting = accept.isPending && accept.variables === invitation.token
    const declining = decline.isPending && decline.variables === invitation.token
    const expires = formatDate(invitation.expiresAt.slice(0, 10), {
      month: 'short',
      day: 'numeric',
    })

    return (
      <li key={invitation.id} className="flex items-center gap-3 py-3.5 sm:gap-4">
        <div className="min-w-0 flex-1">
          <p className="truncate text-[15px] font-medium leading-tight">{invitation.trackerName}</p>
          <p className="eyebrow mt-1 truncate font-semibold text-base-content/80">
            {invitation.inviterName ? `from ${invitation.inviterName}` : 'shared with you'}
            <span className="mx-1.5 text-base-content/40">/</span>
            expires {expires}
          </p>
        </div>

        <div className="flex shrink-0 items-center gap-1.5">
          <button
            type="button"
            className="btn btn-sm btn-primary"
            onClick={() => accept.mutate(invitation.token)}
            disabled={accepting || declining}
          >
            {accepting && <span className="loading loading-spinner loading-xs" />}
            Accept
          </button>
          <button
            type="button"
            className="btn btn-sm btn-ghost"
            onClick={() => decline.mutate(invitation.token)}
            disabled={accepting || declining}
          >
            {declining && <span className="loading loading-spinner loading-xs" />}
            Decline
          </button>
        </div>
      </li>
    )
  }

  return (
    <div className="space-y-8">
      <header className="flex flex-wrap items-end justify-between gap-x-8 gap-y-5 border-b border-base-300 pb-6">
        <div>
          <p className="eyebrow">Shared with you</p>
          <h1 className="display mt-2 text-[40px] leading-none sm:text-5xl">Invitations</h1>
        </div>
        {invitations.length > 0 && (
          <div>
            <p className="eyebrow">waiting</p>
            <p className="num mt-1.5 text-2xl leading-none">{invitations.length}</p>
          </div>
        )}
      </header>

      {isLoading ? (
        <Spinner label="Loading invitations…" />
      ) : isError ? (
        <ErrorState
          title="Couldn't load your invitations"
          description={errorMessage(error)}
          onRetry={() => void refetch()}
          retrying={isFetching}
        />
      ) : invitations.length === 0 ? (
        <EmptyState
          icon={Inbox}
          title="Nothing is waiting"
          description="When someone shares a tracker with your address it appears here until you accept or decline it."
        />
      ) : (
        <ul className="divide-y divide-base-300 border-b border-base-300">
          {invitations.map(renderRow)}
        </ul>
      )}
    </div>
  )
}
