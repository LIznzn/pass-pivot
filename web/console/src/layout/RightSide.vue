<template>
  <aside class="console-module-aside">
    <div class="info-card">
      <div class="section-title">最近变更</div>
      <div class="record-list">
        <div v-for="item in items" :key="item.id" class="record-row">
          <div>
            <strong>{{ item.eventType }}</strong>
            <div class="record-meta">{{ item.result }}</div>
          </div>
          <code>{{ formatDateTime(item.createdAt) }}</code>
        </div>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
defineProps<{
  items: any[]
  formatDateTime: (value?: string) => string
}>()
</script>
