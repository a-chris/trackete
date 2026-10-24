import { useState, type FormEvent } from 'react'
import { Mail, RotateCw, Send, UserMinus } from 'lucide-react'
import { ConfirmDialog } from '../ui/ConfirmDialog'
import { Spinner } from '../ui/Spinner'
import { useToast } from '../ui/Toast'
import { errorMessage } from '../../lib/api'
import { useInviteSuggestions, useSharingMutations, useTrackerSharing } from '../../lib/queries'
import type { Invitation, TrackerMember } from '../../lib/types'

const label = 'mb-1.5 block font-mono text-[10px] uppercase tracking-[0.14em] text-base-content/65'
const suggestionListId = 'tracker-sharing-suggestions'

const statusLabel: Record<Invitation['status'], string> = {
  pending: 'Not delivered',
  delivered: 'Invited',
  accepted: 'Joined',
  rejected: 'Declined',
  removed: 'Removed',
}

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
  const { invite, resend, removeMember } = useSharingMutations(trackerId)
  const [email, setEmail] = useState('')
  const [removing, setRemoving] = useState<TrackerMember | null>(null)

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
  const pendingInvitations = (sharing.data?.invitations ?? []).filter(
    (invitation) => invitation.status === 'pending' || invitation.status === 'delivered',
  )

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
        <p className={label}>Members</p>
        <ul className="divide-y divide-base-300 border-y border-base-300">
          {members.map((member) => (
            <li key={member.userId} className="flex items-center gap-3 py-2.5">
              <span className="min-w-0 flex-1">
                <span className="block truncate text-[13px]">{member.name || member.email}</span>
                <span className="block truncate text-[11px] text-base-content/60">{member.email}</span>
              </span>
              <span className="eyebrow">{member.role === 'owner' ? 'Owner' : 'Member'}</span>
              {!readOnly && member.role === 'member' && (
                <button
                  type="button"
                  className="grid size-7 place-items-center rounded-[3px] text-base-content/65 transition-colors hover:bg-error/10 hover:text-error"
                  onClick={() => setRemoving(member)}
                  aria-label={`Remove ${member.email}`}
                >
                  <UserMinus className="size-3.5" />
                </button>
              )}
            </li>
          ))}
        </ul>
      </div>

      {pendingInvitations.length > 0 && (
        <div>
          <p className={label}>Open invitations</p>
          <ul className="divide-y divide-base-300 border-y border-base-300">
            {pendingInvitations.map((invitation) => (
              <li key={invitation.id} className="flex items-center gap-3 py-2.5">
                <Mail className="size-3.5 shrink-0 text-base-content/55" />
                <span className="min-w-0 flex-1 truncate text-[13px]">{invitation.email}</span>
                <span className="eyebrow">{statusLabel[invitation.status]}</span>
                {!readOnly && (
                  <button
                    type="button"
                    className="grid size-7 place-items-center rounded-[3px] text-base-content/65 transition-colors hover:bg-base-content/5 hover:text-base-content"
                    onClick={() => resendInvite(invitation)}
                    disabled={resend.isPending}
                    aria-label={`Resend the invitation to ${invitation.email}`}
                  >
                    <RotateCw className="size-3.5" />
                  </button>
                )}
              </li>
            ))}
          </ul>
        </div>
      )}

      <ConfirmDialog
        open={Boolean(removing)}
        title="Remove member"
        message={`${(removing?.name || removing?.email) ?? 'This person'} will lose access to this tracker. Their entries stay in the history.`}
        confirmLabel="Remove"
        busy={removeMember.isPending}
        onConfirm={confirmRemove}
        onClose={() => setRemoving(null)}
      />
    </div>
  )
}
