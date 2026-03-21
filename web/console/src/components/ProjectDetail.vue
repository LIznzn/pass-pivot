<template>
  <section class="console-module-shell">
    <ProjectUserAssignmentModal
      :visible="projectUserAssignmentModalVisible"
      :selected-user-ids="localAssignedUserIds"
      :format-role-labels="formatRoleLabels"
      @update:visible="projectUserAssignmentModalVisible = $event"
      @confirm="confirmProjectUserAssignmentModal"
    />
    <div class="console-module-summary-card">
      <div class="console-module-hero">
        <div class="console-module-hero-copy">
          <button type="button" class="console-back-button" @click="emit('back')" aria-label="返回项目列表">
            <i class="bi bi-arrow-left console-back-button-icon" aria-hidden="true"></i>
          </button>
          <div>
            <div class="console-module-eyebrow">项目</div>
            <h2 class="console-module-title">{{ currentProject?.name || '项目' }}</h2>
            <p class="console-module-subtitle">{{ currentProject?.name ? '从项目列表选择条目后，在详情区维护项目及其应用配置。' : '管理项目与应用的结构、协议模式与接入配置。' }}</p>
          </div>
        </div>
        <div class="console-action-menu" role="group" aria-label="项目操作">
          <button type="button" class="btn btn-primary console-action-menu-toggle">
            操作
            <i class="bi bi-chevron-down" aria-hidden="true"></i>
          </button>
          <div class="console-action-menu-list">
            <button type="button" class="console-action-menu-item" @click="emit('disable')">停用</button>
            <button type="button" class="console-action-menu-item console-action-menu-item-danger" @click="emit('delete')">删除</button>
          </div>
        </div>
      </div>
      <div class="console-module-metrics">
        <div v-for="item in currentModuleMetrics" :key="item.label" class="console-module-metric">
          <span class="console-module-metric-label">{{ item.label }}</span>
          <div class="console-module-metric-value-row">
            <strong class="console-module-metric-value">{{ item.value }}</strong>
            <button
              v-if="item.copyable"
              type="button"
              class="console-module-metric-copy"
              :aria-label="`复制${item.label}`"
              @click="consoleStore.copyMetricValue(item.copyValue || item.value)"
            >
              <i class="bi bi-copy" aria-hidden="true"></i>
            </button>
          </div>
        </div>
      </div>
    </div>
    <div class="console-module-workspace">
      <aside class="console-module-sidebar">
        <button v-for="item in currentModulePanels" :key="item.id" type="button" class="console-module-sidebar-link" @click="consoleStore.scrollToPanel(item.id)">{{ item.label }}</button>
      </aside>
      <div class="console-module-main">
        <div id="project-application" class="info-card">
          <div class="section-title">应用列表</div>
          <div class="record-meta mb-3">显示当前项目下的应用列表，点击卡片进入应用详情页。</div>
          <div class="record-list project-application-grid mb-3">
            <button
              v-for="application in applications"
              :key="application.id"
              type="button"
              class="record-card record-card-button"
              @click="emit('go-application-detail', application)"
            >
              <div class="project-card-id mb-1">{{ application.id }}</div>
              <div class="record-head align-items-center mb-2">
                <div class="project-card-name">{{ application.name || application.id }}</div>
                <span class="badge rounded-pill" :class="application.status === 'disabled' ? 'text-bg-secondary' : 'text-bg-success'">
                  {{ application.status === 'disabled' ? '停用' : '启用' }}
                </span>
              </div>
              <div class="record-meta">Token Type</div>
              <div class="project-card-value mb-1">{{ formatApplicationTokenType(application.tokenType) }}</div>
              <div class="record-meta">Grant Type</div>
              <div class="project-card-value mb-1">{{ formatApplicationGrantType(application.grantType) }}</div>
              <div class="record-meta">应用角色</div>
              <div class="project-card-value mb-1">{{ formatRoleLabels(application.roles) }}</div>
              <div class="record-meta">Client Authentication Type</div>
              <div class="project-card-value">{{ formatApplicationClientAuthenticationType(application.clientAuthenticationType) }}</div>
            </button>
            <div class="record-card project-create-card application-create-card">
              <button type="button" class="project-create-trigger" @click="emit('go-application-create')">
                <div class="project-create-plus text-secondary lh-1 mb-2">+</div>
                <div class="project-create-title">创建新应用</div>
              </button>
            </div>
          </div>
        </div>
        <div id="project-user-assignment" class="info-card">
          <div class="section-title">用户分配</div>
          <div class="record-meta mb-3">
            {{ projectUpdateForm.userAclEnabled
              ? '已开启用户访问控制。只有已分配用户，才能访问该项目下的系统应用；如果分配列表为空，则所有用户都不可访问。'
              : '当前未开启用户访问控制。关闭时，当前组织下所有用户都可以访问该项目下的系统应用。' }}
          </div>
          <div class="d-flex justify-content-between align-items-center gap-2 mb-3">
            <div class="record-meta mb-0">已分配 {{ localAssignedUserIds.length }} / {{ users.length }} 个用户。</div>
            <BButton variant="outline-primary" size="sm" @click="projectUserAssignmentModalVisible = true">添加用户</BButton>
          </div>
          <div class="table-responsive project-user-assignment-wrap mb-3">
            <table class="table align-middle console-list-table project-user-assignment-table mb-0">
              <thead>
                <tr>
                  <th>用户 ID</th>
                  <th>用户名</th>
                  <th>名称</th>
                  <th>邮箱 / 手机号</th>
                  <th>状态</th>
                  <th>角色</th>
                  <th class="text-end">操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="user in assignedProjectUsers" :key="user.id">
                  <td class="console-list-id">{{ user.id }}</td>
                  <td>{{ user.username || '-' }}</td>
                  <td>{{ user.name || '-' }}</td>
                  <td>{{ user.email || user.phoneNumber || '-' }}</td>
                  <td>
                    <span class="badge rounded-pill" :class="user.status === 'disabled' ? 'text-bg-secondary' : 'text-bg-success'">
                      {{ user.status === 'disabled' ? '停用' : '启用' }}
                    </span>
                  </td>
                  <td>{{ formatRoleLabels(user.roles) }}</td>
                  <td class="text-end">
                    <BButton size="sm" variant="outline-danger" @click="removeProjectAssignedUser(user.id)">移出</BButton>
                  </td>
                </tr>
                <tr v-if="assignedProjectUsers.length === 0">
                  <td colspan="7" class="text-center text-secondary py-4">当前 ACL 中还没有用户。</td>
                </tr>
              </tbody>
            </table>
          </div>
          <BButton variant="primary" size="sm" @click="emit('save-project-user-assignments', localAssignedUserIds)">保存用户分配</BButton>
        </div>
        <div id="project-setting" class="info-card">
          <div class="section-title">项目设置</div>
          <div class="record-meta mb-3">维护当前项目的基础名称和描述。</div>
          <BForm @submit.prevent="emit('update-project')">
            <div class="mb-3">
              <label class="form-label">项目名称</label>
              <BFormInput v-model="projectUpdateForm.name" placeholder="请输入项目名称" />
            </div>
            <div class="mb-3">
              <label class="form-label">项目描述</label>
              <BFormInput v-model="projectUpdateForm.description" placeholder="请输入项目描述" />
            </div>
            <div class="mb-3">
              <label class="form-label d-block">用户访问控制</label>
              <BFormCheckbox v-model="projectUpdateForm.userAclEnabled">
                开启后，仅允许已分配到项目的用户访问该项目下应用
              </BFormCheckbox>
            </div>
            <BButton type="submit" variant="outline-primary">保存项目设置</BButton>
          </BForm>
        </div>
      </div>
      <RightSide :items="moduleRecentChanges" />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { BButton, BForm, BFormCheckbox, BFormInput } from 'bootstrap-vue-next'
import RightSide from '../layout/RightSide.vue'
import ProjectUserAssignmentModal from '../modal/ProjectUserAssignmentModal.vue'
import { useAuditStore } from '../stores/audit'
import { useConsoleStore } from '../stores/console'
import { useUserStore } from '../stores/user'

const props = defineProps<{
  currentProject: any
  applications: any[]
  projectUpdateForm: { name: string; description: string; userAclEnabled: boolean }
  projectAssignedUserIds: string[]
  formatApplicationTokenType: (value?: string | string[]) => string
  formatApplicationGrantType: (value?: string | string[]) => string
  formatRoleLabels: (roles?: string[]) => string
  formatApplicationClientAuthenticationType: (value?: string) => string
}>()

const auditStore = useAuditStore()
const consoleStore = useConsoleStore()
const userStore = useUserStore()
const users = computed(() => userStore.users)
const moduleRecentChanges = computed(() => auditStore.moduleRecentChanges)
const formatDateTime = consoleStore.formatDateTime

const currentModulePanels = [
  { id: 'project-application', label: '应用列表' },
  { id: 'project-user-assignment', label: '用户分配' },
  { id: 'project-setting', label: '项目设置' }
]

const currentModuleMetrics = computed(() => [
  { label: '项目 ID', value: props.currentProject?.id || '-', copyable: Boolean(props.currentProject?.id), copyValue: props.currentProject?.id || '' },
  { label: '应用数', value: String(props.currentProject?.applications?.length ?? props.applications.length) },
  { label: '创建时间', value: formatDateTime(props.currentProject?.createdAt) },
  { label: '最近变更', value: formatDateTime(props.currentProject?.updatedAt) }
])

const localAssignedUserIds = ref<string[]>([])
const projectUserAssignmentModalVisible = ref(false)

watch(
  () => [props.projectAssignedUserIds, props.currentProject?.id],
  () => {
    localAssignedUserIds.value = [...props.projectAssignedUserIds]
  },
  { immediate: true, deep: true }
)

const assignedProjectUsers = computed(() => users.value.filter((item: any) => localAssignedUserIds.value.includes(item.id)))

function removeProjectAssignedUser(userId: string) {
  localAssignedUserIds.value = localAssignedUserIds.value.filter((item) => item !== userId)
}

function confirmProjectUserAssignmentModal(userIds: string[]) {
  localAssignedUserIds.value = [...userIds]
  projectUserAssignmentModalVisible.value = false
}

const emit = defineEmits<{
  back: []
  disable: []
  delete: []
  'go-application-detail': [application: any]
  'go-application-create': []
  'save-project-user-assignments': [userIds: string[]]
  'update-project': []
}>()
</script>
