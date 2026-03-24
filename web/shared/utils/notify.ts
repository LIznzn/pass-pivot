type ToastVariant = 'success' | 'danger'

type ToastController = {
  create: (options: {
    body: string
    variant: ToastVariant
    pos: 'top-center'
    modelValue: number
    noProgress: boolean
    noCloseButton: boolean
    toastClass: string[]
    bodyClass: string
  }) => void
}

type NotifyOptions = {
  toast: ToastController
  source: string
  message: string
  variant: ToastVariant
  trigger?: string
  error?: unknown
  metadata?: Record<string, unknown>
}

function normalizeError(error: unknown) {
  if (error instanceof Error) {
    return {
      type: 'Error',
      name: error.name,
      message: error.message,
      stack: error.stack
    }
  }

  if (error && typeof error === 'object') {
    const value = error as Record<string, unknown>
    const response = value.response && typeof value.response === 'object'
      ? value.response as Record<string, unknown>
      : null
    return {
      type: value.constructor?.name || 'Object',
      name: typeof value.name === 'string' ? value.name : undefined,
      message: typeof value.message === 'string' ? value.message : String(error),
      code: typeof value.code === 'string' ? value.code : undefined,
      status: typeof response?.status === 'number' ? response.status : undefined,
      statusText: typeof response?.statusText === 'string' ? response.statusText : undefined,
      responseData: response?.data
    }
  }

  if (error === undefined) {
    return undefined
  }

  return {
    type: typeof error,
    message: String(error)
  }
}

export function notifyToast(options: NotifyOptions) {
  const body = String(options.message || '').trim()
  if (!body) {
    return
  }

  options.toast.create({
    body,
    variant: options.variant,
    pos: 'top-center',
    modelValue: 2800,
    noProgress: true,
    noCloseButton: true,
    toastClass: ['ppvt-toast', 'ppvt-toast-' + options.variant],
    bodyClass: 'ppvt-toast-body'
  })

  const payload = {
    timestamp: new Date().toISOString(),
    source: options.source,
    trigger: options.trigger || options.source,
    variant: options.variant,
    message: body,
    error: normalizeError(options.error),
    metadata: options.metadata
  }

  if (options.variant === 'danger') {
    console.error('[ppvt-toast]', payload)
    return
  }

  console.info('[ppvt-toast]', payload)
}
