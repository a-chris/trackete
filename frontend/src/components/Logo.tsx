export function Logo({ withText = true }: { withText?: boolean }) {
  return (
    <span className="flex items-center gap-2.5">
      <svg viewBox="0 0 64 64" fill="none" className="size-7 text-base-content" aria-hidden="true">
        <path d="M10 46V32" stroke="currentColor" strokeWidth="9" strokeLinecap="butt" />
        <path d="M26 46V20" stroke="currentColor" strokeWidth="9" strokeLinecap="butt" opacity="0.7" />
        <path d="M42 46V37" stroke="currentColor" strokeWidth="9" strokeLinecap="butt" opacity="0.45" />
        <circle cx="46" cy="18" r="5" className="fill-secondary" />
      </svg>
      {withText && (
        <span className="text-[17px] font-semibold tracking-[-0.04em]">
          Trackete<span className="text-secondary">.</span>
        </span>
      )}
    </span>
  )
}
