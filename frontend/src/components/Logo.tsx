export function Logo({ withText = true }: { withText?: boolean }) {
  return (
    <span className="flex items-center gap-2.5">
      <span className="grid size-9 place-items-center rounded-xl bg-primary/15 p-1.5 text-primary">
        <svg viewBox="0 0 64 64" fill="none" className="size-full" aria-hidden="true">
          <path d="M10 46V32" stroke="currentColor" strokeWidth="7" strokeLinecap="round" />
          <path d="M26 46V20" stroke="currentColor" strokeWidth="7" strokeLinecap="round" opacity="0.75" />
          <path d="M42 46V37" stroke="currentColor" strokeWidth="7" strokeLinecap="round" opacity="0.55" />
          <circle cx="46" cy="18" r="5" fill="currentColor" />
        </svg>
      </span>
      {withText && (
        <span className="text-lg font-semibold tracking-tight">
          Trackete
        </span>
      )}
    </span>
  )
}
