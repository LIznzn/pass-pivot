<template>
  <BModal :model-value="visible" :title="title" centered @update:model-value="emit('update:visible', $event)">
    <div class="record-meta mb-3">请立即保存以下私钥。系统不会再次展示该私钥。</div>
    <textarea class="form-control" rows="12" :value="snapshot" readonly />
  </BModal>
</template>

<script setup lang="ts">
import { BModal } from 'bootstrap-vue-next'

defineProps<{
  visible: boolean
  title: string
  snapshot: string
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
}>()
</script>
