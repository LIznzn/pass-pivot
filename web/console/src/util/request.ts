import axios from 'axios'
import type { AxiosError, AxiosInstance, AxiosRequestConfig, InternalAxiosRequestConfig } from 'axios'
import { getCurrentAccessToken } from '../api/auth'

const baseURL = import.meta.env.PPVT_CONSOLE_API_BASE_URL ?? 'http://localhost:8090'
const authSessionKeys = [
  'ppvt-oauth-state',
  'ppvt-oauth-code-verifier',
  'ppvt-oauth-nonce',
  'ppvt-oauth-target'
] as const

export type RequestConfig = AxiosRequestConfig & {
  skipAuthHeader?: boolean
  skipUnauthorizedRedirect?: boolean
}

function redirectToPortalLogin() {
  for (const key of authSessionKeys) {
    sessionStorage.removeItem(key)
  }
  const target = encodeURIComponent(window.location.href)
  window.location.assign(`/console?target=${target}`)
}

const request: AxiosInstance = axios.create({
  baseURL,
  withCredentials: true
})

request.interceptors.request.use((config: InternalAxiosRequestConfig & RequestConfig) => {
  const accessToken = getCurrentAccessToken()
  if (accessToken && !config.skipAuthHeader) {
    config.headers.Authorization = `Bearer ${accessToken}`
  }
  return config
})

request.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const config = (error.config || {}) as RequestConfig
    if (error.response?.status === 401 && !config.skipUnauthorizedRedirect) {
      redirectToPortalLogin()
    }
    const message = typeof error.response?.data === 'string'
      ? error.response.data
      : error.message
    return Promise.reject(new Error(message))
  }
)

export async function requestGet<T>(url: string, config?: RequestConfig) {
  const response = await request.get<T>(url, config)
  return response.data
}

export async function requestPost<T>(url: string, data?: unknown, config?: RequestConfig) {
  const response = await request.post<T>(url, data, config)
  return response.data
}

export default request
