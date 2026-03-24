<template>
  <div class="portal-center">
    <header class="portal-topbar border-bottom bg-white">
      <div class="container-fluid">
        <div class="portal-topbar-main">
          <div>
            <div class="portal-title">用户中心</div>
            <div class="portal-subtitle">维护个人资料、安全设置、身份绑定与设备。</div>
          </div>
          <BButton variant="outline-secondary" size="sm" @click="authStore.startLogoutFlow">退出登录</BButton>
        </div>
      </div>
    </header>

    <main class="container-fluid py-4">
      <div class="row g-4">
        <aside class="col-lg-3">
          <div class="console-module-sidebar">
            <button v-for="section in portalSections" :key="section.id" type="button" class="console-module-sidebar-link" @click="scrollTo(section.id)">
              {{ section.label }}
            </button>
          </div>
        </aside>

        <section class="col-lg-9">
          <div class="d-grid gap-4 portal-sections">
            <ProfileSection />
            <LoginSection />
            <BindingSection />
            <MfaSection />
            <DeviceSection />
          </div>
        </section>
      </div>
    </main>

    <MfaModal />
    <SecureKeyModal />
  </div>
</template>

<script setup lang="ts">
import { nextTick, onMounted } from 'vue'
import { BButton } from 'bootstrap-vue-next'
import BindingSection from '@/components/BindingSection.vue'
import DeviceSection from '@/components/DeviceSection.vue'
import LoginSection from '@/components/LoginSection.vue'
import SecureKeyModal from '@/modal/SecureKeyModal.vue'
import MfaModal from '@/modal/MfaModal.vue'
import MfaSection from '@/components/MfaSection.vue'
import ProfileSection from '@/components/ProfileSection.vue'
import { usePortalAuthStore } from '@/stores/auth'
import { usePortalStore } from '@/stores/portal'
import { portalSections } from '@/utils/portal'

const authStore = usePortalAuthStore()
const portalStore = usePortalStore()

async function scrollTo(id: string) {
  await nextTick()
  document.getElementById(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

onMounted(() => {
  authStore.syncSession()
  if (authStore.isAuthenticated) {
    void portalStore.initialize()
    return
  }
  portalStore.reset()
})
</script>

<style scoped>
.portal-center {
  min-height: 100vh;
  background: #f8f9fa;
}

.portal-topbar {
  position: sticky;
  top: 0;
  z-index: 1000;
}

.portal-topbar-main {
  min-height: 4.25rem;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.portal-title {
  font-size: 1.15rem;
  font-weight: 700;
}

.portal-subtitle {
  color: #6c757d;
  font-size: 0.9rem;
}

.portal-sections {
  padding-bottom: 8rem;
}

:deep(.detail-card),
:deep(.record-card) {
  background: #fff;
  border: 1px solid #dee2e6;
  border-radius: 0.65rem;
  padding: 0.9rem 1rem;
}

:deep(.login-card-title) {
  font-size: 0.98rem;
  font-weight: 600;
  margin-bottom: 0.75rem;
}

:deep(.login-toggle-list) {
  display: grid;
  gap: 0.25rem;
}

:deep(.login-setting-row) {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.75rem 0;
  border-top: 1px solid #edf0f2;
}

:deep(.login-setting-name) {
  font-size: 1rem;
  font-weight: 500;
}

:deep(.record-meta) {
  color: #6c757d;
  font-size: 0.9rem;
}

:deep(.record-head) {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1rem;
  margin-bottom: 0.5rem;
}

:deep(.record-actions) {
  display: flex;
  gap: 0.5rem;
  margin-top: 0.75rem;
}

:deep(.mfa-summary-row) {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

:deep(.record-list) {
  display: grid;
  gap: 0.75rem;
}

:deep(.portal-code-grid) {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(8rem, 1fr));
  gap: 0.5rem;
}

:deep(.portal-code-grid code) {
  display: block;
  background: #f8f9fa;
  border: 1px solid #dee2e6;
  border-radius: 0.5rem;
  padding: 0.5rem 0.75rem;
}

:deep(.small-break) {
  word-break: break-all;
}

:deep(.info-card) {
  background: #fff;
  border: 1px solid #dee2e6;
  border-radius: 0.75rem;
  padding: 1.25rem;
  box-shadow: 0 0.125rem 0.25rem rgba(0, 0, 0, 0.04);
}

:deep(.section-title) {
  font-size: 1.05rem;
  font-weight: 700;
  margin-bottom: 1rem;
}
</style>
