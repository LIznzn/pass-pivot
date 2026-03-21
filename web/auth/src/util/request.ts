import axios from 'axios'
import type { AxiosError, AxiosInstance, AxiosRequestConfig } from 'axios'

export class RequestError extends Error {
  code?: string

  constructor(message: string, code?: string) {
    super(message)
    this.name = 'RequestError'
    this.code = code
  }
}

const request: AxiosInstance = axios.create({
  withCredentials: true
})

request.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const responseData = error.response?.data as { message?: string; code?: string } | string | undefined
    const code = typeof responseData === 'string' ? undefined : responseData?.code
    const message = typeof responseData === 'string'
      ? responseData
      : responseData?.message || error.message
    return Promise.reject(new RequestError(message, code))
  }
)

export async function requestPost<T>(url: string, data?: unknown, config?: AxiosRequestConfig) {
  const response = await request.post<T>(url, data, config)
  return response.data
}

export default request
