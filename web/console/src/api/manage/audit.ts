import { requestPost } from '@/utils/request'

export function queryAuditLogs(payload: { organizationId?: string }) {
  return requestPost<{ items: any[] }>('/api/manage/v1/audit_log/query', payload)
}
