import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { queryAuditLogs as apiQueryAuditLogs } from '../api/manage/policy'

export const useAuditStore = defineStore('audit', () => {
  const auditLogs = ref<any[]>([])
  const recentAuditLogs = computed(() => auditLogs.value.slice(0, 12))
  const moduleRecentChanges = computed(() => recentAuditLogs.value.slice(0, 6))

  async function loadAudit() {
    const response = await apiQueryAuditLogs()
    auditLogs.value = response.items
  }

  return {
    auditLogs,
    recentAuditLogs,
    moduleRecentChanges,
    loadAudit
  }
})
