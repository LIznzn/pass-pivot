<template>
  <div id="profile-basic" class="info-card">
    <div class="section-title">基本信息</div>
    <BForm @submit.prevent="portalStore.saveProfileAction">
      <div class="row g-3">
        <div class="col-md-6">
          <label class="form-label">姓名</label>
          <BFormInput v-model="portalStore.profile.name" />
        </div>
        <div class="col-md-6">
          <label class="form-label">用户名</label>
          <BFormInput v-model="portalStore.profile.username" />
        </div>
        <div class="col-md-6">
          <label class="form-label">邮箱</label>
          <BFormInput v-model="portalStore.profile.email" type="email" />
        </div>
        <div class="col-md-6">
          <label class="form-label">手机</label>
          <BFormInput v-model="portalStore.profile.phoneNumber" />
        </div>
      </div>
      <div class="d-flex justify-content-end mt-3">
        <BButton type="submit" variant="primary" size="sm">保存基本信息</BButton>
      </div>
    </BForm>
  </div>
</template>

<script setup lang="ts">
import { BButton, BForm, BFormInput } from 'bootstrap-vue-next'
import { usePortalStore } from '@/stores/portal'

const portalStore = usePortalStore()
</script>
