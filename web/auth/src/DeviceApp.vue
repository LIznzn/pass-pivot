<template>
  <div class="device-shell">
    <main class="device-main">
      <section class="device-card">
        <header class="device-header">
          <div class="device-badge">P</div>
          <div class="device-heading">
            <p class="device-eyebrow">Pass Pivot</p>
            <h1>Device Verification</h1>
            <p class="device-subtitle">
              <span v-if="applicationName">{{ applicationName }}</span>
              <span v-if="organizationName"> · {{ organizationName }}</span>
            </p>
          </div>
        </header>

        <div class="device-code-block">
          <span class="device-code-label">User code</span>
          <code>{{ bootstrap.userCode || 'N/A' }}</code>
        </div>

        <p v-if="bootstrap.error" class="device-alert device-alert-error">
          {{ bootstrap.error }}
        </p>

        <template v-if="bootstrap.status === 'done'">
          <p class="device-alert" :class="bootstrap.denied ? 'device-alert-muted' : 'device-alert-success'">
            {{ bootstrap.denied ? 'The request has been denied.' : 'The request has been approved.' }}
          </p>
          <p v-if="currentUserLabel" class="device-current-user">
            Signed in as <strong>{{ currentUserLabel }}</strong>
          </p>
        </template>

        <template v-else-if="currentUser">
          <div class="device-user-panel">
            <span class="device-user-label">Signed in as</span>
            <strong>{{ currentUserLabel }}</strong>
          </div>
          <form :action="bootstrap.confirmAction" method="post" class="device-form">
            <input type="hidden" name="user_code" :value="bootstrap.userCode" />
            <button type="submit" class="device-button device-button-primary">Approve</button>
          </form>
          <form :action="bootstrap.confirmAction" method="post" class="device-form">
            <input type="hidden" name="user_code" :value="bootstrap.userCode" />
            <input type="hidden" name="deny" value="true" />
            <button type="submit" class="device-button device-button-secondary">Deny</button>
          </form>
        </template>

        <form v-else :action="bootstrap.loginAction" method="post" class="device-form device-form-stack">
          <input type="hidden" name="user_code" :value="bootstrap.userCode" />
          <label class="device-field">
            <span>Email or username</span>
            <input name="identifier" autocomplete="username" required />
          </label>
          <label class="device-field">
            <span>Password</span>
            <input name="secret" type="password" autocomplete="current-password" required />
          </label>
          <button type="submit" class="device-button device-button-primary">Sign in</button>
        </form>
      </section>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'

const bootstrap = window.__PPVT_DEVICE_BOOTSTRAP__

if (!bootstrap) {
  throw new Error('missing device bootstrap payload')
}

const applicationName = computed(() => String(bootstrap.applicationName || '').trim())
const organizationName = computed(() => String(bootstrap.organizationName || '').trim())
const currentUser = computed(() => bootstrap.currentUser)
const currentUserLabel = computed(() =>
  String(
    bootstrap.currentUser?.email ||
    bootstrap.currentUser?.name ||
    bootstrap.currentUser?.username ||
    bootstrap.currentUser?.phoneNumber ||
    ''
  ).trim()
)

watch(
  () => bootstrap.title,
  (value) => {
    document.title = `Pass Pivot · ${value || 'Device Verification'}`
  },
  { immediate: true }
)
</script>

<style scoped>
.device-shell {
  min-height: 100vh;
  background:
    radial-gradient(circle at top, rgba(217, 119, 6, 0.18), transparent 38%),
    linear-gradient(180deg, #f7f1e4 0%, #efe5d0 52%, #e7dac1 100%);
  color: #2f2418;
}

.device-main {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32px 18px;
}

.device-card {
  width: min(100%, 560px);
  padding: 32px;
  border-radius: 28px;
  background: rgba(255, 251, 245, 0.92);
  border: 1px solid rgba(111, 78, 55, 0.12);
  box-shadow: 0 24px 72px rgba(95, 63, 36, 0.12);
}

.device-header {
  display: flex;
  align-items: center;
  gap: 18px;
  margin-bottom: 24px;
}

.device-badge {
  width: 52px;
  height: 52px;
  border-radius: 18px;
  display: grid;
  place-items: center;
  background: linear-gradient(135deg, #9a3412, #d97706);
  color: #fff7ed;
  font-size: 24px;
  font-weight: 700;
}

.device-heading h1 {
  margin: 0;
  font-size: 30px;
}

.device-eyebrow,
.device-subtitle,
.device-code-label,
.device-user-label {
  margin: 0;
  color: #7c5a3c;
}

.device-code-block {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 18px 20px;
  margin-bottom: 20px;
  border-radius: 20px;
  background: #fff7ed;
  border: 1px solid rgba(194, 120, 52, 0.18);
}

.device-code-block code {
  font-size: 28px;
  font-weight: 700;
  letter-spacing: 0.12em;
  color: #9a3412;
}

.device-form {
  margin-top: 14px;
}

.device-form-stack {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.device-field {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.device-field input {
  width: 100%;
  padding: 12px 14px;
  border-radius: 14px;
  border: 1px solid rgba(122, 90, 60, 0.18);
  background: rgba(255, 255, 255, 0.9);
}

.device-button {
  width: 100%;
  padding: 13px 16px;
  border: none;
  border-radius: 999px;
  font-weight: 600;
}

.device-button-primary {
  background: linear-gradient(135deg, #b45309, #ea580c);
  color: #fff7ed;
}

.device-button-secondary {
  background: #f3e7d2;
  color: #6c4a2f;
}

.device-alert,
.device-user-panel {
  margin: 0 0 18px;
  padding: 14px 16px;
  border-radius: 16px;
}

.device-alert-success {
  background: rgba(22, 163, 74, 0.12);
  color: #166534;
}

.device-alert-muted,
.device-user-panel {
  background: rgba(140, 102, 68, 0.1);
  color: #5b4632;
}

.device-alert-error {
  background: rgba(220, 38, 38, 0.1);
  color: #991b1b;
}

.device-current-user {
  margin: 0;
}

@media (max-width: 640px) {
  .device-card {
    padding: 24px 20px;
  }

  .device-header {
    align-items: flex-start;
  }

  .device-heading h1 {
    font-size: 25px;
  }

  .device-code-block code {
    font-size: 22px;
  }
}
</style>
