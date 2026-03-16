const authBaseUrl = import.meta.env.PPVT_PORTAL_AUTH_BASE_URL ?? 'http://localhost:8091'
const portalApplicationId = import.meta.env.PPVT_PORTAL_APPLICATION_ID ?? ''

const stateKey = 'ppvt-portal-oauth-state'
const verifierKey = 'ppvt-portal-oauth-code-verifier'
const nonceKey = 'ppvt-portal-oauth-nonce'
const targetKey = 'ppvt-portal-oauth-target'
const accessTokenKey = 'ppvt-portal-access-token'
const refreshTokenKey = 'ppvt-portal-refresh-token'
const idTokenKey = 'ppvt-portal-id-token'

type TokenResponse = {
  access_token: string
  refresh_token?: string
  id_token?: string
  token_type: string
  expires_in: number
  scope?: string
}

function randomBase64Url(bytes = 32) {
  const buffer = new Uint8Array(bytes)
  crypto.getRandomValues(buffer)
  return toBase64Url(buffer)
}

function toBase64Url(input: Uint8Array) {
  let text = ''
  input.forEach((byte) => {
    text += String.fromCharCode(byte)
  })
  return btoa(text).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '')
}

async function sha256Base64Url(value: string) {
  const digest = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(value))
  return toBase64Url(new Uint8Array(digest))
}

function callbackUrl() {
  return `${window.location.origin}/portal/callback`
}

function postLogoutRedirectUrl() {
  return `${window.location.origin}/portal/my`
}

function clearOAuthHandshake() {
  sessionStorage.removeItem(stateKey)
  sessionStorage.removeItem(verifierKey)
  sessionStorage.removeItem(nonceKey)
  sessionStorage.removeItem(targetKey)
}

export function clearPortalAuthSession() {
  sessionStorage.removeItem(accessTokenKey)
  sessionStorage.removeItem(refreshTokenKey)
  sessionStorage.removeItem(idTokenKey)
  clearOAuthHandshake()
}

export function getCurrentAccessToken() {
  return sessionStorage.getItem(accessTokenKey) ?? ''
}

export async function startPortalAuthorization(target?: string) {
  if (!portalApplicationId) {
    throw new Error('missing portal application id')
  }
  const verifier = randomBase64Url(32)
  const challenge = await sha256Base64Url(verifier)
  const state = randomBase64Url(24)
  const nonce = randomBase64Url(24)
  const finalTarget = target || `${window.location.origin}/portal/my`

  sessionStorage.setItem(verifierKey, verifier)
  sessionStorage.setItem(stateKey, state)
  sessionStorage.setItem(nonceKey, nonce)
  sessionStorage.setItem(targetKey, finalTarget)

  const url = new URL(`${authBaseUrl}/auth/authorize`)
  url.searchParams.set('client_id', portalApplicationId)
  url.searchParams.set('response_type', 'code')
  url.searchParams.set('redirect_uri', callbackUrl())
  url.searchParams.set('scope', 'openid profile email phone')
  url.searchParams.set('state', state)
  url.searchParams.set('nonce', nonce)
  url.searchParams.set('code_challenge', challenge)
  url.searchParams.set('code_challenge_method', 'S256')
  window.location.assign(url.toString())
}

export function startPortalLogout() {
  const url = new URL(`${window.location.origin}/portal/logout`)
  url.searchParams.set('post_logout_redirect_uri', postLogoutRedirectUrl())
  clearPortalAuthSession()
  window.location.assign(url.toString())
}

export async function finishPortalAuthorization(code: string, state: string) {
  const expectedState = sessionStorage.getItem(stateKey) ?? ''
  const verifier = sessionStorage.getItem(verifierKey) ?? ''
  const target = sessionStorage.getItem(targetKey) ?? `${window.location.origin}/portal/my`

  if (!code) {
    throw new Error('missing authorization code')
  }
  if (!expectedState || expectedState !== state) {
    throw new Error('oauth state mismatch')
  }
  if (!verifier) {
    throw new Error('missing pkce verifier')
  }

  const body = new URLSearchParams()
  body.set('grant_type', 'authorization_code')
  body.set('client_id', portalApplicationId)
  body.set('code', code)
  body.set('redirect_uri', callbackUrl())
  body.set('code_verifier', verifier)

  const response = await fetch(`${authBaseUrl}/auth/token`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: body.toString(),
    credentials: 'include'
  })
  if (!response.ok) {
    throw new Error(await response.text())
  }

  const tokenSet = (await response.json()) as TokenResponse
  if (!tokenSet.access_token) {
    throw new Error('missing access_token')
  }
  sessionStorage.setItem(accessTokenKey, tokenSet.access_token)
  if (tokenSet.refresh_token) {
    sessionStorage.setItem(refreshTokenKey, tokenSet.refresh_token)
  } else {
    sessionStorage.removeItem(refreshTokenKey)
  }
  if (tokenSet.id_token) {
    sessionStorage.setItem(idTokenKey, tokenSet.id_token)
  } else {
    sessionStorage.removeItem(idTokenKey)
  }
  clearOAuthHandshake()
  return target
}
