import { requestPost } from '../util/request'

const authBaseUrl = import.meta.env.PPVT_CONSOLE_AUTH_BASE_URL ?? 'http://localhost:8091'
const consoleApplicationId = import.meta.env.PPVT_CONSOLE_APPLICATION_ID ?? ''

const storageKeys = {
  handshakes: 'ppvt-oauth-handshakes',
  state: 'ppvt-oauth-state',
  verifier: 'ppvt-oauth-code-verifier',
  nonce: 'ppvt-oauth-nonce',
  target: 'ppvt-oauth-target',
  accessToken: 'ppvt-access-token',
  refreshToken: 'ppvt-refresh-token',
  idToken: 'ppvt-id-token',
  loginIdentifier: 'ppvt-login-identifier',
  loginName: 'ppvt-login-name',
  loginEmail: 'ppvt-login-email'
} as const

type TokenResponse = {
  access_token: string
  refresh_token?: string
  id_token?: string
  token_type: string
  expires_in: number
  scope?: string
}

type IDTokenClaims = {
  preferred_username?: string
  name?: string
  email?: string
  sub?: string
}

type OAuthHandshake = {
  verifier: string
  nonce: string
  target: string
  createdAt: number
}

function getSessionValue(key: keyof typeof storageKeys) {
  return sessionStorage.getItem(storageKeys[key]) ?? ''
}

function setSessionValue(key: keyof typeof storageKeys, value: string) {
  sessionStorage.setItem(storageKeys[key], value)
}

function removeSessionValue(key: keyof typeof storageKeys) {
  sessionStorage.removeItem(storageKeys[key])
}

function parseOAuthHandshakes(raw: string) {
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
  return parseOAuthHandshakes(getSessionValue('handshakes'))
}

function writeOAuthHandshakes(handshakes: Record<string, OAuthHandshake>) {
  const entries = Object.entries(handshakes)
    .filter(([state, item]) => state && item?.verifier)
    .sort((a, b) => (b[1].createdAt || 0) - (a[1].createdAt || 0))
    .slice(0, 8)
  setSessionValue('handshakes', JSON.stringify(Object.fromEntries(entries)))
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
    removeSessionValue('handshakes')
  }
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

function getCallbackUrl() {
  return `${window.location.origin}/console/callback`
}

function getDefaultTarget() {
  return `${window.location.origin}/console/dashboard`
}

function clearLegacyOAuthHandshake() {
  removeSessionValue('state')
  removeSessionValue('verifier')
  removeSessionValue('nonce')
  removeSessionValue('target')
}

function clearOAuthHandshake() {
  removeSessionValue('handshakes')
  clearLegacyOAuthHandshake()
}

export function clearConsoleAuthSession() {
  localStorage.removeItem(storageKeys.accessToken)
  localStorage.removeItem(storageKeys.refreshToken)
  localStorage.removeItem(storageKeys.idToken)
  sessionStorage.removeItem(storageKeys.loginIdentifier)
  sessionStorage.removeItem(storageKeys.loginName)
  sessionStorage.removeItem(storageKeys.loginEmail)
}

export function clearConsoleOAuthHandshake() {
  clearOAuthHandshake()
}

export function getCurrentAccessToken() {
  return localStorage.getItem(storageKeys.accessToken) ?? ''
}

export function getCurrentRefreshToken() {
  return localStorage.getItem(storageKeys.refreshToken) ?? ''
}

function decodeBase64Url(value: string) {
  const normalized = value.replace(/-/g, '+').replace(/_/g, '/')
  const padded = normalized.padEnd(Math.ceil(normalized.length / 4) * 4, '=')
  return atob(padded)
}

function parseIDTokenClaims(idToken: string): IDTokenClaims | null {
  const parts = idToken.split('.')
  if (parts.length < 2 || !parts[1]) {
    return null
  }
  try {
    return JSON.parse(decodeBase64Url(parts[1])) as IDTokenClaims
  } catch {
    return null
  }
}

export async function buildConsoleAuthorizationUrl(target?: string) {
  if (!consoleApplicationId) {
    throw new Error('missing console application id')
  }

  const verifier = randomBase64Url(32)
  const challenge = await sha256Base64Url(verifier)
  const state = randomBase64Url(24)
  const nonce = randomBase64Url(24)

  storeOAuthHandshake(state, {
    verifier,
    nonce,
    target: target || window.location.href,
    createdAt: Date.now()
  })
  setSessionValue('verifier', verifier)
  setSessionValue('state', state)
  setSessionValue('nonce', nonce)
  setSessionValue('target', target || window.location.href)

  const url = new URL(`${authBaseUrl}/auth/authorize`)
  url.searchParams.set('client_id', consoleApplicationId)
  url.searchParams.set('response_type', 'code')
  url.searchParams.set('redirect_uri', getCallbackUrl())
  url.searchParams.set('scope', 'openid profile email phone')
  url.searchParams.set('state', state)
  url.searchParams.set('nonce', nonce)
  url.searchParams.set('code_challenge', challenge)
  url.searchParams.set('code_challenge_method', 'S256')
  return url.toString()
}

export async function startConsoleAuthorization(target?: string) {
  window.location.assign(await buildConsoleAuthorizationUrl(target))
}

export function startConsoleLogout() {
  clearConsoleAuthSession()
  clearConsoleOAuthHandshake()
  window.location.assign('/console')
}

export async function finishConsoleAuthorization(code: string, state: string) {
  const handshake = loadOAuthHandshake(state)
  const expectedState = handshake ? state : getSessionValue('state')
  const verifier = handshake?.verifier || getSessionValue('verifier')
  const target = handshake?.target || getSessionValue('target') || getDefaultTarget()

  if (!code) {
    throw new Error('missing authorization code')
  }
  if (!expectedState || expectedState !== state) {
    throw new Error('oauth state mismatch')
  }
  if (!verifier) {
    throw new Error('missing pkce verifier')
  }

  const body = new URLSearchParams({
    grant_type: 'authorization_code',
    client_id: consoleApplicationId,
    code,
    redirect_uri: getCallbackUrl(),
    code_verifier: verifier
  })

  const tokenSet = await requestPost<TokenResponse>(`${authBaseUrl}/auth/token`, body, {
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    skipAuthHeader: true,
    skipUnauthorizedRedirect: true,
    withCredentials: true
  })

  if (!tokenSet.access_token) {
    throw new Error('missing access_token')
  }

  localStorage.setItem(storageKeys.accessToken, tokenSet.access_token)
  if (tokenSet.refresh_token) {
    localStorage.setItem(storageKeys.refreshToken, tokenSet.refresh_token)
  } else {
    localStorage.removeItem(storageKeys.refreshToken)
  }
  if (tokenSet.id_token) {
    localStorage.setItem(storageKeys.idToken, tokenSet.id_token)
    const claims = parseIDTokenClaims(tokenSet.id_token)
    const identifier = claims?.preferred_username?.trim() || claims?.email?.trim() || claims?.name?.trim() || claims?.sub?.trim() || ''
    const displayName = claims?.name?.trim() || claims?.preferred_username?.trim() || claims?.email?.trim() || ''
    const email = claims?.email?.trim() || ''
    if (identifier) {
      sessionStorage.setItem(storageKeys.loginIdentifier, identifier)
    } else {
      sessionStorage.removeItem(storageKeys.loginIdentifier)
    }
    if (displayName) {
      sessionStorage.setItem(storageKeys.loginName, displayName)
    } else {
      sessionStorage.removeItem(storageKeys.loginName)
    }
    if (email) {
      sessionStorage.setItem(storageKeys.loginEmail, email)
    } else {
      sessionStorage.removeItem(storageKeys.loginEmail)
    }
  } else {
    localStorage.removeItem(storageKeys.idToken)
    sessionStorage.removeItem(storageKeys.loginIdentifier)
    sessionStorage.removeItem(storageKeys.loginName)
    sessionStorage.removeItem(storageKeys.loginEmail)
  }
  deleteOAuthHandshake(state)
  clearLegacyOAuthHandshake()
  return target
}
