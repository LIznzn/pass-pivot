<template>
  <BModal :model-value="visible" :title="currentTitle" centered @update:model-value="emit('update:visible', $event)">
    <template v-if="method === 'totp'">
      <div v-for="item in activeTotpEnrollments.slice(0, 1)" :key="item.id" class="record-row mb-3">
        <div>
          <strong>{{ item.label || '身份验证器（TOTP）' }}</strong>
          <div class="record-meta">最近使用：{{ formatDateTime(item.lastUsedAt) }}</div>
        </div>
        <code>{{ item.status }}</code>
      </div>
      <div v-if="!activeTotpEnrollments.length" class="record-meta mb-3">当前没有已激活的身份验证器。</div>
      <div v-if="activeTotpEnrollments.length" class="d-flex gap-2 mb-3">
        <BButton size="sm" variant="outline-danger" @click="emit('delete-totp-enrollments')">关闭</BButton>
      </div>
      <div v-if="totpSetup && !activeTotpEnrollments.length" class="detail-card mb-3">
        <div v-if="totpQrCodeDataUrl" class="text-center mb-3">
          <img :src="totpQrCodeDataUrl" alt="身份验证器二维码" class="img-fluid border rounded p-2 bg-white" />
        </div>
        <div class="record-meta">待激活 Enrollment：{{ pendingTotpEnrollmentId || '-' }}</div>
        <div class="record-meta">手动输入密钥：{{ pendingTotpManualEntryKey || '-' }}</div>
      </div>
      <BForm v-if="!activeTotpEnrollments.length" @submit.prevent>
        <BFormInput v-model="totpVerifyForm.code" placeholder="6 位验证码" class="mb-3" />
      </BForm>
    </template>

    <template v-else-if="method === 'email_code'">
      <div class="record-meta mb-3">邮箱地址来自基本信息页。这里只控制邮箱验证码是否启用。</div>
      <BForm @submit.prevent>
        <div class="mb-3">
          <div class="record-meta mb-2">当前邮箱：{{ currentUserRecord?.email || '未配置邮箱' }}</div>
          <BFormSelect v-model="mfaSettingForm.emailEnabled" :options="booleanSettingOptions" />
        </div>
      </BForm>
    </template>

    <template v-else-if="method === 'sms_code'">
      <div class="record-meta mb-3">手机号来自基本信息页。这里只控制手机验证码是否启用。</div>
      <BForm @submit.prevent>
        <div class="mb-3">
          <div class="record-meta mb-2">当前手机号：{{ currentUserRecord?.phoneNumber || '未配置手机号' }}</div>
          <BFormSelect v-model="mfaSettingForm.smsEnabled" :options="booleanSettingOptions" />
        </div>
      </BForm>
    </template>

    <template v-else-if="method === 'u2f'">
      <div class="record-meta mb-3">安全密钥的注册与删除已迁移到“登录设置 &gt; 密钥管理”。</div>
    </template>

    <template v-else-if="method === 'recovery_code'">
      <div class="record-meta mb-2">剩余有效码：{{ userDetail?.recoverySummary?.available ?? 0 }}</div>
      <div class="record-meta mb-3">上次生成时间：{{ formatDateTime(userDetail?.recoverySummary?.lastGeneratedAt) }}</div>
      <div class="record-meta mb-3">当前有效备用验证码：{{ recoveryCodeList.length ? `共 ${recoveryCodeList.length} 个` : '暂无' }}</div>
      <div v-if="recoveryCodeList.length" class="detail-card mb-3">
        <div class="record-meta mb-2">以下为当前有效的备用验证码，请妥善保管。</div>
        <div v-for="code in recoveryCodeList" :key="code" class="record-row">
          <code>{{ code }}</code>
        </div>
      </div>
    </template>
    <template #footer>
      <div class="d-flex justify-content-end gap-2 w-100">
        <BButton type="button" variant="outline-secondary" @click="emit('update:visible', false)">关闭</BButton>
        <BButton type="button" variant="primary" @click="emit('submit')">{{ currentActionLabel }}</BButton>
      </div>
    </template>
  </BModal>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { BButton, BForm, BFormInput, BFormSelect, BModal } from 'bootstrap-vue-next'
import { useConsoleStore } from '../stores/console'

const props = defineProps<{
  visible: boolean
  method: string
  activeTotpEnrollments: any[]
  totpSetup: any
  totpQrCodeDataUrl: string
  pendingTotpEnrollmentId: string
  pendingTotpManualEntryKey: string
  totpVerifyForm: { code: string }
  currentUserRecord: any
  mfaSettingForm: { emailEnabled: string; smsEnabled: string }
  u2fSecureKeys: any[]
  userDetail: any
  recoveryCodeList: string[]
}>()

const console = useConsoleStore()
const formatDateTime = console.formatDateTime

const booleanSettingOptions = [
  { value: 'active', text: '开启' },
  { value: 'disabled', text: '关闭' }
]

const currentTitle = computed(() => {
  if (props.method === 'totp') return '身份验证器（TOTP）'
  if (props.method === 'email_code') return '邮箱验证码'
  if (props.method === 'sms_code') return '手机验证码'
  if (props.method === 'u2f') return '安全密钥'
  if (props.method === 'recovery_code') return '备用验证码'
  return '多因素验证设置'
})

const currentActionLabel = computed(() => {
  if (props.method === 'totp') return '激活身份验证器'
  if (props.method === 'recovery_code') return '生成备用验证码'
  return '保存设置'
})

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'delete-totp-enrollments': []
  'delete-secure-key': [id: string]
  submit: []
}>()
</script>
