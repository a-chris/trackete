import { useEffect, useId, useRef, type ReactNode } from 'react'
import { X } from 'lucide-react'

interface ModalProps {
  open: boolean
  title: string
  onClose: () => void
  children: ReactNode
  footer?: ReactNode
  wide?: boolean
}

const FOCUSABLE = [
  'a[href]',
  'area[href]',
  'button:not([disabled])',
  'input:not([disabled]):not([type="hidden"])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  'iframe',
  '[contenteditable="true"]',
  '[tabindex]:not([tabindex="-1"])',
].join(', ')

function focusableAncestor(element: Element | null): HTMLElement | null {
  let node = element instanceof Element ? element : null
  while (node) {
    if (node.matches(FOCUSABLE)) return node as HTMLElement
    node = node.parentElement
  }
  return null
}

function focusableWithin(container: HTMLElement): HTMLElement[] {
  return Array.from(container.querySelectorAll<HTMLElement>(FOCUSABLE)).filter(
    (element) => element.offsetParent !== null || element === document.activeElement,
  )
}

// Remember the element the user last engaged with, across modal mounts, so focus
// can return to the trigger even in browsers (Safari/WebKit) that do not focus a
// button when it is clicked.
let lastEngaged: HTMLElement | null = null
if (typeof document !== 'undefined') {
  document.addEventListener(
    'pointerdown',
    (event) => {
      const target = focusableAncestor(event.target as Element | null)
      if (target) lastEngaged = target
    },
    true,
  )
}

export function Modal({ open, title, onClose, children, footer, wide = false }: ModalProps) {
  const titleId = useId()
  const boxRef = useRef<HTMLDivElement>(null)
  const restoreFocusRef = useRef<HTMLElement | null>(null)

  useEffect(() => {
    if (!open) return
    const box = boxRef.current
    const active = document.activeElement
    const activeOutside = active instanceof HTMLElement && active !== document.body && !box?.contains(active)
    restoreFocusRef.current = activeOutside ? active : lastEngaged

    if (box && !box.contains(document.activeElement)) {
      const initial = focusableWithin(box)[0] ?? box
      initial.focus()
    }

    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault()
        onClose()
        return
      }
      if (event.key !== 'Tab') return
      if (!box) return
      const focusable = focusableWithin(box)
      if (focusable.length === 0) {
        event.preventDefault()
        box.focus()
        return
      }
      const first = focusable[0]
      const last = focusable[focusable.length - 1]
      const current = document.activeElement
      if (!(current instanceof HTMLElement) || !box.contains(current)) {
        event.preventDefault()
        ;(event.shiftKey ? last : first).focus()
        return
      }
      // Drive Tab navigation ourselves: WebKit skips non-text controls by
      // default, so leaving it to the browser would let focus escape the dialog
      // (and make buttons unreachable by keyboard).
      const index = focusable.indexOf(current)
      const next = event.shiftKey
        ? focusable[index <= 0 ? focusable.length - 1 : index - 1]
        : focusable[index === -1 || index === focusable.length - 1 ? 0 : index + 1]
      event.preventDefault()
      next.focus()
    }

    document.addEventListener('keydown', onKey)
    const previousOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.removeEventListener('keydown', onKey)
      document.body.style.overflow = previousOverflow
      const restore = restoreFocusRef.current
      if (restore && document.contains(restore)) restore.focus()
    }
  }, [open, onClose])

  if (!open) return null

  return (
    <div className="modal modal-open modal-bottom sm:modal-middle">
      <div
        ref={boxRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        tabIndex={-1}
        className={`modal-box ${wide ? 'max-w-2xl' : 'max-w-lg'}`}
      >
        <div className="flex items-start justify-between gap-4">
          <h3 id={titleId} className="text-lg font-semibold tracking-tight">
            {title}
          </h3>
          <button
            type="button"
            className="btn btn-ghost btn-sm btn-circle -mr-2 -mt-1"
            onClick={onClose}
            aria-label="Close"
          >
            <X className="size-4" />
          </button>
        </div>
        <div className="mt-4">{children}</div>
        {footer && <div className="modal-action">{footer}</div>}
      </div>
      <button
        type="button"
        className="modal-backdrop"
        onClick={onClose}
        tabIndex={-1}
        aria-hidden="true"
      >
        close
      </button>
    </div>
  )
}
