export function loginRedirectTarget(): string {
  const saved = sessionStorage.getItem('ppvt-login-redirect')
  return saved || `${window.location.origin}/portal/my`
}

export function captureLoginRedirect(input: unknown) {
  const redirect = typeof input === 'string' ? input.trim() : ''
  if (redirect) {
    sessionStorage.setItem('ppvt-login-redirect', redirect)
    return
  }
  sessionStorage.removeItem('ppvt-login-redirect')
}

export function clearLoginRedirect() {
  sessionStorage.removeItem('ppvt-login-redirect')
}
