<template>
  <section class="console-module-shell">
    <div class="console-module-summary-card">
      <div class="console-module-hero">
          <div>
            <div class="console-module-eyebrow">组织</div>
            <h2 class="console-module-title">{{ currentOrganization?.name || '组织' }}</h2>
          <p class="console-module-subtitle">{{ currentOrganization?.description || '描述' }}</p>
          </div>
        <div class="console-action-menu" role="group" aria-label="组织操作">
          <button type="button" class="btn btn-primary console-action-menu-toggle">
            操作
            <i class="bi bi-chevron-down" aria-hidden="true"></i>
          </button>
          <div class="console-action-menu-list">
            <button type="button" class="console-action-menu-item" @click="organizationStore.showOrganizationDisableNotice">停用</button>
            <button type="button" class="console-action-menu-item console-action-menu-item-danger" @click="organizationStore.showOrganizationDeleteNotice">删除</button>
          </div>
        </div>
      </div>
      <div class="console-module-metrics console-module-metrics-inline">
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
        <div id="organization-basic" class="info-card">
          <div class="section-title">基本设置</div>
          <BForm @submit.prevent="saveOrganizationBasicSettings">
            <div class="row g-3">
              <div class="col-12">
                <label class="form-label">名称</label>
                <BFormInput v-model="organizationBasicSettingForm.name" />
              </div>
              <div class="col-12">
                <label class="form-label">描述</label>
                <BFormInput v-model="organizationBasicSettingForm.description" />
              </div>
            </div>
            <div class="d-flex justify-content-end mt-3">
              <BButton type="submit" variant="primary">保存基本设置</BButton>
            </div>
          </BForm>
        </div>

        <div id="organization-metadata" class="info-card">
          <div class="section-title">维护元信息</div>
          <div class="record-meta mb-3">这些元信息会作为可用变量，用于自定义登录页等组织级展示场景。</div>
          <div v-if="currentOrganization" class="detail-card">
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
                  <tr v-for="(item, index) in organizationMetadataRows" :key="item.id">
                    <td>
                      <BFormInput v-model="item.key" placeholder="例如 login_title" />
                    </td>
                    <td>
                      <BFormInput v-model="item.value" placeholder="例如 PPVT 控制台" />
                    </td>
                    <td class="text-end">
                      <BButton size="sm" variant="outline-danger" @click="removeOrganizationMetadataRow(index)">删除</BButton>
                    </td>
                  </tr>
                  <tr v-if="organizationMetadataRows.length === 0">
                    <td colspan="3" class="text-center text-secondary py-4">当前还没有元信息，新增后可作为组织级变量使用。</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <div class="d-flex gap-2 mt-3">
              <BButton variant="outline-secondary" @click="addOrganizationMetadataRow">新增条目</BButton>
              <BButton variant="primary" @click="organizationStore.saveOrganizationMetadata(organizationMetadataRows)">保存元信息</BButton>
            </div>
          </div>
        </div>

        <div id="organization-mail" class="info-card">
          <div class="section-title">邮箱配置</div>
          <BForm @submit.prevent="saveOrganizationMailSettings">
            <div class="row g-3">
              <div class="col-md-4">
                <label class="form-label">邮件服务类型</label>
                <BFormSelect v-model="organizationMailSettingForm.provider" :options="mailProviderOptions" />
              </div>
              <div v-if="organizationMailSettingForm.provider !== 'disabled'" class="col-md-8">
                <label class="form-label">发件人邮箱</label>
                <BFormInput v-model="organizationMailSettingForm.from" type="email" placeholder="例如 noreply@example.com" />
              </div>

              <template v-if="organizationMailSettingForm.provider === 'smtp'">
                <div class="col-md-4">
                  <label class="form-label">SMTP 主机</label>
                  <BFormInput v-model="organizationMailSettingForm.smtpHost" />
                </div>
                <div class="col-md-4">
                  <label class="form-label">SMTP 端口</label>
                  <BFormInput v-model="organizationMailSettingForm.smtpPort" type="number" min="1" />
                </div>
                <div class="col-md-4">
                  <label class="form-label">SMTP 用户名</label>
                  <BFormInput v-model="organizationMailSettingForm.smtpUser" />
                </div>
                <div class="col-md-6">
                  <label class="form-label">SMTP 密码</label>
                  <BFormInput v-model="organizationMailSettingForm.smtpPass" type="password" />
                </div>
              </template>

              <template v-if="organizationMailSettingForm.provider === 'mailgun'">
                <div class="col-md-4">
                  <label class="form-label">Mailgun Domain</label>
                  <BFormInput v-model="organizationMailSettingForm.mailgunDomain" placeholder="mg.example.com" />
                </div>
                <div class="col-md-4">
                  <label class="form-label">Mailgun API Key</label>
                  <BFormInput v-model="organizationMailSettingForm.mailgunApiKey" type="password" />
                </div>
                <div class="col-md-4">
                  <label class="form-label">Mailgun API Base</label>
                  <BFormInput v-model="organizationMailSettingForm.mailgunApiBase" placeholder="https://api.mailgun.net" />
                </div>
              </template>

              <template v-if="organizationMailSettingForm.provider === 'sendgrid'">
                <div class="col-md-6">
                  <label class="form-label">SendGrid API Key</label>
                  <BFormInput v-model="organizationMailSettingForm.sendgridApiKey" type="password" />
                </div>
              </template>
            </div>
            <div class="record-meta mt-3">支持 `mailgun`、`sendgrid`、`SMTP`。邮箱验证码和密码重置都会使用这里的组织级发信配置。</div>
            <div class="d-flex justify-content-end mt-3">
              <BButton type="submit" variant="primary">保存邮箱配置</BButton>
            </div>
          </BForm>
        </div>
      </div>
      <RightSide :items="moduleRecentChanges" />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch, watchEffect } from 'vue'
import { BButton, BForm, BFormInput, BFormSelect, useToast } from 'bootstrap-vue-next'
import RightSide from '@/layout/RightSide.vue'
import { useAuditStore } from '@/stores/audit'
import { useConsoleStore } from '@/stores/console'
import { useOrganizationStore } from '@/stores/organization'
import { useRoleStore } from '@/stores/role'
import { useUserStore } from '@/stores/user'
import { notifyToast } from '@shared/utils/notify'

const auditStore = useAuditStore()
const consoleStore = useConsoleStore()
const organizationStore = useOrganizationStore()
const roleStore = useRoleStore()
const userStore = useUserStore()
const toast = useToast()

function showToast(
  message: string,
  variant: 'success' | 'danger',
  options: {
    source: string
    trigger?: string
    error?: unknown
    metadata?: Record<string, unknown>
  } = {
    source: 'console/Organization'
  }
) {
  notifyToast({
    toast,
    message,
    variant,
    source: options.source,
    trigger: options.trigger,
    error: options.error,
    metadata: options.metadata
  })
}

watchEffect(() => {
  consoleStore.setPageHeader(
    '组织',
    '当前组织的基础设置与维护元信息。'
  )
})

const organizationMetadataRows = ref<Array<{ id: string; key: string; value: string }>>([])
const organizationBasicSettingForm = reactive({
  name: '',
  description: ''
})
const organizationMailSettingForm = reactive({
  provider: 'disabled' as 'disabled' | 'smtp' | 'mailgun' | 'sendgrid',
  from: '',
  smtpHost: '',
  smtpPort: 587,
  smtpUser: '',
  smtpPass: '',
  mailgunDomain: '',
  mailgunApiKey: '',
  mailgunApiBase: '',
  sendgridApiKey: ''
})
const mailProviderOptions = [
  { value: 'disabled', text: '不开启' },
  { value: 'smtp', text: 'SMTP' },
  { value: 'mailgun', text: 'Mailgun' },
  { value: 'sendgrid', text: 'SendGrid' }
]

const currentModulePanels = [
  { id: 'organization-basic', label: '基本设置' },
  { id: 'organization-metadata', label: '维护元信息' },
  { id: 'organization-mail', label: '邮箱配置' }
]

const currentModuleMetrics = computed(() => {
  const projectCount = organizationStore.currentOrganization?.projects?.length ?? 0
  const applicationCount = (organizationStore.currentOrganization?.projects ?? []).reduce((count: number, project: any) => count + (project.applications?.length ?? 0), 0)
  return [
    { label: '组织 ID', value: organizationStore.currentOrganization?.id || '-', copyable: Boolean(organizationStore.currentOrganization?.id), copyValue: organizationStore.currentOrganization?.id || '' },
    { label: '项目数', value: String(projectCount) },
    { label: '应用数', value: String(applicationCount) },
    { label: '用户数', value: String(userStore.users.length) },
    { label: '角色数', value: String(roleStore.roles.length) },
    { label: '创建时间', value: consoleStore.formatDateTime(organizationStore.currentOrganization?.createdAt) },
    { label: '更新时间', value: consoleStore.formatDateTime(organizationStore.currentOrganization?.updatedAt) }
  ]
})

watch(
  () => organizationStore.currentOrganization,
  (organization) => {
    organizationBasicSettingForm.name = organization?.name || ''
    organizationBasicSettingForm.description = organization?.description || ''
    const mail = parseOrganizationMailSettings(organization?.consoleSettings)
    organizationMailSettingForm.provider = mail.provider
    organizationMailSettingForm.from = mail.from
    organizationMailSettingForm.smtpHost = mail.smtpHost
    organizationMailSettingForm.smtpPort = mail.smtpPort
    organizationMailSettingForm.smtpUser = mail.smtpUser
    organizationMailSettingForm.smtpPass = mail.smtpPass
    organizationMailSettingForm.mailgunDomain = mail.mailgunDomain
    organizationMailSettingForm.mailgunApiKey = mail.mailgunApiKey
    organizationMailSettingForm.mailgunApiBase = mail.mailgunApiBase
    organizationMailSettingForm.sendgridApiKey = mail.sendgridApiKey
  },
  { immediate: true }
)

watch(
  () => organizationStore.currentOrganization?.metadata,
  (metadata) => {
    const normalized = (!metadata || typeof metadata !== 'object' || Array.isArray(metadata))
      ? {}
      : Object.fromEntries(Object.entries(metadata as Record<string, unknown>).map(([key, value]) => [key, String(value ?? '')]))
    organizationMetadataRows.value = Object.entries(normalized).map(([key, value]) => ({
      id: createLocalRowId(),
      key,
      value
    }))
  },
  { immediate: true, deep: true }
)

function createLocalRowId() {
  return `${Date.now()}-${Math.random().toString(16).slice(2)}`
}

function addOrganizationMetadataRow() {
  organizationMetadataRows.value.push({
    id: createLocalRowId(),
    key: '',
    value: ''
  })
}

function removeOrganizationMetadataRow(index: number) {
  organizationMetadataRows.value.splice(index, 1)
}

function buildOrganizationConsoleSettings() {
  const currentSettings = organizationStore.currentOrganization?.consoleSettings
  const normalizedCurrentSettings = currentSettings && typeof currentSettings === 'object' && !Array.isArray(currentSettings)
    ? currentSettings
    : {}
  const { supportEmail: _supportEmail, logoUrl: _logoUrl, ...rest } = normalizedCurrentSettings as Record<string, unknown>
  return rest
}

function parseOrganizationMailSettings(settings?: unknown) {
  const parsed = settings && typeof settings === 'object' && !Array.isArray(settings)
    ? settings as Record<string, any>
    : {}
  const mail = parsed.mail && typeof parsed.mail === 'object' && !Array.isArray(parsed.mail)
    ? parsed.mail
    : {}
  const provider = ['disabled', 'smtp', 'mailgun', 'sendgrid'].includes(String(mail.provider || '').toLowerCase())
    ? String(mail.provider).toLowerCase() as 'disabled' | 'smtp' | 'mailgun' | 'sendgrid'
    : 'disabled'
  return {
    provider,
    from: String(mail.from || ''),
    smtpHost: String(mail.smtpHost || ''),
    smtpPort: Number(mail.smtpPort || 587),
    smtpUser: String(mail.smtpUser || ''),
    smtpPass: String(mail.smtpPass || ''),
    mailgunDomain: String(mail.mailgunDomain || ''),
    mailgunApiKey: String(mail.mailgunApiKey || ''),
    mailgunApiBase: String(mail.mailgunApiBase || ''),
    sendgridApiKey: String(mail.sendgridApiKey || '')
  }
}

async function saveOrganizationBasicSettings() {
  try {
    await organizationStore.saveOrganizationConsoleSettings(buildOrganizationConsoleSettings(), {
      name: organizationBasicSettingForm.name,
      description: organizationBasicSettingForm.description
    })
    showToast('基本设置已保存', 'success', {
      source: 'console/Organization.saveOrganizationBasicSettings',
      trigger: 'saveOrganizationBasicSettings'
    })
  } catch (error) {
    showToast(String(error), 'danger', {
      source: 'console/Organization.saveOrganizationBasicSettings',
      trigger: 'saveOrganizationBasicSettings',
      error
    })
  }
}

async function saveOrganizationMailSettings() {
  try {
    await organizationStore.saveOrganizationConsoleSettings({
      ...buildOrganizationConsoleSettings(),
      mail: {
        provider: organizationMailSettingForm.provider,
        from: organizationMailSettingForm.from.trim(),
        smtpHost: organizationMailSettingForm.smtpHost.trim(),
        smtpPort: Number(organizationMailSettingForm.smtpPort),
        smtpUser: organizationMailSettingForm.smtpUser.trim(),
        smtpPass: organizationMailSettingForm.smtpPass,
        mailgunDomain: organizationMailSettingForm.mailgunDomain.trim(),
        mailgunApiKey: organizationMailSettingForm.mailgunApiKey.trim(),
        mailgunApiBase: organizationMailSettingForm.mailgunApiBase.trim(),
        sendgridApiKey: organizationMailSettingForm.sendgridApiKey.trim()
      }
    })
    showToast('邮箱配置已保存', 'success', {
      source: 'console/Organization.saveOrganizationMailSettings',
      trigger: 'saveOrganizationMailSettings'
    })
  } catch (error) {
    showToast(String(error), 'danger', {
      source: 'console/Organization.saveOrganizationMailSettings',
      trigger: 'saveOrganizationMailSettings',
      error
    })
  }
}

const currentOrganization = computed(() => organizationStore.currentOrganization)
const moduleRecentChanges = computed(() => auditStore.moduleRecentChanges)
const formatDateTime = consoleStore.formatDateTime
</script>
