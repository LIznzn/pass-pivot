import { requestPost } from '@/utils/request'

export function queryExternalIdps(organizationId: string) {
  return requestPost<{ items: any[] }>('/api/manage/v1/external_idp/query', { organizationId })
}

export function createExternalIdp(payload: any) {
  return requestPost('/api/manage/v1/external_idp/create', payload)
}

export function updateExternalIdp(payload: any) {
  return requestPost('/api/manage/v1/external_idp/update', payload)
}
