<template>
  <section class="section-grid">
    <div class="info-card">
      <div class="d-flex justify-content-between align-items-center gap-2 mb-3">
        <div class="section-title mb-0">组织列表</div>
        <div class="d-flex gap-2">
          <BButton size="sm" variant="outline-primary" @click="organizationStore.loadOrganizations">刷新</BButton>
          <BButton size="sm" variant="primary" @click="organizationStore.openCreateOrganizationModal">创建组织</BButton>
        </div>
      </div>
      <div class="record-meta mb-3">选择一个组织后，会将控制台上下文切换到该组织，并同步刷新项目、用户、角色和设置视图。</div>
      <div class="record-list">
        <button
          v-for="organization in organizations"
          :key="organization.id"
          type="button"
          class="record-card record-card-button text-start"
          :class="{ 'record-card-active': organization.id === currentOrganizationId }"
          @click="organizationStore.handleOrganizationSwitch(organization.id)"
        >
          <div class="project-card-id mb-1">{{ organization.id }}</div>
          <div class="record-head mb-1">
            <strong>{{ organization.name || organization.id }}</strong>
            <span v-if="organization.id === currentOrganizationId" class="badge text-bg-primary">当前组织</span>
          </div>
          <div class="record-meta">{{ organization.description || '暂无组织简介' }}</div>
          <div class="record-meta">创建时间：{{ formatDateTime(organization.createdAt) }}</div>
          <div class="record-meta">最近变更：{{ formatDateTime(organization.updatedAt) }}</div>
          <div class="record-meta">
            角色数 {{ organization.roles?.length ?? 0 }} · 用户数 {{ organization.users?.length ?? 0 }} · 项目数 {{ organization.projects?.length ?? 0 }} · 应用数 {{ applicationCount(organization) }}
          </div>
        </button>
        <div v-if="organizations.length === 0" class="detail-card">
          <div class="record-meta">当前还没有可切换的组织。</div>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { watchEffect } from 'vue'
import { storeToRefs } from 'pinia'
import { BButton } from 'bootstrap-vue-next'
import { useConsoleStore } from '@/stores/console'
import { useOrganizationStore } from '@/stores/organization'

const consoleStore = useConsoleStore()
const organizationStore = useOrganizationStore()

watchEffect(() => {
  consoleStore.setPageHeader('组织切换', '在这里切换当前控制台所属组织，必要时可直接创建新的组织；组织基础信息调整请前往该组织的设置页。')
})

function applicationCount(organization: any) {
  return (organization.projects ?? []).reduce((count: number, project: any) => count + (project.applications?.length ?? 0), 0)
}

const { organizations } = storeToRefs(organizationStore)
const { currentOrganizationId } = storeToRefs(consoleStore)
const formatDateTime = consoleStore.formatDateTime
</script>
