<template>
  <div id="profile-device" class="info-card">
    <div class="section-title">会话管理</div>
    <div v-if="!portalStore.deviceRows.length" class="detail-card">
      <div class="record-meta">当前没有设备记录。</div>
    </div>
    <div v-for="device in portalStore.deviceRows" :key="device.id" class="record-card mb-2">
      <div class="record-head">
        <strong>{{ device.label }}</strong>
        <div class="d-flex align-items-center gap-2">
          <span v-if="device.trusted" class="badge text-bg-primary">可信</span>
          <span class="badge" :class="device.online ? 'text-bg-success' : 'text-bg-secondary'">{{ device.online ? '在线' : '离线' }}</span>
        </div>
      </div>
      <div class="record-meta">上次登录 IP：{{ formatIPLine(device.ipAddress, device.ipLocation) }}</div>
      <div class="record-meta">上次登录时间：{{ formatDateTime(device.lastLoginAt) }}</div>
      <div class="record-meta">初次登录日期：{{ formatDateTime(device.firstLoginAt) }}</div>
      <div v-if="device.fingerprint" class="record-meta small-break">设备指纹：{{ device.fingerprint }}</div>
      <div v-if="device.trusted" class="record-actions">
        <BButton size="sm" variant="outline-danger" @click="portalStore.untrustDeviceAction(device.id)">取消可信</BButton>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { BButton } from 'bootstrap-vue-next'
import { usePortalStore } from '@/stores/portal'
import { formatDateTime, formatIPLine } from '@/utils/portal'

const portalStore = usePortalStore()
</script>
