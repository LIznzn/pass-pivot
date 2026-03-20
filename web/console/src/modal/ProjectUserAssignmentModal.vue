<template>
  <BModal :model-value="visible" title="添加项目用户" size="xl" centered @update:model-value="emit('update:visible', $event)">
    <div class="d-flex justify-content-between align-items-center gap-2 mb-3">
      <div class="record-meta mb-0">支持多选和反选。保存后会同步项目 ACL。</div>
      <div class="d-flex gap-2">
        <BButton size="sm" variant="outline-secondary" @click="selectAll">全选</BButton>
        <BButton size="sm" variant="outline-secondary" @click="invertSelection">反选</BButton>
        <BButton size="sm" variant="outline-secondary" @click="clearSelection">清空</BButton>
      </div>
    </div>
    <div class="table-responsive project-user-assignment-wrap">
      <table class="table align-middle console-list-table project-user-assignment-table mb-0">
        <thead>
          <tr>
            <th class="console-list-check-col">选择</th>
            <th>用户 ID</th>
            <th>用户名</th>
            <th>名称</th>
            <th>邮箱 / 手机号</th>
            <th>状态</th>
            <th>角色</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="user in users" :key="user.id">
            <td class="console-list-check-col">
              <input
                class="form-check-input console-list-checkbox"
                type="checkbox"
                :checked="draftUserIds.includes(user.id)"
                @change="toggleUser(user.id, ($event.target as HTMLInputElement).checked)"
              />
            </td>
            <td class="console-list-id">{{ user.id }}</td>
            <td>{{ user.username || '-' }}</td>
            <td>{{ user.name || '-' }}</td>
            <td>{{ user.email || user.phoneNumber || '-' }}</td>
            <td>
              <span class="badge rounded-pill" :class="user.status === 'disabled' ? 'text-bg-secondary' : 'text-bg-success'">
                {{ user.status === 'disabled' ? '停用' : '启用' }}
              </span>
            </td>
            <td>{{ formatRoleLabels(user.roles) }}</td>
          </tr>
          <tr v-if="users.length === 0">
            <td colspan="7" class="text-center text-secondary py-4">当前组织下还没有用户。</td>
          </tr>
        </tbody>
      </table>
    </div>
    <template #footer>
      <div class="d-flex justify-content-end gap-2 w-100">
        <BButton type="button" variant="outline-secondary" @click="emit('update:visible', false)">取消</BButton>
        <BButton type="button" variant="primary" @click="emit('confirm', draftUserIds)">确认添加</BButton>
      </div>
    </template>
  </BModal>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { BButton, BModal } from 'bootstrap-vue-next'
import { useUserStore } from '../stores/user'

const props = defineProps<{
  visible: boolean
  selectedUserIds: string[]
  formatRoleLabels: (roles?: string[]) => string
}>()

const userStore = useUserStore()
const users = computed(() => userStore.users)
const draftUserIds = ref<string[]>([])

watch(
  () => [props.visible, props.selectedUserIds],
  () => {
    if (!props.visible) {
      return
    }
    draftUserIds.value = [...props.selectedUserIds]
  },
  { immediate: true, deep: true }
)

const emit = defineEmits<{
  'update:visible': [value: boolean]
  confirm: [userIds: string[]]
}>()

function toggleUser(userId: string, checked: boolean) {
  if (checked) {
    if (!draftUserIds.value.includes(userId)) {
      draftUserIds.value = [...draftUserIds.value, userId]
    }
    return
  }
  draftUserIds.value = draftUserIds.value.filter((item) => item !== userId)
}

function selectAll() {
  draftUserIds.value = users.value.map((item: any) => item.id)
}

function invertSelection() {
  const selectedSet = new Set(draftUserIds.value)
  draftUserIds.value = users.value
    .map((item: any) => item.id)
    .filter((id: string) => !selectedSet.has(id))
}

function clearSelection() {
  draftUserIds.value = []
}
</script>
