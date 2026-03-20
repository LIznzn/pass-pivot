import { requestPost } from '../../util/request'

export function queryRoles(payload: any) {
  return requestPost<{ items: any[] }>('/api/manage/v1/role/query', payload)
}

export function createRole(payload: any) {
  return requestPost('/api/manage/v1/role/create', payload)
}

export function updateRole(payload: any) {
  return requestPost('/api/manage/v1/role/update', payload)
}

export function deleteRoles(payload: { roleId?: string; roleIds?: string[] }) {
  return requestPost('/api/manage/v1/role/delete', payload)
}
