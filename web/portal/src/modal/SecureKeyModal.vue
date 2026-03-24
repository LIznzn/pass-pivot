<template>
  <BModal v-model="portalStore.showKeyModal" title="密钥管理" size="lg" centered>
    <div class="d-flex gap-2 flex-wrap mb-3">
      <BButton size="sm" variant="outline-primary" @click="portalStore.registerSecureKeyAction('webauthn')">注册为通行密钥（WebAuthn）</BButton>
      <BButton size="sm" variant="outline-secondary" @click="portalStore.registerSecureKeyAction('u2f')">注册为安全密钥（U2F）</BButton>
    </div>
    <div v-if="!portalStore.allSecureKeys.length" class="record-meta">当前没有已注册的密钥。</div>
    <div v-for="secureKey in portalStore.allSecureKeys" :key="secureKey.id" class="record-card mb-2">
      <div class="record-head">
        <div>
          <BFormInput
            v-model="portalStore.keyNameDrafts[secureKey.id]"
            placeholder="密钥名称"
            class="mb-2"
          />
          <div class="record-meta">{{ keyCapabilityLabel(secureKey) }}</div>
        </div>
        <div class="d-flex align-items-center gap-2">
          <BButton size="sm" variant="outline-primary" @click="portalStore.updateSecureKeyAction(secureKey.id, portalStore.keyNameDrafts[secureKey.id] || '')">保存名称</BButton>
          <BButton size="sm" variant="outline-danger" @click="portalStore.deleteSecureKeyAction(secureKey.id)">删除密钥</BButton>
        </div>
      </div>
      <div class="record-meta small-break">{{ secureKey.publicKeyId }}</div>
      <div class="record-meta mt-2">注册时间：{{ formatDateTime(secureKey.createdAt) }}</div>
    </div>
    <template #footer>
      <BButton variant="outline-secondary" @click="portalStore.closeKeyModal">关闭</BButton>
    </template>
  </BModal>
</template>

<script setup lang="ts">
import { BButton, BFormInput, BModal } from 'bootstrap-vue-next'
import { usePortalStore } from '@/stores/portal'
import { formatDateTime, keyCapabilityLabel } from '@/utils/portal'

const portalStore = usePortalStore()
</script>
