import axios from 'axios'
import type { AxiosError, AxiosInstance, AxiosRequestConfig } from 'axios'
import { clearPortalAuthSession, getCurrentAccessToken, startPortalAuthorization } from '@/auth'

const baseURL = import.meta.env.PPVT_PORTAL_API_BASE_URL ?? 'http://localhost:8090'

function redirectToPortalLogin() {
  clearPortalAuthSession()
  void startPortalAuthorization(window.location.href)
}

const request: AxiosInstance = axios.create({
  baseURL,
  withCredentials: true
})

request.interceptors.request.use((config) => {
  const accessToken = getCurrentAccessToken()
  if (accessToken) {
    config.headers.Authorization = `Bearer ${accessToken}`
  }
  return config
})

request.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    if (error.response?.status === 401) {
      redirectToPortalLogin()
    }
    const responseData = error.response?.data as { message?: string; code?: string } | string | undefined
    const message = typeof responseData === 'string'
      ? responseData
      : responseData?.message || error.message
    return Promise.reject(new Error(message))
  }
)

export async function requestGet<T>(url: string, config?: AxiosRequestConfig) {
  const response = await request.get<T>(url, config)
  return response.data
}

export async function requestPost<T>(url: string, data?: unknown, config?: AxiosRequestConfig) {
  const response = await request.post<T>(url, data, config)
  return response.data
}

export default request
