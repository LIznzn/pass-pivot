<template>
  <section class="audit-page">
    <div id="audit-filter" class="info-card">
      <div class="audit-toolbar">
        <div>
          <div class="section-title">查询条件</div>
          <div class="record-meta mt-2">
            按时间范围和模块筛选当前组织的审计日志。
          </div>
        </div>
      </div>
      <div class="audit-filter-grid">
        <div class="audit-filter-field">
          <label class="form-label">开始时间</label>
          <BFormInput v-model="filters.from" type="datetime-local" />
        </div>
        <div class="audit-filter-field">
          <label class="form-label">结束时间</label>
          <BFormInput v-model="filters.to" type="datetime-local" />
        </div>
        <div class="audit-filter-field">
          <label class="form-label">模块</label>
          <BFormSelect v-model="filters.module" :options="moduleOptions" />
        </div>
      </div>
      <div class="audit-preset-row">
        <button
          v-for="item in presetOptions"
          :key="item.value"
          type="button"
          class="audit-preset-chip"
          :class="{ 'audit-preset-chip-active': filters.preset === item.value }"
          @click="applyPreset(item.value)"
        >
          {{ item.label }}
        </button>
      </div>
      <div class="detail-card mt-3">
        <div class="record-meta">
          当前组织：{{
            currentOrganization?.name || currentOrganization?.id || "-"
          }}
        </div>
        <div class="record-meta">
          当前页日志：{{ filteredAuditLogs.length }} / 总数：{{
            auditStore.total
          }}
        </div>
      </div>
      <div class="audit-action-row">
        <BButton variant="outline-secondary" @click="resetFilters"
          >清空筛选条件</BButton
        >
        <BButton variant="primary" @click="() => auditStore.loadAudit(1)"
          >查询</BButton
        >
      </div>
    </div>

    <div id="audit-list" class="info-card mt-4">
      <div class="d-flex justify-content-between align-items-center gap-3 flex-wrap mb-3">
        <div>
          <div class="section-title mb-1">审计日志</div>
          <div class="record-meta">
            第 {{ auditStore.page }} /
            {{ Math.max(auditStore.totalPages, 1) }} 页，每页
            {{ auditStore.pageSize }} 条。
          </div>
        </div>
      </div>
      <div v-if="filteredAuditLogs.length" class="audit-log-list">
        <article
          v-for="item in filteredAuditLogs"
          :key="item.id"
          class="audit-log-card"
        >
          <button
            type="button"
            class="audit-log-summary"
            @click="toggleExpanded(item.id)"
          >
            <div class="audit-log-mainline">
              <span class="audit-log-module">{{
                resolveModuleLabel(item)
              }}</span>
              <span
                class="audit-log-result-badge"
                :class="resolveResultBadgeClass(item.result)"
              >
                {{ resolveResultText(item.result) }}
              </span>
              <strong class="audit-log-event">{{ item.eventType }}</strong>
              <span class="audit-log-request-badge">{{
                [item.requestMethod, item.requestPath]
                  .filter(Boolean)
                  .join(" ") || "-"
              }}</span>
            </div>
            <div class="audit-log-meta">
              <span class="audit-log-time">{{
                formatDateTime(item.createdAt)
              }}</span>
              <span class="audit-log-expand">{{
                isExpanded(item.id) ? "收起" : "展开"
              }}</span>
            </div>
          </button>
          <div v-if="isExpanded(item.id)" class="audit-log-detail">
            <div class="audit-log-detail-row audit-log-detail-row-primary">
              <span
                >Actor:
                {{
                  formatPrincipal(
                    item.actorType,
                    item.actorName,
                    item.actorId,
                  )
                }}</span
              >
              <span
                >Target:
                {{
                  formatPrincipal(
                    item.targetType,
                    item.targetName,
                    item.targetId,
                  )
                }}</span
              >
            </div>
            <div class="audit-log-detail-row">
              <span
                >From:
                {{
                  formatSource(
                    item.applicationName,
                    item.applicationId,
                    item.ipAddress,
                    item.ipLocation,
                  )
                }}</span
              >
              <span>UA: {{ formatAuditValue(item.userAgent) }}</span>
            </div>
            <div
              v-if="item.detailJson?.changes?.length"
              class="audit-change-list"
            >
              <div
                v-for="change in item.detailJson.changes"
                :key="`${item.id}-${change.field}`"
                class="audit-change-item"
              >
                <strong class="audit-change-field">{{ change.field }}</strong>
                <code>{{ formatAuditValue(change.before) }}</code>
                <span>→</span>
                <code>{{ formatAuditValue(change.after) }}</code>
              </div>
            </div>
            <div
              v-if="metadataEntries(item).length"
              class="audit-metadata-list"
            >
              <div
                v-for="entry in metadataEntries(item)"
                :key="`${item.id}-${entry.key}`"
                class="audit-metadata-item"
              >
                <span class="audit-metadata-key">{{ entry.key }}</span>
                <code>{{ formatAuditValue(entry.value) }}</code>
              </div>
            </div>
          </div>
        </article>
      </div>
      <div v-else class="detail-card">
        <div class="record-meta">当前筛选条件下没有审计日志。</div>
      </div>
      <div class="audit-pagination-bar">
        <div class="audit-page-size">
          <span class="record-meta">每页显示</span>
          <BFormSelect
            :model-value="String(auditStore.pageSize)"
            :options="pageSizeOptions"
            @update:model-value="handlePageSizeChange"
          />
          <span class="record-meta">行</span>
        </div>
        <div class="audit-pagination">
          <BButton
            variant="outline-secondary"
            :disabled="auditStore.page <= 1"
            @click="changePage(auditStore.page - 1)"
          >
            上一页
          </BButton>
          <button
            v-for="item in visiblePageItems"
            :key="item.key"
            type="button"
            class="audit-page-button"
            :class="{ 'audit-page-button-active': item.page === auditStore.page }"
            :disabled="item.page === null"
            @click="item.page && changePage(item.page)"
          >
            {{ item.label }}
          </button>
          <BButton
            variant="outline-secondary"
            :disabled="auditStore.page >= auditStore.totalPages"
            @click="changePage(auditStore.page + 1)"
          >
            下一页
          </BButton>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watchEffect } from "vue";
import { BButton, BFormInput, BFormSelect } from "bootstrap-vue-next";
import { useAuditStore } from "@/stores/audit";
import { useConsoleStore } from "@/stores/console";
import { useOrganizationStore } from "@/stores/organization";

const auditStore = useAuditStore();
const consoleStore = useConsoleStore();
const organizationStore = useOrganizationStore();

const presetOptions = [
  { value: "24h", label: "最近 24 小时" },
  { value: "7d", label: "最近 7 天" },
  { value: "30d", label: "最近 30 天" },
  { value: "all", label: "全部时间" },
] as const;
const pageSizeOptions = [
  { value: "10", text: "10" },
  { value: "20", text: "20" },
  { value: "50", text: "50" },
  { value: "100", text: "100" },
];

const filters = reactive({
  from: "",
  to: "",
  module: "all",
  preset: "7d",
});
const expandedAuditIds = ref<string[]>([]);

watchEffect(() => {
  consoleStore.setPageHeader(
    "审计",
    "查看平台关键事件、登录轨迹与策略变更审计。",
  );
});

const moduleOptions = computed(() => {
  const values = new Set<string>();
  for (const item of auditStore.auditLogs) {
    values.add(resolveModuleValue(item));
  }
  return [
    { value: "all", text: "全部模块" },
    ...Array.from(values)
      .filter(Boolean)
      .sort((a, b) => a.localeCompare(b))
      .map((value) => ({
        value,
        text: resolveModuleText(value),
      })),
  ];
});

const filteredAuditLogs = computed(() => {
  const fromTime = parseLocalDateTime(filters.from);
  const toTime = parseLocalDateTime(filters.to);
  return auditStore.auditLogs.filter((item) => {
    const createdAt = Date.parse(String(item.createdAt || ""));
    if (Number.isNaN(createdAt)) {
      return false;
    }
    if (fromTime !== null && createdAt < fromTime) {
      return false;
    }
    if (toTime !== null && createdAt > toTime) {
      return false;
    }
    if (
      filters.module !== "all" &&
      resolveModuleValue(item) !== filters.module
    ) {
      return false;
    }
    return true;
  });
});

const visiblePageItems = computed(() => {
  const totalPages = Math.max(auditStore.totalPages, 1);
  const currentPage = Math.min(auditStore.page, totalPages);
  const pages = new Set<number>([1, totalPages, currentPage]);
  for (let offset = -2; offset <= 2; offset += 1) {
    const page = currentPage + offset;
    if (page >= 1 && page <= totalPages) {
      pages.add(page);
    }
  }
  const sortedPages = Array.from(pages).sort((a, b) => a - b);
  const items: Array<{ key: string; label: string; page: number | null }> = [];
  for (let index = 0; index < sortedPages.length; index += 1) {
    const page = sortedPages[index];
    const previous = sortedPages[index - 1];
    if (previous && page - previous > 1) {
      items.push({
        key: `gap-${previous}-${page}`,
        label: "...",
        page: null,
      });
    }
    items.push({
      key: `page-${page}`,
      label: String(page),
      page,
    });
  }
  return items;
});

function formatIpLine(ipAddress?: string, ipLocation?: string) {
  const ip = String(ipAddress || "").trim();
  const location = String(ipLocation || "").trim();
  if (ip && location) return `${ip} (${location})`;
  return ip || location || "-";
}

function formatAuditValue(value: unknown) {
  if (value === null || value === undefined || value === "") return "-";
  if (typeof value === "string") return value;
  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
}

function formatPrincipal(type?: string, display?: string, id?: string) {
  const subjectType = String(type || "").trim() || "-";
  const name = String(display || "").trim();
  const identifier = String(id || "").trim();
  if (name && identifier && name !== identifier) {
    return `${subjectType}/${name}(${identifier})`;
  }
  if (name) {
    return `${subjectType}/${name}`;
  }
  if (identifier) {
    return `${subjectType}/${identifier}`;
  }
  return subjectType;
}

function formatSource(
  applicationName?: string,
  applicationId?: string,
  ipAddress?: string,
  ipLocation?: string,
) {
  const appName = String(applicationName || "").trim();
  const appId = String(applicationId || "").trim();
  const ipLine = formatIpLine(ipAddress, ipLocation);
  if (appName && appId && appName !== appId) {
    return `${appName}(${appId}) ${ipLine}`;
  }
  if (appName || appId) {
    return `${appName || appId} ${ipLine}`.trim();
  }
  return ipLine;
}

function metadataEntries(item: any) {
  const metadata = item?.detailJson?.metadata;
  if (!metadata || typeof metadata !== "object") {
    return [];
  }
  return Object.entries(metadata).map(([key, value]) => ({ key, value }));
}

function resolveModuleValue(item: any) {
  const eventType = String(item?.eventType || "").trim();
  if (!eventType) {
    return "other";
  }
  const [prefix] = eventType.split(".");
  return prefix || "other";
}

function resolveModuleText(value: string) {
  if (value === "auth") return "认证";
  if (value === "token") return "令牌";
  if (value === "user") return "用户";
  if (value === "role") return "角色";
  if (value === "policy") return "策略";
  if (value === "project") return "项目";
  if (value === "application") return "应用";
  if (value === "organization") return "组织";
  if (value === "external") return "外部身份";
  if (value === "session") return "会话";
  return value;
}

function resolveModuleLabel(item: any) {
  return resolveModuleText(resolveModuleValue(item));
}

function resolveResultText(value: string) {
  const normalized = String(value || "")
    .trim()
    .toLowerCase();
  if (normalized === "success") return "成功";
  if (normalized === "denied") return "失败";
  if (normalized === "failed") return "失败";
  return value || "-";
}

function resolveResultBadgeClass(value: string) {
  const normalized = String(value || "")
    .trim()
    .toLowerCase();
  if (normalized === "success") return "is-success";
  if (normalized === "denied" || normalized === "failed") return "is-danger";
  return "is-neutral";
}

function parseLocalDateTime(input: string) {
  const value = String(input || "").trim();
  if (!value) {
    return null;
  }
  const timestamp = new Date(value).getTime();
  return Number.isNaN(timestamp) ? null : timestamp;
}

function applyPreset(preset: (typeof presetOptions)[number]["value"]) {
  filters.preset = preset;
  if (preset === "all") {
    filters.from = "";
    filters.to = "";
    return;
  }
  const now = new Date();
  const start = new Date(now);
  if (preset === "24h") {
    start.setHours(now.getHours() - 24);
  } else if (preset === "7d") {
    start.setDate(now.getDate() - 7);
  } else {
    start.setDate(now.getDate() - 30);
  }
  filters.from = toDateTimeLocalValue(start);
  filters.to = toDateTimeLocalValue(now);
}

function toDateTimeLocalValue(value: Date) {
  const year = value.getFullYear();
  const month = String(value.getMonth() + 1).padStart(2, "0");
  const day = String(value.getDate()).padStart(2, "0");
  const hour = String(value.getHours()).padStart(2, "0");
  const minute = String(value.getMinutes()).padStart(2, "0");
  return `${year}-${month}-${day}T${hour}:${minute}`;
}

function resetFilters() {
  filters.module = "all";
  applyPreset("7d");
}

async function changePage(page: number) {
  if (
    page <= 0 ||
    (auditStore.totalPages > 0 && page > auditStore.totalPages)
  ) {
    return;
  }
  await auditStore.loadAudit(page);
  expandedAuditIds.value = [];
}

async function handlePageSizeChange(value: string | number) {
  const nextPageSize = Number(value);
  if (![10, 20, 50, 100].includes(nextPageSize)) {
    return;
  }
  await auditStore.setPageSize(nextPageSize);
  expandedAuditIds.value = [];
}

function isExpanded(id: string) {
  return expandedAuditIds.value.includes(id);
}

function toggleExpanded(id: string) {
  if (isExpanded(id)) {
    expandedAuditIds.value = expandedAuditIds.value.filter(
      (item) => item !== id,
    );
    return;
  }
  expandedAuditIds.value = [...expandedAuditIds.value, id];
}

applyPreset("7d");

const currentOrganization = computed(
  () => organizationStore.currentOrganization,
);
const formatDateTime = consoleStore.formatDateTime;
</script>

<style scoped>
.audit-page {
  width: 100%;
}

.audit-toolbar {
  display: flex;
  align-items: flex-start;
  gap: 1rem;
}

.audit-action-row {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 1rem;
}

.audit-pagination {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.75rem;
}

.audit-pagination-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 0.9rem;
  margin-top: 1rem;
  padding-top: 0.9rem;
  border-top: 1px solid #e5e7eb;
  flex-wrap: wrap;
}

.audit-page-size {
  display: flex;
  align-items: center;
  gap: 0.55rem;
}

.audit-page-size :deep(select) {
  width: 5.25rem;
}

.audit-page-button {
  min-width: 2.2rem;
  height: 2.2rem;
  padding: 0 0.65rem;
  border: 1px solid #d1d5db;
  border-radius: 0.6rem;
  background: #fff;
  color: #4b5563;
  font-size: 0.84rem;
  font-weight: 600;
}

.audit-page-button:disabled {
  cursor: default;
  opacity: 0.7;
}

.audit-page-button-active {
  border-color: #1f6feb;
  background: #1f6feb;
  color: #fff;
}

.audit-log-list {
  display: grid;
  gap: 0.65rem;
}

.audit-log-card {
  border: 1px solid #e5e7eb;
  border-radius: 0.7rem;
  background: #fff;
}

.audit-log-summary {
  width: 100%;
  border: 0;
  background: transparent;
  padding: 0.45rem 0.6rem;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  text-align: left;
}

.audit-log-mainline {
  display: flex;
  align-items: center;
  gap: 0.55rem;
  min-width: 0;
  flex-wrap: wrap;
}

.audit-log-event {
  font-size: 0.88rem;
  line-height: 1.2;
}

.audit-log-request-badge {
  display: inline-flex;
  align-items: center;
  min-height: 1.45rem;
  padding: 0 0.55rem;
  border-radius: 999px;
  background: #f3f4f6;
  color: #4b5563;
  font-size: 0.74rem;
  font-weight: 600;
}

.audit-log-meta {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  flex: 0 0 auto;
  white-space: nowrap;
}

.audit-log-expand {
  color: #6b7280;
  font-size: 0.76rem;
}

.audit-log-time {
  color: #6b7280;
  font-size: 0.76rem;
}

.audit-log-result-badge {
  display: inline-flex;
  align-items: center;
  min-height: 1.45rem;
  padding: 0 0.55rem;
  border-radius: 999px;
  font-size: 0.74rem;
  font-weight: 700;
}

.audit-log-result-badge.is-success {
  background: #edfdf3;
  color: #15803d;
}

.audit-log-result-badge.is-danger {
  background: #fef2f2;
  color: #dc2626;
}

.audit-log-result-badge.is-neutral {
  background: #f3f4f6;
  color: #4b5563;
}

.audit-log-detail {
  padding: 0 0.6rem 0.38rem;
}

.audit-log-detail-row {
  display: flex;
  flex-wrap: wrap;
  gap: 0.2rem 0.75rem;
  margin-top: 0.18rem;
  color: #6b7280;
  font-size: 0.78rem;
  line-height: 1.2;
}

.audit-change-list,
.audit-metadata-list {
  display: grid;
  gap: 0.18rem;
  margin-top: 0.18rem;
}

.audit-change-item,
.audit-metadata-item {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  flex-wrap: wrap;
  font-size: 0.78rem;
  line-height: 1.2;
  color: #111827;
}

.audit-change-field {
  min-width: 7rem;
  font-size: 0.78rem;
  font-weight: 400;
  color: #111827;
}

.audit-filter-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 1rem;
}

.audit-filter-field {
  display: grid;
}

.audit-preset-row {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  margin-top: 1rem;
}

.audit-preset-chip {
  border: 1px solid #d0d7de;
  background: #fff;
  color: #57606a;
  border-radius: 999px;
  padding: 0.45rem 0.85rem;
  font-size: 0.88rem;
  font-weight: 600;
}

.audit-preset-chip-active {
  background: #1f6feb;
  border-color: #1f6feb;
  color: #fff;
}

.audit-log-module {
  display: inline-flex;
  align-items: center;
  min-height: 1.45rem;
  padding: 0 0.55rem;
  border-radius: 999px;
  background: #eaf2ff;
  color: #1f6feb;
  font-size: 0.74rem;
  font-weight: 700;
}

.audit-log-item {
  border-radius: 10px;
  border: 1px solid #d8dee4;
  background: #fff;
  padding: 0.75rem 0.85rem;
  display: grid;
  gap: 0.28rem;
}

.audit-log-item span {
  color: #656d76;
  font-size: 0.8rem;
}

.audit-log-item strong {
  font-size: 0.92rem;
  word-break: break-word;
}

@media (max-width: 1100px) {
  .audit-toolbar,
  .audit-filter-grid,
  .audit-log-grid {
    grid-template-columns: 1fr;
  }

  .audit-toolbar {
    flex-direction: column;
  }

  .audit-action-row {
    justify-content: stretch;
  }

  .audit-pagination-bar {
    align-items: stretch;
  }
}
</style>
