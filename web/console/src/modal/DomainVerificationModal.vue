<template>
  <BModal :model-value="visible" title="验证域名所有权" centered @update:model-value="emit('update:visible', $event)">
    <div class="d-grid gap-3">
      <div>
        <label class="form-label">域名</label>
        <BFormInput :model-value="host" disabled />
      </div>
      <div>
        <label class="form-label">验证方式</label>
        <BFormSelect v-model="draftMethod" :options="methodOptions" />
      </div>
      <div v-if="!challenge.token" class="alert alert-secondary mb-0">
        先点击“生成验证信息”，再按提示配置域名验证。
      </div>
      <div v-if="challenge.token" class="detail-card">
        <div class="record-meta mb-2">验证信息</div>
        <div v-if="draftMethod === 'http_file'" class="record-meta">
          在 `{{ challenge.fileUrl || `https://${host}/.well-known/ppvt-domain-verification.txt` }}` 提供以下内容：
        </div>
        <div v-else class="record-meta">
          为 `{{ host }}` 配置 TXT 记录 `{{ challenge.txtRecordName || `_ppvt-domain-verification.${host}` }}`，值如下：
        </div>
        <div class="mt-2"><code>{{ challenge.fileContent || challenge.txtRecordValue || challenge.token }}</code></div>
      </div>
      <div v-if="verified" class="alert alert-success mb-0">该域名已验证。</div>
    </div>
    <template #footer>
      <div class="d-flex justify-content-end gap-2 w-100">
        <BButton type="button" variant="outline-secondary" @click="emit('update:visible', false)">关闭</BButton>
        <BButton type="button" variant="outline-primary" @click="emit('prepare', draftMethod)">生成验证信息</BButton>
        <BButton type="button" variant="success" :disabled="!challenge.token || verifyCooldown > 0" @click="handleVerify">
          {{ verifyCooldown > 0 ? `${verifyCooldown}s 后可重试` : '执行验证' }}
        </BButton>
      </div>
    </template>
  </BModal>
</template>

<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import { BButton, BFormInput, BFormSelect, BModal } from 'bootstrap-vue-next'

const props = defineProps<{
  visible: boolean
  host: string
  method: 'http_file' | 'dns_txt'
  verified: boolean
  challenge: {
    token?: string
    fileUrl?: string
    fileContent?: string
    txtRecordName?: string
    txtRecordValue?: string
  }
}>()

const methodOptions = [
  { value: 'http_file', text: '文件验证' },
  { value: 'dns_txt', text: 'TXT 记录验证' }
] as const

const draftMethod = ref<'http_file' | 'dns_txt'>('http_file')
const verifyCooldown = ref(0)
let verifyCooldownTimer: number | null = null

watch(
  () => [props.visible, props.method] as const,
  () => {
    if (!props.visible) {
      stopVerifyCooldown()
      return
    }
    draftMethod.value = props.method
  },
  { immediate: true }
)

function handleVerify() {
  if (!props.challenge.token || verifyCooldown.value > 0) {
    return
  }
  startVerifyCooldown()
  emit('verify')
}

function startVerifyCooldown() {
  stopVerifyCooldown()
  verifyCooldown.value = 30
  verifyCooldownTimer = window.setInterval(() => {
    if (verifyCooldown.value <= 1) {
      stopVerifyCooldown()
      return
    }
    verifyCooldown.value -= 1
  }, 1000)
}

function stopVerifyCooldown() {
  if (verifyCooldownTimer !== null) {
    window.clearInterval(verifyCooldownTimer)
    verifyCooldownTimer = null
  }
  verifyCooldown.value = 0
}

onBeforeUnmount(() => {
  stopVerifyCooldown()
})

const emit = defineEmits<{
  'update:visible': [value: boolean]
  prepare: [method: 'http_file' | 'dns_txt']
  verify: []
}>()
</script>
