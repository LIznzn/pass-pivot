import { requestPost } from '@/utils/request'

const authBaseUrl = import.meta.env.PPVT_PORTAL_AUTH_BASE_URL ?? 'http://localhost:8091'

export type PortalTokenResponse = {
  access_token: string
  refresh_token?: string
  id_token?: string
  token_type: string
  expires_in: number
  scope?: string
}

export function exchangePortalToken(payload: URLSearchParams) {
  return requestPost<PortalTokenResponse>(`${authBaseUrl}/auth/token`, payload, {
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    skipAuthHeader: true,
    skipUnauthorizedRedirect: true
  })
}

async function revokePortalToken(token: string, clientId: string) {
  if (!token || !clientId) {
    return
  }
  const body = new URLSearchParams({
    client_id: clientId,
    token
  })
  await requestPost<any>(`${authBaseUrl}/auth/revoke`, body, {
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    skipAuthHeader: true,
    skipUnauthorizedRedirect: true
  })
}

export async function revokePortalAuthSession(payload: { accessToken: string; refreshToken: string; clientId: string }) {
  await revokePortalToken(payload.accessToken, payload.clientId)
  await revokePortalToken(payload.refreshToken, payload.clientId)
}
