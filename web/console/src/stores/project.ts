import { computed, reactive, ref } from 'vue'
import { defineStore } from 'pinia'
import {
  createProject as apiCreateProject,
  deleteProject as apiDeleteProject,
  disableProject as apiDisableProject,
  queryProjects as apiQueryProjects,
  updateProject as apiUpdateProject,
  updateProjectUserAssignments as apiUpdateProjectUserAssignments
} from '@/api/manage/project'
import { useConsoleStore } from './console'
import { useOrganizationStore } from './organization'

export const useProjectStore = defineStore('project', () => {
  const console = useConsoleStore()
  const organizationStore = useOrganizationStore()
  const projects = ref<any[]>([])
  const selectedProjectId = ref('')
  const projectAssignedUserIds = ref<string[]>([])

  const projectForm = reactive({ organizationId: '', name: '', userAclEnabled: false })
  const projectUpdateForm = reactive({ id: '', name: '', description: '', userAclEnabled: false })

  const currentProject = computed(() => projects.value.find((item: any) => item.id === selectedProjectId.value) || projects.value[0])

  function syncProjectForm(project?: any) {
    if (!project) {
      projectUpdateForm.id = ''
      projectUpdateForm.name = ''
      projectUpdateForm.description = ''
      projectUpdateForm.userAclEnabled = false
      projectAssignedUserIds.value = []
      return
    }
    projectUpdateForm.id = project.id ?? ''
    projectUpdateForm.name = project.name ?? ''
    projectUpdateForm.description = project.description ?? ''
    projectUpdateForm.userAclEnabled = Boolean(project.userAclEnabled)
    projectAssignedUserIds.value = Array.isArray(project.assignedUserIds) ? [...project.assignedUserIds] : []
  }

  function clearProjectState() {
    projects.value = []
    selectedProjectId.value = ''
    projectAssignedUserIds.value = []
    syncProjectForm()
  }

  function resetProjectCreateForm() {
    projectForm.organizationId = console.currentOrganizationId || organizationStore.currentOrganization?.id || ''
    projectForm.name = ''
    projectForm.userAclEnabled = false
  }

  function setSelectedProjectId(projectId: string) {
    selectedProjectId.value = projectId
    syncProjectForm(currentProject.value)
  }

  async function loadProjects(organizationId = console.currentOrganizationId) {
    projectForm.organizationId = organizationId || ''
    if (!organizationId) {
      clearProjectState()
      return
    }
    const response = await apiQueryProjects({ organizationId })
    projects.value = response.items
    if (!projects.value.some((item: any) => item.id === selectedProjectId.value)) {
      selectedProjectId.value = projects.value[0]?.id ?? ''
    }
    syncProjectForm(currentProject.value)
  }

  async function createProject() {
    const response = await apiCreateProject(projectForm)
    await loadProjects(projectForm.organizationId)
    return response
  }

  async function updateProject() {
    const response = await apiUpdateProject(projectUpdateForm)
    await loadProjects(projectForm.organizationId || console.currentOrganizationId)
    syncProjectForm(currentProject.value)
    return response
  }

  async function saveProjectUserAssignments(userIds: string[]) {
    if (!selectedProjectId.value) {
      return { userIds: [] }
    }
    const response = await apiUpdateProjectUserAssignments(selectedProjectId.value, userIds)
    projectAssignedUserIds.value = [...(response.userIds ?? [])]
    await loadProjects(projectForm.organizationId || console.currentOrganizationId)
    return response
  }

  async function disableProject() {
    if (!selectedProjectId.value) {
      return
    }
    await apiDisableProject(selectedProjectId.value)
    await loadProjects(projectForm.organizationId || console.currentOrganizationId)
  }

  async function deleteProject() {
    if (!selectedProjectId.value) {
      return
    }
    await apiDeleteProject(selectedProjectId.value)
    selectedProjectId.value = ''
    await loadProjects(projectForm.organizationId || console.currentOrganizationId)
  }

  return {
    projects,
    selectedProjectId,
    projectAssignedUserIds,
    projectForm,
    projectUpdateForm,
    syncProjectForm,
    clearProjectState,
    resetProjectCreateForm,
    setSelectedProjectId,
    loadProjects,
    createProject,
    updateProject,
    saveProjectUserAssignments,
    disableProject,
    deleteProject
  }
})
