import { requestPost } from '../../util/request'

export function queryPolicies(payload: any) {
  return requestPost<{ items: any[] }>('/api/manage/v1/policy/query', payload)
}

export function createPolicy(payload: any) {
  return requestPost('/api/manage/v1/policy/create', payload)
}

export function updatePolicy(payload: any) {
  return requestPost('/api/manage/v1/policy/update', payload)
}

export function deletePolicy(policyId: string) {
  return requestPost('/api/manage/v1/policy/delete', { policyId })
}

export function checkPolicy(payload: any) {
  return requestPost('/api/authz/v1/policy/check', payload)
}

export function queryAuditLogs() {
  return requestPost<{ items: any[] }>('/api/manage/v1/audit_log/query', {})
}
