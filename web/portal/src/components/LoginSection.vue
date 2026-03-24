<template>
  <div id="profile-login" class="info-card">
    <div class="section-title">登录设置</div>
    <div class="row g-3">
      <div class="col-lg-6">
        <div class="detail-card h-100">
          <div class="login-card-title">登录方式</div>
          <div class="login-toggle-list">
            <div class="login-setting-row">
              <div>
                <div class="login-setting-name">密码登录</div>
                <div class="record-meta">{{ portalStore.detail?.passwordCredential ? '已启用' : '未配置' }}</div>
              </div>
              <span class="record-meta">{{ portalStore.detail?.passwordCredential ? '当前账号已设置密码' : '当前账号尚未设置密码' }}</span>
            </div>
            <div class="login-setting-row">
              <div>
                <div class="login-setting-name">通行密钥登录</div>
                <div class="record-meta">{{ portalStore.webauthnLoginEnabled ? '已启用' : '未启用' }}</div>
              </div>
              <BButton size="sm" :variant="portalStore.webauthnLoginEnabled ? 'outline-danger' : 'outline-primary'" @click="portalStore.toggleWebAuthnLoginAction(!portalStore.webauthnLoginEnabled)">
                {{ portalStore.webauthnLoginEnabled ? '关闭' : '开启' }}
              </BButton>
            </div>
          </div>
        </div>
      </div>
      <div class="col-lg-6">
        <div class="detail-card h-100">
          <div class="login-card-title">密码修改</div>
          <BForm @submit.prevent="portalStore.savePasswordAction">
            <BFormInput v-model="portalStore.passwordForm.currentPassword" type="password" placeholder="当前密码" class="mb-2" />
            <BFormInput v-model="portalStore.passwordForm.newPassword" type="password" placeholder="新密码" class="mb-3" />
            <BButton type="submit" variant="outline-primary" size="sm">更新密码</BButton>
          </BForm>
        </div>
      </div>
      <div class="col-12">
        <div class="detail-card">
          <div class="d-flex justify-content-between align-items-center gap-3 flex-wrap">
            <div>
              <div class="login-card-title mb-1">密钥管理</div>
              <div class="record-meta">{{ portalStore.allSecureKeys.length ? `当前账号已绑定 ${portalStore.allSecureKeys.length} 把密钥，可查看每把密钥支持的能力。` : '当前账号还没有绑定密钥。' }}</div>
            </div>
            <BButton size="sm" variant="outline-primary" @click="portalStore.openKeyModal">管理密钥</BButton>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { BButton, BForm, BFormInput } from 'bootstrap-vue-next'
import { usePortalStore } from '@/stores/portal'

const portalStore = usePortalStore()
</script>
