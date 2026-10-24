import type { ReactNode } from 'react'
import type { LucideIcon } from 'lucide-react'

interface EmptyStateProps {
  icon: LucideIcon
  title: string
  description: string
  action?: ReactNode
}

/**
 * Empty states are left-aligned prose, not a dashed box in the middle of the
 * screen: this is the first thing a new user reads, so it reads like a page.
 */
export function EmptyState({ icon: Icon, title, description, action }: EmptyStateProps) {
  return (
    <div className="flex max-w-xl flex-col items-start gap-5 py-12">
      <span className="grid size-9 place-items-center rounded-[3px] border border-base-300 text-base-content/60">
        <Icon className="size-4" />
      </span>
      <div>
        <h3 className="display text-xl">{title}</h3>
        <p className="mt-2 text-sm leading-relaxed text-base-content/50">{description}</p>
      </div>
      {action}
    </div>
  )
}
