import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { exchangePortalToken, revokePortalAuthSession } from '@/api/auth'

const authBaseUrl = import.meta.env.PPVT_PORTAL_AUTH_BASE_URL ?? 'http://localhost:8091'
const portalApplicationId = import.meta.env.PPVT_PORTAL_APPLICATION_ID ?? ''

const storageKeys = {
  handshakes: 'ppvt-portal-oauth-handshakes',
  state: 'ppvt-portal-oauth-state',
  verifier: 'ppvt-portal-oauth-code-verifier',
  nonce: 'ppvt-portal-oauth-nonce',
  target: 'ppvt-portal-oauth-target',
  accessToken: 'ppvt-portal-access-token',
  refreshToken: 'ppvt-portal-refresh-token',
  idToken: 'ppvt-portal-id-token'
} as const

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
  return `${window.location.origin}/portal/callback`
}

function getDefaultTarget() {
  return `${window.location.origin}/portal/my`
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

export function clearPortalAuthSession() {
  localStorage.removeItem(storageKeys.accessToken)
  localStorage.removeItem(storageKeys.refreshToken)
  localStorage.removeItem(storageKeys.idToken)
  clearOAuthHandshake()
}

export function getCurrentAccessToken() {
  return localStorage.getItem(storageKeys.accessToken) ?? ''
}

export function getCurrentRefreshToken() {
  return localStorage.getItem(storageKeys.refreshToken) ?? ''
}

export async function startPortalAuthorization(target?: string) {
  if (!portalApplicationId) {
    throw new Error('missing portal application id')
  }

  const verifier = randomBase64Url(32)
  const challenge = await sha256Base64Url(verifier)
  const state = randomBase64Url(24)
  const nonce = randomBase64Url(24)
  const finalTarget = target || getDefaultTarget()

  storeOAuthHandshake(state, {
    verifier,
    nonce,
    target: finalTarget,
    createdAt: Date.now()
  })
  setSessionValue('verifier', verifier)
  setSessionValue('state', state)
  setSessionValue('nonce', nonce)
  setSessionValue('target', finalTarget)

  const url = new URL(`${authBaseUrl}/auth/authorize`)
  url.searchParams.set('client_id', portalApplicationId)
  url.searchParams.set('response_type', 'code')
  url.searchParams.set('redirect_uri', getCallbackUrl())
  url.searchParams.set('scope', 'openid profile email phone')
  url.searchParams.set('state', state)
  url.searchParams.set('nonce', nonce)
  url.searchParams.set('code_challenge', challenge)
  url.searchParams.set('code_challenge_method', 'S256')
  window.location.assign(url.toString())
}

export async function startPortalLogout() {
  try {
    await revokePortalAuthSession({
      accessToken: getCurrentAccessToken(),
      refreshToken: getCurrentRefreshToken(),
      clientId: portalApplicationId
    })
  } catch {
    // Keep local logout deterministic even if remote revoke fails.
  }
  clearPortalAuthSession()
  window.location.assign('/portal/my')
}

export async function finishPortalAuthorization(code: string, state: string) {
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
    client_id: portalApplicationId,
    code,
    redirect_uri: getCallbackUrl(),
    code_verifier: verifier
  })

  const tokenSet = await exchangePortalToken(body)
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
  } else {
    localStorage.removeItem(storageKeys.idToken)
  }

  deleteOAuthHandshake(state)
  clearLegacyOAuthHandshake()
  return target
}

export const usePortalAuthStore = defineStore('portal-auth', () => {
  const accessToken = ref(getCurrentAccessToken())
  const refreshToken = ref(getCurrentRefreshToken())
  const isAuthenticated = computed(() => Boolean(accessToken.value))

  function syncSession() {
    accessToken.value = getCurrentAccessToken()
    refreshToken.value = getCurrentRefreshToken()
  }

  function clearSession() {
    clearPortalAuthSession()
    syncSession()
  }

  async function startAuthorization(target?: string) {
    await startPortalAuthorization(target)
  }

  async function finishAuthorization(code: string, state: string) {
    const target = await finishPortalAuthorization(code, state)
    syncSession()
    return target
  }

  async function startLogoutFlow() {
    await startPortalLogout()
  }

  return {
    accessToken,
    refreshToken,
    isAuthenticated,
    syncSession,
    clearSession,
    startAuthorization,
    finishAuthorization,
    startLogoutFlow
  }
})
