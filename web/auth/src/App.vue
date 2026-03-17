<template>
  <div class="oauth-ui-shell">
    <ToastHost />
    <div class="oauth-ui-panel">
      <div class="oauth-eyebrow">PPVT OAuth</div>
      <h1 class="oauth-title">{{ bootstrap.title }}</h1>

      <div v-if="bootstrap.error" class="alert alert-danger">{{ bootstrap.error }}</div>

      <div class="oauth-target-card">
        <div class="oauth-target-row">
          <span>组织</span>
          <strong>{{ bootstrap.target.organizationName || '-' }}</strong>
        </div>
        <div class="oauth-target-row">
          <span>项目</span>
          <strong>{{ bootstrap.target.projectName || '-' }}</strong>
        </div>
        <div class="oauth-target-row">
          <span>应用</span>
          <strong>{{ bootstrap.target.applicationName || '-' }}</strong>
        </div>
      </div>

      <form v-if="bootstrap.stage === 'login'" :action="bootstrap.loginAction" method="post" class="oauth-form-grid">
        <div class="text-secondary small">浏览器直接从 <code>/auth/authorize</code> 进入认证交互。</div>
        <input type="hidden" name="interaction" value="login" />
        <div>
          <label class="form-label">账号</label>
          <input name="identifier" class="form-control" autocomplete="username" placeholder="请输入邮箱 / 手机 / 用户名" />
        </div>
        <div>
          <label class="form-label">密码</label>
          <input name="secret" type="password" class="form-control" autocomplete="current-password" placeholder="请输入密码" />
        </div>
        <div class="d-flex flex-wrap gap-2 pt-2">
          <button type="submit" class="btn btn-primary">登录</button>
          <button
            v-if="supportsWebAuthnLogin"
            type="button"
            class="btn btn-outline-primary"
            @click="loginWithWebAuthn"
          >
            使用通行密钥登录
          </button>
        </div>
      </form>

      <div v-else-if="bootstrap.stage === 'confirmation'" class="oauth-form-grid">
        <div class="text-secondary small">二次确认先于 MFA，用于风险提示、协议变更确认和公告确认。</div>
        <div class="oauth-confirm-card">
          <ul class="mb-0">
            <li>新设备登录提示</li>
            <li>组织公告确认</li>
            <li>高风险行为继续确认</li>
          </ul>
        </div>
        <div class="d-flex flex-wrap gap-2 pt-2">
          <form :action="bootstrap.confirmAction" method="post">
            <input type="hidden" name="interaction" value="confirm" />
            <input type="hidden" name="accept" value="true" />
            <button type="submit" class="btn btn-primary">我已知晓并继续</button>
          </form>
          <form :action="bootstrap.confirmAction" method="post">
            <input type="hidden" name="interaction" value="confirm" />
            <input type="hidden" name="accept" value="false" />
            <button type="submit" class="btn btn-outline-danger">拒绝并终止</button>
          </form>
        </div>
      </div>

      <form v-else :action="bootstrap.mfaAction" method="post" class="oauth-form-grid">
        <div class="text-secondary small">完成多因素认证后，将继续原始 OAuth/OIDC 授权流程。</div>
        <input type="hidden" name="interaction" value="mfa" />
        <div>
          <label class="form-label">验证方式</label>
          <select v-model="selectedMethod" name="method" class="form-select">
            <option v-for="option in bootstrap.mfaOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
          </select>
        </div>
        <div>
          <label class="form-label">验证码</label>
          <input name="code" class="form-control" placeholder="请输入验证码" />
        </div>
        <div class="form-check">
          <input id="trustDevice" name="trustDevice" value="true" type="checkbox" class="form-check-input" checked />
          <label for="trustDevice" class="form-check-label">将当前浏览器设为可信设备</label>
        </div>
        <div class="d-flex flex-wrap gap-2 pt-2">
          <button type="submit" class="btn btn-primary">完成验证</button>
          <button
            v-if="supportsEmailChallenge"
            type="button"
            class="btn btn-outline-secondary"
            @click="sendEmailChallenge"
          >
            发送邮箱验证码
          </button>
          <button
            v-if="supportsSessionU2F"
            type="button"
            class="btn btn-outline-primary"
            @click="verifyMFAWithU2F"
          >
            使用安全密钥
          </button>
        </div>
        <div v-if="challengeFeedback" class="alert alert-light border mb-0">{{ challengeFeedback }}</div>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { normalizeRequestOptions, serializeCredential } from '@shared/api/webauthn'
import ToastHost from '@shared/components/ToastHost.vue'
import { useToast } from '@shared/composables/toast'

const toast = useToast()
const bootstrapPayload = window.__PPVT_OAUTH_BOOTSTRAP__

if (!bootstrapPayload) {
  throw new Error('missing oauth bootstrap payload')
}
const bootstrap = bootstrapPayload

const selectedMethod = ref(bootstrap.secondFactorMethod || 'totp')
const challengeFeedback = ref('')
const supportsWebAuthnLogin = Boolean(bootstrap.api.webauthnLoginBegin && bootstrap.api.webauthnLoginEnd)
const supportsSessionU2F = Boolean(bootstrap.api.sessionU2fBegin && bootstrap.api.sessionU2fFinish)
const supportsEmailChallenge = Boolean(bootstrap.api.mfaChallenge)

async function readJSON<T>(response: Response): Promise<T> {
  if (!response.ok) {
    throw new Error(await response.text())
  }
  return response.json() as Promise<T>
}

async function loginWithWebAuthn() {
  try {
    const identifierInput = document.querySelector<HTMLInputElement>('input[name="identifier"]')
    const identifier = identifierInput?.value.trim() ?? ''
    if (!identifier) {
      toast.error('请先输入账号，再使用通行密钥登录。')
      return
    }
    const begin = await readJSON<{ challengeId: string; options: any }>(
      await fetch(bootstrap.api.webauthnLoginBegin, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ identifier })
      })
    )
    const credential = await navigator.credentials.get({
      publicKey: normalizeRequestOptions(begin.options)
    })
    if (!credential) {
      return
    }
    await readJSON(
      await fetch(bootstrap.api.webauthnLoginEnd, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          challengeId: begin.challengeId,
          response: serializeCredential(credential as PublicKeyCredential),
          applicationId: bootstrap.applicationId
        })
      })
    )
    window.location.assign(bootstrap.authorizeReturnUrl)
  } catch (error) {
    toast.error(String(error))
  }
}

async function sendEmailChallenge() {
  try {
    const result = await readJSON<{ demoCode?: string }>(
      await fetch(bootstrap.api.mfaChallenge, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ method: 'email_code' })
      })
    )
    challengeFeedback.value = result.demoCode
      ? `验证码已生成，演示码：${result.demoCode}`
      : '验证码已发送到邮箱。'
  } catch (error) {
    toast.error(String(error))
  }
}

async function verifyMFAWithU2F() {
  try {
    const begin = await readJSON<{ challengeId: string; options: any }>(
      await fetch(bootstrap.api.sessionU2fBegin, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({})
      })
    )
    const credential = await navigator.credentials.get({
      publicKey: normalizeRequestOptions(begin.options)
    })
    if (!credential) {
      return
    }
    await readJSON(
      await fetch(bootstrap.api.sessionU2fFinish, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          challengeId: begin.challengeId,
          response: serializeCredential(credential as PublicKeyCredential),
          trustDevice: true
        })
      })
    )
    window.location.assign(bootstrap.authorizeReturnUrl)
  } catch (error) {
    toast.error(String(error))
  }
}
</script>

<style scoped>
.oauth-ui-shell {
  min-height: 100vh;
  display: grid;
  place-items: center;
  padding: 2rem 1rem;
  background: linear-gradient(180deg, #f8fafc 0%, #eef2f7 100%);
}

.oauth-ui-panel {
  width: min(34rem, 100%);
  background: #fff;
  border: 1px solid #d9dee7;
  border-radius: 1rem;
  padding: 2rem;
  box-shadow: 0 18px 48px rgba(15, 23, 42, 0.08);
}

.oauth-eyebrow {
  color: #0d6efd;
  font-size: 0.75rem;
  font-weight: 700;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.oauth-title {
  margin: 0.35rem 0 1rem;
  font-size: 1.8rem;
  font-weight: 700;
}

.oauth-target-card {
  display: grid;
  gap: 0.5rem;
  margin-bottom: 1.25rem;
  border: 1px solid #e5e7eb;
  border-radius: 0.75rem;
  padding: 0.875rem 1rem;
  background: #f8fafc;
}

.oauth-target-row {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
}

.oauth-target-row span {
  color: #6b7280;
}

.oauth-form-grid {
  display: grid;
  gap: 0.95rem;
}

.oauth-confirm-card {
  border: 1px solid #e5e7eb;
  border-radius: 0.75rem;
  padding: 1rem;
  background: #f8fafc;
}
</style>
