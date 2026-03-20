<template>
  <BModal :model-value="visible" title="创建组织" centered @update:model-value="emit('update:visible', $event)" @hidden="emit('hidden')">
    <BForm @submit.prevent="emit('create')">
      <label class="form-label">组织标识</label>
      <BFormInput v-model="form.name" placeholder="仅支持字母、数字和 -" autofocus class="mb-3" />
      <div class="record-meta mb-3">组织标识会作为系统内的唯一名称使用，建议采用 `internal-team` 这样的格式。</div>
      <label class="form-label">组织简介</label>
      <BFormInput v-model="form.description" placeholder="请输入组织简介" />
    </BForm>
    <template #footer>
      <div class="d-flex justify-content-end gap-2 w-100">
        <BButton type="button" variant="outline-secondary" @click="emit('update:visible', false)">取消</BButton>
        <BButton type="button" variant="primary" @click="emit('create')">创建组织</BButton>
      </div>
    </template>
  </BModal>
</template>

<script setup lang="ts">
import { BButton, BForm, BFormInput, BModal } from 'bootstrap-vue-next'

defineProps<{
  visible: boolean
  form: { name: string; description: string }
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  hidden: []
  create: []
}>()
</script>
