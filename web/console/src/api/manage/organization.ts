import { requestPost } from '../../util/request'

export function queryOrganizations() {
  return requestPost<{ items: any[] }>('/api/manage/v1/organization/query', {})
}

export function createOrganization(payload: { name: string; description: string }) {
  return requestPost('/api/manage/v1/organization/create', payload)
}

export function updateOrganization(payload: any) {
  return requestPost('/api/manage/v1/organization/update', payload)
}

export function disableOrganization(organizationId: string) {
  return requestPost('/api/manage/v1/organization/disable', { organizationId })
}

export function deleteOrganization(organizationId: string) {
  return requestPost('/api/manage/v1/organization/delete', { organizationId })
}
