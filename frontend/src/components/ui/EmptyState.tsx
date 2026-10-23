import type { ReactNode } from 'react'
import type { LucideIcon } from 'lucide-react'

interface EmptyStateProps {
  icon: LucideIcon
  title: string
  description: string
  action?: ReactNode
}

export function EmptyState({ icon: Icon, title, description, action }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center rounded-box border border-dashed border-base-300 bg-base-100/60 px-6 py-14 text-center">
      <div className="mb-4 grid size-12 place-items-center rounded-2xl bg-primary/10 text-primary">
        <Icon className="size-6" />
      </div>
      <h3 className="text-base font-semibold">{title}</h3>
      <p className="mt-1 max-w-md text-sm text-base-content/60">{description}</p>
      {action && <div className="mt-5">{action}</div>}
    </div>
  )
}
