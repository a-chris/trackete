import { useEffect, type ReactNode } from 'react'
import { X } from 'lucide-react'

interface ModalProps {
  open: boolean
  title: string
  onClose: () => void
  children: ReactNode
  footer?: ReactNode
  wide?: boolean
}

export function Modal({ open, title, onClose, children, footer, wide = false }: ModalProps) {
  useEffect(() => {
    if (!open) return
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', onKey)
    const previousOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.removeEventListener('keydown', onKey)
      document.body.style.overflow = previousOverflow
    }
  }, [open, onClose])

  if (!open) return null

  return (
    <div className="modal modal-open modal-bottom sm:modal-middle" role="dialog" aria-modal="true">
      <div className={`modal-box ${wide ? 'max-w-2xl' : 'max-w-lg'}`}>
        <div className="flex items-start justify-between gap-4">
          <h3 className="text-lg font-semibold tracking-tight">{title}</h3>
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
      <button type="button" className="modal-backdrop" onClick={onClose} aria-label="Close dialog">
        close
      </button>
    </div>
  )
}
