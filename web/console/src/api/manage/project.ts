import { requestPost } from '@/utils/request'

export function queryProjects(payload: any) {
  return requestPost<{ items: any[] }>('/api/manage/v1/project/query', payload)
}

export function createProject(payload: any) {
  return requestPost('/api/manage/v1/project/create', payload)
}

export function updateProject(payload: any) {
  return requestPost('/api/manage/v1/project/update', payload)
}

export function updateProjectUserAssignments(projectId: string, userIds: string[]) {
  return requestPost<{ userIds: string[] }>('/api/manage/v1/project/user_assignment/update', { projectId, userIds })
}

export function disableProject(projectId: string) {
  return requestPost('/api/manage/v1/project/disable', { projectId })
}

export function deleteProject(projectId: string) {
  return requestPost('/api/manage/v1/project/delete', { projectId })
}
