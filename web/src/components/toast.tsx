import { useEffect } from 'react'
import { Check, X } from 'lucide-react'

interface ToastProps {
  message: string
  visible: boolean
  onClose: () => void
  duration?: number
}

export function Toast({ message, visible, onClose, duration = 3000 }: ToastProps) {
  useEffect(() => {
    if (!visible) return
    const timer = setTimeout(onClose, duration)
    return () => clearTimeout(timer)
  }, [visible, duration, onClose])

  if (!visible) return null

  return (
    <div className="fixed bottom-6 right-6 z-50 animate-in slide-in-from-bottom-4 fade-in duration-300">
      <div className="flex items-center gap-3 rounded-lg border border-green-500/30 bg-green-500/10 px-4 py-3 shadow-lg backdrop-blur-sm dark:bg-green-500/15">
        <Check className="size-4 shrink-0 text-green-600 dark:text-green-400" />
        <p className="text-sm font-medium text-green-700 dark:text-green-300">{message}</p>
        <button
          onClick={onClose}
          className="ml-2 rounded p-0.5 text-green-600 hover:bg-green-500/20 dark:text-green-400"
        >
          <X className="size-3.5" />
        </button>
      </div>
    </div>
  )
}
