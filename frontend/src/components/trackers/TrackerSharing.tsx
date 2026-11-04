import { useState, type FormEvent } from 'react'
import { RotateCw, Send, UserMinus, X } from 'lucide-react'
import { ConfirmDialog } from '../ui/ConfirmDialog'
import { Spinner } from '../ui/Spinner'
import { useToast } from '../ui/Toast'
import { errorMessage } from '../../lib/api'
import { useInviteSuggestions, useSharingMutations, useTrackerSharing } from '../../lib/queries'
import type { Invitation, TrackerMember } from '../../lib/types'

const label = 'mb-1.5 block font-mono text-[10px] uppercase tracking-[0.14em] text-base-content/65'
const suggestionListId = 'tracker-sharing-suggestions'
const badge = 'eyebrow shrink-0'
const action =
  'grid size-7 place-items-center rounded-[3px] text-base-content/65 transition-colors disabled:opacity-50'

// One line of the sharing list: the owner, an accepted member, or an invitation
// nobody has answered yet.
type ShareEntry =
  | { kind: 'owner'; member: TrackerMember }
  | { kind: 'member'; member: TrackerMember }
  | { kind: 'invite'; invitation: Invitation }

interface TrackerSharingProps {
  trackerId: string
  /** Members see who shares the tracker but cannot invite or remove anyone. */
  readOnly?: boolean
}

// TrackerSharing renders the membership of one tracker: who is in, which
// invitations are still open, and the owner's controls for both.
export function TrackerSharing({ trackerId, readOnly = false }: TrackerSharingProps) {
  const toast = useToast()
  const sharing = useTrackerSharing(trackerId)
  const suggestions = useInviteSuggestions()
  const { invite, resend, cancelInvite, removeMember } = useSharingMutations(trackerId)
  const [email, setEmail] = useState('')
  const [removing, setRemoving] = useState<TrackerMember | null>(null)
  const [cancelling, setCancelling] = useState<Invitation | null>(null)

  const submit = (event: FormEvent) => {
    event.preventDefault()
    const address = email.trim()
    if (!address) return
    invite.mutate(address, {
      onSuccess: (invitation) => {
        setEmail('')
        toast.push(
          invitation.status === 'delivered' ? 'success' : 'info',
          invitation.status === 'delivered'
            ? `Invited ${invitation.email}.`
            : `Saved the invitation for ${invitation.email}, but the email could not be sent yet.`,
        )
      },
      onError: (error) => toast.push('error', errorMessage(error)),
    })
  }

  const resendInvite = (invitation: Invitation) => {
    resend.mutate(invitation.id, {
      onSuccess: (updated) =>
        toast.push(
          updated.status === 'delivered' ? 'success' : 'error',
          updated.status === 'delivered'
            ? `Invitation sent again to ${updated.email}.`
            : `Could not deliver the invitation to ${updated.email}.`,
        ),
      onError: (error) => toast.push('error', errorMessage(error)),
    })
  }

  const confirmCancel = () => {
    if (!cancelling) return
    cancelInvite.mutate(cancelling.id, {
      onSuccess: (invitation) => {
        toast.push('success', `Cancelled the invitation for ${invitation.email}.`)
        setCancelling(null)
      },
      onError: (error) => toast.push('error', errorMessage(error)),
    })
  }

  const confirmRemove = () => {
    if (!removing) return
    removeMember.mutate(removing.userId, {
      onSuccess: () => {
        toast.push('success', `${removing.name || removing.email} no longer has access.`)
        setRemoving(null)
      },
      onError: (error) => toast.push('error', errorMessage(error)),
    })
  }

  if (sharing.isLoading) return <Spinner label="Loading sharing…" />
  if (sharing.isError) {
    return (
      <p role="alert" className="text-sm text-base-content/70">
        Couldn't load sharing. {errorMessage(sharing.error)}
      </p>
    )
  }

  const members = sharing.data?.members ?? []
  const entries: ShareEntry[] = [
    ...members
      .filter((member) => member.role === 'owner')
      .map((member): ShareEntry => ({ kind: 'owner', member })),
    ...members
      .filter((member) => member.role !== 'owner')
      .map((member): ShareEntry => ({ kind: 'member', member })),
    ...(sharing.data?.invitations ?? [])
      .filter((invitation) => invitation.status === 'pending' || invitation.status === 'delivered')
      .map((invitation): ShareEntry => ({ kind: 'invite', invitation })),
  ]

  return (
    <div className="space-y-6">
      {!readOnly && (
        <form className="space-y-2" onSubmit={submit}>
          <label className="form-control">
            <span className={label}>Invite by email</span>
            <div className="flex gap-2">
              <input
                type="email"
                className="input w-full rounded-[3px] border-base-300"
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                placeholder="friend@example.com"
                list={suggestionListId}
                required
              />
              <button type="submit" className="btn btn-primary gap-2" disabled={invite.isPending}>
                {invite.isPending ? (
                  <span className="loading loading-spinner loading-xs" />
                ) : (
                  <Send className="size-3.5" />
                )}
                Invite
              </button>
            </div>
          </label>
          <datalist id={suggestionListId}>
            {(suggestions.data ?? []).map((suggestion) => (
              <option key={suggestion.email} value={suggestion.email}>
                {suggestion.name}
              </option>
            ))}
          </datalist>
          <p className="text-xs text-base-content/65">
            They get a link that lets them accept and start logging. Only you can edit the
            tracker, invite people or remove them.
          </p>
        </form>
      )}

      <div>
        <p className={label}>Shared with</p>
        <ul className="divide-y divide-base-300 border-y border-base-300">
          {entries.map((entry) => {
            if (entry.kind === 'invite') {
              const { invitation } = entry
              return (
                <li key={invitation.id} className="flex items-center gap-3 py-2.5">
                  <span className="min-w-0 flex-1 truncate text-[13px]">{invitation.email}</span>
                  <span
                    className={badge}
                    title={
                      invitation.status === 'pending'
                        ? 'The invitation email could not be sent yet.'
                        : undefined
                    }
                  >
                    pending
                  </span>
                  {!readOnly && (
                    <span className="flex shrink-0 items-center gap-0.5">
                      <button
                        type="button"
                        className={`${action} hover:bg-base-content/5 hover:text-base-content`}
                        onClick={() => resendInvite(invitation)}
                        disabled={resend.isPending}
                        title={`Resend the invitation to ${invitation.email}`}
                        aria-label={`Resend the invitation to ${invitation.email}`}
                      >
                        <RotateCw className="size-3.5" />
                      </button>
                      <button
                        type="button"
                        className={`${action} hover:bg-error/10 hover:text-error`}
                        onClick={() => setCancelling(invitation)}
                        title={`Cancel the invitation to ${invitation.email}`}
                        aria-label={`Cancel the invitation to ${invitation.email}`}
                      >
                        <X className="size-3.5" />
                      </button>
                    </span>
                  )}
                </li>
              )
            }

            const { member } = entry
            return (
              <li key={member.userId} className="flex items-center gap-3 py-2.5">
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-[13px]">{member.name || member.email}</span>
                  <span className="block truncate text-[11px] text-base-content/60">
                    {member.email}
                  </span>
                </span>
                <span className={badge}>{entry.kind === 'owner' ? 'owner' : 'accepted'}</span>
                {!readOnly && entry.kind === 'member' && (
                  <button
                    type="button"
                    className={`${action} hover:bg-error/10 hover:text-error`}
                    onClick={() => setRemoving(member)}
                    title={`Remove ${member.email}`}
                    aria-label={`Remove ${member.email}`}
                  >
                    <UserMinus className="size-3.5" />
                  </button>
                )}
              </li>
            )
          })}
        </ul>
        {!readOnly && entries.length === 1 && (
          <p className="mt-2 text-xs text-base-content/65">
            Only you so far. Invite someone to log alongside you.
          </p>
        )}
      </div>

      <ConfirmDialog
        open={Boolean(removing)}
        title="Remove member"
        message={`${(removing?.name || removing?.email) ?? 'This person'} will lose access to this tracker. Their entries stay in the history.`}
        confirmLabel="Remove"
        busy={removeMember.isPending}
        onConfirm={confirmRemove}
        onClose={() => setRemoving(null)}
      />

      <ConfirmDialog
        open={Boolean(cancelling)}
        title="Cancel this invitation?"
        message={`The invitation link for ${cancelling?.email ?? 'this address'} stops working. You can invite the address again later.`}
        confirmLabel="Cancel invite"
        busy={cancelInvite.isPending}
        onConfirm={confirmCancel}
        onClose={() => setCancelling(null)}
      />
    </div>
  )
}
