import { requestPost } from '@/utils/request'

export function queryOrganizations() {
  return requestPost<{ items: any[] }>('/api/manage/v1/organization/query', {})
}

export function createOrganization(payload: { name: string; description: string }) {
  return requestPost('/api/manage/v1/organization/create', payload)
}

export function updateOrganization(payload: any) {
  return requestPost('/api/manage/v1/organization/update', payload)
}

export function prepareOrganizationDomainVerification(payload: {
  organizationId: string
  host: string
  method: string
}) {
  return requestPost<{
    host: string
    method: string
    token: string
    fileUrl?: string
    fileContent?: string
    txtRecordName?: string
    txtRecordValue?: string
  }>('/api/manage/v1/organization/domain/verification/prepare', payload)
}

export function verifyOrganizationDomain(payload: {
  organizationId: string
  host: string
}) {
  return requestPost<{
    host: string
    verified?: boolean
    verificationMethod?: string
    verificationToken?: string
    verifiedAt?: string
  }>('/api/manage/v1/organization/domain/verification/verify', payload)
}

export function disableOrganization(organizationId: string) {
  return requestPost('/api/manage/v1/organization/disable', { organizationId })
}

export function deleteOrganization(organizationId: string) {
  return requestPost('/api/manage/v1/organization/delete', { organizationId })
}
