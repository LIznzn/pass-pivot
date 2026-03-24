<template>
  <section class="console-module-shell">
    <div class="console-module-summary-card">
      <div class="console-module-hero">
        <div class="console-module-hero-copy">
          <button type="button" class="console-back-button" @click="emit('back')" aria-label="返回项目详情">
            <i class="bi bi-arrow-left console-back-button-icon" aria-hidden="true"></i>
          </button>
          <div>
            <div class="console-module-eyebrow">应用</div>
            <h2 class="console-module-title">{{ currentApplication?.name || '应用' }}</h2>
            <p class="console-module-subtitle">查看并维护当前应用的协议能力、令牌参数与接入配置。</p>
          </div>
        </div>
        <div class="console-action-menu" role="group" aria-label="应用操作">
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
        <div v-for="item in applicationDetailMetrics" :key="item.label" class="console-module-metric">
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
        <button v-for="item in applicationDetailPanels" :key="item.id" type="button" class="console-module-sidebar-link" @click="consoleStore.scrollToPanel(item.id)">{{ item.label }}</button>
      </aside>
      <div class="console-module-main">
        <div id="application-protocol" class="info-card">
          <div class="section-title">协议配置</div>
          <div class="record-meta mb-3">Application Type 仅作为内部标记。真正控制协议行为的是 Grant Type、Token Type、Enable Refresh Token、Client Authentication Type 四个维度。</div>
          <div class="detail-card mb-3">
            <div class="record-meta">Grant Type：表示 OAuth 授权流程。</div>
            <div class="record-meta">Token Type：表示主令牌类型，可选 `access_token`、`access_token + id_token`、`id_token`。</div>
            <div class="record-meta">Enable Refresh Token：表示是否额外签发 `refresh_token`，后续可用 `grant_type=refresh_token` 更新访问令牌。</div>
            <div class="record-meta">Client Authentication Type：表示客户端调用 token endpoint 时如何证明自己。</div>
            <div class="record-meta">应用角色：表示该 Application 具备哪些策略组，例如 `api:manage`、`api:user`。</div>
            <div class="record-meta mt-2">约束：</div>
            <div class="record-meta">1. `client_credentials` 只能使用 `access_token`，且不能启用 Refresh Token。</div>
            <div class="record-meta">2. `implicit` 只允许 `access_token`、`access_token + id_token`、`id_token`，且不能启用 Refresh Token。</div>
            <div class="record-meta">3. `authorization_code_pkce` 必须使用 `client_authentication_type=none`。</div>
            <div class="record-meta">4. 其他 Grant Type 不能使用 `client_authentication_type=none`。</div>
            <div class="record-meta mt-2">{{ currentApplicationProtocolHint }}</div>
          </div>
          <BForm @submit.prevent="emit('update-application')">
            <div class="mb-3">
              <label class="form-label">应用名称</label>
              <BFormInput v-model="applicationUpdateForm.name" placeholder="请输入应用名称" />
            </div>
            <div v-if="supportsLoginPresentation" class="mb-3">
              <label class="form-label">回调地址</label>
              <BFormInput v-model="applicationUpdateForm.redirectUris" placeholder="请输入回调地址" />
            </div>
            <div class="mb-3">
              <label class="form-label">Application Type</label>
              <BFormSelect v-model="applicationUpdateForm.applicationType" :options="applicationTypeOptions" />
            </div>
            <div class="detail-card mb-3">
              <div class="form-label mb-2">Token Type</div>
              <div class="d-flex flex-wrap gap-3">
                <label v-for="item in applicationUpdateTokenTypeOptions" :key="item.value" class="d-inline-flex align-items-center gap-2">
                  <input
                    class="form-check-input mt-0"
                    type="checkbox"
                    :checked="applicationUpdateForm.tokenType.includes(item.value)"
                    @change="toggleApplicationTokenType(applicationUpdateForm.tokenType, item.value, ($event.target as HTMLInputElement).checked)"
                  />
                  <span>{{ item.text }}</span>
                </label>
              </div>
            </div>
            <div class="mb-3">
              <label class="form-label d-block">Refresh Token</label>
              <BFormCheckbox v-model="applicationUpdateForm.enableRefreshToken">启用 Refresh Token</BFormCheckbox>
            </div>
            <div class="detail-card mb-3">
              <div class="form-label mb-2">Grant Type</div>
              <div class="d-flex flex-wrap gap-3">
                <label v-for="item in grantTypeOptions" :key="item.value" class="d-inline-flex align-items-center gap-2">
                  <input
                    class="form-check-input mt-0"
                    type="checkbox"
                    :checked="applicationUpdateForm.grantType.includes(item.value)"
                    @change="toggleApplicationGrantType(applicationUpdateForm.grantType, item.value, ($event.target as HTMLInputElement).checked)"
                  />
                  <span>{{ item.text }}</span>
                </label>
              </div>
            </div>
            <div class="mb-3">
              <label class="form-label">Client Authentication Type</label>
              <BFormSelect v-model="applicationUpdateForm.clientAuthenticationType" :options="applicationUpdateClientAuthenticationTypeOptions" />
            </div>
            <BButton type="submit" variant="outline-primary">保存协议配置</BButton>
          </BForm>
        </div>
        <div v-if="supportsLoginPresentation" id="application-metadata" class="info-card">
          <div class="section-title">维护元信息</div>
          <div class="record-meta mb-3">这些元信息会作为应用级变量，用于登录页显示名称等展示场景。多语言显示名称建议使用 `displayName`、`displayName.en`、`displayName.ja`、`displayName.chs`、`displayName.cht`。</div>
          <div class="detail-card">
            <div class="metadata-table-wrap">
              <table class="table table-sm align-middle mb-0">
                <thead>
                  <tr>
                    <th class="metadata-col-key">键</th>
                    <th>值</th>
                    <th class="metadata-col-action"></th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="(item, index) in applicationMetadataRows" :key="item.id">
                    <td>
                      <BFormInput v-model="item.key" placeholder="例如 displayName.chs" />
                    </td>
                    <td>
                      <BFormInput v-model="item.value" placeholder="例如 控制台" />
                    </td>
                    <td class="text-end">
                      <BButton size="sm" variant="outline-danger" @click="removeApplicationMetadataRow(index)">删除</BButton>
                    </td>
                  </tr>
                  <tr v-if="applicationMetadataRows.length === 0">
                    <td colspan="3" class="text-center text-secondary py-4">当前还没有元信息，新增后可作为应用级变量使用。</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <div class="d-flex gap-2 mt-3">
              <BButton variant="outline-secondary" @click="addApplicationMetadataRow">新增条目</BButton>
              <BButton variant="primary" @click="emit('save-application-metadata', applicationMetadataRows)">保存元信息</BButton>
            </div>
          </div>
        </div>
        <div id="application-role-assignment" class="info-card">
          <div class="section-title">角色分配</div>
          <div class="record-meta mb-3">维护当前应用可授予或可使用的应用角色标签。</div>
          <BForm @submit.prevent="emit('update-application')">
            <div class="detail-card mb-3">
              <div class="form-label mb-2">应用角色</div>
              <div class="d-flex flex-wrap gap-3">
                <label v-for="item in applicationAssignableRoles" :key="item.id" class="d-inline-flex align-items-center gap-2">
                  <input
                    class="form-check-input mt-0"
                    type="checkbox"
                    :checked="applicationUpdateForm.roles.includes(item.name)"
                    @change="toggleRoleName(applicationUpdateForm.roles, item.name, ($event.target as HTMLInputElement).checked)"
                  />
                  <span>{{ item.name }}</span>
                </label>
              </div>
            </div>
            <BButton type="submit" variant="outline-primary">保存角色分配</BButton>
          </BForm>
        </div>
        <div id="application-token" class="info-card">
          <div class="section-title">令牌设置</div>
          <div class="record-meta mb-3">Issuer 为实例级统一配置。`private_key_jwt` 应用的公钥以 Ed25519 裸公钥形式保存在系统中；普通应用的私钥只在创建或重置时显示一次，系统内置 API 应用的私钥固化在代码中。</div>
          <BForm @submit.prevent="emit('update-application')">
            <div v-if="applicationUpdateForm.clientAuthenticationType === 'private_key_jwt'" class="mb-3">
              <label class="form-label">应用公钥</label>
              <textarea
                class="form-control"
                rows="8"
                :value="applicationUpdateForm.publicKey"
                readonly
              />
            </div>
            <div class="mb-3">
              <label class="form-label">Access Token TTL Minutes</label>
              <BFormInput v-model="applicationUpdateForm.accessTokenTTLMinutes" type="number" placeholder="请输入 access token ttl minutes" />
            </div>
            <div class="mb-3">
              <label class="form-label">Refresh Token TTL Hours</label>
              <BFormInput v-model="applicationUpdateForm.refreshTokenTTLHours" type="number" placeholder="请输入 refresh token ttl hours" />
            </div>
            <div class="d-flex gap-2">
              <BButton type="submit" variant="outline-primary">保存令牌设置</BButton>
              <BButton
                v-if="applicationUpdateForm.clientAuthenticationType === 'private_key_jwt' && applicationUpdateForm.id"
                type="button"
                variant="outline-danger"
                @click="emit('reset-application-key')"
              >
                重置密钥
              </BButton>
            </div>
          </BForm>
        </div>
      </div>
      <RightSide :items="moduleRecentChanges" />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { BButton, BForm, BFormCheckbox, BFormInput, BFormSelect } from 'bootstrap-vue-next'
import RightSide from '@/layout/RightSide.vue'
import { useAuditStore } from '@/stores/audit'
import { useConsoleStore } from '@/stores/console'

const props = defineProps<{
  currentApplication: any
  applicationUpdateForm: {
    id: string
    name: string
    metadata: Record<string, string>
    displayName: string
    displayNameEn: string
    displayNameJa: string
    displayNameChs: string
    displayNameCht: string
    redirectUris: string
    applicationType: string
    grantType: string[]
    clientAuthenticationType: string
    tokenType: string[]
    enableRefreshToken: boolean
    roles: string[]
    publicKey: string
    accessTokenTTLMinutes: number
    refreshTokenTTLHours: number
  }
  applicationTypeOptions: Array<{ value: string; text: string }>
  grantTypeOptions: Array<{ value: string; text: string }>
  tokenTypeOptions: Array<{ value: string; text: string }>
  clientAuthenticationTypeOptions: Array<{ value: string; text: string }>
  applicationAssignableRoles: any[]
  formatApplicationType: (value?: string) => string
  formatApplicationTokenType: (value?: string | string[]) => string
  formatApplicationGrantType: (value?: string | string[]) => string
  formatApplicationClientAuthenticationType: (value?: string) => string
  formatRoleLabels: (roles?: string[]) => string
}>()

const auditStore = useAuditStore()
const consoleStore = useConsoleStore()
const moduleRecentChanges = computed(() => auditStore.moduleRecentChanges)
const formatDateTime = consoleStore.formatDateTime

const tokenTypeOptionsByGrantType: Record<string, string[]> = {
  authorization_code: ['access_token', 'id_token'],
  authorization_code_pkce: ['access_token', 'id_token'],
  client_credentials: ['access_token'],
  device_code: ['access_token'],
  implicit: ['access_token', 'id_token'],
  password: ['access_token']
}

const clientAuthenticationTypeOptionsByGrantType: Record<string, string[]> = {
  authorization_code: ['client_secret_basic', 'client_secret_post', 'private_key_jwt'],
  authorization_code_pkce: ['none'],
  client_credentials: ['client_secret_basic', 'client_secret_post', 'private_key_jwt'],
  device_code: ['none'],
  implicit: ['client_secret_basic', 'client_secret_post', 'private_key_jwt'],
  password: ['none', 'client_secret_basic', 'client_secret_post', 'private_key_jwt']
}

const applicationDetailPanels = computed(() => {
  const panels = [{ id: 'application-protocol', label: '协议配置' }]
  if (supportsLoginPresentation.value) {
    panels.push({ id: 'application-metadata', label: '维护元信息' })
  }
  panels.push(
    { id: 'application-role-assignment', label: '角色分配' },
    { id: 'application-token', label: '令牌设置' }
  )
  return panels
})

const applicationDetailMetrics = computed(() => [
  { label: '应用 ID', value: props.currentApplication?.id || '-', copyable: Boolean(props.currentApplication?.id), copyValue: props.currentApplication?.id || '' },
  { label: '应用类型', value: props.formatApplicationType(props.currentApplication?.applicationType) },
  { label: '令牌类型', value: props.formatApplicationTokenType(props.currentApplication?.tokenType) },
  { label: '刷新令牌', value: props.currentApplication?.enableRefreshToken ? '已启用' : '未启用' },
  { label: '授权流程', value: props.formatApplicationGrantType(props.currentApplication?.grantType) },
  { label: '客户端认证', value: props.formatApplicationClientAuthenticationType(props.currentApplication?.clientAuthenticationType) },
  { label: '应用角色', value: props.formatRoleLabels(props.currentApplication?.roles) },
  { label: '创建时间', value: formatDateTime(props.currentApplication?.createdAt) },
  { label: '最近变更', value: formatDateTime(props.currentApplication?.updatedAt) }
])

const currentApplicationProtocolHint = computed(() => {
  const applicationType = props.applicationUpdateForm.applicationType || props.currentApplication?.applicationType
  if (applicationType === 'api') {
    return '推荐 API 类型默认使用 `client_credentials + access_token + private_key_jwt`，并关闭 Refresh Token。'
  }
  if (applicationType === 'native') {
    return '推荐 Native 类型优先使用 `authorization_code_pkce` 或 `device_code`。如需长期会话，可额外开启 Refresh Token。'
  }
  return '推荐 Web 类型优先使用 `authorization_code_pkce + access_token + none`。如需 OIDC 前端消费身份声明，可改为 `access_token_id_token`。'
})

const applicationUpdateTokenTypeOptions = computed(() => filterApplicationTokenTypeOptions(props.applicationUpdateForm.grantType))
const applicationUpdateClientAuthenticationTypeOptions = computed(() => filterApplicationClientAuthenticationTypeOptions(props.applicationUpdateForm.grantType))
const applicationMetadataRows = ref<Array<{ id: string; key: string; value: string }>>([])
const supportsLoginPresentation = computed(() => props.applicationUpdateForm.applicationType === 'web' || props.applicationUpdateForm.applicationType === 'native')

function applyRecommendedApplicationProtocol() {
  const target = props.applicationUpdateForm
  if (target.applicationType === 'api') {
    target.redirectUris = ''
    target.metadata = {}
    target.displayName = ''
    target.displayNameEn = ''
    target.displayNameJa = ''
    target.displayNameChs = ''
    target.displayNameCht = ''
    target.tokenType = ['access_token']
    target.enableRefreshToken = false
    target.grantType = ['client_credentials']
    target.clientAuthenticationType = 'private_key_jwt'
    return
  }
  if (target.applicationType === 'native') {
    target.tokenType = ['access_token']
    target.enableRefreshToken = false
    target.grantType = ['authorization_code_pkce']
    target.clientAuthenticationType = 'none'
    return
  }
  target.tokenType = ['access_token']
  target.enableRefreshToken = false
  target.grantType = ['authorization_code_pkce']
  target.clientAuthenticationType = 'none'
}

function intersectOptionValues(groups: string[][], fallback: string[]) {
  if (!groups.length) {
    return fallback
  }
  return groups.reduce((acc, group) => acc.filter((item) => group.includes(item)))
}

function filterApplicationTokenTypeOptions(grantTypes: string[]) {
  const allowed = intersectOptionValues(
    grantTypes.map((grantType) => tokenTypeOptionsByGrantType[grantType] ?? props.tokenTypeOptions.map((item) => item.value)),
    props.tokenTypeOptions.map((item) => item.value)
  )
  return props.tokenTypeOptions.filter((item) => allowed.includes(item.value))
}

function filterApplicationClientAuthenticationTypeOptions(grantTypes: string[]) {
  const allowed = intersectOptionValues(
    grantTypes.map((grantType) => clientAuthenticationTypeOptionsByGrantType[grantType] ?? props.clientAuthenticationTypeOptions.map((item) => item.value)),
    props.clientAuthenticationTypeOptions.map((item) => item.value)
  )
  return props.clientAuthenticationTypeOptions.filter((item) => allowed.includes(item.value))
}

function normalizeApplicationProtocolSelection() {
  const target = props.applicationUpdateForm
  if (!target.grantType.length) {
    target.grantType = ['authorization_code_pkce']
  }
  const allowedTokenTypes = filterApplicationTokenTypeOptions(target.grantType).map((item) => item.value)
  target.tokenType = target.tokenType.filter((item) => allowedTokenTypes.includes(item))
  if (!target.tokenType.length) {
    target.tokenType = allowedTokenTypes.length ? [allowedTokenTypes[0]] : ['access_token']
  }

  const allowedClientAuthenticationTypes = filterApplicationClientAuthenticationTypeOptions(target.grantType).map((item) => item.value)
  if (!allowedClientAuthenticationTypes.includes(target.clientAuthenticationType)) {
    target.clientAuthenticationType = allowedClientAuthenticationTypes[0] ?? 'none'
  }

  if (target.grantType.includes('client_credentials')) {
    target.tokenType = ['access_token']
    target.enableRefreshToken = false
  }
  if (target.grantType.includes('implicit')) {
    target.enableRefreshToken = false
  }
  if (!target.tokenType.includes('access_token')) {
    target.enableRefreshToken = false
  }
}

function toggleApplicationTokenType(items: string[], value: string, checked: boolean) {
  const next = new Set(items)
  if (checked) next.add(value)
  else next.delete(value)
  items.splice(0, items.length, ...Array.from(next))
}

function toggleApplicationGrantType(items: string[], value: string, checked: boolean) {
  const next = new Set(items)
  if (checked) next.add(value)
  else next.delete(value)
  items.splice(0, items.length, ...Array.from(next))
}

function toggleRoleName(items: string[], value: string, checked: boolean) {
  const next = new Set(items)
  if (checked) {
    next.add(value)
  } else {
    next.delete(value)
  }
  items.splice(0, items.length, ...Array.from(next).sort())
}

function createLocalRowID() {
  return `${Date.now()}-${Math.random().toString(16).slice(2)}`
}

function addApplicationMetadataRow() {
  applicationMetadataRows.value.push({
    id: createLocalRowID(),
    key: '',
    value: ''
  })
}

function removeApplicationMetadataRow(index: number) {
  applicationMetadataRows.value.splice(index, 1)
}

watch(() => props.applicationUpdateForm.applicationType, () => applyRecommendedApplicationProtocol())
watch(() => [...props.applicationUpdateForm.grantType], () => normalizeApplicationProtocolSelection())
watch(() => [...props.applicationUpdateForm.tokenType], () => normalizeApplicationProtocolSelection())
watch(
  () => props.applicationUpdateForm.metadata,
  (metadata) => {
    const normalized = (!metadata || typeof metadata !== 'object' || Array.isArray(metadata))
      ? {}
      : Object.fromEntries(Object.entries(metadata as Record<string, unknown>).map(([key, value]) => [key, String(value ?? '')]))
    applicationMetadataRows.value = Object.entries(normalized).map(([key, value]) => ({
      id: createLocalRowID(),
      key,
      value
    }))
  },
  { immediate: true, deep: true }
)

const emit = defineEmits<{
  back: []
  disable: []
  delete: []
  'update-application': []
  'save-application-metadata': [rows: Array<{ id?: string; key: string; value: string }>]
  'reset-application-key': []
}>()
</script>
