import { requestPost } from '@/utils/request'

export function queryApplications(payload: any) {
  return requestPost<{ items: any[] }>('/api/manage/v1/application/query', payload)
}

export function createApplication(payload: any) {
  return requestPost<any>('/api/manage/v1/application/create', payload)
}

export function updateApplication(payload: any) {
  return requestPost<any>('/api/manage/v1/application/update', payload)
}

export function resetApplicationKey(applicationId: string) {
  return requestPost<any>('/api/manage/v1/application/key/reset', { applicationId })
}

export function disableApplication(applicationId: string) {
  return requestPost('/api/manage/v1/application/disable', { applicationId })
}

export function deleteApplication(applicationId: string) {
  return requestPost('/api/manage/v1/application/delete', { applicationId })
}
