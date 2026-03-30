import { computed, ref } from "vue";
import { defineStore } from "pinia";
import { queryAuditLogs as apiQueryAuditLogs } from "@/api/manage/audit";
import { useConsoleStore } from "./console";

export const useAuditStore = defineStore("audit", () => {
  const console = useConsoleStore();
  const auditLogs = ref<any[]>([]);
  const page = ref(1);
  const pageSize = ref(20);
  const total = ref(0);
  const totalPages = ref(0);
  const recentAuditLogs = computed(() => auditLogs.value.slice(0, 12));
  const moduleRecentChanges = computed(() => recentAuditLogs.value.slice(0, 6));

  async function loadAudit(nextPage?: number) {
    const response = await apiQueryAuditLogs({
      organizationId: console.currentOrganizationId || "",
      page: nextPage || page.value,
      pageSize: pageSize.value,
    });
    auditLogs.value = response.items;
    page.value = response.page;
    pageSize.value = response.pageSize;
    total.value = response.total;
    totalPages.value = response.totalPages;
  }

  async function setPageSize(nextPageSize: number) {
    pageSize.value = nextPageSize;
    await loadAudit(1);
  }

  return {
    auditLogs,
    page,
    pageSize,
    total,
    totalPages,
    recentAuditLogs,
    moduleRecentChanges,
    loadAudit,
    setPageSize,
  };
});
