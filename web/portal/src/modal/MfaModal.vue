<template>
  <BModal v-model="portalStore.showMFAModal" :title="portalStore.mfaModalTitle" centered>
    <template v-if="portalStore.mfaModalType === 'totp'">
      <div v-if="portalStore.activeTotpEnrollment" class="record-meta mb-3">当前已配置身份验证器，可重新生成或关闭。</div>
      <div v-else class="record-meta mb-3">当前没有已激活的身份验证器。</div>
      <div v-if="portalStore.totpQRCode" class="text-center mb-3">
        <img :src="portalStore.totpQRCode" alt="身份验证器二维码" class="img-fluid border rounded bg-white p-2" />
      </div>
      <div v-if="portalStore.totpEnrollment.manualEntryKey" class="record-meta mb-3">手动输入密钥：{{ portalStore.totpEnrollment.manualEntryKey }}</div>
      <BForm @submit.prevent="portalStore.verifyTotpAction">
        <BFormInput v-model="portalStore.totpCode" placeholder="输入 6 位验证码" class="mb-3" />
        <div class="d-flex gap-2">
          <BButton type="button" variant="outline-secondary" @click="portalStore.generateTotpAction">生成身份验证器配置</BButton>
          <BButton type="submit" variant="primary">确认并启用</BButton>
          <BButton v-if="portalStore.activeTotpEnrollment" type="button" variant="outline-danger" @click="portalStore.disableTotpAction">关闭</BButton>
        </div>
      </BForm>
    </template>

    <template v-else-if="portalStore.mfaModalType === 'u2f'">
      <div class="record-meta mb-3">安全密钥的注册与删除已迁移到“登录设置 &gt; 密钥管理”。</div>
    </template>

    <template v-else-if="portalStore.mfaModalType === 'recovery_code'">
      <div class="record-meta mb-2">剩余有效码：{{ portalStore.detail?.recoverySummary?.available ?? 0 }}</div>
      <div class="record-meta mb-3">上次生成时间：{{ formatDateTime(portalStore.detail?.recoverySummary?.lastGeneratedAt) }}</div>
      <div class="record-meta mb-3">当前有效备用验证码：{{ portalStore.recoveryCodes.length ? `共 ${portalStore.recoveryCodes.length} 个` : '暂无' }}</div>
      <div v-if="portalStore.recoveryCodes.length" class="portal-code-grid mb-3">
        <code v-for="code in portalStore.recoveryCodes" :key="code">{{ code }}</code>
      </div>
      <BButton variant="primary" @click="portalStore.generateRecoveryCodesAction">重新生成备用验证码</BButton>
    </template>

    <template #footer>
      <BButton variant="outline-secondary" @click="portalStore.closeMFAModal">关闭</BButton>
    </template>
  </BModal>
</template>

<script setup lang="ts">
import { BButton, BForm, BFormInput, BModal } from 'bootstrap-vue-next'
import { usePortalStore } from '@/stores/portal'
import { formatDateTime } from '@/utils/portal'

const portalStore = usePortalStore()
</script>
