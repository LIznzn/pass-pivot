import { clearPortalAuthSession, startPortalAuthorization } from '../auth'

const baseUrl = import.meta.env.PPVT_PORTAL_API_BASE_URL ?? 'http://localhost:8090'

function buildHeaders() {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  const accessToken = sessionStorage.getItem('ppvt-portal-access-token')
  if (accessToken) {
    headers.Authorization = `Bearer ${accessToken}`
  }
  return headers
}

function redirectToPortalLogin() {
  clearPortalAuthSession()
  void startPortalAuthorization(window.location.href)
}

function handleUnauthorized(response: Response) {
  if (response.status === 401) {
    redirectToPortalLogin()
  }
}

export async function apiGet<T>(path: string): Promise<T> {
  const response = await fetch(`${baseUrl}${path}`, {
    headers: buildHeaders(),
    credentials: 'include'
  })
  handleUnauthorized(response)
  if (!response.ok) {
    throw new Error(await response.text())
  }
  return response.json() as Promise<T>
}

export async function apiPost<T>(path: string, body: unknown): Promise<T> {
  const response = await fetch(`${baseUrl}${path}`, {
    method: 'POST',
    headers: buildHeaders(),
    body: JSON.stringify(body),
    credentials: 'include'
  })
  handleUnauthorized(response)
  if (!response.ok) {
    throw new Error(await response.text())
  }
  return response.json() as Promise<T>
}
