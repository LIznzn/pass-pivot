<template>
  <BModal :model-value="visible" title="创建应用" size="xl" centered @update:model-value="emit('update:visible', $event)" @hidden="emit('hidden')">
      <BForm @submit.prevent="submitForm">
        <BFormSelect v-model="applicationTemplateSelection" :options="applicationProtocolTemplateOptions" class="mb-2" />
        <div class="detail-card mb-2">
          <div class="record-meta mb-2">模板说明</div>
          <div class="record-meta mb-3">模板会自动填充 `grant_type`、`enable_refresh_token`、`token_type`、`client_authentication_type`。仅显示当前应用类型允许使用的模板。</div>
          <div class="metadata-table-wrap">
            <table class="table table-sm align-middle mb-0">
              <thead>
                <tr>
                  <th>模板</th>
                  <th>允许类型</th>
                  <th>Grant Type</th>
                  <th>Refresh Token</th>
                  <th>Token Type</th>
                  <th>Client Authentication Type</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="item in visibleApplicationProtocolTemplates" :key="item.key">
                  <td>{{ item.text }}</td>
                  <td>{{ item.allowedTypes }}</td>
                  <td><code>{{ item.grantType }}</code></td>
                  <td>{{ item.enableRefreshToken }}</td>
                  <td><code>{{ item.tokenType }}</code></td>
                  <td><code>{{ item.clientAuthenticationType }}</code></td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
        <BFormInput v-model="applicationForm.name" placeholder="应用名称" class="mb-2" />
        <div v-if="supportsLoginPresentation" class="row g-2 mb-2">
          <div class="col-md-6">
            <BFormInput v-model="applicationForm.displayName" placeholder="默认显示名称" />
          </div>
          <div class="col-md-6">
            <BFormInput v-model="applicationForm.displayNameEn" placeholder="显示名称（en）" />
          </div>
          <div class="col-md-6">
            <BFormInput v-model="applicationForm.displayNameJa" placeholder="显示名称（ja）" />
          </div>
          <div class="col-md-6">
            <BFormInput v-model="applicationForm.displayNameChs" placeholder="显示名称（chs）" />
          </div>
          <div class="col-md-6">
            <BFormInput v-model="applicationForm.displayNameCht" placeholder="显示名称（cht）" />
          </div>
        </div>
        <BFormSelect v-model="applicationForm.applicationType" :options="applicationTypeOptions" class="mb-2" />
        <BFormInput v-if="supportsLoginPresentation" v-model="applicationForm.redirectUris" placeholder="回调地址，多个值可用逗号或换行分隔" class="mb-2" />
        <div class="detail-card mb-2">
          <div class="record-meta mb-2">Token Type</div>
          <div class="d-flex flex-wrap gap-3">
            <label v-for="item in applicationFormTokenTypeOptions" :key="item.value" class="d-inline-flex align-items-center gap-2">
              <input
                class="form-check-input mt-0"
                type="checkbox"
                :checked="applicationForm.tokenType.includes(item.value)"
                @change="toggleApplicationTokenType(applicationForm.tokenType, item.value, ($event.target as HTMLInputElement).checked)"
              />
              <span>{{ item.text }}</span>
            </label>
          </div>
        </div>
        <BFormCheckbox v-model="applicationForm.enableRefreshToken" class="mb-2">启用 Refresh Token</BFormCheckbox>
        <div class="detail-card mb-2">
          <div class="record-meta mb-2">Grant Type</div>
          <div class="d-flex flex-wrap gap-3">
            <label v-for="item in grantTypeOptions" :key="item.value" class="d-inline-flex align-items-center gap-2">
              <input
                class="form-check-input mt-0"
                type="checkbox"
                :checked="applicationForm.grantType.includes(item.value)"
                @change="toggleApplicationGrantType(applicationForm.grantType, item.value, ($event.target as HTMLInputElement).checked)"
              />
              <span>{{ item.text }}</span>
            </label>
          </div>
        </div>
        <BFormSelect v-model="applicationForm.clientAuthenticationType" :options="applicationFormClientAuthenticationTypeOptions" class="mb-2" />
        <div class="detail-card mb-2">
          <div class="record-meta mb-2">应用角色</div>
          <div class="d-flex flex-wrap gap-3">
            <label v-for="item in applicationAssignableRoles" :key="item.id" class="d-inline-flex align-items-center gap-2">
              <input
                class="form-check-input mt-0"
                type="checkbox"
                :checked="applicationForm.roles.includes(item.name)"
                @change="emit('toggle-role-name', applicationForm.roles, item.name, ($event.target as HTMLInputElement).checked)"
              />
              <span>{{ item.name }}</span>
            </label>
          </div>
        </div>
        <div v-if="applicationForm.clientAuthenticationType === 'private_key_jwt'" class="record-meta mb-2">
          创建后系统会自动生成一组 RSA 密钥对。数据库仅保存公钥，私钥只会显示一次。
        </div>
        <BFormInput v-model="applicationForm.accessTokenTTLMinutes" type="number" placeholder="access token ttl minutes" class="mb-2" />
        <BFormInput v-model="applicationForm.refreshTokenTTLHours" type="number" placeholder="refresh token ttl hours" class="mb-2" />
        <div class="d-flex gap-2">
          <BButton type="submit" variant="primary">创建应用</BButton>
          <BButton type="button" variant="outline-secondary" @click="emit('update:visible', false)">取消</BButton>
        </div>
      </BForm>
  </BModal>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { BButton, BForm, BFormCheckbox, BFormInput, BFormSelect, BModal } from 'bootstrap-vue-next'

type ProtocolTemplate = {
  text: string
  allowedTypes: string[]
  grantType: string[]
  enableRefreshToken: boolean
  tokenType: string[]
  clientAuthenticationType: string
}

const props = defineProps<{
  visible: boolean
  applicationForm: {
    name: string
    metadata: Record<string, string>
    displayName: string
    displayNameEn: string
    displayNameJa: string
    displayNameChs: string
    displayNameCht: string
    redirectUris: string
    applicationType: string
    tokenType: string[]
    enableRefreshToken: boolean
    grantType: string[]
    clientAuthenticationType: string
    roles: string[]
    accessTokenTTLMinutes: number
    refreshTokenTTLHours: number
  }
  applicationAssignableRoles: any[]
  applicationTypeOptions: Array<{ value: string; text: string }>
  grantTypeOptions: Array<{ value: string; text: string }>
  tokenTypeOptions: Array<{ value: string; text: string }>
  clientAuthenticationTypeOptions: Array<{ value: string; text: string }>
  applicationProtocolTemplates: Record<string, ProtocolTemplate>
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  hidden: []
  submit: []
  'validation-error': [message: string]
  'toggle-role-name': [target: string[], value: string, checked: boolean]
}>()

const applicationTemplateSelection = ref('')

const applicationProtocolTemplateOptions = computed(() => {
  const items = Object.entries(props.applicationProtocolTemplates)
    .filter(([, template]) => template.allowedTypes.includes(props.applicationForm.applicationType))
    .map(([value, template]) => ({ value, text: template.text }))
  return [{ value: '', text: '选择授权模板' }, ...items]
})

const visibleApplicationProtocolTemplates = computed(() => Object.entries(props.applicationProtocolTemplates)
  .filter(([, template]) => template.allowedTypes.includes(props.applicationForm.applicationType))
  .map(([key, template]) => ({
    key,
    text: template.text,
    allowedTypes: template.allowedTypes.join('、'),
    grantType: template.grantType.join(' + '),
    enableRefreshToken: template.enableRefreshToken ? 'true' : 'false',
    tokenType: template.tokenType.join(' + '),
    clientAuthenticationType: template.clientAuthenticationType
  })))

const tokenTypeOptionsByGrantType: Record<string, string[]> = {
  authorization_code: ['access_token', 'id_token'],
  authorization_code_pkce: ['access_token', 'id_token'],
  implicit: ['access_token', 'id_token'],
  client_credentials: ['access_token'],
  device_code: ['access_token', 'id_token'],
  password: ['access_token', 'id_token']
}

const clientAuthenticationTypeOptionsByGrantType: Record<string, string[]> = {
  authorization_code: ['client_secret_basic', 'client_secret_post', 'client_secret_jwt', 'private_key_jwt', 'tls_client_auth', 'self_signed_tls_client_auth'],
  authorization_code_pkce: ['none', 'client_secret_basic', 'client_secret_post', 'client_secret_jwt', 'private_key_jwt', 'tls_client_auth', 'self_signed_tls_client_auth'],
  implicit: ['client_secret_basic', 'client_secret_post', 'client_secret_jwt', 'private_key_jwt', 'tls_client_auth', 'self_signed_tls_client_auth'],
  client_credentials: ['client_secret_basic', 'client_secret_post', 'client_secret_jwt', 'private_key_jwt', 'tls_client_auth', 'self_signed_tls_client_auth'],
  device_code: ['none', 'client_secret_basic', 'client_secret_post', 'client_secret_jwt', 'private_key_jwt', 'tls_client_auth', 'self_signed_tls_client_auth'],
  password: ['none', 'client_secret_basic', 'client_secret_post', 'client_secret_jwt', 'private_key_jwt', 'tls_client_auth', 'self_signed_tls_client_auth']
}

const applicationFormTokenTypeOptions = computed(() => filterApplicationTokenTypeOptions(props.applicationForm.grantType))
const applicationFormClientAuthenticationTypeOptions = computed(() => filterApplicationClientAuthenticationTypeOptions(props.applicationForm.grantType))
const supportsLoginPresentation = computed(() => props.applicationForm.applicationType === 'web' || props.applicationForm.applicationType === 'native')

function applyRecommendedApplicationProtocol() {
  if (props.applicationForm.applicationType === 'api') {
    props.applicationForm.redirectUris = ''
    props.applicationForm.displayName = ''
    props.applicationForm.displayNameEn = ''
    props.applicationForm.displayNameJa = ''
    props.applicationForm.displayNameChs = ''
    props.applicationForm.displayNameCht = ''
    props.applicationForm.tokenType = ['access_token']
    props.applicationForm.enableRefreshToken = false
    props.applicationForm.grantType = ['client_credentials']
    props.applicationForm.clientAuthenticationType = 'private_key_jwt'
    return
  }
  if (props.applicationForm.applicationType === 'native') {
    props.applicationForm.tokenType = ['access_token']
    props.applicationForm.enableRefreshToken = false
    props.applicationForm.grantType = ['authorization_code_pkce']
    props.applicationForm.clientAuthenticationType = 'none'
    return
  }
  props.applicationForm.tokenType = ['access_token']
  props.applicationForm.enableRefreshToken = false
  props.applicationForm.grantType = ['authorization_code_pkce']
  props.applicationForm.clientAuthenticationType = 'none'
}

function intersectOptionValues(groups: string[][], fallback: string[]) {
  if (!groups.length) {
    return fallback
  }
  return groups.reduce((acc, group) => acc.filter((item) => group.includes(item)))
}

function filterApplicationTokenTypeOptions(grantTypes: string[]) {
  const selectedGrantTypes = grantTypes.length ? grantTypes : []
  const allowed = intersectOptionValues(
    selectedGrantTypes.map((grantType) => tokenTypeOptionsByGrantType[grantType] ?? props.tokenTypeOptions.map((item) => item.value)),
    props.tokenTypeOptions.map((item) => item.value)
  )
  return props.tokenTypeOptions.filter((item) => allowed.includes(item.value))
}

function filterApplicationClientAuthenticationTypeOptions(grantTypes: string[]) {
  const selectedGrantTypes = grantTypes.length ? grantTypes : []
  const allowed = intersectOptionValues(
    selectedGrantTypes.map((grantType) => clientAuthenticationTypeOptionsByGrantType[grantType] ?? props.clientAuthenticationTypeOptions.map((item) => item.value)),
    props.clientAuthenticationTypeOptions.map((item) => item.value)
  )
  return props.clientAuthenticationTypeOptions.filter((item) => allowed.includes(item.value))
}

function normalizeApplicationProtocolSelection() {
  if (!props.applicationForm.grantType.length) {
    props.applicationForm.grantType = ['authorization_code_pkce']
  }
  const allowedTokenTypes = filterApplicationTokenTypeOptions(props.applicationForm.grantType).map((item) => item.value)
  props.applicationForm.tokenType = props.applicationForm.tokenType.filter((item) => allowedTokenTypes.includes(item))
  if (!props.applicationForm.tokenType.length) {
    props.applicationForm.tokenType = allowedTokenTypes.length ? [allowedTokenTypes[0]] : ['access_token']
  }

  const allowedClientAuthenticationTypes = filterApplicationClientAuthenticationTypeOptions(props.applicationForm.grantType).map((item) => item.value)
  if (!allowedClientAuthenticationTypes.includes(props.applicationForm.clientAuthenticationType)) {
    props.applicationForm.clientAuthenticationType = allowedClientAuthenticationTypes[0] ?? 'none'
  }

  if (props.applicationForm.grantType.includes('client_credentials')) {
    props.applicationForm.tokenType = ['access_token']
    props.applicationForm.enableRefreshToken = false
  }
  if (props.applicationForm.grantType.includes('implicit')) {
    props.applicationForm.enableRefreshToken = false
  }
  if (!props.applicationForm.tokenType.includes('access_token')) {
    props.applicationForm.enableRefreshToken = false
  }
}

function validateApplicationProtocolInput() {
  if (!props.applicationForm.grantType.length) {
    return '至少需要选择一个 Grant Type。'
  }
  if (!props.applicationForm.tokenType.length) {
    return '至少需要选择一个 Token Type。'
  }
  if (props.applicationForm.grantType.includes('client_credentials') && !(props.applicationForm.tokenType.length === 1 && props.applicationForm.tokenType[0] === 'access_token')) {
    return 'client_credentials 只允许 token_type=access_token。'
  }
  if (props.applicationForm.grantType.includes('client_credentials') && props.applicationForm.enableRefreshToken) {
    return 'client_credentials 不允许启用 Refresh Token。'
  }
  if (props.applicationForm.grantType.includes('implicit') && props.applicationForm.tokenType.some((item) => !['access_token', 'id_token'].includes(item))) {
    return 'implicit 只允许 access_token 和/或 id_token。'
  }
  if (props.applicationForm.grantType.includes('implicit') && props.applicationForm.enableRefreshToken) {
    return 'implicit 不允许启用 Refresh Token。'
  }
  if (props.applicationForm.clientAuthenticationType === 'none' && props.applicationForm.grantType.some((item) => item !== 'authorization_code_pkce' && item !== 'device_code' && item !== 'password')) {
    return 'client_authentication_type=none 只允许用于 authorization_code_pkce、device_code 或 password。'
  }
  if (!props.applicationForm.tokenType.includes('access_token') && props.applicationForm.enableRefreshToken) {
    return '未签发 access_token 时不能启用 Refresh Token。'
  }
  return ''
}

function toggleApplicationTokenType(items: string[], value: string, checked: boolean) {
  const next = new Set(items)
  if (checked) {
    next.add(value)
  } else {
    next.delete(value)
  }
  items.splice(0, items.length, ...Array.from(next))
}

function toggleApplicationGrantType(items: string[], value: string, checked: boolean) {
  const next = new Set(items)
  if (checked) {
    next.add(value)
  } else {
    next.delete(value)
  }
  items.splice(0, items.length, ...Array.from(next))
}

function applyApplicationProtocolTemplate(templateKey: string) {
  const template = props.applicationProtocolTemplates[templateKey]
  if (!template) {
    return
  }
  props.applicationForm.grantType.splice(0, props.applicationForm.grantType.length, ...template.grantType)
  props.applicationForm.tokenType.splice(0, props.applicationForm.tokenType.length, ...template.tokenType)
  props.applicationForm.enableRefreshToken = template.enableRefreshToken
  props.applicationForm.clientAuthenticationType = template.clientAuthenticationType
  normalizeApplicationProtocolSelection()
}

function submitForm() {
  normalizeApplicationProtocolSelection()
  const protocolError = validateApplicationProtocolInput()
  if (protocolError) {
    emit('validation-error', protocolError)
    return
  }
  emit('submit')
}

watch(() => props.applicationForm.applicationType, (value) => {
  applyRecommendedApplicationProtocol()
  if (!applicationTemplateSelection.value) {
    return
  }
  const template = props.applicationProtocolTemplates[applicationTemplateSelection.value]
  if (!template || !template.allowedTypes.includes(value)) {
    applicationTemplateSelection.value = ''
  }
})

watch(applicationTemplateSelection, (value) => {
  if (!value) {
    return
  }
  applyApplicationProtocolTemplate(value)
})

watch(() => [...props.applicationForm.grantType], () => normalizeApplicationProtocolSelection())
watch(() => [...props.applicationForm.tokenType], () => normalizeApplicationProtocolSelection())
</script>
