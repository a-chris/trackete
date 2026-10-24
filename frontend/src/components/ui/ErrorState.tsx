import { RotateCw, TriangleAlert } from 'lucide-react'

interface ErrorStateProps {
  title: string
  description?: string
  onRetry?: () => void
  retrying?: boolean
}

export function ErrorState({ title, description, onRetry, retrying = false }: ErrorStateProps) {
  return (
    <div
      role="alert"
      className="flex flex-col items-center justify-center rounded-box border border-error/30 bg-error/5 px-6 py-14 text-center"
    >
      <div className="mb-3 grid size-12 place-items-center rounded-2xl bg-error/10 text-error">
        <TriangleAlert className="size-6" />
      </div>
      <h3 className="text-base font-semibold">{title}</h3>
      {description && <p className="mt-1 max-w-md text-sm text-base-content/70">{description}</p>}
      {onRetry && (
        <button
          type="button"
          className="btn btn-sm btn-outline mt-5"
          onClick={onRetry}
          disabled={retrying}
        >
          {retrying ? (
            <span className="loading loading-spinner loading-xs" />
          ) : (
            <RotateCw className="size-3.5" />
          )}
          Retry
        </button>
      )}
    </div>
  )
}
