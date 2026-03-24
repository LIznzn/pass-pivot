<template>
  <div id="profile-mfa" class="info-card">
    <div class="section-title">多因素验证</div>
    <div class="record-card mb-3">
      <div class="mfa-summary-row">
        <div>
          <strong>启用多因素验证</strong>
          <div class="record-meta">{{ portalStore.mfaSummaryText }}</div>
        </div>
        <div class="d-flex gap-2">
          <BButton v-if="portalStore.mfaEnabled" size="sm" variant="outline-primary" @click="portalStore.prepareMFAModal('recovery_code')">查看备用验证码</BButton>
          <BButton size="sm" :variant="portalStore.mfaEnabled ? 'outline-danger' : 'outline-primary'" @click="portalStore.toggleMFAEnabledAction(!portalStore.mfaEnabled)">
            {{ portalStore.mfaEnabled ? '关闭' : '开启' }}
          </BButton>
        </div>
      </div>
    </div>
    <div v-if="portalStore.mfaEnabled" class="record-list">
      <div v-for="item in portalStore.mfaRows" :key="item.id" class="record-card">
        <div class="mfa-summary-row">
          <div>
            <strong>{{ item.label }}</strong>
            <div class="record-meta">{{ item.summary }}</div>
          </div>
          <div class="d-flex gap-2">
            <BButton
              v-if="item.id === 'email_code' || item.id === 'sms_code' || item.id === 'u2f'"
              size="sm"
              :variant="item.enabled ? 'outline-danger' : 'outline-primary'"
              @click="portalStore.toggleSimpleMFAAction(item.id, !item.enabled)"
            >
              {{ item.enabled ? '关闭' : '开启' }}
            </BButton>
            <BButton v-else size="sm" variant="outline-primary" @click="portalStore.prepareMFAModal(item.id as 'totp' | 'u2f' | 'recovery_code')">
              配置
            </BButton>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { BButton } from 'bootstrap-vue-next'
import { usePortalStore } from '@/stores/portal'

const portalStore = usePortalStore()
</script>
