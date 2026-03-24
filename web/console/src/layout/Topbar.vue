<template>
  <header class="admin-topbar border-bottom bg-white">
    <div class="container-fluid">
      <div class="admin-topbar-main">
        <div class="admin-topbar-left">
          <div class="admin-brand">
            <div class="admin-brand-badge">P</div>
          </div>
          <nav class="nav admin-nav-tabs">
            <button
              v-for="item in navItems"
              :key="item.id"
              type="button"
              class="nav-link"
              :class="{ active: consoleStore.tab === item.id }"
              @click="consoleStore.setTab(item.id)"
            >
              {{ item.label }}
            </button>
          </nav>
        </div>
        <div class="admin-toolbar-group">
          <BDropdown right no-caret variant="link" class="organization-context-dropdown" toggle-class="organization-context-toggle" menu-class="organization-context-menu">
            <template #button-content>
              <span class="organization-context">
                <span class="organization-context-name">{{ currentOrganizationLabel }}</span>
                <span class="organization-context-arrow">▾</span>
              </span>
            </template>
            <div class="organization-context-menu-label">切换组织</div>
            <BDropdownItem
              v-for="organization in organizations"
              :key="organization.id"
              class="organization-context-item"
              @click="organizationStore.handleOrganizationSwitch(organization.id)"
            >
              <span class="organization-context-item-row">
                <span>{{ organization.name || organization.id }}</span>
                <span v-if="organization.id === consoleStore.currentOrganizationId" class="organization-context-check">当前</span>
              </span>
            </BDropdownItem>
            <BDropdownDivider />
            <BDropdownItem class="organization-context-manage" @click="consoleStore.toggleManageOrganization">管理组织</BDropdownItem>
          </BDropdown>
          <BDropdown right no-caret variant="link" class="user-context-dropdown" toggle-class="user-context-toggle" menu-class="user-context-menu">
            <template #button-content>
              <span class="user-context">
                <span class="user-context-avatar">{{ currentUserInitials }}</span>
              </span>
            </template>
            <div class="user-context-menu-header">
              <span class="user-context-menu-avatar">{{ currentUserInitials }}</span>
              <div class="user-context-menu-copy">
                <strong>{{ currentUserDisplayName }}</strong>
                <span>{{ currentUserEmail }}</span>
              </div>
            </div>
            <BDropdownDivider />
            <BDropdownItem @click="consoleStore.goMy()">用户中心</BDropdownItem>
            <BDropdownItem @click="consoleStore.logout()">退出登录</BDropdownItem>
          </BDropdown>
        </div>
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { BDropdown, BDropdownDivider, BDropdownItem } from 'bootstrap-vue-next'
import { computed } from 'vue'
import { useConsoleStore } from '@/stores/console'
import { useOrganizationStore } from '@/stores/organization'

const consoleStore = useConsoleStore()
const organizationStore = useOrganizationStore()
const organizations = computed(() => organizationStore.organizations)
const currentOrganizationLabel = computed(() => organizationStore.currentOrganization?.name || organizationStore.currentOrganization?.id || '选择组织')
const currentUserDisplayName = computed(() => consoleStore.currentLoginName || consoleStore.currentLoginUser || '当前登录用户')
const currentUserEmail = computed(() => consoleStore.currentLoginEmail || consoleStore.currentLoginUser || '-')
const currentUserInitials = computed(() => {
  const source = currentUserDisplayName.value || currentUserEmail.value
  const cleaned = source.replace(/[^A-Za-z0-9\u4e00-\u9fa5 ]/g, ' ').trim()
  if (!cleaned) {
    return 'U'
  }
  const parts = cleaned.split(/\s+/).filter(Boolean)
  if (parts.length >= 2) {
    return `${parts[0][0] ?? ''}${parts[1][0] ?? ''}`.toUpperCase()
  }
  return cleaned.slice(0, 2).toUpperCase()
})

const navItems = [
  { id: 'dashboard', label: '仪表盘' },
  { id: 'organization', label: '组织' },
  { id: 'project', label: '项目' },
  { id: 'user', label: '用户' },
  { id: 'role', label: '角色' },
  { id: 'audit', label: '审计' },
  { id: 'setting', label: '设置' }
] as const
</script>
