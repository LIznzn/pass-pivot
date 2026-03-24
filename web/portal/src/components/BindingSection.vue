<template>
  <div id="profile-binding" class="info-card">
    <div class="section-title">账号绑定</div>
    <div class="record-meta mb-3">绑定外部 OAuth / OIDC 身份的 Subject，用于后续识别已有账号。</div>
    <div class="row g-3">
      <div class="col-lg-7">
        <div v-if="!(portalStore.detail?.bindings?.length ?? 0)" class="detail-card">
          <div class="record-meta">当前没有第三方身份绑定。</div>
        </div>
        <div v-for="binding in portalStore.detail?.bindings || []" :key="binding.id" class="record-card mb-2">
          <div class="record-head">
            <strong>{{ binding.providerName || binding.externalIdpId }}</strong>
            <code>{{ binding.subject }}</code>
          </div>
          <div class="record-meta">Issuer：{{ binding.issuer }}</div>
          <div class="record-meta">绑定时间：{{ formatDateTime(binding.createdAt) }}</div>
          <div class="record-actions">
            <BButton size="sm" variant="outline-danger" @click="portalStore.deleteBindingAction(binding.id)">解绑</BButton>
          </div>
        </div>
      </div>
      <div class="col-lg-5">
        <BForm @submit.prevent="portalStore.createBindingAction">
          <label class="form-label">外部 IdP</label>
          <BFormSelect v-model="portalStore.bindingForm.externalIdpId" class="mb-2" @update:model-value="portalStore.syncBindingIssuer">
            <option value="">请选择</option>
            <option v-for="provider in portalStore.detail?.externalIdps || []" :key="provider.id" :value="provider.id">{{ provider.name }}</option>
          </BFormSelect>
          <label class="form-label">Issuer</label>
          <BFormInput v-model="portalStore.bindingForm.issuer" class="mb-2" />
          <label class="form-label">Subject</label>
          <BFormInput v-model="portalStore.bindingForm.subject" class="mb-3" />
          <BButton type="submit" variant="primary" size="sm">新增绑定</BButton>
        </BForm>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { BButton, BForm, BFormInput, BFormSelect } from 'bootstrap-vue-next'
import { usePortalStore } from '@/stores/portal'
import { formatDateTime } from '@/utils/portal'

const portalStore = usePortalStore()
</script>
