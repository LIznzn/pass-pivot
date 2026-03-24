import { requestGet, requestPost } from '@/utils/request'

export async function apiGet<T>(path: string): Promise<T> {
  return requestGet<T>(path)
}

export async function apiPost<T>(path: string, body: unknown): Promise<T> {
  return requestPost<T>(path, body)
}
