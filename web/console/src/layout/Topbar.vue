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
              :class="{ active: activeTab === item.id }"
              @click="emit('set-tab', item.id)"
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
              @click="emit('switch-organization', organization.id)"
            >
              <span class="organization-context-item-row">
                <span>{{ organization.name || organization.id }}</span>
                <span v-if="organization.id === currentOrganizationId" class="organization-context-check">当前</span>
              </span>
            </BDropdownItem>
            <BDropdownDivider />
            <BDropdownItem class="organization-context-manage" @click="emit('manage-organization')">管理组织</BDropdownItem>
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
            <BDropdownItem @click="emit('go-my')">用户中心</BDropdownItem>
            <BDropdownItem @click="emit('logout')">退出登录</BDropdownItem>
          </BDropdown>
        </div>
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { BDropdown, BDropdownDivider, BDropdownItem } from 'bootstrap-vue-next'

defineProps<{
  activeTab: 'dashboard' | 'organization' | 'project' | 'user' | 'role' | 'audit' | 'setting'
  organizations: any[]
  currentOrganizationId: string
  currentOrganizationLabel: string
  currentUserInitials: string
  currentUserDisplayName: string
  currentUserEmail: string
}>()

const emit = defineEmits<{
  'set-tab': [tab: 'dashboard' | 'organization' | 'project' | 'user' | 'role' | 'audit' | 'setting']
  'switch-organization': [organizationId: string]
  'manage-organization': []
  'go-my': []
  logout: []
}>()

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
