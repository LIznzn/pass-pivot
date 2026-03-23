<template>
  <BModal :model-value="visible" title="添加域名" centered @update:model-value="emit('update:visible', $event)" @hidden="emit('hidden')">
    <BForm @submit.prevent="emit('submit')">
      <label class="form-label">域名</label>
      <BFormInput v-model="form.host" placeholder="login.example.com" autofocus />
      <div class="record-meta mt-3">添加后状态默认为未验证，后续请通过“验证所有权”完成校验。</div>
    </BForm>
    <template #footer>
      <div class="d-flex justify-content-end gap-2 w-100">
        <BButton type="button" variant="outline-secondary" @click="emit('update:visible', false)">取消</BButton>
        <BButton type="button" variant="primary" @click="emit('submit')">保存</BButton>
      </div>
    </template>
  </BModal>
</template>

<script setup lang="ts">
import { BButton, BForm, BFormInput, BModal } from 'bootstrap-vue-next'

defineProps<{
  visible: boolean
  form: { host: string }
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  hidden: []
  submit: []
}>()
</script>
