import axios from 'axios'
import type { AxiosError, AxiosInstance, AxiosRequestConfig, InternalAxiosRequestConfig } from 'axios'
import { clearPortalAuthSession, getCurrentAccessToken, startPortalAuthorization } from '@/stores/auth'

const baseURL = import.meta.env.PPVT_PORTAL_API_BASE_URL ?? 'http://localhost:8090'

export type RequestConfig = AxiosRequestConfig & {
  skipAuthHeader?: boolean
  skipUnauthorizedRedirect?: boolean
}

function redirectToPortalLogin() {
  clearPortalAuthSession()
  void startPortalAuthorization(window.location.href)
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
    const responseData = error.response?.data as { message?: string; code?: string } | string | undefined
    const message = typeof responseData === 'string'
      ? responseData
      : responseData?.message || error.message
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
