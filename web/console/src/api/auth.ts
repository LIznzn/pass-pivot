import { requestPost } from '../util/request'

const authBaseUrl = import.meta.env.PPVT_CONSOLE_AUTH_BASE_URL ?? 'http://localhost:8091'
const portalBaseUrl = import.meta.env.PPVT_CONSOLE_PORTAL_BASE_URL ?? 'http://localhost:8092'
const consoleApplicationId = import.meta.env.PPVT_CONSOLE_APPLICATION_ID ?? ''

const storageKeys = {
  state: 'ppvt-oauth-state',
  verifier: 'ppvt-oauth-code-verifier',
  nonce: 'ppvt-oauth-nonce',
  target: 'ppvt-oauth-target',
  accessToken: 'ppvt-access-token',
  refreshToken: 'ppvt-refresh-token',
  idToken: 'ppvt-id-token'
} as const

type TokenResponse = {
  access_token: string
  refresh_token?: string
  id_token?: string
  token_type: string
  expires_in: number
  scope?: string
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

function clearOAuthHandshake() {
  removeSessionValue('state')
  removeSessionValue('verifier')
  removeSessionValue('nonce')
  removeSessionValue('target')
}

function persistTokenSet(tokenSet: TokenResponse) {
  setSessionValue('accessToken', tokenSet.access_token)
  if (tokenSet.refresh_token) {
    setSessionValue('refreshToken', tokenSet.refresh_token)
  } else {
    removeSessionValue('refreshToken')
  }
  if (tokenSet.id_token) {
    setSessionValue('idToken', tokenSet.id_token)
  } else {
    removeSessionValue('idToken')
  }
}

export function clearConsoleAuthSession() {
  removeSessionValue('accessToken')
  removeSessionValue('refreshToken')
  removeSessionValue('idToken')
  clearOAuthHandshake()
}

export function getCurrentAccessToken() {
  return getSessionValue('accessToken')
}

export async function buildConsoleAuthorizationUrl(target?: string) {
  if (!consoleApplicationId) {
    throw new Error('missing console application id')
  }

  const verifier = randomBase64Url(32)
  const challenge = await sha256Base64Url(verifier)
  const state = randomBase64Url(24)
  const nonce = randomBase64Url(24)

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
  const url = new URL(`${portalBaseUrl}/portal/logout`)
  url.searchParams.set('post_logout_redirect_uri', getCallbackUrl())
  clearConsoleAuthSession()
  window.location.assign(url.toString())
}

export async function finishConsoleAuthorization(code: string, state: string) {
  const expectedState = getSessionValue('state')
  const verifier = getSessionValue('verifier')
  const target = getSessionValue('target') || getDefaultTarget()

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

  persistTokenSet(tokenSet)
  clearOAuthHandshake()
  return target
}
