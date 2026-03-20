import axios from 'axios'

const authBaseUrl = import.meta.env.PPVT_CONSOLE_AUTH_BASE_URL ?? 'http://localhost:8091'
const portalBaseUrl = import.meta.env.PPVT_CONSOLE_PORTAL_BASE_URL ?? 'http://localhost:8092'
const consoleApplicationId = import.meta.env.PPVT_CONSOLE_APPLICATION_ID ?? ''

const stateKey = 'ppvt-oauth-state'
const verifierKey = 'ppvt-oauth-code-verifier'
const nonceKey = 'ppvt-oauth-nonce'
const targetKey = 'ppvt-oauth-target'
const accessTokenKey = 'ppvt-access-token'
const refreshTokenKey = 'ppvt-refresh-token'
const idTokenKey = 'ppvt-id-token'

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
  return `${window.location.origin}/console/callback`
}

function logoutRedirectUrl() {
  return callbackUrl()
}

function clearOAuthHandshake() {
  sessionStorage.removeItem(stateKey)
  sessionStorage.removeItem(verifierKey)
  sessionStorage.removeItem(nonceKey)
  sessionStorage.removeItem(targetKey)
}

export function clearConsoleAuthSession() {
  sessionStorage.removeItem(accessTokenKey)
  sessionStorage.removeItem(refreshTokenKey)
  sessionStorage.removeItem(idTokenKey)
  clearOAuthHandshake()
}

export function getCurrentAccessToken() {
  return sessionStorage.getItem(accessTokenKey) ?? ''
}

export async function buildConsoleAuthorizationUrl(target?: string) {
  if (!consoleApplicationId) {
    throw new Error('missing console application id')
  }
  const verifier = randomBase64Url(32)
  const challenge = await sha256Base64Url(verifier)
  const state = randomBase64Url(24)
  const nonce = randomBase64Url(24)
  const finalTarget = target || window.location.href

  sessionStorage.setItem(verifierKey, verifier)
  sessionStorage.setItem(stateKey, state)
  sessionStorage.setItem(nonceKey, nonce)
  sessionStorage.setItem(targetKey, finalTarget)

  const url = new URL(`${authBaseUrl}/auth/authorize`)
  url.searchParams.set('client_id', consoleApplicationId)
  url.searchParams.set('response_type', 'code')
  url.searchParams.set('redirect_uri', callbackUrl())
  url.searchParams.set('scope', 'openid profile email phone')
  url.searchParams.set('state', state)
  url.searchParams.set('nonce', nonce)
  url.searchParams.set('code_challenge', challenge)
  url.searchParams.set('code_challenge_method', 'S256')
  return url.toString()
}

export async function startConsoleAuthorization(target?: string) {
  const url = await buildConsoleAuthorizationUrl(target)
  window.location.assign(url)
}

export function startConsoleLogout() {
  const url = new URL(`${portalBaseUrl}/portal/logout`)
  url.searchParams.set('post_logout_redirect_uri', logoutRedirectUrl())
  clearConsoleAuthSession()
  window.location.assign(url.toString())
}

export async function finishConsoleAuthorization(code: string, state: string) {
  const expectedState = sessionStorage.getItem(stateKey) ?? ''
  const verifier = sessionStorage.getItem(verifierKey) ?? ''
  const target = sessionStorage.getItem(targetKey) ?? `${window.location.origin}/console/dashboard`

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
  body.set('client_id', consoleApplicationId)
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
