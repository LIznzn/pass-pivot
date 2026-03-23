<template>
  <BModal :model-value="visible" :title="currentTitle" centered @update:model-value="emit('update:visible', $event)">
    <BForm @submit.prevent>
      <div class="row g-3">
        <div class="col-md-4">
          <label class="form-label">协议</label>
          <BFormInput :model-value="currentProtocol.toUpperCase()" disabled />
        </div>
        <div class="col-md-6">
          <label class="form-label">名称</label>
          <BFormInput v-model="draftForm.name" />
        </div>
        <div class="col-md-6">
          <label class="form-label">Issuer</label>
          <BFormInput v-model="draftForm.issuer" />
        </div>
        <div class="col-md-6">
          <label class="form-label">Client ID / App Key</label>
          <BFormInput v-model="draftForm.clientId" />
        </div>
        <div class="col-md-6">
          <label class="form-label">Client Secret / App Secret</label>
          <BFormInput v-model="draftForm.clientSecret" type="password" placeholder="留空则保持原值" />
        </div>
        <div class="col-md-12">
          <label class="form-label">Scopes</label>
          <BFormInput v-model="draftForm.scopes" />
        </div>
        <div class="col-md-6">
          <label class="form-label">Authorization URL</label>
          <BFormInput v-model="draftForm.authorizationUrl" />
        </div>
        <div class="col-md-6">
          <label class="form-label">Token URL</label>
          <BFormInput v-model="draftForm.tokenUrl" />
        </div>
        <div class="col-md-6">
          <label class="form-label">UserInfo URL</label>
          <BFormInput v-model="draftForm.userInfoUrl" />
        </div>
        <div v-if="currentProtocol === 'oidc'" class="col-md-6">
          <label class="form-label">JWKS URL</label>
          <BFormInput v-model="draftForm.jwksUrl" />
        </div>
      </div>
    </BForm>
    <template #footer>
      <div class="d-flex justify-content-end gap-2 w-100">
        <BButton type="button" variant="outline-secondary" @click="emit('update:visible', false)">关闭</BButton>
        <BButton type="button" variant="primary" @click="submitForm">{{ currentActionLabel }}</BButton>
      </div>
    </template>
  </BModal>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { BButton, BForm, BFormInput, BModal } from 'bootstrap-vue-next'

const props = defineProps<{
  visible: boolean
  kind: 'google' | 'github' | 'apple'
  form: {
    id?: string
    name: string
    protocol: string
    issuer: string
    clientId: string
    clientSecret: string
    scopes: string
    authorizationUrl: string
    tokenUrl: string
    userInfoUrl: string
    jwksUrl: string
  }
}>()

const currentTitle = computed(() => {
  if (props.kind === 'google') return '配置 Google 登录'
  if (props.kind === 'github') return '配置 GitHub 登录'
  return '配置 Apple 登录'
})

const currentProtocol = computed(() => props.form.protocol || (props.kind === 'apple' ? 'oidc' : 'oauth'))
const currentActionLabel = computed(() => props.form.id ? '保存配置' : '启用 Provider')

const draftForm = reactive({
  id: '',
  name: '',
  protocol: '',
  issuer: '',
  clientId: '',
  clientSecret: '',
  scopes: '',
  authorizationUrl: '',
  tokenUrl: '',
  userInfoUrl: '',
  jwksUrl: ''
})

watch(
  () => [props.visible, props.form, props.kind],
  () => {
    if (!props.visible) {
      return
    }
    draftForm.id = props.form.id || ''
    draftForm.name = props.form.name || ''
    draftForm.protocol = props.form.protocol || (props.kind === 'apple' ? 'oidc' : 'oauth')
    draftForm.issuer = props.form.issuer || ''
    draftForm.clientId = props.form.clientId || ''
    draftForm.clientSecret = ''
    draftForm.scopes = props.form.scopes || ''
    draftForm.authorizationUrl = props.form.authorizationUrl || ''
    draftForm.tokenUrl = props.form.tokenUrl || ''
    draftForm.userInfoUrl = props.form.userInfoUrl || ''
    draftForm.jwksUrl = props.form.jwksUrl || ''
  },
  { immediate: true, deep: true }
)

const emit = defineEmits<{
  'update:visible': [value: boolean]
  submit: [form: {
    id?: string
    name: string
    protocol: string
    issuer: string
    clientId: string
    clientSecret: string
    scopes: string
    authorizationUrl: string
    tokenUrl: string
    userInfoUrl: string
    jwksUrl: string
  }]
}>()

function submitForm() {
  emit('submit', {
    id: draftForm.id,
    name: draftForm.name,
    protocol: draftForm.protocol,
    issuer: draftForm.issuer,
    clientId: draftForm.clientId,
    clientSecret: draftForm.clientSecret,
    scopes: draftForm.scopes,
    authorizationUrl: draftForm.authorizationUrl,
    tokenUrl: draftForm.tokenUrl,
    userInfoUrl: draftForm.userInfoUrl,
    jwksUrl: draftForm.jwksUrl
  })
}
</script>
