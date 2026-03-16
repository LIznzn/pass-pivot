import { reactive } from 'vue'

export type ToastVariant = 'success' | 'danger' | 'warning' | 'info'

type ToastItem = {
  id: number
  message: string
  variant: ToastVariant
}

const state = reactive({
  items: [] as ToastItem[]
})

let seed = 1

export function useToast() {
  function show(message: string, variant: ToastVariant = 'info', duration = 2800) {
    const content = String(message || '').trim()
    if (!content) {
      return
    }
    const id = seed
    seed += 1
    state.items.push({ id, message: content, variant })
    window.setTimeout(() => dismiss(id), duration)
  }

  function dismiss(id: number) {
    const index = state.items.findIndex((item) => item.id === id)
    if (index >= 0) {
      state.items.splice(index, 1)
    }
  }

  return {
    items: state.items,
    show,
    dismiss,
    success(message: string, duration?: number) {
      show(message, 'success', duration)
    },
    error(message: string, duration?: number) {
      show(message, 'danger', duration)
    },
    warning(message: string, duration?: number) {
      show(message, 'warning', duration)
    },
    info(message: string, duration?: number) {
      show(message, 'info', duration)
    }
  }
}
