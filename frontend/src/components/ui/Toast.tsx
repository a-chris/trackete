import { createContext, useCallback, useContext, useMemo, useRef, useState, type ReactNode } from 'react'
import { CircleAlert, CircleCheck, Info, X } from 'lucide-react'

type ToastType = 'success' | 'error' | 'info'

interface ToastItem {
  id: number
  type: ToastType
  message: string
}

interface ToastContextValue {
  push: (type: ToastType, message: string) => void
}

const ToastContext = createContext<ToastContextValue | null>(null)

const icons: Record<ToastType, ReactNode> = {
  success: <CircleCheck className="size-4" />,
  error: <CircleAlert className="size-4" />,
  info: <Info className="size-4" />,
}

const alertClass: Record<ToastType, string> = {
  success: 'alert-success',
  error: 'alert-error',
  info: 'alert-info',
}

export function ToastProvider({ children }: { children: ReactNode }) {
  const [items, setItems] = useState<ToastItem[]>([])
  const counter = useRef(0)

  const remove = useCallback((id: number) => {
    setItems((current) => current.filter((item) => item.id !== id))
  }, [])

  const push = useCallback(
    (type: ToastType, message: string) => {
      const id = ++counter.current
      setItems((current) => [...current.slice(-2), { id, type, message }])
      window.setTimeout(() => remove(id), 2800)
    },
    [remove],
  )

  const value = useMemo(() => ({ push }), [push])

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="toast toast-end toast-bottom z-50">
        {items.map((item) => (
          <div key={item.id} className={`alert ${alertClass[item.type]} shadow-lg`}>
            {icons[item.type]}
            <span className="max-w-xs text-sm">{item.message}</span>
            <button
              type="button"
              className="btn btn-ghost btn-xs btn-circle"
              onClick={() => remove(item.id)}
              aria-label="Dismiss"
            >
              <X className="size-3" />
            </button>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  )
}

// eslint-disable-next-line react-refresh/only-export-components
export function useToast(): ToastContextValue {
  const context = useContext(ToastContext)
  if (!context) throw new Error('useToast must be used inside ToastProvider')
  return context
}
