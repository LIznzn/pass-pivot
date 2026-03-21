import axios from 'axios'

const authBaseUrl = import.meta.env.PPVT_PORTAL_AUTH_BASE_URL ?? 'http://localhost:8091'
const portalApplicationId = import.meta.env.PPVT_PORTAL_APPLICATION_ID ?? ''

const handshakesKey = 'ppvt-portal-oauth-handshakes'
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

type OAuthHandshake = {
  verifier: string
  nonce: string
  target: string
  createdAt: number
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

function parseOAuthHandshakes(raw: string | null) {
  if (!raw) {
    return {} as Record<string, OAuthHandshake>
  }
  try {
    const parsed = JSON.parse(raw) as Record<string, OAuthHandshake> | null
    if (!parsed || typeof parsed !== 'object') {
      return {} as Record<string, OAuthHandshake>
    }
    return parsed
  } catch {
    return {} as Record<string, OAuthHandshake>
  }
}

function readOAuthHandshakes() {
  return parseOAuthHandshakes(sessionStorage.getItem(handshakesKey))
}

function writeOAuthHandshakes(handshakes: Record<string, OAuthHandshake>) {
  const entries = Object.entries(handshakes)
    .filter(([state, item]) => state && item?.verifier)
    .sort((a, b) => (b[1].createdAt || 0) - (a[1].createdAt || 0))
    .slice(0, 8)
  sessionStorage.setItem(handshakesKey, JSON.stringify(Object.fromEntries(entries)))
}

function storeOAuthHandshake(state: string, handshake: OAuthHandshake) {
  const handshakes = readOAuthHandshakes()
  handshakes[state] = handshake
  writeOAuthHandshakes(handshakes)
}

function loadOAuthHandshake(state: string) {
  if (!state) {
    return null
  }
  const handshakes = readOAuthHandshakes()
  return handshakes[state] ?? null
}

function deleteOAuthHandshake(state: string) {
  if (!state) {
    return
  }
  const handshakes = readOAuthHandshakes()
  if (!(state in handshakes)) {
    return
  }
  delete handshakes[state]
  if (Object.keys(handshakes).length) {
    writeOAuthHandshakes(handshakes)
  } else {
    sessionStorage.removeItem(handshakesKey)
  }
}

function clearLegacyOAuthHandshake() {
  sessionStorage.removeItem(stateKey)
  sessionStorage.removeItem(verifierKey)
  sessionStorage.removeItem(nonceKey)
  sessionStorage.removeItem(targetKey)
}

function clearOAuthHandshake() {
  sessionStorage.removeItem(handshakesKey)
  clearLegacyOAuthHandshake()
}

export function clearPortalAuthSession() {
  localStorage.removeItem(accessTokenKey)
  localStorage.removeItem(refreshTokenKey)
  localStorage.removeItem(idTokenKey)
  clearOAuthHandshake()
}

export function getCurrentAccessToken() {
  return localStorage.getItem(accessTokenKey) ?? ''
}

export function getCurrentRefreshToken() {
  return localStorage.getItem(refreshTokenKey) ?? ''
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

  storeOAuthHandshake(state, {
    verifier,
    nonce,
    target: finalTarget,
    createdAt: Date.now()
  })
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
  const handshake = loadOAuthHandshake(state)
  const expectedState = handshake ? state : (sessionStorage.getItem(stateKey) ?? '')
  const verifier = handshake?.verifier || (sessionStorage.getItem(verifierKey) ?? '')
  const target = handshake?.target || sessionStorage.getItem(targetKey) || `${window.location.origin}/portal/my`

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

  const response = await axios.post<TokenResponse>(`${authBaseUrl}/auth/token`, body, {
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    withCredentials: true
  })
  const tokenSet = response.data
  if (!tokenSet.access_token) {
    throw new Error('missing access_token')
  }
  localStorage.setItem(accessTokenKey, tokenSet.access_token)
  if (tokenSet.refresh_token) {
    localStorage.setItem(refreshTokenKey, tokenSet.refresh_token)
  } else {
    localStorage.removeItem(refreshTokenKey)
  }
  if (tokenSet.id_token) {
    localStorage.setItem(idTokenKey, tokenSet.id_token)
  } else {
    localStorage.removeItem(idTokenKey)
  }
  deleteOAuthHandshake(state)
  clearLegacyOAuthHandshake()
  return target
}
