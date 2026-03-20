<template>
  <BModal :model-value="visible" title="创建项目" centered @update:model-value="emit('update:visible', $event)" @hidden="emit('hidden')">
    <BForm @submit.prevent="emit('submit')">
      <label class="form-label">项目名称</label>
      <BFormInput v-model="projectForm.name" placeholder="project name" autofocus class="mb-3" />
    </BForm>
    <template #footer>
      <div class="d-flex justify-content-end gap-2 w-100">
        <BButton type="button" variant="outline-secondary" @click="emit('update:visible', false)">取消</BButton>
        <BButton type="button" variant="primary" @click="emit('submit')">创建项目</BButton>
      </div>
    </template>
  </BModal>
</template>

<script setup lang="ts">
import { BButton, BForm, BFormInput, BModal } from 'bootstrap-vue-next'

defineProps<{
  visible: boolean
  projectForm: {
    name: string
  }
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  hidden: []
  submit: []
}>()
</script>
