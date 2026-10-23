export function Spinner({ label }: { label?: string }) {
  return (
    <div className="flex items-center justify-center gap-2 py-10 text-sm text-base-content/60">
      <span className="loading loading-spinner loading-sm" />
      {label && <span>{label}</span>}
    </div>
  )
}

export function FullPageSpinner() {
  return (
    <div className="flex min-h-screen items-center justify-center">
      <span className="loading loading-spinner loading-lg text-primary" />
    </div>
  )
}
