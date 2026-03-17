<template>
  <div class="admin-classic">
    <header class="admin-topbar border-bottom bg-white">
      <div class="container-fluid">
        <div class="admin-topbar-main">
          <div class="admin-topbar-left">
            <div class="admin-brand">
              <div class="admin-brand-badge">P</div>
            </div>
            <nav class="nav admin-nav-tabs">
              <button type="button" class="nav-link" :class="{ active: isTabActive('dashboard') }" @click="setTab('dashboard')">仪表盘</button>
              <button type="button" class="nav-link" :class="{ active: isTabActive('organization') }" @click="setTab('organization')">组织</button>
              <button type="button" class="nav-link" :class="{ active: isTabActive('project') }" @click="setTab('project')">项目</button>
              <button type="button" class="nav-link" :class="{ active: isTabActive('user') }" @click="setTab('user')">用户</button>
              <button type="button" class="nav-link" :class="{ active: isTabActive('role') }" @click="setTab('role')">角色</button>
              <button type="button" class="nav-link" :class="{ active: isTabActive('audit') }" @click="setTab('audit')">审计</button>
              <button type="button" class="nav-link" :class="{ active: isTabActive('setting') }" @click="setTab('setting')">设置</button>
            </nav>
          </div>
          <div class="admin-toolbar-group">
            <BDropdown right no-caret variant="link" toggle-class="organization-context-toggle">
              <template #button-content>
                <span class="organization-context">
                  <span class="organization-context-name">{{ currentOrganizationLabel }}</span>
                  <span class="organization-context-arrow">▾</span>
                </span>
              </template>
              <div class="organization-context-menu-label">切换组织</div>
              <BDropdownItem
                v-for="organization in organizations"
                :key="organization.id"
                class="organization-context-item"
                @click="handleOrganizationSwitch(organization.id)"
              >
                <span class="organization-context-item-row">
                  <span>{{ organization.name || organization.id }}</span>
                  <span v-if="organization.id === currentOrganizationId" class="organization-context-check">当前</span>
                </span>
              </BDropdownItem>
              <BDropdownDivider />
              <BDropdownItem class="organization-context-manage" @click="toggleManageOrganization">管理组织</BDropdownItem>
            </BDropdown>
            <BDropdown right no-caret variant="link" toggle-class="user-context-toggle">
              <template #button-content>
                <span class="user-context">
                  <span class="user-context-avatar">{{ currentUserInitials }}</span>
                </span>
              </template>
              <div class="user-context-menu-header">
                <span class="user-context-menu-avatar">{{ currentUserInitials }}</span>
                <div class="user-context-menu-copy">
                  <strong>{{ currentUserDisplayName }}</strong>
                  <span>{{ currentUserEmail }}</span>
                </div>
              </div>
              <BDropdownDivider />
              <BDropdownItem @click="() => goMy()">用户中心</BDropdownItem>
              <BDropdownItem @click="logout">退出登录</BDropdownItem>
            </BDropdown>
          </div>
        </div>
      </div>
    </header>

    <main class="admin-content container-fluid py-4">
      <div class="admin-page-header">
        <div>
          <h2 v-if="pageHeaderTitle" class="section-page-title">{{ pageHeaderTitle }}</h2>
          <p v-if="pageHeaderDescription" class="section-page-subtitle">{{ pageHeaderDescription }}</p>
        </div>
      </div>

      <ToastHost />

      <section v-if="currentView === 'my'" class="console-module-layout">
        <aside class="console-module-sidebar">
          <button v-for="item in currentMyPanels" :key="item.id" type="button" class="console-module-sidebar-link" @click="scrollToPanel(item.id)">{{ item.label }}</button>
        </aside>
        <div class="console-module-main">
          <div id="my-basic" class="info-card">
            <div class="section-title">基本信息</div>
            <BForm @submit.prevent="saveProfile">
              <div class="row g-3">
                <div class="col-md-6">
                  <label class="form-label">姓名</label>
                  <BFormInput v-model="profileForm.name" />
                </div>
                <div class="col-md-6">
                  <label class="form-label">用户名</label>
                  <BFormInput v-model="profileForm.username" />
                </div>
                <div class="col-md-6">
                  <label class="form-label">邮箱</label>
                  <BFormInput v-model="profileForm.email" type="email" />
                </div>
                <div class="col-md-6">
                  <label class="form-label">手机</label>
                  <div class="phone-input-group">
                    <BFormSelect v-model="profilePhoneInput.countryCode" :options="phoneCountryOptions" class="phone-country-select" />
                    <BFormInput v-model="profilePhoneInput.localNumber" class="phone-local-input" />
                  </div>
                </div>
              </div>
              <div class="d-flex justify-content-between align-items-center mt-3">
                <div class="record-meta mb-0">登录标识：{{ currentLoginUserLabel || '-' }}</div>
                <BButton type="submit" variant="primary">保存基本信息</BButton>
              </div>
            </BForm>
          </div>

          <div id="my-login-setting" class="info-card">
            <div class="section-title">登录设置</div>
            <div class="row g-3">
              <div class="col-lg-6">
                <div class="detail-card h-100">
                  <div class="record-meta mb-3">密码登录：{{ userDetail?.passwordCredential ? '已启用' : '未配置' }}</div>
                  <BForm @submit.prevent="savePassword">
                    <BFormInput v-model="passwordForm.currentPassword" type="password" placeholder="当前密码" class="mb-2" />
                    <BFormInput v-model="passwordForm.newPassword" type="password" placeholder="新密码" class="mb-3" />
                    <BButton type="submit" variant="outline-primary" size="sm">更新密码</BButton>
                  </BForm>
                </div>
              </div>
                <div class="col-lg-6">
                  <div class="detail-card h-100">
                    <div class="d-flex justify-content-between align-items-center mb-2">
                    <div class="record-meta mb-0">通行密钥数：{{ loginSecureKeys.length }} · {{ webauthnLoginEnabled ? '已启用登录' : '已关闭登录' }}</div>
                    <div class="d-flex gap-2">
                      <BButton size="sm" :variant="webauthnLoginEnabled ? 'outline-danger' : 'outline-secondary'" :disabled="!loginSecureKeys.length" @click="toggleWebAuthnLogin(!webauthnLoginEnabled)">
                        {{ webauthnLoginEnabled ? '关闭登录' : '启用登录' }}
                      </BButton>
                      <BButton size="sm" variant="outline-primary" @click="registerSecureKey('webauthn')">注册通行密钥</BButton>
                    </div>
                    </div>
                  <div v-if="!loginSecureKeys.length" class="record-meta">当前没有通行密钥，注册后才可启用通行密钥登录。</div>
                  <div v-for="secureKey in loginSecureKeys" :key="secureKey.id" class="record-row">
                    <div>
                      <strong>{{ secureKey.identifier || '通行密钥' }}</strong>
                      <div class="record-meta">{{ secureKey.publicKeyId }}</div>
                    </div>
                    <div class="d-flex align-items-center gap-2">
                      <code>{{ formatDateTime(secureKey.createdAt) }}</code>
                      <BButton size="sm" variant="outline-danger" @click="deleteSecureKey(secureKey.id)">删除</BButton>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div id="my-binding" class="info-card">
            <div class="section-title">账号绑定</div>
            <div class="record-meta mb-3">绑定外部 OAuth/OIDC 身份的 UUID/Subject，用于登录识别，不会自动创建用户。</div>
            <div class="row g-3">
              <div class="col-lg-7">
                <div v-if="!(userDetail?.bindings?.length ?? 0)" class="detail-card">
                  <div class="record-meta">当前没有第三方身份绑定。</div>
                </div>
                <div v-for="binding in userDetail?.bindings || []" :key="binding.id" class="record-card mb-2">
                  <div class="record-head">
                    <strong>{{ binding.providerName || binding.externalIdpId }}</strong>
                    <code>{{ binding.subject }}</code>
                  </div>
                  <div class="record-meta">Issuer：{{ binding.issuer }}</div>
                  <div class="record-meta">绑定时间：{{ formatDateTime(binding.createdAt) }}</div>
                  <div class="record-actions">
                    <BButton size="sm" variant="outline-danger" @click="deleteExternalBinding(binding.id)">解绑</BButton>
                  </div>
                </div>
              </div>
              <div class="col-lg-5">
                <BForm @submit.prevent="createExternalBinding">
                  <label class="form-label">外部 IdP</label>
                  <BFormSelect v-model="externalBindingForm.externalIdpId" class="mb-2" @update:model-value="syncExternalBindingIssuer">
                    <option v-for="item in userDetail?.externalIdps || []" :key="item.id" :value="item.id">{{ item.name }}</option>
                  </BFormSelect>
                  <label class="form-label">Issuer</label>
                  <BFormInput v-model="externalBindingForm.issuer" class="mb-2" />
                  <label class="form-label">Subject / UUID</label>
                  <BFormInput v-model="externalBindingForm.subject" class="mb-3" />
                  <BButton type="submit" variant="primary" size="sm">新增绑定</BButton>
                </BForm>
              </div>
            </div>
          </div>

          <div id="my-mfa" class="info-card">
            <div class="section-title">两步验证</div>
            <div class="record-list">
              <div v-for="item in userMFAMethodRows" :key="item.id" class="record-row">
                <div>
                  <strong>{{ item.label }}</strong>
                  <div class="record-meta">{{ item.summary }}</div>
                </div>
                <BButton
                  v-if="item.id === 'email_code' || item.id === 'sms_code'"
                  size="sm"
                  :variant="item.enabled ? 'outline-danger' : 'outline-primary'"
                  @click="handleInlineMFAMethodAction(item)"
                >
                  {{ item.enabled ? '关闭' : '开启' }}
                </BButton>
                <BButton v-else size="sm" variant="outline-primary" @click="openMFAModal(item.id)">{{ item.enabled ? '配置' : '开启' }}</BButton>
              </div>
            </div>
          </div>

          <div id="my-session" class="info-card">
            <div class="section-title">会话管理</div>
            <div v-if="!userDeviceList.length" class="detail-card">
              <div class="record-meta">当前没有设备记录。</div>
            </div>
            <div v-for="device in userDeviceList" :key="device.id" class="record-card mb-2">
              <div class="record-head">
                <strong>{{ device.label }}</strong>
                <div class="d-flex align-items-center gap-2">
                  <span class="badge text-bg-primary" v-if="device.trusted">可信</span>
                  <span class="badge text-bg-success" v-if="device.online">在线</span>
                  <span class="badge text-bg-secondary" v-else>离线</span>
                </div>
              </div>
              <div class="record-meta">上次登录 IP：{{ formatIPLine(device.ipAddress, device.ipLocation) }}</div>
              <div class="record-meta">上次登录时间：{{ formatDateTime(device.lastLoginAt) }}</div>
              <div class="record-meta">初次登录日期：{{ formatDateTime(device.firstLoginAt) }}</div>
              <div v-if="device.fingerprint" class="record-meta">设备指纹：{{ device.fingerprint }}</div>
              <div v-if="device.trusted" class="record-actions">
                <BButton size="sm" variant="outline-danger" @click="untrustDevice(device.id)">取消可信</BButton>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section v-else-if="currentView === 'organization-manage'" class="section-grid">
        <div class="info-card">
          <div class="section-title">Organization 列表</div>
          <BButton size="sm" variant="outline-primary" class="mb-3" @click="loadOrganizations">刷新</BButton>
          <div class="record-list">
            <div v-for="organization in organizations" :key="organization.id" class="record-card">
              <div class="record-head">
                <strong>{{ organization.name || organization.id }}</strong>
                <code>{{ organization.id }}</code>
              </div>
              <div class="record-meta">Projects: {{ organization.projects?.length ?? 0 }}</div>
              <div class="record-meta">Organization ID: {{ organization.id }}</div>
            </div>
          </div>
        </div>
        <div class="info-card">
          <div class="section-title">创建 Organization</div>
          <BForm @submit.prevent="createOrganization">
            <BFormInput v-model="organizationForm.name" placeholder="organization name" class="mb-2" />
            <BButton type="submit" variant="primary">创建 Organization</BButton>
          </BForm>
        </div>
        <div class="info-card">
          <div class="section-title">更新 Organization</div>
          <BForm @submit.prevent="updateOrganization">
            <BFormInput v-model="organizationUpdateForm.id" placeholder="organizationId" class="mb-2" />
            <BFormInput v-model="organizationUpdateForm.name" placeholder="organization name" class="mb-2" />
            <BButton type="submit" variant="outline-primary">更新 Organization</BButton>
          </BForm>
        </div>
      </section>

      <section v-else-if="currentView === 'project-create'" class="section-grid">
        <div class="info-card">
          <div class="section-title">创建项目</div>
          <BForm @submit.prevent="submitProjectCreatePage">
            <BFormInput v-model="projectForm.name" placeholder="project name" class="mb-3" />
            <div class="d-flex gap-2">
              <BButton type="submit" variant="primary">创建项目</BButton>
              <BButton type="button" variant="outline-secondary" @click="closeOverlayView">返回列表</BButton>
            </div>
          </BForm>
        </div>
      </section>

      <section v-else-if="currentView === 'application-create'" class="section-grid">
        <div class="info-card">
          <div class="section-title">创建应用</div>
          <BForm @submit.prevent="submitApplicationCreatePage">
            <BFormSelect v-model="applicationTemplateSelection" :options="applicationProtocolTemplateOptions" class="mb-2" />
            <div class="detail-card mb-2">
              <div class="record-meta mb-2">模板说明</div>
              <div class="record-meta mb-3">模板会自动填充 `grant_type`、`enable_refresh_token`、`token_type`、`client_authentication_type`。仅显示当前应用类型允许使用的模板。</div>
              <div class="metadata-table-wrap">
                <table class="table table-sm align-middle mb-0">
                  <thead>
                    <tr>
                      <th>模板</th>
                      <th>允许类型</th>
                      <th>Grant Type</th>
                      <th>Refresh Token</th>
                      <th>Token Type</th>
                      <th>Client Authentication Type</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="item in visibleApplicationProtocolTemplates" :key="item.key">
                      <td>{{ item.text }}</td>
                      <td>{{ item.allowedTypes }}</td>
                      <td><code>{{ item.grantType }}</code></td>
                      <td>{{ item.enableRefreshToken }}</td>
                      <td><code>{{ item.tokenType }}</code></td>
                      <td><code>{{ item.clientAuthenticationType }}</code></td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
            <BFormInput v-model="applicationForm.name" placeholder="应用名称" class="mb-2" />
            <BFormInput v-model="applicationForm.redirectUris" placeholder="回调地址，多个值可用逗号或换行分隔" class="mb-2" />
            <BFormSelect v-model="applicationForm.applicationType" :options="applicationTypeOptions" class="mb-2" />
            <div class="detail-card mb-2">
              <div class="record-meta mb-2">Token Type</div>
              <div class="d-flex flex-wrap gap-3">
                <label v-for="item in applicationFormTokenTypeOptions" :key="item.value" class="d-inline-flex align-items-center gap-2">
                  <input
                    class="form-check-input mt-0"
                    type="checkbox"
                    :checked="applicationForm.tokenType.includes(item.value)"
                    @change="toggleApplicationTokenType(applicationForm.tokenType, item.value, ($event.target as HTMLInputElement).checked)"
                  />
                  <span>{{ item.text }}</span>
                </label>
              </div>
            </div>
            <BFormCheckbox v-model="applicationForm.enableRefreshToken" class="mb-2">启用 Refresh Token</BFormCheckbox>
            <div class="detail-card mb-2">
              <div class="record-meta mb-2">Grant Type</div>
              <div class="d-flex flex-wrap gap-3">
                <label v-for="item in grantTypeOptions" :key="item.value" class="d-inline-flex align-items-center gap-2">
                  <input
                    class="form-check-input mt-0"
                    type="checkbox"
                    :checked="applicationForm.grantType.includes(item.value)"
                    @change="toggleApplicationGrantType(applicationForm.grantType, item.value, ($event.target as HTMLInputElement).checked)"
                  />
                  <span>{{ item.text }}</span>
                </label>
              </div>
            </div>
            <BFormSelect v-model="applicationForm.clientAuthenticationType" :options="applicationFormClientAuthenticationTypeOptions" class="mb-2" />
            <div class="detail-card mb-2">
              <div class="record-meta mb-2">应用角色</div>
              <div class="d-flex flex-wrap gap-3">
                <label v-for="item in applicationAssignableRoles" :key="item.id" class="d-inline-flex align-items-center gap-2">
                  <input
                    class="form-check-input mt-0"
                    type="checkbox"
                    :checked="applicationForm.roles.includes(item.name)"
                    @change="toggleRoleName(applicationForm.roles, item.name, ($event.target as HTMLInputElement).checked)"
                  />
                  <span>{{ item.name }}</span>
                </label>
              </div>
            </div>
            <div v-if="applicationForm.clientAuthenticationType === 'private_key_jwt'" class="record-meta mb-2">
              创建后系统会自动生成一组 RSA 密钥对。数据库仅保存公钥，私钥只会显示一次。
            </div>
            <BFormInput v-model="applicationForm.accessTokenTTLMinutes" type="number" placeholder="access token ttl minutes" class="mb-2" />
            <BFormInput v-model="applicationForm.refreshTokenTTLHours" type="number" placeholder="refresh token ttl hours" class="mb-2" />
            <div class="d-flex gap-2">
              <BButton type="submit" variant="primary">创建应用</BButton>
              <BButton type="button" variant="outline-secondary" @click="closeOverlayView">返回项目</BButton>
            </div>
          </BForm>
        </div>
      </section>

      <section v-else-if="currentView === 'application-detail'" class="console-module-shell">
        <div class="console-module-summary-card">
          <div class="console-module-hero">
            <div class="console-module-hero-copy">
              <button type="button" class="console-back-button" @click="backToProjectDetail" aria-label="返回项目详情">
                <i class="bi bi-arrow-left console-back-button-icon" aria-hidden="true"></i>
              </button>
              <div>
                <div class="console-module-eyebrow">应用</div>
                <h2 class="console-module-title">{{ currentApplication?.name || '应用' }}</h2>
                <p class="console-module-subtitle">查看并维护当前应用的协议能力、令牌参数与接入配置。</p>
              </div>
            </div>
            <div class="console-action-menu" role="group" aria-label="应用操作">
              <button type="button" class="btn btn-primary console-action-menu-toggle">
                操作
                <i class="bi bi-chevron-down" aria-hidden="true"></i>
              </button>
              <div class="console-action-menu-list">
                <button type="button" class="console-action-menu-item" @click="showApplicationDisableNotice">停用</button>
                <button type="button" class="console-action-menu-item console-action-menu-item-danger" @click="showApplicationDeleteNotice">删除</button>
              </div>
            </div>
          </div>
          <div class="console-module-metrics">
            <div v-for="item in applicationDetailMetrics" :key="item.label" class="console-module-metric">
              <span class="console-module-metric-label">{{ item.label }}</span>
              <div class="console-module-metric-value-row">
                <strong class="console-module-metric-value">{{ item.value }}</strong>
                <button
                  v-if="item.copyable"
                  type="button"
                  class="console-module-metric-copy"
                  :aria-label="`复制${item.label}`"
                  @click="copyMetricValue(item.copyValue || item.value)"
                >
                  <i class="bi bi-copy" aria-hidden="true"></i>
                </button>
              </div>
            </div>
          </div>
        </div>
        <div class="console-module-workspace">
          <aside class="console-module-sidebar">
            <button v-for="item in applicationDetailPanels" :key="item.id" type="button" class="console-module-sidebar-link" @click="scrollToPanel(item.id)">{{ item.label }}</button>
          </aside>
          <div class="console-module-main">
            <div id="application-protocol" class="info-card">
              <div class="section-title">协议配置</div>
              <div class="record-meta mb-3">Application Type 仅作为内部标记。真正控制协议行为的是 Grant Type、Token Type、Enable Refresh Token、Client Authentication Type 四个维度。</div>
              <div class="detail-card mb-3">
                <div class="record-meta">Grant Type：表示 OAuth 授权流程。</div>
                <div class="record-meta">Token Type：表示主令牌类型，可选 `access_token`、`access_token + id_token`、`id_token`。</div>
                <div class="record-meta">Enable Refresh Token：表示是否额外签发 `refresh_token`，后续可用 `grant_type=refresh_token` 更新访问令牌。</div>
                <div class="record-meta">Client Authentication Type：表示客户端调用 token endpoint 时如何证明自己。</div>
                <div class="record-meta">应用角色：表示该 Application 具备哪些策略组，例如 `api:manage`、`api:user`。</div>
                <div class="record-meta mt-2">约束：</div>
                <div class="record-meta">1. `client_credentials` 只能使用 `access_token`，且不能启用 Refresh Token。</div>
                <div class="record-meta">2. `implicit` 只允许 `access_token`、`access_token + id_token`、`id_token`，且不能启用 Refresh Token。</div>
                <div class="record-meta">3. `authorization_code_pkce` 必须使用 `client_authentication_type=none`。</div>
                <div class="record-meta">4. 其他 Grant Type 不能使用 `client_authentication_type=none`。</div>
                <div class="record-meta mt-2">{{ currentApplicationProtocolHint }}</div>
              </div>
              <BForm @submit.prevent="updateApplication">
                <div class="mb-3">
                  <label class="form-label">应用名称</label>
                  <BFormInput v-model="applicationUpdateForm.name" placeholder="请输入应用名称" />
                </div>
                <div class="mb-3">
                  <label class="form-label">回调地址</label>
                  <BFormInput v-model="applicationUpdateForm.redirectUris" placeholder="请输入回调地址" />
                </div>
                <div class="mb-3">
                  <label class="form-label">Application Type</label>
                  <BFormSelect v-model="applicationUpdateForm.applicationType" :options="applicationTypeOptions" />
                </div>
                <div class="detail-card mb-3">
                  <div class="form-label mb-2">Token Type</div>
                  <div class="d-flex flex-wrap gap-3">
                    <label v-for="item in applicationUpdateTokenTypeOptions" :key="item.value" class="d-inline-flex align-items-center gap-2">
                      <input
                        class="form-check-input mt-0"
                        type="checkbox"
                        :checked="applicationUpdateForm.tokenType.includes(item.value)"
                        @change="toggleApplicationTokenType(applicationUpdateForm.tokenType, item.value, ($event.target as HTMLInputElement).checked)"
                      />
                      <span>{{ item.text }}</span>
                    </label>
                  </div>
                </div>
                <div class="mb-3">
                  <label class="form-label d-block">Refresh Token</label>
                  <BFormCheckbox v-model="applicationUpdateForm.enableRefreshToken">启用 Refresh Token</BFormCheckbox>
                </div>
                <div class="detail-card mb-3">
                  <div class="form-label mb-2">Grant Type</div>
                  <div class="d-flex flex-wrap gap-3">
                    <label v-for="item in grantTypeOptions" :key="item.value" class="d-inline-flex align-items-center gap-2">
                      <input
                        class="form-check-input mt-0"
                        type="checkbox"
                        :checked="applicationUpdateForm.grantType.includes(item.value)"
                        @change="toggleApplicationGrantType(applicationUpdateForm.grantType, item.value, ($event.target as HTMLInputElement).checked)"
                      />
                      <span>{{ item.text }}</span>
                    </label>
                  </div>
                </div>
                <div class="mb-3">
                  <label class="form-label">Client Authentication Type</label>
                  <BFormSelect v-model="applicationUpdateForm.clientAuthenticationType" :options="applicationUpdateClientAuthenticationTypeOptions" />
                </div>
                <BButton type="submit" variant="outline-primary">保存协议配置</BButton>
              </BForm>
            </div>
            <div id="application-role-assignment" class="info-card">
              <div class="section-title">角色分配</div>
              <div class="record-meta mb-3">维护当前应用可授予或可使用的应用角色标签。</div>
              <BForm @submit.prevent="updateApplication">
                <div class="detail-card mb-3">
                  <div class="form-label mb-2">应用角色</div>
                  <div class="d-flex flex-wrap gap-3">
                    <label v-for="item in applicationAssignableRoles" :key="item.id" class="d-inline-flex align-items-center gap-2">
                      <input
                        class="form-check-input mt-0"
                        type="checkbox"
                        :checked="applicationUpdateForm.roles.includes(item.name)"
                        @change="toggleRoleName(applicationUpdateForm.roles, item.name, ($event.target as HTMLInputElement).checked)"
                      />
                      <span>{{ item.name }}</span>
                    </label>
                  </div>
                </div>
                <BButton type="submit" variant="outline-primary">保存角色分配</BButton>
              </BForm>
            </div>
            <div id="application-token" class="info-card">
              <div class="section-title">令牌设置</div>
              <div class="record-meta mb-3">Issuer 为实例级统一配置。`private_key_jwt` 应用的公钥以 Ed25519 裸公钥形式保存在系统中；普通应用的私钥只在创建或重置时显示一次，系统内置 API 应用的私钥固化在代码中。</div>
              <BForm @submit.prevent="updateApplication">
                <div v-if="applicationUpdateForm.clientAuthenticationType === 'private_key_jwt'" class="mb-3">
                  <label class="form-label">应用公钥</label>
                  <textarea
                    class="form-control"
                    rows="8"
                    :value="applicationUpdateForm.publicKey"
                    readonly
                  />
                </div>
                <div class="mb-3">
                  <label class="form-label">Access Token TTL Minutes</label>
                  <BFormInput v-model="applicationUpdateForm.accessTokenTTLMinutes" type="number" placeholder="请输入 access token ttl minutes" />
                </div>
                <div class="mb-3">
                  <label class="form-label">Refresh Token TTL Hours</label>
                  <BFormInput v-model="applicationUpdateForm.refreshTokenTTLHours" type="number" placeholder="请输入 refresh token ttl hours" />
                </div>
                <div class="d-flex gap-2">
                  <BButton type="submit" variant="outline-primary">保存令牌设置</BButton>
                  <BButton
                    v-if="applicationUpdateForm.clientAuthenticationType === 'private_key_jwt' && applicationUpdateForm.id"
                    type="button"
                    variant="outline-danger"
                    @click="resetApplicationKey"
                  >
                    重置密钥
                  </BButton>
                </div>
              </BForm>
            </div>
          </div>
          <aside class="console-module-aside">
            <div class="info-card">
              <div class="section-title">最近变更</div>
              <div class="record-list">
                <div v-for="item in moduleRecentChanges" :key="item.id" class="record-row">
                  <div>
                    <strong>{{ item.eventType }}</strong>
                    <div class="record-meta">{{ item.result }}</div>
                  </div>
                  <code>{{ formatDateTime(item.createdAt) }}</code>
                </div>
              </div>
            </div>
          </aside>
        </div>
      </section>

      <section v-else-if="tab === 'dashboard'" class="console-module-shell">
        <div class="console-module-summary-card">
          <div class="console-module-hero">
            <div>
              <div class="console-module-eyebrow">{{ currentTabLabel }}</div>
              <h2 class="console-module-title">{{ currentModuleEntityTitle }}</h2>
              <p class="console-module-subtitle">{{ currentModuleSummaryText }}</p>
            </div>
            <BButton v-if="showModuleActionButton" variant="primary" @click="runModuleAction">{{ currentModuleActionLabel }}</BButton>
          </div>
          <div class="console-module-metrics" :class="currentModuleMetricsClass">
            <div v-for="item in currentModuleMetrics" :key="item.label" class="console-module-metric">
              <span class="console-module-metric-label">{{ item.label }}</span>
              <div class="console-module-metric-value-row">
                <strong class="console-module-metric-value">{{ item.value }}</strong>
                <button
                  v-if="item.copyable"
                  type="button"
                  class="console-module-metric-copy"
                  :aria-label="`复制${item.label}`"
                  @click="copyMetricValue(item.copyValue || item.value)"
                >
                  <i class="bi bi-copy" aria-hidden="true"></i>
                </button>
              </div>
            </div>
          </div>
        </div>
        <div class="console-module-workspace">
          <aside class="console-module-sidebar">
            <button v-for="item in currentModulePanels" :key="item.id" type="button" class="console-module-sidebar-link" @click="scrollToPanel(item.id)">{{ item.label }}</button>
          </aside>
          <div class="console-module-main">
        <div id="dashboard-overview" class="info-card">
          <div class="section-title">平台概览</div>
          <div class="small text-secondary mb-3">组织、项目、应用、用户与策略概览</div>
          <div class="summary-grid">
            <div class="summary-tile" v-for="item in summaryTiles" :key="item.label">
              <span class="summary-label">{{ item.label }}</span>
              <strong class="summary-value">{{ item.value }}</strong>
            </div>
          </div>
        </div>
        <div id="dashboard-audit" class="info-card">
          <div class="section-title">审计摘要</div>
          <BButton size="sm" variant="outline-primary" class="mb-3" @click="loadAudit">刷新</BButton>
          <div class="record-list">
            <div v-for="item in recentAuditLogs" :key="item.id" class="record-row">
              <div>
                <strong>{{ item.eventType }}</strong>
                <div class="record-meta">{{ item.actorType }} · {{ item.result }}</div>
              </div>
              <code>{{ formatDateTime(item.createdAt) }}</code>
            </div>
          </div>
        </div>
          </div>
          <aside class="console-module-aside">
            <div class="info-card">
              <div class="section-title">最近变更</div>
              <div class="record-list">
                <div v-for="item in moduleRecentChanges" :key="item.id" class="record-row">
                  <div>
                    <strong>{{ item.eventType }}</strong>
                    <div class="record-meta">{{ item.result }}</div>
                  </div>
                  <code>{{ formatDateTime(item.createdAt) }}</code>
                </div>
              </div>
            </div>
          </aside>
        </div>
      </section>

      <section v-else-if="tab === 'organization'" class="console-module-shell">
        <div class="console-module-summary-card">
          <div class="console-module-hero">
            <div>
              <div class="console-module-eyebrow">{{ currentTabLabel }}</div>
              <h2 class="console-module-title">{{ currentModuleEntityTitle }}</h2>
              <p class="console-module-subtitle">{{ currentModuleSummaryText }}</p>
            </div>
            <div class="console-action-menu" role="group" aria-label="组织操作">
              <button type="button" class="btn btn-primary console-action-menu-toggle">
                操作
                <i class="bi bi-chevron-down" aria-hidden="true"></i>
              </button>
              <div class="console-action-menu-list">
                <button type="button" class="console-action-menu-item" @click="showOrganizationDisableNotice">停用</button>
                <button type="button" class="console-action-menu-item console-action-menu-item-danger" @click="showOrganizationDeleteNotice">删除</button>
              </div>
            </div>
          </div>
          <div class="console-module-metrics" :class="currentModuleMetricsClass">
            <div v-for="item in currentModuleMetrics" :key="item.label" class="console-module-metric">
              <span class="console-module-metric-label">{{ item.label }}</span>
              <div class="console-module-metric-value-row">
                <strong class="console-module-metric-value">{{ item.value }}</strong>
                <button
                  v-if="item.copyable"
                  type="button"
                  class="console-module-metric-copy"
                  :aria-label="`复制${item.label}`"
                  @click="copyMetricValue(item.copyValue || item.value)"
                >
                  <i class="bi bi-copy" aria-hidden="true"></i>
                </button>
              </div>
            </div>
          </div>
        </div>
        <div class="console-module-workspace">
          <aside class="console-module-sidebar">
            <button v-for="item in currentModulePanels" :key="item.id" type="button" class="console-module-sidebar-link" @click="scrollToPanel(item.id)">{{ item.label }}</button>
          </aside>
          <div class="console-module-main">
            <div id="organization-metadata" class="info-card">
              <div class="section-title">维护元信息</div>
              <div class="record-meta mb-3">这些元信息会作为可用变量，用于自定义登录页等组织级展示场景。</div>
              <div v-if="currentOrganization" class="detail-card">
                <div class="metadata-table-wrap">
                  <table class="table table-sm align-middle mb-0">
                    <thead>
                      <tr>
                        <th class="metadata-col-key">键</th>
                        <th>值</th>
                        <th class="metadata-col-action"></th>
                      </tr>
                    </thead>
                    <tbody>
                      <tr v-for="(item, index) in organizationMetadataRows" :key="item.id">
                        <td>
                          <BFormInput v-model="item.key" placeholder="例如 login_title" />
                        </td>
                        <td>
                          <BFormInput v-model="item.value" placeholder="例如 PPVT 控制台" />
                        </td>
                        <td class="text-end">
                          <BButton size="sm" variant="outline-danger" @click="removeOrganizationMetadataRow(index)">删除</BButton>
                        </td>
                      </tr>
                      <tr v-if="organizationMetadataRows.length === 0">
                        <td colspan="3" class="text-center text-secondary py-4">当前还没有元信息，新增后可作为组织级变量使用。</td>
                      </tr>
                    </tbody>
                  </table>
                </div>
                <div class="d-flex gap-2 mt-3">
                  <BButton variant="outline-secondary" @click="addOrganizationMetadataRow">新增条目</BButton>
                  <BButton variant="primary" @click="saveOrganizationMetadata">保存元信息</BButton>
                </div>
              </div>
            </div>
          </div>
          <aside class="console-module-aside">
            <div class="info-card">
              <div class="section-title">最近变更</div>
              <div class="record-list">
                <div v-for="item in moduleRecentChanges" :key="item.id" class="record-row">
                  <div>
                    <strong>{{ item.eventType }}</strong>
                    <div class="record-meta">{{ item.result }}</div>
                  </div>
                  <code>{{ formatDateTime(item.createdAt) }}</code>
                </div>
              </div>
            </div>
          </aside>
        </div>
      </section>

      <section v-else-if="tab === 'project' && projectViewMode === 'list'" class="section-grid">
        <div class="info-card">
          <div class="section-title">当前组织下可用的项目</div>
          <div class="record-list project-list-records">
            <button
              v-for="project in projects"
              :key="project.id"
              type="button"
              class="record-card record-card-button"
              @click="selectProject(project)"
            >
              <div class="project-card-id mb-1">{{ project.id }}</div>
              <div class="record-head align-items-center mb-1">
                <div class="project-card-name">{{ project.name || '-' }}</div>
                <span class="badge rounded-pill" :class="project.status === 'disabled' ? 'text-bg-secondary' : 'text-bg-success'">
                  {{ project.status === 'disabled' ? '停用' : '启用' }}
                </span>
              </div>
              <div class="record-meta">创建时间</div>
              <div class="project-card-value mb-1">{{ formatDateTime(project.createdAt) }}</div>
              <div class="record-meta">更新时间</div>
              <div class="project-card-value mb-1">{{ formatDateTime(project.updatedAt) }}</div>
              <div class="record-meta">应用数</div>
              <div class="project-card-value">{{ project.applications?.length ?? 0 }}</div>
            </button>
            <div class="record-card project-create-card">
              <button
                type="button"
                class="project-create-trigger"
                @click="goProjectCreate"
              >
                <div class="project-create-plus text-secondary lh-1 mb-2">+</div>
                <div class="project-create-title">创建新项目</div>
              </button>
            </div>
          </div>
        </div>
      </section>

      <section v-else-if="tab === 'user' && userViewMode === 'list'" class="section-grid">
        <div class="info-card">
          <div class="section-title">当前组织下可用的用户</div>
          <div class="d-flex align-items-center justify-content-between gap-3 mb-3 flex-wrap">
            <div class="d-flex align-items-center gap-2 flex-wrap">
              <BButton size="sm" variant="outline-danger" :disabled="selectedUserIds.length === 0" @click="deleteSelectedUsers">删除用户</BButton>
            </div>
            <BButton size="sm" variant="primary" @click="showCreateUserForm = !showCreateUserForm">{{ showCreateUserForm ? '收起添加用户' : '添加用户' }}</BButton>
          </div>
          <div v-if="showCreateUserForm" class="detail-card mb-3">
            <BForm @submit.prevent="submitUserCreateFromList">
              <BFormInput v-model="userForm.username" placeholder="username" class="mb-2" />
              <BFormInput v-model="userForm.name" placeholder="name" class="mb-2" />
              <BFormInput v-model="userForm.email" placeholder="email" class="mb-2" />
              <div class="phone-input-group mb-2">
                <BFormSelect v-model="userPhoneInput.countryCode" :options="phoneCountryOptions" class="phone-country-select" />
                <BFormInput v-model="userPhoneInput.localNumber" placeholder="phoneNumber" class="phone-local-input" />
              </div>
              <BFormInput v-model="userForm.roleLabels" placeholder="role labels, comma separated" class="mb-2" />
              <BFormInput v-model="userForm.identifier" placeholder="login identifier" class="mb-2" />
              <BFormInput v-model="userForm.password" type="password" placeholder="initial password" class="mb-2" />
              <div class="d-flex gap-2">
                <BButton type="submit" variant="primary">创建用户</BButton>
                <BButton type="button" variant="outline-secondary" @click="showCreateUserForm = false">取消</BButton>
              </div>
            </BForm>
          </div>
          <div class="table-responsive">
            <table class="table align-middle console-list-table mb-0">
              <thead>
                <tr>
                  <th class="console-list-check-col">
                    <input
                      class="form-check-input console-list-checkbox"
                      type="checkbox"
                      :checked="users.length > 0 && selectedUserIds.length === users.length"
                      @change="toggleAllUsers(($event.target as HTMLInputElement).checked)"
                    />
                  </th>
                  <th>用户 ID</th>
                  <th>用户名</th>
                  <th>名称</th>
                  <th>邮箱 / 手机号</th>
                  <th>状态</th>
                  <th>角色</th>
                  <th class="text-end">操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="user in users" :key="user.id">
                  <td class="console-list-check-col">
                    <input class="form-check-input console-list-checkbox" type="checkbox" :checked="selectedUserIds.includes(user.id)" @change="toggleUserSelection(user.id, ($event.target as HTMLInputElement).checked)" />
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
                  <td class="text-end">
                    <div class="d-inline-flex gap-2">
                      <BButton size="sm" variant="outline-primary" @click="selectUser(user)">查看详情</BButton>
                      <BButton size="sm" variant="outline-danger" @click="deleteSingleUser(user.id)">删除</BButton>
                    </div>
                  </td>
                </tr>
                <tr v-if="users.length === 0">
                  <td colspan="8" class="text-center text-secondary py-4">当前组织下还没有用户。</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </section>

      <section v-else-if="tab === 'role' && roleViewMode === 'list'" class="section-grid">
        <div class="info-card">
          <div class="section-title">当前组织下可用的用户角色</div>
          <div class="d-flex align-items-center justify-content-between gap-3 mb-3 flex-wrap">
            <div class="d-flex align-items-center gap-2 flex-wrap">
              <BButton size="sm" variant="outline-danger" :disabled="selectedRoleIds.length === 0" @click="deleteSelectedRoles">删除角色</BButton>
            </div>
            <BButton size="sm" variant="primary" @click="showCreateRoleForm = !showCreateRoleForm">{{ showCreateRoleForm ? '收起添加角色' : '添加角色' }}</BButton>
          </div>
          <div v-if="showCreateRoleForm" class="detail-card mb-3">
            <BForm @submit.prevent="submitRoleCreateFromList">
              <BFormInput v-model="roleForm.name" placeholder="role label" class="mb-2" />
              <BFormSelect v-model="roleForm.type" :options="roleTypeOptions" class="mb-2" />
              <BFormInput v-model="roleForm.description" placeholder="description" class="mb-2" />
              <div class="d-flex gap-2">
                <BButton type="submit" variant="primary">创建角色</BButton>
                <BButton type="button" variant="outline-secondary" @click="showCreateRoleForm = false">取消</BButton>
              </div>
            </BForm>
          </div>
          <div class="table-responsive">
            <table class="table align-middle console-list-table mb-0">
              <thead>
                <tr>
                  <th class="console-list-check-col">
                    <input
                      class="form-check-input console-list-checkbox"
                      type="checkbox"
                      :checked="userAssignableRoles.length > 0 && userAssignableRoles.every((role) => selectedRoleIds.includes(role.id))"
                      @change="toggleRolesByType('user', ($event.target as HTMLInputElement).checked)"
                    />
                  </th>
                  <th>角色 ID</th>
                  <th>角色标签</th>
                  <th>描述</th>
                  <th>策略数</th>
                  <th class="text-end">操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="role in userAssignableRoles" :key="role.id">
                  <td class="console-list-check-col">
                    <input class="form-check-input console-list-checkbox" type="checkbox" :checked="selectedRoleIds.includes(role.id)" @change="toggleRoleSelection(role.id, ($event.target as HTMLInputElement).checked)" />
                  </td>
                  <td class="console-list-id">{{ role.id }}</td>
                  <td>{{ role.name || '-' }}</td>
                  <td>{{ role.description || '-' }}</td>
                  <td>{{ policies.filter((item) => item.roleId === role.id).length }}</td>
                  <td class="text-end">
                    <div class="d-inline-flex gap-2">
                      <BButton size="sm" variant="outline-primary" @click="selectRole(role)">查看详情</BButton>
                      <BButton size="sm" variant="outline-danger" @click="deleteSingleRole(role.id)">删除</BButton>
                    </div>
                  </td>
                </tr>
                <tr v-if="userAssignableRoles.length === 0">
                  <td colspan="6" class="text-center text-secondary py-4">当前组织下还没有用户角色。</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
        <div class="info-card">
          <div class="section-title">当前组织下可用的应用角色</div>
          <div class="table-responsive">
            <table class="table align-middle console-list-table mb-0">
              <thead>
                <tr>
                  <th class="console-list-check-col">
                    <input
                      class="form-check-input console-list-checkbox"
                      type="checkbox"
                      :checked="applicationAssignableRoles.length > 0 && applicationAssignableRoles.every((role) => selectedRoleIds.includes(role.id))"
                      @change="toggleRolesByType('application', ($event.target as HTMLInputElement).checked)"
                    />
                  </th>
                  <th>角色 ID</th>
                  <th>角色标签</th>
                  <th>描述</th>
                  <th>策略数</th>
                  <th class="text-end">操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="role in applicationAssignableRoles" :key="role.id">
                  <td class="console-list-check-col">
                    <input class="form-check-input console-list-checkbox" type="checkbox" :checked="selectedRoleIds.includes(role.id)" @change="toggleRoleSelection(role.id, ($event.target as HTMLInputElement).checked)" />
                  </td>
                  <td class="console-list-id">{{ role.id }}</td>
                  <td>{{ role.name || '-' }}</td>
                  <td>{{ role.description || '-' }}</td>
                  <td>{{ policies.filter((item) => item.roleId === role.id).length }}</td>
                  <td class="text-end">
                    <div class="d-inline-flex gap-2">
                      <BButton size="sm" variant="outline-primary" @click="selectRole(role)">查看详情</BButton>
                      <BButton size="sm" variant="outline-danger" @click="deleteSingleRole(role.id)">删除</BButton>
                    </div>
                  </td>
                </tr>
                <tr v-if="applicationAssignableRoles.length === 0">
                  <td colspan="6" class="text-center text-secondary py-4">当前组织下还没有应用角色。</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </section>

      <section v-else-if="tab === 'project'" class="console-module-shell">
        <div class="console-module-summary-card">
          <div class="console-module-hero">
            <div class="console-module-hero-copy">
              <button type="button" class="console-back-button" @click="backToProjectList" aria-label="返回项目列表">
                <i class="bi bi-arrow-left console-back-button-icon" aria-hidden="true"></i>
              </button>
              <div>
              <div class="console-module-eyebrow">{{ currentTabLabel }}</div>
              <h2 class="console-module-title">{{ currentModuleEntityTitle }}</h2>
              <p class="console-module-subtitle">{{ currentModuleSummaryText }}</p>
              </div>
            </div>
            <div class="console-action-menu" role="group" aria-label="项目操作">
              <button type="button" class="btn btn-primary console-action-menu-toggle">
                操作
                <i class="bi bi-chevron-down" aria-hidden="true"></i>
              </button>
              <div class="console-action-menu-list">
                <button type="button" class="console-action-menu-item" @click="showProjectDisableNotice">停用</button>
                <button type="button" class="console-action-menu-item console-action-menu-item-danger" @click="showProjectDeleteNotice">删除</button>
              </div>
            </div>
          </div>
          <div class="console-module-metrics" :class="currentModuleMetricsClass">
            <div v-for="item in currentModuleMetrics" :key="item.label" class="console-module-metric">
              <span class="console-module-metric-label">{{ item.label }}</span>
              <div class="console-module-metric-value-row">
                <strong class="console-module-metric-value">{{ item.value }}</strong>
                <button
                  v-if="item.copyable"
                  type="button"
                  class="console-module-metric-copy"
                  :aria-label="`复制${item.label}`"
                  @click="copyMetricValue(item.copyValue || item.value)"
                >
                  <i class="bi bi-copy" aria-hidden="true"></i>
                </button>
              </div>
            </div>
          </div>
        </div>
        <div class="console-module-workspace">
          <aside class="console-module-sidebar">
            <button v-for="item in currentModulePanels" :key="item.id" type="button" class="console-module-sidebar-link" @click="scrollToPanel(item.id)">{{ item.label }}</button>
          </aside>
          <div class="console-module-main">
            <div id="project-application" class="info-card">
              <div class="section-title">应用列表</div>
              <div class="record-meta mb-3">显示当前项目下的应用列表，点击卡片进入应用详情页。</div>
              <div class="record-list project-application-grid mb-3">
                <button
                  v-for="application in applications"
                  :key="application.id"
                  type="button"
                  class="record-card record-card-button"
                  @click="goApplicationDetail(application)"
                >
                  <div class="project-card-id mb-1">{{ application.id }}</div>
                  <div class="record-head align-items-center mb-2">
                    <div class="project-card-name">{{ application.name || application.id }}</div>
                    <span
                      class="badge rounded-pill"
                      :class="application.status === 'disabled' ? 'text-bg-secondary' : 'text-bg-success'"
                    >
                      {{ application.status === 'disabled' ? '停用' : '启用' }}
                    </span>
                  </div>
                  <div class="record-meta">Token Type</div>
                  <div class="project-card-value mb-1">{{ formatApplicationTokenType(application.tokenType) }}</div>
                  <div class="record-meta">Grant Type</div>
                  <div class="project-card-value mb-1">{{ formatApplicationGrantType(application.grantType) }}</div>
                  <div class="record-meta">应用角色</div>
                  <div class="project-card-value mb-1">{{ formatRoleLabels(application.roles) }}</div>
                  <div class="record-meta">Client Authentication Type</div>
                  <div class="project-card-value">{{ formatApplicationClientAuthenticationType(application.clientAuthenticationType) }}</div>
                </button>
                <div class="record-card project-create-card application-create-card">
                  <button
                    type="button"
                    class="project-create-trigger"
                    @click="goApplicationCreate"
                  >
                    <div class="project-create-plus text-secondary lh-1 mb-2">+</div>
                    <div class="project-create-title">创建新应用</div>
                  </button>
                </div>
              </div>
            </div>
            <div id="project-user-assignment" class="info-card">
              <div class="section-title">用户分配</div>
              <div class="record-meta mb-3">
                {{ projectUpdateForm.userAclEnabled
                  ? '已开启用户访问控制。只有已分配用户，才能访问该项目下的系统应用；如果分配列表为空，则所有用户都不可访问。'
                  : '当前未开启用户访问控制。关闭时，当前组织下所有用户都可以访问该项目下的系统应用。' }}
              </div>
              <div class="d-flex justify-content-between align-items-center gap-2 mb-3">
                <div class="record-meta mb-0">已分配 {{ projectAssignedUserIds.length }} / {{ users.length }} 个用户。</div>
                <BButton variant="outline-primary" size="sm" @click="openProjectUserAssignmentModal">添加用户</BButton>
              </div>
              <div class="table-responsive project-user-assignment-wrap mb-3">
                <table class="table align-middle console-list-table project-user-assignment-table mb-0">
                  <thead>
                    <tr>
                      <th>用户 ID</th>
                      <th>用户名</th>
                      <th>名称</th>
                      <th>邮箱 / 手机号</th>
                      <th>状态</th>
                      <th>角色</th>
                      <th class="text-end">操作</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="user in assignedProjectUsers" :key="user.id">
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
                      <td class="text-end">
                        <BButton size="sm" variant="outline-danger" @click="removeProjectAssignedUser(user.id)">移出</BButton>
                      </td>
                    </tr>
                    <tr v-if="assignedProjectUsers.length === 0">
                      <td colspan="7" class="text-center text-secondary py-4">当前 ACL 中还没有用户。</td>
                    </tr>
                  </tbody>
                </table>
              </div>
              <BButton variant="primary" size="sm" @click="saveProjectUserAssignments">保存用户分配</BButton>
            </div>
            <div id="project-setting" class="info-card">
              <div class="section-title">项目设置</div>
              <div class="record-meta mb-3">维护当前项目的基础名称和描述。</div>
              <BForm @submit.prevent="updateProject">
                <div class="mb-3">
                  <label class="form-label">项目名称</label>
                  <BFormInput v-model="projectUpdateForm.name" placeholder="请输入项目名称" />
                </div>
                <div class="mb-3">
                  <label class="form-label">项目描述</label>
                  <BFormInput v-model="projectUpdateForm.description" placeholder="请输入项目描述" />
                </div>
                <div class="mb-3">
                  <label class="form-label d-block">用户访问控制</label>
                  <BFormCheckbox v-model="projectUpdateForm.userAclEnabled">
                    开启后，仅允许已分配到项目的用户访问该项目下应用
                  </BFormCheckbox>
                </div>
                <BButton type="submit" variant="outline-primary">保存项目设置</BButton>
              </BForm>
            </div>
          </div>
          <aside class="console-module-aside">
            <div class="info-card">
              <div class="section-title">最近变更</div>
              <div class="record-list">
                <div v-for="item in moduleRecentChanges" :key="item.id" class="record-row">
                  <div>
                    <strong>{{ item.eventType }}</strong>
                    <div class="record-meta">{{ item.result }}</div>
                  </div>
                  <code>{{ formatDateTime(item.createdAt) }}</code>
                </div>
              </div>
            </div>
          </aside>
        </div>
      </section>

      <section v-else-if="tab === 'user'" class="console-module-shell">
        <div class="console-module-summary-card">
          <div class="console-module-hero">
            <div class="console-module-hero-copy">
              <button type="button" class="console-back-button" @click="backToUserList" aria-label="返回用户列表">
                <i class="bi bi-arrow-left console-back-button-icon" aria-hidden="true"></i>
              </button>
              <div>
                <div class="console-module-eyebrow">{{ currentTabLabel }}</div>
                <h2 class="console-module-title">{{ currentModuleEntityTitle }}</h2>
                <p class="console-module-subtitle">{{ currentModuleSummaryText }}</p>
              </div>
            </div>
            <BButton variant="primary" @click="runModuleAction">{{ currentModuleActionLabel }}</BButton>
          </div>
          <div class="console-module-metrics" :class="currentModuleMetricsClass">
            <div v-for="item in currentModuleMetrics" :key="item.label" class="console-module-metric">
              <span class="console-module-metric-label">{{ item.label }}</span>
              <div class="console-module-metric-value-row">
                <strong class="console-module-metric-value">{{ item.value }}</strong>
                <button
                  v-if="item.copyable"
                  type="button"
                  class="console-module-metric-copy"
                  :aria-label="`复制${item.label}`"
                  @click="copyMetricValue(item.copyValue || item.value)"
                >
                  <i class="bi bi-copy" aria-hidden="true"></i>
                </button>
              </div>
            </div>
          </div>
        </div>
        <div class="console-module-workspace">
          <aside class="console-module-sidebar">
            <button v-for="item in currentModulePanels" :key="item.id" type="button" class="console-module-sidebar-link" @click="scrollToPanel(item.id)">{{ item.label }}</button>
          </aside>
          <div class="console-module-main">
            <div id="user-basic" class="info-card">
              <div class="section-title">基本信息</div>
              <BForm @submit.prevent="updateUser">
                <div class="row g-3">
                  <div class="col-md-6">
                    <label class="form-label">姓名</label>
                    <BFormInput v-model="userUpdateForm.name" />
                  </div>
                  <div class="col-md-6">
                    <label class="form-label">用户名</label>
                    <BFormInput v-model="userUpdateForm.username" />
                  </div>
                  <div class="col-md-6">
                    <label class="form-label">邮箱</label>
                    <BFormInput v-model="userUpdateForm.email" type="email" />
                  </div>
                  <div class="col-md-6">
                    <label class="form-label">手机</label>
                    <div class="phone-input-group">
                      <BFormSelect v-model="userUpdatePhoneInput.countryCode" :options="phoneCountryOptions" class="phone-country-select" />
                      <BFormInput v-model="userUpdatePhoneInput.localNumber" class="phone-local-input" />
                    </div>
                  </div>
                </div>
                <div class="d-flex justify-content-between align-items-center mt-3">
                  <div class="record-meta mb-0">创建时间：{{ formatDateTime(currentUserRecord?.createdAt) }} | 更新时间：{{ formatDateTime(currentUserRecord?.updatedAt) }}</div>
                  <BButton type="submit" variant="primary">保存基本信息</BButton>
                </div>
              </BForm>
            </div>

            <div id="user-login-setting" class="info-card">
              <div class="section-title">登录设置</div>
              <div class="row g-3">
                <div class="col-lg-6">
                  <div class="detail-card h-100">
                    <div class="record-meta mb-3">密码登录：{{ userDetail?.passwordCredential ? '已启用' : '未配置' }}</div>
                    <BForm @submit.prevent="resetUserPassword">
                      <BFormInput v-model="userAdminForm.password" type="password" placeholder="新密码" class="mb-2" />
                      <BButton type="submit" variant="outline-primary" size="sm">重置密码</BButton>
                    </BForm>
                  </div>
                </div>
                <div class="col-lg-6">
                  <div class="detail-card h-100">
                    <div class="d-flex justify-content-between align-items-center mb-2">
                      <div class="record-meta mb-0">通行密钥数：{{ loginSecureKeys.length }} · {{ webauthnLoginEnabled ? '已启用登录' : '已关闭登录' }}</div>
                      <div class="d-flex gap-2">
                        <BButton size="sm" :variant="webauthnLoginEnabled ? 'outline-danger' : 'outline-secondary'" :disabled="!loginSecureKeys.length" @click="toggleWebAuthnLogin(!webauthnLoginEnabled)">
                          {{ webauthnLoginEnabled ? '关闭登录' : '启用登录' }}
                        </BButton>
                        <BButton size="sm" variant="outline-primary" @click="registerSecureKey('webauthn')">注册通行密钥</BButton>
                      </div>
                    </div>
                    <div v-if="!loginSecureKeys.length" class="record-meta">当前没有通行密钥，注册后才可启用通行密钥登录。</div>
                    <div v-for="secureKey in loginSecureKeys" :key="secureKey.id" class="record-row">
                      <div>
                        <strong>{{ secureKey.identifier || '通行密钥' }}</strong>
                        <div class="record-meta">{{ secureKey.publicKeyId }}</div>
                      </div>
                      <code>{{ formatDateTime(secureKey.createdAt) }}</code>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div id="user-binding" class="info-card">
              <div class="section-title">账号绑定</div>
              <div class="record-meta mb-3">绑定外部 OAuth/OIDC 身份的 UUID/Subject，用于登录识别，不会自动创建用户。</div>
              <div class="row g-3">
                <div class="col-lg-7">
                  <div v-if="!(userDetail?.bindings?.length ?? 0)" class="detail-card">
                    <div class="record-meta">当前没有第三方身份绑定。</div>
                  </div>
                  <div v-for="binding in userDetail?.bindings || []" :key="binding.id" class="record-card mb-2">
                    <div class="record-head">
                      <strong>{{ binding.providerName || binding.externalIdpId }}</strong>
                      <code>{{ binding.subject }}</code>
                    </div>
                    <div class="record-meta">Issuer：{{ binding.issuer }}</div>
                    <div class="record-meta">绑定时间：{{ formatDateTime(binding.createdAt) }}</div>
                    <div class="record-actions">
                      <BButton size="sm" variant="outline-danger" @click="deleteExternalBinding(binding.id)">解绑</BButton>
                    </div>
                  </div>
                </div>
                <div class="col-lg-5">
                  <BForm @submit.prevent="createExternalBinding">
                    <label class="form-label">外部 IdP</label>
                    <BFormSelect v-model="externalBindingForm.externalIdpId" class="mb-2" @update:model-value="syncExternalBindingIssuer">
                      <option v-for="item in userDetail?.externalIdps || []" :key="item.id" :value="item.id">{{ item.name }}</option>
                    </BFormSelect>
                    <label class="form-label">Issuer</label>
                    <BFormInput v-model="externalBindingForm.issuer" class="mb-2" />
                    <label class="form-label">Subject / UUID</label>
                    <BFormInput v-model="externalBindingForm.subject" class="mb-3" />
                    <BButton type="submit" variant="primary" size="sm">新增绑定</BButton>
                  </BForm>
                </div>
              </div>
            </div>

            <div id="user-mfa" class="info-card">
              <div class="section-title">两步验证</div>
              <div class="record-list">
                <div v-for="item in userMFAMethodRows" :key="item.id" class="record-row">
                  <div>
                    <strong>{{ item.label }}</strong>
                    <div class="record-meta">{{ item.summary }}</div>
                  </div>
                  <BButton
                    v-if="item.id === 'email_code' || item.id === 'sms_code'"
                    size="sm"
                    :variant="item.enabled ? 'outline-danger' : 'outline-primary'"
                    @click="handleInlineMFAMethodAction(item)"
                  >
                    {{ item.enabled ? '关闭' : '开启' }}
                  </BButton>
                  <BButton v-else size="sm" variant="outline-primary" @click="openMFAModal(item.id)">{{ item.enabled ? '配置' : '开启' }}</BButton>
                </div>
              </div>
            </div>

            <div id="user-session" class="info-card">
              <div class="section-title">会话管理</div>
              <div class="d-flex justify-content-end mb-3">
                <BButton variant="outline-danger" size="sm" @click="revokeAllUserSessions">吊销全部 Session</BButton>
              </div>
              <div v-if="!userDeviceList.length" class="detail-card">
                <div class="record-meta">当前没有设备记录。</div>
              </div>
              <div v-for="device in userDeviceList" :key="device.id" class="record-card mb-2">
                <div class="record-head">
                  <strong>{{ device.label }}</strong>
                  <div class="d-flex align-items-center gap-2">
                    <span class="badge text-bg-success" v-if="device.online">在线</span>
                    <span class="badge text-bg-secondary" v-else>离线</span>
                    <span class="badge text-bg-primary" v-if="device.trusted">可信</span>
                    <span class="badge text-bg-light text-dark border" v-else>非可信</span>
                  </div>
                </div>
                <div class="record-meta">上次登录 IP：{{ formatIPLine(device.ipAddress, device.ipLocation) }}</div>
                <div class="record-meta">上次登录时间：{{ formatDateTime(device.lastLoginAt) }}</div>
                <div class="record-meta">初次登录日期：{{ formatDateTime(device.firstLoginAt) }}</div>
                <div v-if="device.fingerprint" class="record-meta">设备指纹：{{ device.fingerprint }}</div>
                <div v-if="device.trusted" class="record-actions">
                  <BButton size="sm" variant="outline-danger" @click="untrustManagedDevice(device.id)">取消可信</BButton>
                </div>
              </div>
            </div>

            <div id="user-role-assignment" class="info-card">
              <div class="section-title">角色分配</div>
              <div class="record-meta mb-3">用户表中的角色以标签数组保存，若角色标签未来被删除，则自动忽略。</div>
              <div class="row g-2 mb-3">
                <div v-for="role in userAssignableRoles" :key="role.id" class="col-md-6 col-xl-4">
                  <label class="detail-card d-flex align-items-center gap-2 mb-0">
                    <input class="form-check-input mt-0" type="checkbox" :checked="userRoleAssignments.includes(role.name)" @change="toggleUserRole(role.name, ($event.target as HTMLInputElement).checked)" />
                    <span>
                      <strong>{{ role.name }}</strong>
                      <span class="record-meta d-block">{{ role.description || '无描述' }}</span>
                    </span>
                  </label>
                </div>
              </div>
              <BButton variant="primary" size="sm" @click="updateUser">保存角色分配</BButton>
            </div>

            <div id="user-danger-zone" class="info-card">
              <div class="section-title text-danger">危险区</div>
              <div class="record-meta mb-3">以下操作会直接影响该用户的凭据、会话与访问状态，请谨慎执行。</div>
              <div class="d-flex gap-2 flex-wrap mb-3">
                <BButton v-if="currentUserRecord?.status !== 'disabled'" variant="outline-warning" size="sm" @click="disableUser">停用用户</BButton>
                <BButton v-else variant="outline-success" size="sm" @click="enableUser">启用用户</BButton>
              </div>
              <div class="d-flex gap-2 flex-wrap">
                <BButton variant="outline-warning" size="sm" @click="resetUserUkid">轮换用户主密钥</BButton>
                <BButton variant="outline-warning" size="sm" @click="rotateUserToken">轮换用户主 Token</BButton>
                <BButton variant="outline-danger" size="sm" @click="revokeAllUserSessions">吊销全部 Session</BButton>
                <BButton variant="outline-danger" size="sm" @click="deleteSingleUser(selectedUserId)">删除用户</BButton>
              </div>
              <div class="detail-card mt-3">
                <div class="record-meta">最近管理员动作结果：{{ formatAdminResult(userAdminResult) }}</div>
              </div>
            </div>
          </div>
          <aside class="console-module-aside">
            <div class="info-card">
              <div class="section-title">最近变更</div>
              <div class="record-list">
                <div v-for="item in moduleRecentChanges" :key="item.id" class="record-row">
                  <div>
                    <strong>{{ item.eventType }}</strong>
                    <div class="record-meta">{{ item.result }}</div>
                  </div>
                  <code>{{ formatDateTime(item.createdAt) }}</code>
                </div>
              </div>
            </div>
          </aside>
        </div>
      </section>

      <section v-else-if="tab === 'role'" class="console-module-shell">
        <div class="console-module-summary-card">
          <div class="console-module-hero">
            <div class="console-module-hero-copy">
              <button type="button" class="console-back-button" @click="backToRoleList" aria-label="返回角色列表">
                <i class="bi bi-arrow-left console-back-button-icon" aria-hidden="true"></i>
              </button>
              <div>
                <div class="console-module-eyebrow">{{ currentTabLabel }}</div>
                <h2 class="console-module-title">{{ currentModuleEntityTitle }}</h2>
                <p class="console-module-subtitle">{{ currentModuleSummaryText }}</p>
              </div>
            </div>
            <BButton variant="primary" @click="runModuleAction">{{ currentModuleActionLabel }}</BButton>
          </div>
          <div class="console-module-metrics" :class="currentModuleMetricsClass">
            <div v-for="item in currentModuleMetrics" :key="item.label" class="console-module-metric">
              <span class="console-module-metric-label">{{ item.label }}</span>
              <div class="console-module-metric-value-row">
                <strong class="console-module-metric-value">{{ item.value }}</strong>
                <button
                  v-if="item.copyable"
                  type="button"
                  class="console-module-metric-copy"
                  :aria-label="`复制${item.label}`"
                  @click="copyMetricValue(item.copyValue || item.value)"
                >
                  <i class="bi bi-copy" aria-hidden="true"></i>
                </button>
              </div>
            </div>
          </div>
        </div>
        <div class="console-module-workspace">
          <aside class="console-module-sidebar">
            <button v-for="item in currentModulePanels" :key="item.id" type="button" class="console-module-sidebar-link" @click="scrollToPanel(item.id)">{{ item.label }}</button>
          </aside>
          <div class="console-module-main">
            <div id="role-list" class="info-card">
              <div class="section-title">角色标签列表</div>
              <div class="record-meta mb-3">角色是策略组。用户只能分配用户角色，应用只能分配应用角色。</div>
              <div class="record-list">
                <div v-for="role in roles" :key="role.id" class="record-card" :class="{ 'record-card-active': role.id === selectedRoleId }">
                  <div class="record-head">
                    <strong>{{ role.name }}</strong>
                    <code>{{ role.type === 'application' ? '应用角色' : '用户角色' }}</code>
                  </div>
                  <div class="record-meta">{{ role.description || 'no description' }}</div>
                  <div class="record-meta">Role ID: {{ role.id }}</div>
                  <div class="record-meta">策略数：{{ policies.filter((item) => item.roleId === role.id).length }}</div>
                  <div class="record-actions">
                    <BButton size="sm" variant="outline-primary" @click="selectRole(role)">查看详情</BButton>
                  </div>
                </div>
              </div>
            </div>
            <div id="role-detail" class="info-card">
              <div class="section-title">角色详情</div>
              <div class="record-meta mb-3">当前详情区只展示一个角色，点击左侧列表即可切换。</div>
              <div v-if="selectedRole" class="detail-card mb-3">
                <div class="record-meta">角色 ID：{{ selectedRole.id }}</div>
                <div class="record-meta">角色标签：{{ selectedRole.name || '-' }}</div>
                <div class="record-meta">角色类型：{{ selectedRole.type === 'application' ? '应用角色' : '用户角色' }}</div>
                <div class="record-meta">描述：{{ selectedRole.description || '-' }}</div>
                <div class="record-meta">策略数：{{ selectedRolePolicies.length }}</div>
                <div class="record-meta">创建时间：{{ formatDateTime(selectedRole.createdAt) }}</div>
                <div class="record-meta">更新时间：{{ formatDateTime(selectedRole.updatedAt) }}</div>
              </div>
              <BForm v-if="selectedRole" @submit.prevent="updateRole">
                <BFormInput v-model="roleForm.name" placeholder="role label" class="mb-2" />
                <BFormSelect v-model="roleForm.type" :options="roleTypeOptions" class="mb-2" />
                <BFormInput v-model="roleForm.description" placeholder="description" class="mb-2" />
                <BButton type="submit" variant="outline-primary">保存角色</BButton>
              </BForm>
            </div>
            <div id="policy-list" class="info-card">
          <div class="section-title">策略列表</div>
          <div class="record-list">
            <div v-for="policy in selectedRolePolicies" :key="policy.id" class="record-card">
              <div class="record-head">
                <strong>{{ policy.name }}</strong>
                <code>{{ policy.effect }} · {{ policy.priority }}</code>
              </div>
              <div class="record-meta">Policy ID：{{ policy.id }}</div>
              <div class="record-meta">API Rules：{{ formatPolicyRules(policy.apiRules) }}</div>
              <div class="record-actions">
                <BButton size="sm" variant="outline-primary" @click="editPolicy(policy)">编辑</BButton>
                <BButton size="sm" variant="outline-danger" @click="deletePolicy(policy.id)">删除</BButton>
              </div>
            </div>
            <div v-if="selectedRolePolicies.length === 0" class="detail-card">
              <div class="record-meta">当前角色还没有策略。</div>
            </div>
          </div>
            </div>
            <div id="policy-editor" class="info-card">
          <div class="section-title">策略编辑</div>
          <div class="record-meta mb-3">策略直接挂在角色下，`apiRules.path` 支持 `keyMatch2`。</div>
          <BForm @submit.prevent="savePolicy">
            <BFormInput v-model="policyForm.name" placeholder="policy name" class="mb-2" />
            <div class="row g-2 mb-2">
              <div class="col-md-4">
                <BFormSelect v-model="policyForm.effect" :options="[{ value: 'allow', text: 'allow' }, { value: 'deny', text: 'deny' }]" />
              </div>
              <div class="col-md-4">
                <BFormInput v-model="policyForm.priority" type="number" placeholder="priority" />
              </div>
            </div>
            <textarea v-model="policyForm.apiRulesText" class="form-control mb-2" rows="8" />
            <div class="d-flex gap-2">
              <BButton type="submit" variant="primary">{{ policyForm.id ? '保存策略' : '创建策略' }}</BButton>
              <BButton type="button" variant="outline-secondary" @click="resetPolicyForm">重置</BButton>
            </div>
          </BForm>
            </div>
            <div id="role-decision" class="info-card">
          <div class="section-title">Policy Check</div>
          <BForm @submit.prevent="evaluatePolicyCheck">
            <BFormSelect v-model="policyCheckForm.subjectType" :options="[{ value: 'application', text: 'application' }, { value: 'user', text: 'user' }]" class="mb-2" />
            <BFormInput v-model="policyCheckForm.subjectId" placeholder="subjectId" class="mb-2" />
            <BFormInput v-model="policyCheckForm.method" placeholder="POST" class="mb-2" />
            <BFormInput v-model="policyCheckForm.path" placeholder="/api/manage/v1/organization/query" class="mb-2" />
            <BButton type="submit" variant="primary">检查</BButton>
          </BForm>
          <pre class="json-block mt-3">{{ JSON.stringify(decisionResult, null, 2) }}</pre>
            </div>
          </div>
          <aside class="console-module-aside">
            <div class="info-card">
              <div class="section-title">最近变更</div>
              <div class="record-list">
                <div v-for="item in moduleRecentChanges" :key="item.id" class="record-row">
                  <div>
                    <strong>{{ item.eventType }}</strong>
                    <div class="record-meta">{{ item.result }}</div>
                  </div>
                  <code>{{ formatDateTime(item.createdAt) }}</code>
                </div>
              </div>
            </div>
          </aside>
        </div>
      </section>

      <section v-else-if="tab === 'audit'" class="console-module-shell">
        <div class="console-module-summary-card">
          <div class="console-module-hero">
            <div>
              <div class="console-module-eyebrow">{{ currentTabLabel }}</div>
              <h2 class="console-module-title">{{ currentModuleEntityTitle }}</h2>
              <p class="console-module-subtitle">{{ currentModuleSummaryText }}</p>
            </div>
            <BButton variant="primary" @click="runModuleAction">{{ currentModuleActionLabel }}</BButton>
          </div>
          <div class="console-module-metrics" :class="currentModuleMetricsClass">
            <div v-for="item in currentModuleMetrics" :key="item.label" class="console-module-metric">
              <span class="console-module-metric-label">{{ item.label }}</span>
              <div class="console-module-metric-value-row">
                <strong class="console-module-metric-value">{{ item.value }}</strong>
                <button
                  v-if="item.copyable"
                  type="button"
                  class="console-module-metric-copy"
                  :aria-label="`复制${item.label}`"
                  @click="copyMetricValue(item.copyValue || item.value)"
                >
                  <i class="bi bi-copy" aria-hidden="true"></i>
                </button>
              </div>
            </div>
          </div>
        </div>
        <div class="console-module-workspace">
          <aside class="console-module-sidebar">
            <button v-for="item in currentModulePanels" :key="item.id" type="button" class="console-module-sidebar-link" @click="scrollToPanel(item.id)">{{ item.label }}</button>
          </aside>
          <div class="console-module-main">
        <div class="info-card">
          <div class="section-title">模块基础信息</div>
          <div class="detail-card">
            <div class="record-meta">当前组织：{{ currentOrganization?.name || currentOrganization?.id || '-' }}</div>
            <div class="record-meta">审计事件数量：{{ auditLogs.length }}</div>
            <div class="record-meta">这里聚合登录、失败、令牌签发/吊销、策略变更、发现导入、UKID 重置等关键事件。</div>
          </div>
        </div>
        <div id="audit-list" class="info-card">
          <div class="section-title">审计日志</div>
          <BButton size="sm" variant="outline-primary" class="mb-3" @click="loadAudit">刷新</BButton>
          <div class="record-list">
            <div v-for="item in auditLogs" :key="item.id" class="record-card">
              <div class="record-head">
                <strong>{{ item.eventType }}</strong>
                <code>{{ item.result }}</code>
              </div>
              <div class="record-meta">Actor: {{ item.actorType }} / {{ item.actorId || '-' }}</div>
              <div class="record-meta">Target: {{ item.targetType }} / {{ item.targetId || '-' }}</div>
              <div class="record-meta">组织: {{ item.organizationId || '-' }}</div>
              <div class="record-meta">IP: {{ formatIPLine(item.ipAddress, item.ipLocation) }}</div>
              <div class="record-meta">时间: {{ formatDateTime(item.createdAt) }}</div>
            </div>
          </div>
        </div>
          </div>
          <aside class="console-module-aside">
            <div class="info-card">
              <div class="section-title">最近变更</div>
              <div class="record-list">
                <div v-for="item in moduleRecentChanges" :key="item.id" class="record-row">
                  <div>
                    <strong>{{ item.eventType }}</strong>
                    <div class="record-meta">{{ item.result }}</div>
                  </div>
                  <code>{{ formatDateTime(item.createdAt) }}</code>
                </div>
              </div>
            </div>
          </aside>
        </div>
      </section>

      <section v-else class="console-module-layout">
        <aside class="console-module-sidebar">
          <button v-for="item in currentModulePanels" :key="item.id" type="button" class="console-module-sidebar-link" @click="scrollToPanel(item.id)">{{ item.label }}</button>
        </aside>
        <div class="console-module-main">
          <div id="setting-basic" class="info-card">
            <div class="section-title">基本设置</div>
            <BForm @submit.prevent="saveOrganizationBasicSettings">
              <div class="row g-3">
                <div class="col-md-6">
                  <label class="form-label">组织名称</label>
                  <BFormInput v-model="organizationBasicSettingForm.name" />
                </div>
                <div class="col-md-6">
                  <label class="form-label">企业支持邮箱</label>
                  <BFormInput v-model="organizationBasicSettingForm.supportEmail" type="email" />
                </div>
                <div class="col-md-6">
                  <label class="form-label">ToS 地址</label>
                  <BFormInput v-model="organizationBasicSettingForm.tosUrl" />
                </div>
                <div class="col-md-6">
                  <label class="form-label">隐私策略地址</label>
                  <BFormInput v-model="organizationBasicSettingForm.privacyPolicyUrl" />
                </div>
                <div class="col-12">
                  <label class="form-label">组织 Logo 地址</label>
                  <BFormInput v-model="organizationBasicSettingForm.logoUrl" />
                </div>
              </div>
              <div class="d-flex justify-content-end mt-3">
                <BButton type="submit" variant="primary">保存基本设置</BButton>
              </div>
            </BForm>
          </div>

          <div id="setting-domain" class="info-card">
            <div class="section-title">域名设置</div>
            <div class="record-meta mb-3">绑定登录域名并记录当前验证状态。</div>
            <div v-for="(item, index) in organizationDomainRows" :key="item.id" class="row g-2 align-items-center mb-2">
              <div class="col-md-6">
                <BFormInput v-model="item.host" placeholder="login.example.com" />
              </div>
              <div class="col-md-2">
                <span class="badge" :class="item.verified ? 'text-bg-success' : 'text-bg-secondary'">{{ item.verified ? '已验证' : '未验证' }}</span>
              </div>
              <div class="col-md-4 d-flex gap-2">
                <BButton type="button" size="sm" variant="outline-primary" @click="verifyOrganizationDomain(index)">验证域名</BButton>
                <BButton type="button" size="sm" variant="outline-danger" @click="removeOrganizationDomainRow(index)">删除</BButton>
              </div>
            </div>
            <div class="d-flex justify-content-between align-items-center mt-3">
              <BButton type="button" variant="outline-secondary" @click="addOrganizationDomainRow">添加域名</BButton>
              <BButton type="button" variant="primary" @click="saveOrganizationDomainSettings">保存域名设置</BButton>
            </div>
          </div>

          <div id="setting-login-policy" class="info-card">
            <div class="section-title">登录策略设置</div>
            <BForm @submit.prevent="saveOrganizationLoginPolicy">
              <div class="row g-3">
                <div class="col-md-6">
                  <div class="form-check">
                    <input id="setting-password-login-enabled" v-model="organizationLoginPolicyForm.passwordLoginEnabled" class="form-check-input" type="checkbox" />
                    <label class="form-check-label" for="setting-password-login-enabled">启用密码登录</label>
                  </div>
                </div>
                <div class="col-md-6">
                  <div class="form-check">
                    <input id="setting-webauthn-login-enabled" v-model="organizationLoginPolicyForm.webauthnLoginEnabled" class="form-check-input" type="checkbox" />
                    <label class="form-check-label" for="setting-webauthn-login-enabled">启用通行密钥登录</label>
                  </div>
                </div>
                <div class="col-md-4">
                  <div class="form-check mb-2">
                    <input id="setting-allow-username" v-model="organizationLoginPolicyForm.allowUsername" class="form-check-input" type="checkbox" />
                    <label class="form-check-label" for="setting-allow-username">支持用户名</label>
                  </div>
                  <BFormSelect v-model="organizationLoginPolicyForm.usernameMode" :options="fieldVisibilityOptions" />
                </div>
                <div class="col-md-4">
                  <div class="form-check mb-2">
                    <input id="setting-allow-email" v-model="organizationLoginPolicyForm.allowEmail" class="form-check-input" type="checkbox" />
                    <label class="form-check-label" for="setting-allow-email">支持邮箱</label>
                  </div>
                  <BFormSelect v-model="organizationLoginPolicyForm.emailMode" :options="fieldVisibilityOptions" />
                </div>
                <div class="col-md-4">
                  <div class="form-check mb-2">
                    <input id="setting-allow-phone" v-model="organizationLoginPolicyForm.allowPhone" class="form-check-input" type="checkbox" />
                    <label class="form-check-label" for="setting-allow-phone">支持手机</label>
                  </div>
                  <BFormSelect v-model="organizationLoginPolicyForm.phoneMode" :options="fieldVisibilityOptions" />
                </div>
              </div>
              <div class="d-flex justify-content-end mt-3">
                <BButton type="submit" variant="primary">保存登录策略</BButton>
              </div>
            </BForm>
          </div>

          <div id="setting-password-policy" class="info-card">
            <div class="section-title">密码策略设置</div>
            <BForm @submit.prevent="saveOrganizationPasswordPolicy">
              <div class="row g-3">
                <div class="col-md-4">
                  <label class="form-label">最少位数</label>
                  <BFormInput v-model="organizationPasswordPolicyForm.minLength" type="number" min="6" />
                </div>
                <div class="col-md-4">
                  <div class="form-check mt-4">
                    <input id="setting-password-uppercase" v-model="organizationPasswordPolicyForm.requireUppercase" class="form-check-input" type="checkbox" />
                    <label class="form-check-label" for="setting-password-uppercase">必须包含大写字母</label>
                  </div>
                </div>
                <div class="col-md-4">
                  <div class="form-check mt-4">
                    <input id="setting-password-lowercase" v-model="organizationPasswordPolicyForm.requireLowercase" class="form-check-input" type="checkbox" />
                    <label class="form-check-label" for="setting-password-lowercase">必须包含小写字母</label>
                  </div>
                </div>
                <div class="col-md-4">
                  <div class="form-check">
                    <input id="setting-password-number" v-model="organizationPasswordPolicyForm.requireNumber" class="form-check-input" type="checkbox" />
                    <label class="form-check-label" for="setting-password-number">必须包含数字</label>
                  </div>
                </div>
                <div class="col-md-4">
                  <div class="form-check">
                    <input id="setting-password-symbol" v-model="organizationPasswordPolicyForm.requireSymbol" class="form-check-input" type="checkbox" />
                    <label class="form-check-label" for="setting-password-symbol">必须包含特殊符号</label>
                  </div>
                </div>
                <div class="col-md-4">
                  <div class="form-check">
                    <input id="setting-password-expires" v-model="organizationPasswordPolicyForm.passwordExpires" class="form-check-input" type="checkbox" />
                    <label class="form-check-label" for="setting-password-expires">密码过期</label>
                  </div>
                </div>
                <div class="col-md-4">
                  <label class="form-label">过期时间（天）</label>
                  <BFormInput v-model="organizationPasswordPolicyForm.expiryDays" type="number" min="1" :disabled="!organizationPasswordPolicyForm.passwordExpires" />
                </div>
              </div>
              <div class="d-flex justify-content-end mt-3">
                <BButton type="submit" variant="primary">保存密码策略</BButton>
              </div>
            </BForm>
          </div>

          <div id="setting-mfa-policy" class="info-card">
            <div class="section-title">两步验证策略</div>
            <BForm @submit.prevent="saveOrganizationMFAPolicy">
              <div class="row g-3">
                <div class="col-12">
                  <div class="form-check">
                    <input id="setting-mfa-required" v-model="organizationMFAPolicyForm.requireForAllUsers" class="form-check-input" type="checkbox" />
                    <label class="form-check-label" for="setting-mfa-required">强制所有用户启用两步验证</label>
                  </div>
                </div>
                <div class="col-md-4"><div class="form-check"><input id="setting-mfa-webauthn" v-model="organizationMFAPolicyForm.allowWebauthn" class="form-check-input" type="checkbox" /><label class="form-check-label" for="setting-mfa-webauthn">通行密钥</label></div></div>
                <div class="col-md-4"><div class="form-check"><input id="setting-mfa-totp" v-model="organizationMFAPolicyForm.allowTotp" class="form-check-input" type="checkbox" /><label class="form-check-label" for="setting-mfa-totp">身份验证器（TOTP）</label></div></div>
                <div class="col-md-4"><div class="form-check"><input id="setting-mfa-email" v-model="organizationMFAPolicyForm.allowEmailCode" class="form-check-input" type="checkbox" /><label class="form-check-label" for="setting-mfa-email">邮箱验证码</label></div></div>
                <div class="col-md-4"><div class="form-check"><input id="setting-mfa-sms" v-model="organizationMFAPolicyForm.allowSmsCode" class="form-check-input" type="checkbox" /><label class="form-check-label" for="setting-mfa-sms">手机验证码</label></div></div>
                <div class="col-md-4"><div class="form-check"><input id="setting-mfa-u2f" v-model="organizationMFAPolicyForm.allowU2f" class="form-check-input" type="checkbox" /><label class="form-check-label" for="setting-mfa-u2f">安全密钥</label></div></div>
                <div class="col-md-4"><div class="form-check"><input id="setting-mfa-recovery" v-model="organizationMFAPolicyForm.allowRecoveryCode" class="form-check-input" type="checkbox" /><label class="form-check-label" for="setting-mfa-recovery">备用验证码</label></div></div>
                <div class="col-12">
                  <div class="detail-card h-100">
                    <div class="d-flex align-items-center justify-content-between mb-3">
                      <div>
                        <div class="section-subtitle mb-1">邮箱验证码通道</div>
                        <div class="record-meta">用于组织级邮箱验证码发送配置。</div>
                      </div>
                      <div class="form-check m-0">
                        <input id="setting-mfa-email-channel-enabled" v-model="organizationMFAPolicyForm.emailChannelEnabled" class="form-check-input" type="checkbox" />
                        <label class="form-check-label ms-2" for="setting-mfa-email-channel-enabled">启用 SMTP</label>
                      </div>
                    </div>
                    <div class="row g-3">
                      <div class="col-md-6">
                        <label class="form-label">发件人邮箱</label>
                        <BFormInput v-model="organizationMFAPolicyForm.emailChannelFrom" type="email" />
                      </div>
                      <div class="col-md-6">
                        <label class="form-label">SMTP 主机</label>
                        <BFormInput v-model="organizationMFAPolicyForm.emailChannelHost" />
                      </div>
                      <div class="col-md-4">
                        <label class="form-label">SMTP 端口</label>
                        <BFormInput v-model="organizationMFAPolicyForm.emailChannelPort" type="number" min="1" />
                      </div>
                      <div class="col-md-4">
                        <label class="form-label">SMTP 用户名</label>
                        <BFormInput v-model="organizationMFAPolicyForm.emailChannelUsername" />
                      </div>
                      <div class="col-md-4">
                        <label class="form-label">SMTP 密码</label>
                        <BFormInput v-model="organizationMFAPolicyForm.emailChannelPassword" type="password" />
                      </div>
                    </div>
                  </div>
                </div>
              </div>
              <div class="d-flex justify-content-end mt-3">
                <BButton type="submit" variant="primary">保存两步验证策略</BButton>
              </div>
            </BForm>
          </div>

          <div id="setting-external-idp" class="info-card">
            <div class="section-title">外部 IdP 设置</div>
            <div class="record-meta mb-3">采用预置 Provider 模板，点击启用后在弹窗中配置应用参数。</div>
            <div class="record-list">
              <div v-for="item in externalIDPProviderRows" :key="item.id" class="record-row">
                <div>
                  <strong>{{ item.label }}</strong>
                  <div class="record-meta">{{ item.summary }}</div>
                </div>
                <BButton size="sm" variant="outline-primary" @click="openExternalIDPModal(item.id)">{{ item.enabled ? '配置' : (item.id.startsWith('custom_') ? '添加' : '启用') }}</BButton>
              </div>
            </div>
            <div v-if="customExternalIDPs.length" class="detail-card mt-3">
              <div class="record-meta mb-2">已添加的自定义 Provider</div>
              <div v-for="item in customExternalIDPs" :key="item.id" class="record-row">
                <div>
                  <strong>{{ item.name }}</strong>
                  <div class="record-meta">{{ String(item.protocol || '').toUpperCase() }} · {{ item.issuer || '-' }}</div>
                </div>
                <BButton size="sm" variant="outline-primary" @click="openExistingExternalIDP(item)">配置</BButton>
              </div>
            </div>
          </div>
        </div>
      </section>
      <div v-if="showBackToTopButton" class="console-back-to-top-wrap" :class="backToTopWrapClass">
        <button type="button" class="console-back-to-top" @click="scrollToTop" aria-label="回到顶部">
          <i class="bi bi-arrow-up" aria-hidden="true"></i>
        </button>
      </div>
      <BModal v-model="externalIDPConfigModalVisible" :title="currentExternalIDPModalTitle" centered>
        <BForm @submit.prevent>
            <div class="row g-3">
            <div class="col-md-4">
              <label class="form-label">协议</label>
              <BFormInput :model-value="currentExternalIDPProtocol.toUpperCase()" disabled />
            </div>
            <div class="col-md-6">
              <label class="form-label">名称</label>
              <BFormInput v-model="externalIDPForm.name" />
            </div>
            <div class="col-md-6">
              <label class="form-label">Issuer</label>
              <BFormInput v-model="externalIDPForm.issuer" />
            </div>
            <div class="col-md-6">
              <label class="form-label">Client ID / App Key</label>
              <BFormInput v-model="externalIDPForm.clientId" />
            </div>
            <div class="col-md-6">
              <label class="form-label">Client Secret / App Secret</label>
              <BFormInput v-model="externalIDPForm.clientSecret" type="password" placeholder="留空则保持原值" />
            </div>
            <div class="col-md-12">
              <label class="form-label">Scopes</label>
              <BFormInput v-model="externalIDPForm.scopes" />
            </div>
            <div class="col-md-6">
              <label class="form-label">Authorization URL</label>
              <BFormInput v-model="externalIDPForm.authorizationUrl" />
            </div>
            <div class="col-md-6">
              <label class="form-label">Token URL</label>
              <BFormInput v-model="externalIDPForm.tokenUrl" />
            </div>
            <div class="col-md-6">
              <label class="form-label">UserInfo URL</label>
              <BFormInput v-model="externalIDPForm.userInfoUrl" />
            </div>
            <div v-if="currentExternalIDPProtocol === 'oidc'" class="col-md-6">
              <label class="form-label">JWKS URL</label>
              <BFormInput v-model="externalIDPForm.jwksUrl" />
            </div>
          </div>
        </BForm>
        <template #footer>
          <div class="d-flex justify-content-end gap-2 w-100">
            <BButton type="button" variant="outline-secondary" @click="externalIDPConfigModalVisible = false">关闭</BButton>
            <BButton type="button" variant="primary" @click="submitExternalIDPConfig">{{ currentExternalIDPActionLabel }}</BButton>
          </div>
        </template>
      </BModal>
      <BModal v-model="mfaConfigModalVisible" :title="currentMFAModalTitle" centered>
        <template v-if="currentMFAMethod === 'totp'">
          <div v-for="item in activeTOTPEnrollments.slice(0, 1)" :key="item.id" class="record-row mb-3">
            <div>
              <strong>{{ item.label || '身份验证器（TOTP）' }}</strong>
              <div class="record-meta">最近使用：{{ formatDateTime(item.lastUsedAt) }}</div>
            </div>
            <code>{{ item.status }}</code>
          </div>
          <div v-if="!activeTOTPEnrollments.length" class="record-meta mb-3">当前没有已激活的身份验证器。</div>
          <div v-if="activeTOTPEnrollments.length" class="d-flex gap-2 mb-3">
            <BButton size="sm" variant="outline-danger" @click="deleteTotpEnrollments">关闭</BButton>
          </div>
          <div v-if="totpSetup && !activeTOTPEnrollments.length" class="detail-card mb-3">
            <div v-if="totpQRCodeDataURL" class="text-center mb-3">
              <img :src="totpQRCodeDataURL" alt="身份验证器二维码" class="img-fluid border rounded p-2 bg-white" />
            </div>
            <div class="record-meta">待激活 Enrollment：{{ pendingTotpEnrollmentId || '-' }}</div>
            <div class="record-meta">手动输入密钥：{{ pendingTotpManualEntryKey || '-' }}</div>
          </div>
          <BForm v-if="!activeTOTPEnrollments.length" @submit.prevent>
            <BFormInput v-model="totpVerifyForm.code" placeholder="6 位验证码" class="mb-3" />
          </BForm>
        </template>

        <template v-else-if="currentMFAMethod === 'email_code'">
          <div class="record-meta mb-3">邮箱地址来自基本信息页。这里只控制邮箱验证码是否启用。</div>
          <BForm @submit.prevent>
            <div class="mb-3">
              <div class="record-meta mb-2">当前邮箱：{{ currentUserRecord?.email || '未配置邮箱' }}</div>
              <BFormSelect v-model="mfaSettingForm.emailEnabled" :options="booleanSettingOptions" />
            </div>
          </BForm>
        </template>

        <template v-else-if="currentMFAMethod === 'sms_code'">
          <div class="record-meta mb-3">手机号来自基本信息页。这里只控制手机验证码是否启用。</div>
          <BForm @submit.prevent>
            <div class="mb-3">
              <div class="record-meta mb-2">当前手机号：{{ currentUserRecord?.phoneNumber || '未配置手机号' }}</div>
              <BFormSelect v-model="mfaSettingForm.smsEnabled" :options="booleanSettingOptions" />
            </div>
          </BForm>
        </template>

        <template v-else-if="currentMFAMethod === 'u2f'">
          <div class="record-meta mb-3">当前已注册 {{ u2fSecureKeys.length }} 个安全密钥。</div>
          <div v-for="secureKey in u2fSecureKeys" :key="secureKey.id" class="record-row mb-2">
            <div>
              <strong>{{ secureKey.identifier || '安全密钥' }}</strong>
              <div class="record-meta">{{ secureKey.publicKeyId }}</div>
            </div>
            <div class="d-flex align-items-center gap-2">
              <code>{{ formatDateTime(secureKey.createdAt) }}</code>
              <BButton size="sm" variant="outline-danger" @click="deleteSecureKey(secureKey.id)">删除</BButton>
            </div>
          </div>
          <div v-if="!u2fSecureKeys.length" class="record-meta mb-3">当前没有已注册的安全密钥。</div>
        </template>

        <template v-else-if="currentMFAMethod === 'recovery_code'">
          <div class="record-meta mb-2">剩余有效码：{{ userDetail?.recoverySummary?.available ?? 0 }}</div>
          <div class="record-meta mb-3">上次生成时间：{{ formatDateTime(userDetail?.recoverySummary?.lastGeneratedAt) }}</div>
          <div class="record-meta mb-3">最近生成结果：{{ generatedRecoveryCodeList.length ? '已生成新备用验证码' : '未生成' }}</div>
          <div v-if="generatedRecoveryCodeList.length" class="detail-card mb-3">
            <div class="record-meta mb-2">请立即保存以下备用验证码，这些明文只会在生成后显示一次。</div>
            <div v-for="code in generatedRecoveryCodeList" :key="code" class="record-row">
              <code>{{ code }}</code>
            </div>
          </div>
        </template>
        <template #footer>
          <div class="d-flex justify-content-end gap-2 w-100">
            <BButton type="button" variant="outline-secondary" @click="mfaConfigModalVisible = false">关闭</BButton>
            <BButton type="button" variant="primary" @click="submitCurrentMFAModal">{{ currentMFAModalActionLabel }}</BButton>
          </div>
        </template>
      </BModal>

      <BModal v-model="applicationKeyModalVisible" :title="applicationKeyModalTitle" centered>
        <div class="record-meta mb-3">请立即保存以下私钥。系统不会再次展示该私钥。</div>
        <textarea class="form-control" rows="12" :value="applicationPrivateKeySnapshot" readonly />
      </BModal>
      <BModal v-model="projectUserAssignmentModalVisible" title="添加项目用户" size="xl" centered>
        <div class="d-flex justify-content-between align-items-center gap-2 mb-3">
          <div class="record-meta mb-0">支持多选和反选。保存后会同步项目 ACL。</div>
          <div class="d-flex gap-2">
            <BButton size="sm" variant="outline-secondary" @click="selectAllProjectAssignmentUsers">全选</BButton>
            <BButton size="sm" variant="outline-secondary" @click="invertProjectAssignmentUsers">反选</BButton>
            <BButton size="sm" variant="outline-secondary" @click="clearProjectAssignmentUsers">清空</BButton>
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
                    :checked="projectAssignmentDraftUserIds.includes(user.id)"
                    @change="toggleProjectAssignmentDraftUser(user.id, ($event.target as HTMLInputElement).checked)"
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
            <BButton type="button" variant="outline-secondary" @click="projectUserAssignmentModalVisible = false">取消</BButton>
            <BButton type="button" variant="primary" @click="confirmProjectUserAssignmentModal">确认添加</BButton>
          </div>
        </template>
      </BModal>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { BButton, BDropdown, BDropdownDivider, BDropdownItem, BForm, BFormCheckbox, BFormInput, BFormSelect, BModal } from 'bootstrap-vue-next'
import QRCode from 'qrcode'
import { apiPost } from '../api/client'
import { normalizeCreationOptions, serializeCredential } from '@shared/api/webauthn'
import ToastHost from '@shared/components/ToastHost.vue'
import { useToast } from '@shared/composables/toast'
import { startConsoleAuthorization, startConsoleLogout } from '../auth'

const router = useRouter()
const route = useRoute()
const apiBaseUrl = import.meta.env.PPVT_CONSOLE_API_BASE_URL ?? 'http://localhost:8090'
const authBaseUrl = import.meta.env.PPVT_CONSOLE_AUTH_BASE_URL ?? 'http://localhost:8091'
const consoleApplicationId = import.meta.env.PPVT_CONSOLE_APPLICATION_ID ?? ''
const toast = useToast()
const tab = ref<'dashboard' | 'organization' | 'project' | 'user' | 'role' | 'audit' | 'setting'>('dashboard')
const message = ref('')
const messageVariant = ref<'success' | 'danger'>('success')
const organizations = ref<any[]>([])
const projects = ref<any[]>([])
const applications = ref<any[]>([])
const users = ref<any[]>([])
const roles = ref<any[]>([])
const policies = ref<any[]>([])
const auditLogs = ref<any[]>([])
const externalIDPs = ref<any[]>([])
const decisionResult = ref<unknown>(null)
const totpSetup = ref<unknown>(null)
const totpQRCodeDataURL = ref('')
const recoveryCodes = ref<unknown>(null)
const userDetail = ref<any | null>(null)
const mfaConfigModalVisible = ref(false)
const currentMFAMethod = ref<MFAMethod>('totp')
const externalIDPConfigModalVisible = ref(false)
const currentExternalIDPKind = ref<'google' | 'github' | 'apple' | 'qq' | 'weibo' | 'custom_oauth' | 'custom_oidc'>('google')
const applicationKeyModalVisible = ref(false)
const projectUserAssignmentModalVisible = ref(false)
const applicationKeyModalTitle = ref('应用私钥')
const applicationPrivateKeySnapshot = ref('')
const selectedUserId = ref('')
const selectedProjectId = ref('')
const selectedApplicationId = ref('')
const selectedRoleId = ref('')
const currentOrganizationId = ref('')
const organizationSwitcher = ref('')
const currentLoginUser = ref('')
const projectViewMode = ref<'list' | 'detail'>('list')
const userViewMode = ref<'list' | 'detail'>('list')
const roleViewMode = ref<'list' | 'detail'>('list')
const selectedUserIds = ref<string[]>([])
const selectedRoleIds = ref<string[]>([])
const projectAssignedUserIds = ref<string[]>([])
const projectAssignmentDraftUserIds = ref<string[]>([])
const showCreateUserForm = ref(false)
const showCreateRoleForm = ref(false)
const organizationMetadataRows = ref<Array<{ id: string; key: string; value: string }>>([])
const userRoleAssignments = ref<string[]>([])

type MetricItem = {
  label: string
  value: string
  copyable?: boolean
  copyValue?: string
}

type PhoneInputState = {
  countryCode: string
  localNumber: string
}

type MFAMethod = 'totp' | 'email_code' | 'sms_code' | 'u2f' | 'recovery_code'
type OrganizationDomainRow = {
  id: string
  host: string
  verified: boolean
}

type OrganizationConsoleSettings = {
  tosUrl: string
  privacyPolicyUrl: string
  supportEmail: string
  logoUrl: string
  domains: Array<{ host: string; verified: boolean }>
  loginPolicy: {
    passwordLoginEnabled: boolean
    webauthnLoginEnabled: boolean
    allowUsername: boolean
    allowEmail: boolean
    allowPhone: boolean
    usernameMode: 'hidden' | 'optional' | 'required'
    emailMode: 'hidden' | 'optional' | 'required'
    phoneMode: 'hidden' | 'optional' | 'required'
  }
  passwordPolicy: {
    minLength: number
    requireUppercase: boolean
    requireLowercase: boolean
    requireNumber: boolean
    requireSymbol: boolean
    passwordExpires: boolean
    expiryDays: number
  }
  mfaPolicy: {
    requireForAllUsers: boolean
    allowWebauthn: boolean
    allowTotp: boolean
    allowEmailCode: boolean
    allowSmsCode: boolean
    allowU2f: boolean
    allowRecoveryCode: boolean
    emailChannel: {
      enabled: boolean
      from: string
      host: string
      port: number
      username: string
      password: string
    }
  }
}

const organizationForm = reactive({ name: '' })
const organizationUpdateForm = reactive({ id: '', name: '', metadata: {} as Record<string, string> })
const projectForm = reactive({ organizationId: '', name: '', userAclEnabled: false })
const projectUpdateForm = reactive({ id: '', name: '', description: '', userAclEnabled: false })
const applicationForm = reactive({
  projectId: '',
  name: '',
  redirectUris: '',
  applicationType: 'web',
  tokenType: ['access_token'] as string[],
  enableRefreshToken: false,
  grantType: ['authorization_code_pkce'] as string[],
  clientAuthenticationType: 'none',
  roles: [] as string[],
  publicKey: '',
  accessTokenTTLMinutes: 10,
  refreshTokenTTLHours: 168
})
const applicationUpdateForm = reactive({
  id: '',
  name: '',
  redirectUris: '',
  applicationType: 'web',
  tokenType: ['access_token'] as string[],
  enableRefreshToken: false,
  grantType: ['authorization_code_pkce'] as string[],
  clientAuthenticationType: 'none',
  roles: [] as string[],
  publicKey: '',
  accessTokenTTLMinutes: 10,
  refreshTokenTTLHours: 168
})
const applicationTypeOptions = [
  { value: 'web', text: 'Web' },
  { value: 'native', text: 'Native' },
  { value: 'api', text: 'API' }
]
const tokenTypeOptions = [
  { value: 'access_token', text: 'access_token' },
  { value: 'id_token', text: 'id_token' }
]
const grantTypeOptions = [
  { value: 'authorization_code', text: 'authorization_code' },
  { value: 'authorization_code_pkce', text: 'authorization_code_pkce' },
  { value: 'client_credentials', text: 'client_credentials' },
  { value: 'device_code', text: 'device_code' },
  { value: 'implicit', text: 'implicit' },
  { value: 'password', text: 'password' }
]
const clientAuthenticationTypeOptions = [
  { value: 'none', text: 'none' },
  { value: 'client_secret_basic', text: 'client_secret_basic' },
  { value: 'client_secret_post', text: 'client_secret_post' },
  { value: 'client_secret_jwt', text: 'client_secret_jwt' },
  { value: 'private_key_jwt', text: 'private_key_jwt' },
  { value: 'tls_client_auth', text: 'tls_client_auth' },
  { value: 'self_signed_tls_client_auth', text: 'self_signed_tls_client_auth' }
]
const applicationProtocolTemplates: Record<string, { text: string; allowedTypes: string[]; grantType: string[]; enableRefreshToken: boolean; tokenType: string[]; clientAuthenticationType: string }> = {
  'oauth21-oidc-pkce-private-key-jwt': {
    text: 'OAuth2.1 + OIDC 1.0 + Private Key JWT（高安全性）',
    allowedTypes: ['web'],
    grantType: ['authorization_code_pkce'],
    enableRefreshToken: false,
    tokenType: ['access_token'],
    clientAuthenticationType: 'private_key_jwt'
  },
  'oauth21-oidc-pkce-client-secret-basic': {
    text: 'OAuth2.1 + OIDC 1.0 + Client Secret Basic（高安全性）',
    allowedTypes: ['web'],
    grantType: ['authorization_code_pkce'],
    enableRefreshToken: false,
    tokenType: ['access_token'],
    clientAuthenticationType: 'client_secret_basic'
  },
  'oauth21-oidc-pkce-none': {
    text: 'OAuth2.1 + OIDC 1.0（中高安全性）',
    allowedTypes: ['web', 'native'],
    grantType: ['authorization_code_pkce'],
    enableRefreshToken: false,
    tokenType: ['access_token'],
    clientAuthenticationType: 'none'
  },
  'oauth20-oidc-auth-code-private-key-jwt': {
    text: 'OAuth2.0 + OIDC 1.0 + Private Key JWT（中高安全性）',
    allowedTypes: ['web'],
    grantType: ['authorization_code'],
    enableRefreshToken: false,
    tokenType: ['access_token'],
    clientAuthenticationType: 'private_key_jwt'
  },
  'oauth20-oidc-auth-code-client-secret-basic': {
    text: 'OAuth2.0 + OIDC 1.0 + Client Secret Basic（中高安全性）',
    allowedTypes: ['web'],
    grantType: ['authorization_code'],
    enableRefreshToken: false,
    tokenType: ['access_token'],
    clientAuthenticationType: 'client_secret_basic'
  },
  'oauth20-device-code': {
    text: 'OAuth2.0 + Device Code（中安全性）',
    allowedTypes: ['native'],
    grantType: ['device_code'],
    enableRefreshToken: false,
    tokenType: ['access_token'],
    clientAuthenticationType: 'none'
  },
  'oauth20-oidc-client-credentials': {
    text: 'OAuth2.0 + OIDC 1.0 + Client Credentials（中安全性）',
    allowedTypes: ['api'],
    grantType: ['client_credentials'],
    enableRefreshToken: false,
    tokenType: ['access_token'],
    clientAuthenticationType: 'client_secret_basic'
  },
  'oauth20-oidc-implicit-client-secret-basic': {
    text: 'OAuth2.0 + OIDC 1.0 + Implicit + Client Secret Basic（低安全性）',
    allowedTypes: ['web'],
    grantType: ['implicit'],
    enableRefreshToken: false,
    tokenType: ['access_token', 'id_token'],
    clientAuthenticationType: 'client_secret_basic'
  },
  'oauth20-implicit-client-secret-basic': {
    text: 'OAuth2.0 + Implicit + Client Secret Basic（低安全性）',
    allowedTypes: ['web'],
    grantType: ['implicit'],
    enableRefreshToken: false,
    tokenType: ['access_token'],
    clientAuthenticationType: 'client_secret_basic'
  }
}
const roleTypeOptions = [
  { value: 'user', text: '用户角色' },
  { value: 'application', text: '应用角色' }
]
const tokenTypeOptionsByGrantType: Record<string, string[]> = {
  authorization_code: ['access_token', 'id_token'],
  authorization_code_pkce: ['access_token', 'id_token'],
  implicit: ['access_token', 'id_token'],
  client_credentials: ['access_token'],
  device_code: ['access_token', 'id_token'],
  password: ['access_token', 'id_token']
}
const clientAuthenticationTypeOptionsByGrantType: Record<string, string[]> = {
  authorization_code: ['client_secret_basic', 'client_secret_post', 'client_secret_jwt', 'private_key_jwt', 'tls_client_auth', 'self_signed_tls_client_auth'],
  authorization_code_pkce: ['none', 'client_secret_basic', 'client_secret_post', 'client_secret_jwt', 'private_key_jwt', 'tls_client_auth', 'self_signed_tls_client_auth'],
  implicit: ['client_secret_basic', 'client_secret_post', 'client_secret_jwt', 'private_key_jwt', 'tls_client_auth', 'self_signed_tls_client_auth'],
  client_credentials: ['client_secret_basic', 'client_secret_post', 'client_secret_jwt', 'private_key_jwt', 'tls_client_auth', 'self_signed_tls_client_auth'],
  device_code: ['none', 'client_secret_basic', 'client_secret_post', 'client_secret_jwt', 'private_key_jwt', 'tls_client_auth', 'self_signed_tls_client_auth'],
  password: ['none', 'client_secret_basic', 'client_secret_post', 'client_secret_jwt', 'private_key_jwt', 'tls_client_auth', 'self_signed_tls_client_auth']
}
const roleForm = reactive({ organizationId: '', name: '', type: 'user', description: '' })
const applicationTemplateSelection = ref('')
const policyForm = reactive({ id: '', organizationId: '', roleId: '', name: '', effect: 'allow', priority: 10, apiRulesText: '[\n  {\n    "method": "POST",\n    "path": "/api/manage/v1/example/query"\n  }\n]' })
const userForm = reactive({ organizationId: '', applicationId: '', username: '', name: '', email: '', phoneNumber: '', roleLabels: '', identifier: '', password: '' })
const userUpdateForm = reactive({ id: '', username: '', name: '', email: '', phoneNumber: '', roleLabels: '', status: '' })
const userAdminForm = reactive({ password: '' })
const policyCheckForm = reactive({ subjectType: 'application', subjectId: '', method: 'POST', path: '/api/manage/v1/organization/query' })
const externalIDPForm = reactive({
  id: '',
  organizationId: '',
  protocol: 'oidc',
  name: '',
  issuer: '',
  clientId: '',
  clientSecret: '',
  scopes: '',
  authorizationUrl: '',
  tokenUrl: '',
  userInfoUrl: '',
  jwksUrl: ''
})
const externalBindingForm = reactive({ organizationId: '', userId: '', externalIdpId: '', issuer: '', subject: '' })
const organizationBasicSettingForm = reactive({
  name: '',
  tosUrl: '',
  privacyPolicyUrl: '',
  supportEmail: '',
  logoUrl: ''
})
const organizationLoginPolicyForm = reactive({
  passwordLoginEnabled: true,
      webauthnLoginEnabled: true,
  allowUsername: true,
  allowEmail: true,
  allowPhone: true,
  usernameMode: 'optional' as 'hidden' | 'optional' | 'required',
  emailMode: 'required' as 'hidden' | 'optional' | 'required',
  phoneMode: 'optional' as 'hidden' | 'optional' | 'required'
})
const organizationPasswordPolicyForm = reactive({
  minLength: 12,
  requireUppercase: true,
  requireLowercase: true,
  requireNumber: true,
  requireSymbol: false,
  passwordExpires: false,
  expiryDays: 90
})
const organizationMFAPolicyForm = reactive({
  requireForAllUsers: false,
      allowWebauthn: true,
  allowTotp: true,
  allowEmailCode: true,
  allowSmsCode: false,
  allowU2f: true,
  allowRecoveryCode: true,
  emailChannelEnabled: false,
  emailChannelFrom: '',
  emailChannelHost: '',
  emailChannelPort: 587,
  emailChannelUsername: '',
  emailChannelPassword: ''
})
const organizationDomainRows = ref<OrganizationDomainRow[]>([])
const totpVerifyForm = reactive({ enrollmentId: '', code: '' })
const projectQuery = reactive({ organizationId: '' })
const applicationQuery = reactive({ projectId: '' })
const userQuery = reactive({ organizationId: '' })
const roleQuery = reactive({ organizationId: '' })
const userAdminResult = ref<unknown>(null)
const profileForm = reactive({
  id: '',
  organizationId: '',
  username: '',
  name: '',
  email: '',
  phoneNumber: '',
  status: '',
  roles: [] as string[]
})
const passwordForm = reactive({ currentPassword: '', newPassword: '' })
const profilePhoneInput = reactive<PhoneInputState>({ countryCode: '+86', localNumber: '' })
const userPhoneInput = reactive<PhoneInputState>({ countryCode: '+86', localNumber: '' })
const userUpdatePhoneInput = reactive<PhoneInputState>({ countryCode: '+86', localNumber: '' })
const mfaSettingForm = reactive({ emailEnabled: 'disabled', smsEnabled: 'disabled' })
const securitySetting = reactive({
  session: {
    id: '',
    applicationId: '',
    secondFactorMethod: '',
    riskLevel: ''
  },
  devices: [] as Array<any>
})

const summary = computed(() => ({
  organizationCount: organizations.value.length,
  projectCount: projects.value.length,
  applicationCount: applications.value.length,
  userCount: users.value.length,
  roleCount: roles.value.length,
  policyCount: policies.value.length,
  externalIdpCount: externalIDPs.value.length,
  auditCount: auditLogs.value.length
}))

const phoneCountryOptions = [
  { value: '+86', text: '+86 中国' },
  { value: '+852', text: '+852 中国香港' },
  { value: '+853', text: '+853 中国澳门' },
  { value: '+886', text: '+886 中国台湾' },
  { value: '+81', text: '+81 日本' },
  { value: '+82', text: '+82 韩国' },
  { value: '+1', text: '+1 美国/加拿大' },
  { value: '+44', text: '+44 英国' },
  { value: '+49', text: '+49 德国' },
  { value: '+33', text: '+33 法国' },
  { value: '+65', text: '+65 新加坡' },
  { value: '+60', text: '+60 马来西亚' },
  { value: '+61', text: '+61 澳大利亚' }
]
const booleanSettingOptions = [
  { value: 'active', text: '开启' },
  { value: 'disabled', text: '关闭' }
]
const fieldVisibilityOptions = [
  { value: 'hidden', text: '隐藏' },
  { value: 'optional', text: '选填' },
  { value: 'required', text: '必填' }
]

const currentOrganization = computed(() => organizations.value.find((item: any) => item.id === currentOrganizationId.value) || organizations.value[0])
const currentOrganizationLabel = computed(() => currentOrganization.value?.name || currentOrganization.value?.id || '选择组织')
const currentProject = computed(() => projects.value.find((item: any) => item.id === selectedProjectId.value) || projects.value[0])
const currentApplication = computed(() => applications.value.find((item: any) => item.id === selectedApplicationId.value) || applications.value[0])
const assignedProjectUsers = computed(() => users.value.filter((item: any) => projectAssignedUserIds.value.includes(item.id)))
const selectedUser = computed(() => users.value.find((item: any) => item.id === selectedUserId.value))
const currentUserRecord = computed(() => {
  if (currentView.value === 'my') {
    return userDetail.value?.user || profileForm
  }
  if (userDetail.value?.user?.id === selectedUserId.value) {
    return userDetail.value.user
  }
  return selectedUser.value
})
const selectedRole = computed(() => roles.value.find((item: any) => item.id === selectedRoleId.value) || roles.value[0])
const selectedRolePolicies = computed(() => policies.value.filter((item: any) => item.roleId === selectedRole.value?.id))
const applicationAssignableRoles = computed(() => roles.value.filter((item: any) => item.type === 'application'))
const userAssignableRoles = computed(() => roles.value.filter((item: any) => item.type === 'user'))
const currentProjectApplicationCount = computed(() => currentProject.value?.applications?.length ?? applications.value.length)
const currentRouteName = computed(() => String(route.name ?? 'console-dashboard'))
const recentAuditLogs = computed(() => auditLogs.value.slice(0, 12))
const moduleRecentChanges = computed(() => {
  if (tab.value === 'user' && userDetail.value?.recentAuditLogs?.length) {
    return userDetail.value.recentAuditLogs.slice(0, 6)
  }
  return recentAuditLogs.value.slice(0, 6)
})
const currentLoginUserLabel = computed(() => currentLoginUser.value || '当前登录用户')
const currentUserDisplayName = computed(() => profileForm.name || profileForm.username || currentLoginUser.value || '当前登录用户')
const currentUserEmail = computed(() => profileForm.email || currentLoginUser.value || '-')
const currentUserInitials = computed(() => {
  const source = currentUserDisplayName.value || currentUserEmail.value
  const cleaned = source.replace(/[^A-Za-z0-9\u4e00-\u9fa5 ]/g, ' ').trim()
  if (!cleaned) {
    return 'U'
  }
  const parts = cleaned.split(/\s+/).filter(Boolean)
  if (parts.length >= 2) {
    return `${parts[0][0] ?? ''}${parts[1][0] ?? ''}`.toUpperCase()
  }
  return cleaned.slice(0, 2).toUpperCase()
})
const pendingTotpEnrollmentId = computed(() => (totpSetup.value as { enrollmentId?: string } | null)?.enrollmentId || '')
const pendingTotpProvisioningUri = computed(() => (totpSetup.value as { provisioningUri?: string } | null)?.provisioningUri || '')
const pendingTotpManualEntryKey = computed(() => (totpSetup.value as { manualEntryKey?: string } | null)?.manualEntryKey || '')
const generatedRecoveryCodeList = computed(() => (recoveryCodes.value as { codes?: string[] } | null)?.codes || [])
const activeTOTPEnrollments = computed(() => (userDetail.value?.mfaEnrollments || []).filter((item: any) => item.method === 'totp'))
const emailCodeEnrollment = computed(() => (userDetail.value?.mfaEnrollments || []).find((item: any) => item.method === 'email_code'))
const smsCodeEnrollment = computed(() => (userDetail.value?.mfaEnrollments || []).find((item: any) => item.method === 'sms_code'))
const webauthnEnrollment = computed(() => (userDetail.value?.mfaEnrollments || []).find((item: any) => item.method === 'webauthn'))
const loginSecureKeys = computed(() => (userDetail.value?.secureKeys || []).filter((item: any) => item.webauthnEnable))
const u2fSecureKeys = computed(() => (userDetail.value?.secureKeys || []).filter((item: any) => item.u2fEnable))
const u2fEnrollment = computed(() => (userDetail.value?.mfaEnrollments || []).find((item: any) => item.method === 'u2f'))
const webauthnLoginEnabled = computed(() => webauthnEnrollment.value?.status === 'active' && loginSecureKeys.value.length > 0)
const userMFAMethodRows = computed<Array<{ id: MFAMethod; label: string; summary: string; enabled: boolean; disabled?: boolean }>>(() => [
  {
    id: 'totp',
    label: '身份验证器（TOTP）',
    summary: activeTOTPEnrollments.value.length > 0 ? '已配置' : '未开启',
    enabled: activeTOTPEnrollments.value.length > 0
  },
  {
    id: 'email_code',
    label: '邮箱验证码',
    summary: currentUserRecord.value?.email ? `${emailCodeEnrollment.value?.status === 'active' ? '已开启' : '已关闭'}，目标邮箱：${currentUserRecord.value.email}` : '未配置邮箱',
    enabled: emailCodeEnrollment.value?.status === 'active',
    disabled: !currentUserRecord.value?.email
  },
  {
    id: 'sms_code',
    label: '手机验证码',
    summary: currentUserRecord.value?.phoneNumber ? `${smsCodeEnrollment.value?.status === 'active' ? '已开启' : '已关闭'}，目标手机号：${currentUserRecord.value.phoneNumber}` : '未配置手机号',
    enabled: smsCodeEnrollment.value?.status === 'active',
    disabled: !currentUserRecord.value?.phoneNumber
  },
  {
    id: 'u2f',
    label: '安全密钥',
    summary: u2fEnrollment.value?.status === 'active' && u2fSecureKeys.value.length > 0 ? `已注册 ${u2fSecureKeys.value.length} 个密钥` : '未开启',
    enabled: u2fEnrollment.value?.status === 'active' && u2fSecureKeys.value.length > 0
  },
  {
    id: 'recovery_code',
    label: '备用验证码',
    summary: (userDetail.value?.recoverySummary?.total ?? 0) > 0
      ? `剩余有效码 ${userDetail.value?.recoverySummary?.available ?? 0} 个，最近生成于 ${formatDateTime(userDetail.value?.recoverySummary?.lastGeneratedAt)}`
      : '未开启',
    enabled: (userDetail.value?.recoverySummary?.total ?? 0) > 0
  }
])
const currentMFAModalTitle = computed(() => {
  const item = userMFAMethodRows.value.find((entry) => entry.id === currentMFAMethod.value)
  return item?.label || '两步验证设置'
})
const currentMFAModalActionLabel = computed(() => {
  if (currentMFAMethod.value === 'totp') return '激活身份验证器'
  if (currentMFAMethod.value === 'u2f') return '注册安全密钥'
  if (currentMFAMethod.value === 'recovery_code') return '生成备用验证码'
  return '保存设置'
})
const externalIDPProviderRows = computed(() => (['google', 'github', 'apple', 'qq', 'weibo', 'custom_oauth', 'custom_oidc'] as const).map((kind) => {
  const provider = findExistingExternalIDP(kind)
  return {
    id: kind,
    label: kind === 'google' ? 'Google'
      : kind === 'github' ? 'GitHub'
      : kind === 'apple' ? 'Apple'
      : kind === 'qq' ? 'QQ'
      : kind === 'weibo' ? '新浪微博'
      : kind === 'custom_oauth' ? '自定义 OAuth'
      : '自定义 OIDC',
    summary: provider ? `已配置 · ${provider.clientId || provider.issuer || provider.name}` : '未启用',
    enabled: Boolean(provider)
  }
}))
const currentExternalIDPModalTitle = computed(() => currentExternalIDPKind.value === 'google' ? '配置 Google 登录'
  : currentExternalIDPKind.value === 'github' ? '配置 GitHub 登录'
  : currentExternalIDPKind.value === 'apple' ? '配置 Apple 登录'
  : currentExternalIDPKind.value === 'qq' ? '配置 QQ 登录'
  : currentExternalIDPKind.value === 'weibo' ? '配置 新浪微博 登录'
  : currentExternalIDPKind.value === 'custom_oauth' ? '配置自定义 OAuth 提供商'
  : '配置自定义 OIDC 提供商')
const currentExternalIDPActionLabel = computed(() => externalIDPForm.id ? '保存配置' : isCustomExternalIDPKind(currentExternalIDPKind.value) ? '添加 Provider' : '启用 Provider')
const currentExternalIDPProtocol = computed(() => externalIDPForm.protocol)
const customExternalIDPs = computed(() => externalIDPs.value.filter((item: any) => {
  const providerKind = normalizeProviderKind(item)
  return providerKind === 'custom_oauth' || providerKind === 'custom_oidc'
}))
const userDeviceList = computed(() => (userDetail.value?.devices || []).map((device: any) => ({
  id: device.id,
  label: inferDeviceName(device.userAgent),
  online: Boolean(device.online),
  trusted: Boolean(device.trusted),
  ipAddress: device.lastLoginIp || '-',
  ipLocation: device.ipLocation || '',
  lastLoginAt: device.lastLoginAt || '',
  firstLoginAt: device.firstLoginAt || '',
  fingerprint: device.deviceFingerprint || ''
})))
const currentMyPanels = computed(() => [
  { id: 'my-basic', label: '基本信息' },
  { id: 'my-login-setting', label: '登录设置' },
  { id: 'my-binding', label: '账号绑定' },
  { id: 'my-mfa', label: '两步验证' },
  { id: 'my-session', label: '会话管理' }
])
const currentView = computed(() => {
  if (currentRouteName.value === 'console-my') return 'my'
  if (currentRouteName.value === 'console-organization-manage') return 'organization-manage'
  if (currentRouteName.value === 'console-project-create') return 'project-create'
  if (currentRouteName.value === 'console-application-create') return 'application-create'
  if (currentRouteName.value === 'console-application-detail') return 'application-detail'
  return 'main'
})
const currentTabLabel = computed(() => {
  if (currentView.value === 'my') return '个人中心'
  if (currentView.value === 'organization-manage') return '组织管理'
  if (currentView.value === 'project-create') return '创建项目'
  if (currentView.value === 'application-create') return '创建应用'
  if (currentView.value === 'application-detail') return '应用详情'
  if (tab.value === 'dashboard') return '仪表盘'
  if (tab.value === 'organization') return '组织'
  if (tab.value === 'project') return '项目'
  if (tab.value === 'user') return '用户'
  if (tab.value === 'role') return '角色'
  if (tab.value === 'audit') return '审计'
  return '设置'
})
const currentTabDescription = computed(() => {
  if (currentView.value === 'my') return '查看并维护当前登录用户的基本信息、登录方式、账号绑定、两步验证与设备会话。'
  if (currentView.value === 'organization-manage') return '仅内部管理员可以创建和维护组织。'
  if (currentView.value === 'project-create') return '在当前组织下创建新的项目。'
  if (currentView.value === 'application-create') return '在当前项目下创建新的应用。'
  if (currentView.value === 'application-detail') return '查看并维护当前应用的接入配置。'
  if (tab.value === 'dashboard') return '概览当前实例下的核心 IAM 统计和审计摘要。'
  if (tab.value === 'organization') return ''
  if (tab.value === 'project') return '管理项目与应用的结构、协议模式与接入配置。'
  if (tab.value === 'user') return '管理用户、通行密钥、身份验证器、备用验证码与管理员动作。'
  if (tab.value === 'role') return '维护角色标签、策略规则与 Policy Check。'
  if (tab.value === 'audit') return '查看平台关键事件、登录轨迹与策略变更审计。'
  return '配置外部 OAuth/OIDC 联邦与身份绑定。'
})
const pageHeaderTitle = computed(() => {
  if (tab.value === 'organization' && currentView.value === 'main') {
    return ''
  }
  if (currentView.value === 'application-detail') {
    return ''
  }
  if (tab.value === 'project' && currentView.value === 'main' && projectViewMode.value === 'detail') {
    return ''
  }
  if (tab.value === 'user' && userViewMode.value === 'detail') {
    return ''
  }
  if (tab.value === 'role' && roleViewMode.value === 'detail') {
    return ''
  }
  return currentTabLabel.value
})
const pageHeaderDescription = computed(() => {
  if (tab.value === 'organization' && currentView.value === 'main') {
    return ''
  }
  if (currentView.value === 'application-detail') {
    return ''
  }
  if (tab.value === 'project' && currentView.value === 'main' && projectViewMode.value === 'detail') {
    return ''
  }
  if (tab.value === 'user' && userViewMode.value === 'detail') {
    return ''
  }
  if (tab.value === 'role' && roleViewMode.value === 'detail') {
    return ''
  }
  return currentTabDescription.value
})
const summaryTiles = computed(() => [
  { label: '组织', value: summary.value.organizationCount },
  { label: '项目', value: summary.value.projectCount },
  { label: '应用', value: summary.value.applicationCount },
  { label: '用户', value: summary.value.userCount },
  { label: '角色标签', value: summary.value.roleCount },
  { label: '策略', value: summary.value.policyCount },
])
const currentModuleEntityTitle = computed(() => {
  if (tab.value === 'organization') return currentOrganization.value?.name || '组织'
  if (tab.value === 'project') return currentProject.value?.name || '项目'
  if (tab.value === 'user') return currentUserRecord.value?.name || currentUserRecord.value?.username || currentUserRecord.value?.email || '用户'
  if (tab.value === 'role') return selectedRole.value?.name || '角色'
  if (tab.value === 'audit') return currentOrganization.value?.name || '审计'
  if (tab.value === 'setting') return currentOrganization.value?.name || '外部联邦'
  return '实例概览'
})
const currentModuleSummaryText = computed(() => {
  if (tab.value === 'organization') {
    return currentOrganization.value?.name ? `当前组织 ${currentOrganization.value.name} 的基础配置、登录方式和接入边界。` : currentTabDescription.value
  }
  if (tab.value === 'project') {
    if (currentProject.value?.name) {
      return `从项目列表选择条目后，在详情区维护项目及其应用配置。`
    }
    return currentTabDescription.value
  }
  if (tab.value === 'user') {
    return currentUserRecord.value?.name || currentUserRecord.value?.email ? '从用户列表选择条目后，在详情区维护基本信息、登录设置、账号绑定、两步验证、会话与角色分配。' : currentTabDescription.value
  }
  if (tab.value === 'role') {
    return selectedRole.value?.name ? '从角色列表选择条目后，在详情区维护角色元信息、策略列表与 Policy Check。' : currentTabDescription.value
  }
  return currentTabDescription.value
})
const currentModuleMetrics = computed<MetricItem[]>(() => {
  if (tab.value === 'organization') {
    const projectCount = currentOrganization.value?.projects?.length ?? 0
    const applicationCount = (currentOrganization.value?.projects ?? []).reduce((count: number, project: any) => count + (project.applications?.length ?? 0), 0)
    return [
      { label: '组织 ID', value: currentOrganization.value?.id || '-', copyable: Boolean(currentOrganization.value?.id), copyValue: currentOrganization.value?.id || '' },
      { label: '项目数', value: String(projectCount) },
      { label: '应用数', value: String(applicationCount) },
      { label: '用户数', value: String(users.value.length) },
      { label: '角色数', value: String(roles.value.length) },
      { label: '创建时间', value: formatDateTime(currentOrganization.value?.createdAt) },
      { label: '更新时间', value: formatDateTime(currentOrganization.value?.updatedAt) }
    ]
  }
  if (tab.value === 'project') {
    return [
      { label: '项目 ID', value: currentProject.value?.id || '-', copyable: Boolean(currentProject.value?.id), copyValue: currentProject.value?.id || '' },
      { label: '应用数', value: String(currentProjectApplicationCount.value) },
      { label: '创建时间', value: formatDateTime(currentProject.value?.createdAt) },
      { label: '最近变更', value: formatDateTime(currentProject.value?.updatedAt) }
    ]
  }
  if (tab.value === 'user') {
    return [
      { label: '用户 ID', value: currentUserRecord.value?.id || '-', copyable: Boolean(currentUserRecord.value?.id), copyValue: currentUserRecord.value?.id || '' },
      { label: '状态', value: currentUserRecord.value?.status || '-' },
      { label: '通行密钥', value: String(userDetail.value?.secureKeys?.length ?? 0) },
      { label: '绑定数', value: String(userDetail.value?.bindings?.length ?? 0) },
      { label: '会话数', value: String(userDetail.value?.recentSessions?.length ?? 0) },
      { label: '最近变更', value: formatDateTime(currentUserRecord.value?.updatedAt) }
    ]
  }
  if (tab.value === 'role') {
    return [
      { label: '角色 ID', value: selectedRole.value?.id || '-' },
      { label: '角色类型', value: selectedRole.value?.type || '-' },
      { label: '角色数', value: String(roles.value.length) },
      { label: '关联策略', value: String(selectedRolePolicies.value.length) },
      { label: '策略总数', value: String(policies.value.length) },
      { label: '最近变更', value: formatDateTime(selectedRole.value?.updatedAt) }
    ]
  }
  if (tab.value === 'audit') {
    return [
      { label: '组织 ID', value: currentOrganization.value?.id || '-' },
      { label: '创建时间', value: formatDateTime(currentOrganization.value?.createdAt) },
      { label: '审计数', value: String(auditLogs.value.length) },
      { label: '最近登录用户', value: currentLoginUserLabel.value || '-' }
    ]
  }
  if (tab.value === 'setting') {
    return [
      { label: '组织 ID', value: currentOrganization.value?.id || '-' },
      { label: '最近变更', value: formatDateTime(currentOrganization.value?.updatedAt) },
      { label: '外部 IdP 数量', value: String(externalIDPs.value.length) },
      { label: '绑定用户', value: selectedUser.value?.email || selectedUser.value?.id || '-' }
    ]
  }
  return summaryTiles.value.map((item) => ({ label: item.label, value: String(item.value) }))
})
const currentModulePanels = computed(() => {
  if (tab.value === 'organization') return [
    { id: 'organization-metadata', label: '维护元信息' }
  ]
  if (tab.value === 'project') return [
    { id: 'project-application', label: '应用列表' },
    { id: 'project-user-assignment', label: '用户分配' },
    { id: 'project-setting', label: '项目设置' }
  ]
  if (tab.value === 'user') return [
    { id: 'user-basic', label: '基本信息' },
    { id: 'user-login-setting', label: '登录设置' },
    { id: 'user-binding', label: '账号绑定' },
    { id: 'user-mfa', label: '两步验证' },
    { id: 'user-session', label: '会话管理' },
    { id: 'user-role-assignment', label: '角色分配' },
    { id: 'user-danger-zone', label: '危险区' }
  ]
  if (tab.value === 'role') return [
    { id: 'role-list', label: '角色列表' },
    { id: 'role-detail', label: '角色详情' },
    { id: 'policy-list', label: '策略列表' },
    { id: 'policy-editor', label: '策略编辑' },
    { id: 'role-decision', label: 'Policy Check' }
  ]
  if (tab.value === 'audit') return [
    { id: 'audit-list', label: '审计日志' }
  ]
  if (tab.value === 'setting') return [
    { id: 'setting-basic', label: '基本设置' },
    { id: 'setting-domain', label: '域名设置' },
    { id: 'setting-login-policy', label: '登录策略设置' },
    { id: 'setting-password-policy', label: '密码策略设置' },
    { id: 'setting-mfa-policy', label: '两步验证策略' },
    { id: 'setting-external-idp', label: '外部 IdP 设置' }
  ]
  return [
    { id: 'dashboard-overview', label: '平台概览' },
    { id: 'dashboard-audit', label: '审计摘要' }
  ]
})
const applicationDetailMetrics = computed<MetricItem[]>(() => [
  { label: '应用 ID', value: currentApplication.value?.id || '-', copyable: Boolean(currentApplication.value?.id), copyValue: currentApplication.value?.id || '' },
  { label: '应用类型', value: formatApplicationType(currentApplication.value?.applicationType) },
  { label: '令牌类型', value: formatApplicationTokenType(currentApplication.value?.tokenType) },
  { label: '刷新令牌', value: currentApplication.value?.enableRefreshToken ? '已启用' : '未启用' },
  { label: '授权流程', value: formatApplicationGrantType(currentApplication.value?.grantType) },
  { label: '客户端认证', value: formatApplicationClientAuthenticationType(currentApplication.value?.clientAuthenticationType) },
  { label: '应用角色', value: formatRoleLabels(currentApplication.value?.roles) },
  { label: '创建时间', value: formatDateTime(currentApplication.value?.createdAt) },
  { label: '最近变更', value: formatDateTime(currentApplication.value?.updatedAt) }
])
const currentApplicationProtocolHint = computed(() => {
  const applicationType = currentApplication.value?.applicationType || applicationUpdateForm.applicationType
  if (applicationType === 'api') {
    return '推荐 API 类型默认使用 `client_credentials + access_token + private_key_jwt`，并关闭 Refresh Token。'
  }
  if (applicationType === 'native') {
    return '推荐 Native 类型优先使用 `authorization_code_pkce` 或 `device_code`。如需长期会话，可额外开启 Refresh Token。'
  }
  return '推荐 Web 类型优先使用 `authorization_code_pkce + access_token + none`。如需 OIDC 前端消费身份声明，可改为 `access_token_id_token`。'
})
const applicationDetailPanels = computed(() => [
  { id: 'application-protocol', label: '协议配置' },
  { id: 'application-role-assignment', label: '角色分配' },
  { id: 'application-token', label: '令牌设置' }
])
const showBackToTopButton = computed(() => {
  if (currentView.value === 'my' || currentView.value === 'application-detail') {
    return true
  }
  if (tab.value === 'organization') return true
  if (tab.value === 'project' && projectViewMode.value === 'detail') return true
  if (tab.value === 'user' && userViewMode.value === 'detail') return true
  if (tab.value === 'role') return true
  if (tab.value === 'audit') return true
  if (tab.value === 'setting') return true
  return false
})
const backToTopWrapClass = computed(() => currentView.value === 'my' ? 'console-back-to-top-wrap-wide' : 'console-back-to-top-wrap-middle')
const currentModuleActionLabel = computed(() => {
  if (tab.value === 'organization') return '刷新组织'
  if (tab.value === 'project') return '刷新项目'
  if (tab.value === 'user') return '刷新用户'
  if (tab.value === 'role') return '刷新角色'
  if (tab.value === 'audit') return '刷新审计'
  if (tab.value === 'setting') return '刷新联邦'
  return '刷新概览'
})
const showModuleActionButton = computed(() => tab.value !== 'organization')
const currentModuleMetricsClass = computed(() => ({
  'console-module-metrics-inline': tab.value === 'organization'
}))
const applicationProtocolTemplateOptions = computed(() => {
  const type = applicationForm.applicationType
  const items = Object.entries(applicationProtocolTemplates)
    .filter(([, template]) => template.allowedTypes.includes(type))
    .map(([value, template]) => ({ value, text: template.text }))
  return [{ value: '', text: '选择授权模板' }, ...items]
})
const visibleApplicationProtocolTemplates = computed(() => {
  const type = applicationForm.applicationType
  return Object.entries(applicationProtocolTemplates)
    .filter(([, template]) => template.allowedTypes.includes(type))
    .map(([key, template]) => ({
      key,
      text: template.text,
      allowedTypes: template.allowedTypes.join('、'),
      grantType: template.grantType.join(' + '),
      enableRefreshToken: template.enableRefreshToken ? 'true' : 'false',
      tokenType: template.tokenType.join(' + '),
      clientAuthenticationType: template.clientAuthenticationType
    }))
})
const applicationFormTokenTypeOptions = computed(() => filterApplicationTokenTypeOptions(applicationForm.grantType))
const applicationUpdateTokenTypeOptions = computed(() => filterApplicationTokenTypeOptions(applicationUpdateForm.grantType))
const applicationFormClientAuthenticationTypeOptions = computed(() => filterApplicationClientAuthenticationTypeOptions(applicationForm.grantType))
const applicationUpdateClientAuthenticationTypeOptions = computed(() => filterApplicationClientAuthenticationTypeOptions(applicationUpdateForm.grantType))

function applyRecommendedApplicationProtocol(target: { applicationType: string; tokenType: string[]; enableRefreshToken: boolean; grantType: string[]; clientAuthenticationType: string }) {
  if (target.applicationType === 'api') {
    target.tokenType = ['access_token']
    target.enableRefreshToken = false
    target.grantType = ['client_credentials']
    target.clientAuthenticationType = 'private_key_jwt'
    return
  }
  if (target.applicationType === 'native') {
    target.tokenType = ['access_token']
    target.enableRefreshToken = false
    target.grantType = ['authorization_code_pkce']
    target.clientAuthenticationType = 'none'
    return
  }
  target.tokenType = ['access_token']
  target.enableRefreshToken = false
  target.grantType = ['authorization_code_pkce']
  target.clientAuthenticationType = 'none'
}

function toggleApplicationGrantType(target: string[], value: string, checked: boolean) {
  const next = new Set(target)
  if (checked) {
    next.add(value)
  } else {
    next.delete(value)
  }
  target.splice(0, target.length, ...Array.from(next))
}

function toggleApplicationTokenType(target: string[], value: string, checked: boolean) {
  const next = new Set(target)
  if (checked) {
    next.add(value)
  } else {
    next.delete(value)
  }
  target.splice(0, target.length, ...Array.from(next))
}

function applyApplicationProtocolTemplate(templateKey: string) {
  const template = applicationProtocolTemplates[templateKey]
  if (!template) {
    return
  }
  applicationForm.grantType.splice(0, applicationForm.grantType.length, ...template.grantType)
  applicationForm.tokenType.splice(0, applicationForm.tokenType.length, ...template.tokenType)
  applicationForm.enableRefreshToken = template.enableRefreshToken
  applicationForm.clientAuthenticationType = template.clientAuthenticationType
  normalizeApplicationProtocolSelection(applicationForm)
}

function intersectOptionValues(groups: string[][], fallback: string[]) {
  if (!groups.length) {
    return fallback
  }
  return groups.reduce((acc, group) => acc.filter((item) => group.includes(item)))
}

function filterApplicationTokenTypeOptions(grantTypes: string[]) {
  const selectedGrantTypes = grantTypes.length ? grantTypes : []
  const allowed = intersectOptionValues(
    selectedGrantTypes.map((grantType) => tokenTypeOptionsByGrantType[grantType] ?? tokenTypeOptions.map((item) => item.value)),
    tokenTypeOptions.map((item) => item.value)
  )
  return tokenTypeOptions.filter((item) => allowed.includes(item.value))
}

function filterApplicationClientAuthenticationTypeOptions(grantTypes: string[]) {
  const selectedGrantTypes = grantTypes.length ? grantTypes : []
  const allowed = intersectOptionValues(
    selectedGrantTypes.map((grantType) => clientAuthenticationTypeOptionsByGrantType[grantType] ?? clientAuthenticationTypeOptions.map((item) => item.value)),
    clientAuthenticationTypeOptions.map((item) => item.value)
  )
  return clientAuthenticationTypeOptions.filter((item) => allowed.includes(item.value))
}

function normalizeApplicationProtocolSelection(target: { tokenType: string[]; enableRefreshToken: boolean; grantType: string[]; clientAuthenticationType: string }) {
  if (!target.grantType.length) {
    target.grantType = ['authorization_code_pkce']
  }
  const allowedTokenTypes = filterApplicationTokenTypeOptions(target.grantType).map((item) => item.value)
  target.tokenType = target.tokenType.filter((item) => allowedTokenTypes.includes(item))
  if (!target.tokenType.length) {
    target.tokenType = allowedTokenTypes.length ? [allowedTokenTypes[0]] : ['access_token']
  }

  const allowedClientAuthenticationTypes = filterApplicationClientAuthenticationTypeOptions(target.grantType).map((item) => item.value)
  if (!allowedClientAuthenticationTypes.includes(target.clientAuthenticationType)) {
    target.clientAuthenticationType = allowedClientAuthenticationTypes[0] ?? 'none'
  }

  if (target.grantType.includes('client_credentials')) {
    target.tokenType = ['access_token']
    target.enableRefreshToken = false
  }
  if (target.grantType.includes('implicit')) {
    target.enableRefreshToken = false
  }
  if (!target.tokenType.includes('access_token')) {
    target.enableRefreshToken = false
  }
}

function validateApplicationProtocolInput(target: { tokenType: string[]; enableRefreshToken: boolean; grantType: string[]; clientAuthenticationType: string }) {
  if (!target.grantType.length) {
    return '至少需要选择一个 Grant Type。'
  }
  if (!target.tokenType.length) {
    return '至少需要选择一个 Token Type。'
  }
  if (target.grantType.includes('client_credentials') && !(target.tokenType.length === 1 && target.tokenType[0] === 'access_token')) {
    return 'client_credentials 只允许 token_type=access_token。'
  }
  if (target.grantType.includes('client_credentials') && target.enableRefreshToken) {
    return 'client_credentials 不允许启用 Refresh Token。'
  }
  if (target.grantType.includes('implicit') && target.tokenType.some((item) => !['access_token', 'id_token'].includes(item))) {
    return 'implicit 只允许 access_token 和/或 id_token。'
  }
  if (target.grantType.includes('implicit') && target.enableRefreshToken) {
    return 'implicit 不允许启用 Refresh Token。'
  }
  if (target.clientAuthenticationType === 'none' && target.grantType.some((item) => item !== 'authorization_code_pkce' && item !== 'device_code' && item !== 'password')) {
    return 'client_authentication_type=none 只允许用于 authorization_code_pkce、device_code 或 password。'
  }
  if (!target.tokenType.includes('access_token') && target.enableRefreshToken) {
    return '未签发 access_token 时不能启用 Refresh Token。'
  }
  return ''
}

watch(() => applicationForm.applicationType, () => applyRecommendedApplicationProtocol(applicationForm))
watch(() => applicationUpdateForm.applicationType, () => applyRecommendedApplicationProtocol(applicationUpdateForm))
watch(() => applicationForm.applicationType, (value) => {
  if (!applicationTemplateSelection.value) {
    return
  }
  const template = applicationProtocolTemplates[applicationTemplateSelection.value]
  if (!template || !template.allowedTypes.includes(value)) {
    applicationTemplateSelection.value = ''
  }
})
watch(() => [...applicationForm.grantType], () => normalizeApplicationProtocolSelection(applicationForm))
watch(() => [...applicationUpdateForm.grantType], () => normalizeApplicationProtocolSelection(applicationUpdateForm))
watch(() => [...applicationForm.tokenType], () => normalizeApplicationProtocolSelection(applicationForm))
watch(() => [...applicationUpdateForm.tokenType], () => normalizeApplicationProtocolSelection(applicationUpdateForm))
watch(applicationTemplateSelection, (value) => {
  if (!value) {
    return
  }
  applyApplicationProtocolTemplate(value)
})

onMounted(async () => {
  currentLoginUser.value = sessionStorage.getItem('ppvt-login-identifier') ?? ''
  await loadAll()
  if (sessionStorage.getItem('ppvt-access-token')) {
    await loadProfile()
  }
})

watch(
  () => pendingTotpProvisioningUri.value,
  async (value) => {
    if (!value) {
      totpQRCodeDataURL.value = ''
      return
    }
    try {
      totpQRCodeDataURL.value = await QRCode.toDataURL(value, {
        margin: 1,
        width: 192
      })
    } catch {
      totpQRCodeDataURL.value = ''
    }
  },
  { immediate: true }
)

watch(
  () => mfaConfigModalVisible.value,
  (visible) => {
    if (visible) {
      return
    }
    if (currentMFAMethod.value === 'totp') {
      totpSetup.value = null
      totpVerifyForm.enrollmentId = ''
      totpVerifyForm.code = ''
    }
  }
)

watch(
  () => [currentRouteName.value, route.params.organizationId, route.params.projectId, route.params.applicationId, route.params.userId, route.params.roleId],
  async ([routeName, organizationId, projectId, applicationId, userId, roleId]) => {
    if (routeName === 'console-organization') tab.value = 'organization'
    else if (routeName === 'console-project-list' || routeName === 'console-project-create' || routeName === 'console-project-detail' || routeName === 'console-application-create' || routeName === 'console-application-detail') tab.value = 'project'
    else if (routeName === 'console-user-list' || routeName === 'console-user-detail') tab.value = 'user'
    else if (routeName === 'console-role-list' || routeName === 'console-role-detail') tab.value = 'role'
    else if (routeName === 'console-audit') tab.value = 'audit'
    else if (routeName === 'console-settings') tab.value = 'setting'
    else tab.value = 'dashboard'

    if (routeName === 'console-project-list') {
      projectViewMode.value = 'list'
    } else if (routeName === 'console-project-detail' || routeName === 'console-application-create' || routeName === 'console-application-detail') {
      projectViewMode.value = 'detail'
    }
    if (routeName === 'console-user-list') userViewMode.value = 'list'
    else if (routeName === 'console-user-detail') userViewMode.value = 'detail'
    if (routeName === 'console-role-list') roleViewMode.value = 'list'
    else if (routeName === 'console-role-detail') roleViewMode.value = 'detail'

    if (typeof organizationId === 'string' && organizationId) {
      currentOrganizationId.value = organizationId
      organizationSwitcher.value = organizationId
    }
    if (typeof projectId === 'string' && projectId) {
      selectedProjectId.value = projectId
    }
    if (typeof applicationId === 'string' && applicationId) {
      selectedApplicationId.value = applicationId
    }
    if (typeof userId === 'string' && userId) {
      selectedUserId.value = userId
      await loadUserDetail(userId)
    }
    if (typeof roleId === 'string' && roleId) {
      selectedRoleId.value = roleId
    }

    if (routeName === 'console-my') {
      await Promise.all([loadProfile(), loadSecuritySetting(), loadCurrentUserDetail()])
    }
  },
  { immediate: true }
)

watch(
  () => currentApplication.value,
  (value) => {
    syncApplicationEditState(value)
  }
)

watch(
  () => currentProject.value,
  (value) => {
    syncProjectEditState(value)
  }
)

watch(
  () => currentOrganization.value,
  (value) => {
    syncOrganizationMetadataRows(value)
    syncOrganizationSettingForms(value)
  },
  { immediate: true }
)

watch(message, (value) => {
  if (!value) {
    return
  }
  if (messageVariant.value === 'danger') {
    toast.error(value)
  } else {
    toast.success(value)
  }
  message.value = ''
})

async function loadAll() {
  await loadOrganizations()
  const routeOrganizationId = typeof route.params.organizationId === 'string' ? route.params.organizationId : ''
  const routeProjectId = typeof route.params.projectId === 'string' ? route.params.projectId : ''
  const routeApplicationId = typeof route.params.applicationId === 'string' ? route.params.applicationId : ''
  const fallbackOrganization = organizations.value.find((item: any) => item.id === routeOrganizationId) || organizations.value[0]
  if (!currentOrganizationId.value || !organizations.value.some((item: any) => item.id === currentOrganizationId.value)) {
    currentOrganizationId.value = fallbackOrganization?.id ?? ''
  }
  if (!organizationSwitcher.value && currentOrganizationId.value) {
    organizationSwitcher.value = currentOrganizationId.value
  }
  const currentOrg = organizations.value.find((item: any) => item.id === currentOrganizationId.value) || fallbackOrganization
  const builtinProject = currentOrg?.projects?.find((item: any) => (item.applications || []).some((application: any) => application.id === consoleApplicationId))
  const project = currentOrg?.projects?.find((item: any) => item.id === routeProjectId) || builtinProject || currentOrg?.projects?.[0]
  const application = project?.applications?.find((item: any) => item.id === routeApplicationId) || project?.applications?.find((item: any) => item.id === consoleApplicationId) || project?.applications?.[0]
  selectedProjectId.value = project?.id ?? selectedProjectId.value
  selectedApplicationId.value = application?.id ?? selectedApplicationId.value
  organizationUpdateForm.id = currentOrg?.id ?? organizationUpdateForm.id
  organizationUpdateForm.name = currentOrg?.name ?? organizationUpdateForm.name
  organizationUpdateForm.metadata = normalizeMetadataMap(currentOrg?.metadata)
  projectForm.organizationId = currentOrg?.id ?? projectForm.organizationId
  projectQuery.organizationId = currentOrg?.id ?? projectQuery.organizationId
  applicationForm.projectId = project?.id ?? applicationForm.projectId
  syncProjectEditState(project)
  applicationQuery.projectId = project?.id ?? applicationQuery.projectId
  applicationUpdateForm.id = application?.id ?? applicationUpdateForm.id
  applicationUpdateForm.name = application?.name ?? applicationUpdateForm.name
  applicationUpdateForm.redirectUris = application?.redirectUris ?? applicationUpdateForm.redirectUris
  applicationUpdateForm.applicationType = application?.applicationType ?? applicationUpdateForm.applicationType
  applicationUpdateForm.grantType = [...(application?.grantType ?? applicationUpdateForm.grantType)]
  applicationUpdateForm.clientAuthenticationType = application?.clientAuthenticationType ?? applicationUpdateForm.clientAuthenticationType
  applicationUpdateForm.tokenType = [...(application?.tokenType ?? applicationUpdateForm.tokenType)]
  applicationForm.roles = [...(application?.roles ?? applicationForm.roles)]
  applicationForm.publicKey = application?.publicKey ?? applicationForm.publicKey
  applicationForm.accessTokenTTLMinutes = application?.accessTokenTTLMinutes ?? applicationForm.accessTokenTTLMinutes
  applicationForm.refreshTokenTTLHours = application?.refreshTokenTTLHours ?? applicationForm.refreshTokenTTLHours
  applicationUpdateForm.roles = [...(application?.roles ?? applicationUpdateForm.roles)]
  applicationUpdateForm.publicKey = application?.publicKey ?? applicationUpdateForm.publicKey
  applicationUpdateForm.accessTokenTTLMinutes = application?.accessTokenTTLMinutes ?? applicationUpdateForm.accessTokenTTLMinutes
  applicationUpdateForm.refreshTokenTTLHours = application?.refreshTokenTTLHours ?? applicationUpdateForm.refreshTokenTTLHours
  roleForm.organizationId = currentOrg?.id ?? roleForm.organizationId
  userForm.organizationId = currentOrg?.id ?? userForm.organizationId
  roleQuery.organizationId = currentOrg?.id ?? roleQuery.organizationId
  userForm.applicationId = application?.id ?? userForm.applicationId
  policyForm.organizationId = currentOrg?.id ?? policyForm.organizationId
  externalIDPForm.organizationId = currentOrg?.id ?? externalIDPForm.organizationId
  externalBindingForm.organizationId = currentOrg?.id ?? externalBindingForm.organizationId
  userQuery.organizationId = currentOrg?.id ?? userQuery.organizationId
  const results = await Promise.allSettled([
    loadUsers(),
    loadRoles(),
    loadPolicies(),
    loadExternalIDPs(),
    loadAudit(),
    loadProjects(),
    loadApplications()
  ])
  const firstUser = users.value[0]
  const firstRole = roles.value[0]
  selectedUserId.value = firstUser?.id ?? selectedUserId.value
  selectedRoleId.value = firstRole?.id ?? selectedRoleId.value
  syncUserEditState(firstUser)
  await loadUserDetail(selectedUserId.value)
  const rejected = results.find((item) => item.status === 'rejected') as PromiseRejectedResult | undefined
  if (rejected) {
    message.value = String(rejected.reason)
    messageVariant.value = 'danger'
  }
}

async function loadOrganizations() {
  const response = await apiPost<{ items: any[] }>('/api/manage/v1/organization/query', {})
  organizations.value = response.items
  const currentOrg = response.items.find((item: any) => item.id === currentOrganizationId.value) || response.items[0]
  if (currentOrg) {
    organizationUpdateForm.id = currentOrg.id ?? ''
    organizationUpdateForm.name = currentOrg.name ?? ''
    organizationUpdateForm.metadata = normalizeMetadataMap(currentOrg.metadata)
  }
  if (response.items.length === 0) {
    message.value = '当前没有可用组织'
    messageVariant.value = 'danger'
  }
}

async function loadProjects() {
  const response = await apiPost<{ items: any[] }>('/api/manage/v1/project/query', projectQuery)
  projects.value = response.items
  if (!projects.value.some((item: any) => item.id === selectedProjectId.value)) {
    selectedProjectId.value = projects.value[0]?.id ?? ''
  }
  const selectedProject = projects.value.find((item: any) => item.id === selectedProjectId.value) || projects.value[0]
  syncProjectEditState(selectedProject)
  syncProjectUserAssignments(selectedProject)
}

async function loadApplications() {
  const response = await apiPost<{ items: any[] }>('/api/manage/v1/application/query', applicationQuery)
  applications.value = response.items
  if (!applications.value.some((item: any) => item.id === selectedApplicationId.value)) {
    selectedApplicationId.value = applications.value[0]?.id ?? ''
  }
  const selectedApplication = applications.value.find((item: any) => item.id === selectedApplicationId.value) || applications.value[0]
  syncApplicationEditState(selectedApplication)
}

async function loadUsers() {
  const response = await apiPost<{ items: any[] }>('/api/manage/v1/user/query', userQuery)
  users.value = response.items
  selectedUserIds.value = selectedUserIds.value.filter((id) => users.value.some((item: any) => item.id === id))
  if (!users.value.some((item: any) => item.id === selectedUserId.value)) {
    selectedUserId.value = users.value[0]?.id ?? ''
  }
  if (selectedUserId.value) {
    syncUserEditState(users.value.find((item: any) => item.id === selectedUserId.value) || users.value[0])
  }
}

function syncUserEditState(user?: any) {
  if (!user) {
    return
  }
  userUpdateForm.id = user.id ?? ''
  userUpdateForm.username = user.username ?? ''
  userUpdateForm.name = user.name ?? ''
  userUpdateForm.email = user.email ?? ''
  userUpdateForm.phoneNumber = user.phoneNumber ?? ''
  syncPhoneInput(userUpdatePhoneInput, userUpdateForm.phoneNumber)
  userUpdateForm.roleLabels = (user.roles ?? []).join(',')
  userUpdateForm.status = user.status ?? ''
  userRoleAssignments.value = [...(user.roles ?? [])]
}

function syncProjectEditState(project?: any) {
  if (!project) {
    return
  }
  projectUpdateForm.id = project.id ?? ''
  projectUpdateForm.name = project.name ?? ''
  projectUpdateForm.description = project.description ?? ''
  projectUpdateForm.userAclEnabled = Boolean(project.userAclEnabled)
}

function syncApplicationEditState(application?: any) {
  if (!application) {
    return
  }
  applicationUpdateForm.id = application.id ?? ''
  applicationUpdateForm.name = application.name ?? ''
  applicationUpdateForm.redirectUris = application.redirectUris ?? ''
  applicationUpdateForm.applicationType = application.applicationType ?? 'web'
  applicationUpdateForm.grantType = [...(application.grantType ?? ['authorization_code_pkce'])]
  applicationUpdateForm.clientAuthenticationType = application.clientAuthenticationType ?? 'none'
  applicationUpdateForm.tokenType = [...(application.tokenType ?? ['access_token'])]
  applicationUpdateForm.enableRefreshToken = Boolean(application.enableRefreshToken)
  applicationUpdateForm.roles = [...(application.roles ?? [])]
  applicationUpdateForm.publicKey = application.publicKey ?? ''
  applicationUpdateForm.accessTokenTTLMinutes = application.accessTokenTTLMinutes ?? 10
  applicationUpdateForm.refreshTokenTTLHours = application.refreshTokenTTLHours ?? 168
}

function syncProjectUserAssignments(project?: any) {
  projectAssignedUserIds.value = Array.isArray(project?.assignedUserIds) ? [...project.assignedUserIds] : []
}

async function loadUserDetail(userID = selectedUserId.value) {
  if (!userID) {
    userDetail.value = null
    return
  }
  const detail = await apiPost<any>('/api/manage/v1/user/detail/query', { userId: userID })
  userDetail.value = detail
  syncUserEditState(detail.user)
  externalBindingForm.organizationId = detail.user?.organizationId ?? currentOrganizationId.value
  externalBindingForm.userId = detail.user?.id ?? userID
  externalBindingForm.externalIdpId = detail.externalIdps?.[0]?.id ?? externalBindingForm.externalIdpId
  externalBindingForm.issuer = detail.externalIdps?.find((item: any) => item.id === externalBindingForm.externalIdpId)?.issuer ?? externalBindingForm.issuer
}

function selectUser(user: any) {
  userViewMode.value = 'detail'
  selectedUserId.value = user.id
  syncUserEditState(user)
  loadUserDetail(user.id)
  router.push({
    name: 'console-user-detail',
    params: {
      organizationId: currentOrganizationId.value || currentOrganization.value?.id || '',
      userId: user.id ?? ''
    }
  })
}

async function selectProject(project: any) {
  selectedProjectId.value = project.id ?? ''
  syncProjectEditState(project)
  syncProjectUserAssignments(project)
  applicationQuery.projectId = project.id ?? ''
  applicationForm.projectId = project.id ?? ''
  await loadApplications()
  projectViewMode.value = 'detail'
  await router.push({
    name: 'console-project-detail',
    params: {
      organizationId: currentOrganizationId.value || currentOrganization.value?.id || '',
      projectId: project.id ?? ''
    }
  })
}

function selectApplication(application: any) {
  selectedApplicationId.value = application.id ?? ''
  syncApplicationEditState(application)
}

async function goApplicationDetail(application: any) {
  selectApplication(application)
  await router.push({
    name: 'console-application-detail',
    params: {
      organizationId: currentOrganizationId.value,
      projectId: selectedProjectId.value || currentProject.value?.id || '',
      applicationId: application.id ?? ''
    }
  })
}

function backToProjectList() {
  projectViewMode.value = 'list'
  router.push({ name: 'console-project-list', params: { organizationId: currentOrganizationId.value || currentOrganization.value?.id || '' } })
}

async function backToProjectDetail() {
  await router.push({
    name: 'console-project-detail',
    params: {
      organizationId: currentOrganizationId.value || currentOrganization.value?.id || '',
      projectId: selectedProjectId.value || currentProject.value?.id || ''
    }
  })
}

function backToUserList() {
  userViewMode.value = 'list'
  router.push({
    name: 'console-user-list',
    params: {
      organizationId: currentOrganizationId.value || currentOrganization.value?.id || ''
    }
  })
}

function toggleAllUsers(checked: boolean) {
  selectedUserIds.value = checked ? users.value.map((item: any) => item.id) : []
}

function toggleUserSelection(userId: string, checked: boolean) {
  if (checked) {
    if (!selectedUserIds.value.includes(userId)) {
      selectedUserIds.value = [...selectedUserIds.value, userId]
    }
    return
  }
  selectedUserIds.value = selectedUserIds.value.filter((id) => id !== userId)
}

async function deleteSelectedUsers() {
  await withFeedback(async () => {
    await apiPost('/api/manage/v1/user/delete', { userIds: selectedUserIds.value })
    selectedUserIds.value = []
    await loadUsers()
    await loadUserDetail()
  })
}

async function deleteSingleUser(userId: string) {
  await withFeedback(async () => {
    await apiPost('/api/manage/v1/user/delete', { userId })
    selectedUserIds.value = selectedUserIds.value.filter((id) => id !== userId)
    await loadUsers()
    await loadUserDetail()
    if (currentRouteName.value === 'console-user-detail') {
      if (selectedUserId.value) {
        await router.replace({
          name: 'console-user-detail',
          params: {
            organizationId: currentOrganizationId.value || currentOrganization.value?.id || '',
            userId: selectedUserId.value
          }
        })
      } else {
        await router.replace({
          name: 'console-user-list',
          params: {
            organizationId: currentOrganizationId.value || currentOrganization.value?.id || ''
          }
        })
      }
    }
  })
}

async function submitUserCreateFromList() {
  userForm.phoneNumber = composePhoneNumber(userPhoneInput)
  await createUser()
  resetPhoneInput(userPhoneInput)
  showCreateUserForm.value = false
}

async function loadRoles() {
  const response = await apiPost<{ items: any[] }>('/api/manage/v1/role/query', roleQuery)
  roles.value = response.items
  selectedRoleIds.value = selectedRoleIds.value.filter((id) => roles.value.some((item: any) => item.id === id))
  if (!roles.value.some((item: any) => item.id === selectedRoleId.value)) {
    selectedRoleId.value = roles.value[0]?.id ?? ''
  }
  if (selectedRole.value) {
    roleForm.organizationId = selectedRole.value.organizationId ?? currentOrganizationId.value
    roleForm.name = selectedRole.value.name ?? ''
    roleForm.type = selectedRole.value.type ?? 'user'
    roleForm.description = selectedRole.value.description ?? ''
    policyForm.roleId = selectedRole.value.id ?? ''
  }
}

function selectRole(role: any) {
  roleViewMode.value = 'detail'
  selectedRoleId.value = role.id ?? ''
  roleForm.organizationId = role.organizationId ?? currentOrganizationId.value
  roleForm.name = role.name ?? ''
  roleForm.type = role.type ?? 'user'
  roleForm.description = role.description ?? ''
  policyForm.roleId = role.id ?? ''
  policyCheckForm.subjectType = role.type === 'application' ? 'application' : 'user'
  router.push({
    name: 'console-role-detail',
    params: {
      organizationId: currentOrganizationId.value || currentOrganization.value?.id || '',
      roleId: role.id ?? ''
    }
  })
}

function backToRoleList() {
  roleViewMode.value = 'list'
  router.push({
    name: 'console-role-list',
    params: {
      organizationId: currentOrganizationId.value || currentOrganization.value?.id || ''
    }
  })
}

function toggleAllRoles(checked: boolean) {
  selectedRoleIds.value = checked ? roles.value.map((item: any) => item.id) : []
}

function toggleRolesByType(type: 'user' | 'application', checked: boolean) {
  const targetIds = roles.value.filter((item: any) => item.type === type).map((item: any) => item.id)
  if (checked) {
    selectedRoleIds.value = Array.from(new Set([...selectedRoleIds.value, ...targetIds]))
    return
  }
  selectedRoleIds.value = selectedRoleIds.value.filter((id) => !targetIds.includes(id))
}

function toggleRoleSelection(roleId: string, checked: boolean) {
  if (checked) {
    if (!selectedRoleIds.value.includes(roleId)) {
      selectedRoleIds.value = [...selectedRoleIds.value, roleId]
    }
    return
  }
  selectedRoleIds.value = selectedRoleIds.value.filter((id) => id !== roleId)
}

async function deleteSelectedRoles() {
  await withFeedback(async () => {
    await apiPost('/api/manage/v1/role/delete', { roleIds: selectedRoleIds.value })
    selectedRoleIds.value = []
    await loadRoles()
  })
}

async function deleteSingleRole(roleId: string) {
  await withFeedback(async () => {
    await apiPost('/api/manage/v1/role/delete', { roleId })
    selectedRoleIds.value = selectedRoleIds.value.filter((id) => id !== roleId)
    await loadRoles()
  })
}

async function submitRoleCreateFromList() {
  await createRole()
  showCreateRoleForm.value = false
}

async function handleOrganizationSwitch(value: string) {
  organizationSwitcher.value = value
  currentOrganizationId.value = value
  await loadAll()
  if (currentRouteName.value === 'console-organization') {
    await router.push({ name: 'console-organization', params: { organizationId: value } })
    return
  }
  if (currentRouteName.value === 'console-organization-manage') {
    await router.push({ name: 'console-organization-manage' })
    return
  }
  if (currentRouteName.value === 'console-project-list' || currentRouteName.value === 'console-project-create') {
    await router.push({ name: currentRouteName.value, params: { organizationId: value } })
    return
  }
  if (currentRouteName.value === 'console-project-detail') {
    await router.push({
      name: 'console-project-detail',
      params: {
        organizationId: value,
        projectId: selectedProjectId.value || currentProject.value?.id || ''
      }
    })
    return
  }
  if (currentRouteName.value === 'console-application-detail') {
    await router.push({
      name: 'console-application-detail',
      params: {
        organizationId: value,
        projectId: selectedProjectId.value || currentProject.value?.id || '',
        applicationId: selectedApplicationId.value || currentApplication.value?.id || ''
      }
    })
    return
  }
  if (currentRouteName.value === 'console-application-create') {
    await router.push({
      name: 'console-application-create',
      params: {
        organizationId: value,
        projectId: selectedProjectId.value || currentProject.value?.id || ''
      }
    })
    return
  }
  if (currentRouteName.value === 'console-user-list') {
    await router.push({ name: 'console-user-list', params: { organizationId: value } })
    return
  }
  if (currentRouteName.value === 'console-user-detail') {
    await router.push({
      name: 'console-user-detail',
      params: {
        organizationId: value,
        userId: selectedUserId.value || selectedUser.value?.id || ''
      }
    })
    return
  }
  if (currentRouteName.value === 'console-role-list') {
    await router.push({ name: 'console-role-list', params: { organizationId: value } })
    return
  }
  if (currentRouteName.value === 'console-role-detail') {
    await router.push({
      name: 'console-role-detail',
      params: {
        organizationId: value,
        roleId: selectedRoleId.value || selectedRole.value?.id || ''
      }
    })
  }
}

function scrollToPanel(id: string) {
  const target = document.getElementById(id)
  if (target) {
    const topbar = document.querySelector('.admin-topbar') as HTMLElement | null
    const offset = (topbar?.offsetHeight ?? 0) + 32
    const targetTop = target.getBoundingClientRect().top + window.scrollY - offset
    window.scrollTo({
      top: Math.max(targetTop, 0),
      behavior: 'smooth'
    })
  }
}

function scrollToTop() {
  window.scrollTo({
    top: 0,
    behavior: 'smooth'
  })
}

async function runModuleAction() {
  if (tab.value === 'organization') {
    await loadOrganizations()
    return
  }
  if (tab.value === 'project') {
    await Promise.all([loadProjects(), loadApplications(), loadOrganizations()])
    return
  }
  if (tab.value === 'user') {
    await Promise.all([loadUsers(), loadUserDetail()])
    return
  }
  if (tab.value === 'role') {
    await Promise.all([loadRoles(), loadPolicies()])
    return
  }
  if (tab.value === 'audit') {
    await loadAudit()
    return
  }
  if (tab.value === 'setting') {
    await loadExternalIDPs()
    return
  }
  await Promise.all([loadOrganizations(), loadUsers(), loadRoles(), loadPolicies(), loadExternalIDPs(), loadAudit()])
}

async function setTab(nextTab: 'dashboard' | 'organization' | 'project' | 'user' | 'role' | 'audit' | 'setting') {
  tab.value = nextTab
  if (nextTab === 'organization') await router.push({ name: 'console-organization', params: { organizationId: currentOrganizationId.value || currentOrganization.value?.id || '' } })
  else if (nextTab === 'project') {
    projectViewMode.value = 'list'
    await router.push({ name: 'console-project-list', params: { organizationId: currentOrganizationId.value || currentOrganization.value?.id || '' } })
  } else if (nextTab === 'user') {
    userViewMode.value = 'list'
    await router.push({ name: 'console-user-list', params: { organizationId: currentOrganizationId.value || currentOrganization.value?.id || '' } })
  } else if (nextTab === 'role') {
    roleViewMode.value = 'list'
    await router.push({ name: 'console-role-list', params: { organizationId: currentOrganizationId.value || currentOrganization.value?.id || '' } })
  }
  else if (nextTab === 'audit') await router.push({ name: 'console-audit', params: { organizationId: currentOrganizationId.value || currentOrganization.value?.id || '' } })
  else if (nextTab === 'setting') await router.push({ name: 'console-settings', params: { organizationId: currentOrganizationId.value || currentOrganization.value?.id || '' } })
  else await router.push({ name: 'console-dashboard' })
}

async function toggleManageOrganization() {
  await router.push({ name: 'console-organization-manage' })
}

async function closeOverlayView() {
  if (tab.value === 'project' && currentOrganizationId.value && selectedProjectId.value) {
    await router.push({
      name: 'console-project-detail',
      params: {
        organizationId: currentOrganizationId.value,
        projectId: selectedProjectId.value
      }
    })
    return
  }
  await setTab(tab.value)
}

async function goProjectCreate() {
  await router.push({
    name: 'console-project-create',
    params: {
      organizationId: currentOrganizationId.value
    }
  })
}

async function goApplicationCreate() {
  await router.push({
    name: 'console-application-create',
    params: {
      organizationId: currentOrganizationId.value || currentOrganization.value?.id || '',
      projectId: selectedProjectId.value || currentProject.value?.id || ''
    }
  })
}

async function goMy(hash = '') {
  const portalBaseUrl = import.meta.env.PPVT_CONSOLE_PORTAL_BASE_URL ?? 'http://localhost:8092'
  const suffix = hash.startsWith('#') ? hash : ''
  window.location.assign(`${portalBaseUrl}/portal/my${suffix}`)
}

async function goProfile() {
  await goMy('#my-basic')
}

async function goSecuritySetting() {
  await goMy('#my-login-setting')
}

async function logout() {
  sessionStorage.removeItem('ppvt-login-identifier')
  sessionStorage.removeItem('ppvt-external-idp-application-id')
  startConsoleLogout()
}

async function loadProfile() {
  if (!sessionStorage.getItem('ppvt-access-token')) {
    return
  }
  try {
    const accessToken = sessionStorage.getItem('ppvt-access-token') ?? ''
    const response = await fetch(`${authBaseUrl}/auth/userinfo`, {
      method: 'GET',
      headers: {
        Authorization: `Bearer ${accessToken}`
      },
      credentials: 'include'
    })
    if (!response.ok) {
      throw new Error(await response.text())
    }
    const result = await response.json()
    profileForm.id = result.sub ?? ''
    profileForm.organizationId = ''
    profileForm.username = result.preferred_username ?? ''
    profileForm.name = result.name ?? ''
    profileForm.email = result.email ?? ''
    profileForm.phoneNumber = result.phone_number ?? ''
    syncPhoneInput(profilePhoneInput, profileForm.phoneNumber)
    profileForm.status = 'active'
    profileForm.roles = []
    currentLoginUser.value = result.email ?? result.preferred_username ?? currentLoginUser.value
  } catch (error) {
    message.value = String(error)
    messageVariant.value = 'danger'
  }
}

async function loadCurrentUserDetail() {
  if (!sessionStorage.getItem('ppvt-access-token')) {
    userDetail.value = null
    return
  }
  try {
    const detail = await apiPost<any>('/api/user/v1/detail/query', {})
    userDetail.value = detail
    externalBindingForm.organizationId = detail.user?.organizationId ?? profileForm.organizationId
    externalBindingForm.userId = detail.user?.id ?? ''
    externalBindingForm.externalIdpId = detail.externalIdps?.[0]?.id ?? ''
    externalBindingForm.issuer = detail.externalIdps?.find((item: any) => item.id === externalBindingForm.externalIdpId)?.issuer ?? ''
  } catch (error) {
    message.value = String(error)
    messageVariant.value = 'danger'
  }
}

async function saveProfile() {
  if (!sessionStorage.getItem('ppvt-access-token')) {
    return
  }
  await withFeedback(async () => {
    const result = await apiPost<any>('/api/user/v1/profile/update', {
      username: profileForm.username,
      name: profileForm.name,
      email: profileForm.email,
      phoneNumber: composePhoneNumber(profilePhoneInput)
    })
    profileForm.username = result.username ?? ''
    profileForm.name = result.name ?? ''
    profileForm.email = result.email ?? ''
    profileForm.phoneNumber = result.phoneNumber ?? ''
    syncPhoneInput(profilePhoneInput, profileForm.phoneNumber)
    if (currentView.value === 'my') {
      await loadCurrentUserDetail()
    }
  })
}

async function loadSecuritySetting() {
  if (!sessionStorage.getItem('ppvt-access-token')) {
    return
  }
  try {
    const result = await apiPost<any>('/api/user/v1/setting/query', {})
    securitySetting.session = result.session ?? securitySetting.session
    securitySetting.devices = result.devices ?? []
  } catch (error) {
    message.value = String(error)
    messageVariant.value = 'danger'
  }
}

async function savePassword() {
  if (!sessionStorage.getItem('ppvt-access-token')) {
    return
  }
  await withFeedback(async () => {
    await apiPost('/api/user/v1/setting/update', {
      currentPassword: passwordForm.currentPassword,
      newPassword: passwordForm.newPassword
    })
    passwordForm.currentPassword = ''
    passwordForm.newPassword = ''
  })
}

async function untrustDevice(deviceId: string) {
  if (!sessionStorage.getItem('ppvt-access-token')) {
    return
  }
  await withFeedback(async () => {
    await apiPost('/api/user/v1/device/untrust', { deviceId })
    await loadSecuritySetting()
    if (currentView.value === 'my') {
      await loadCurrentUserDetail()
    }
  })
}

async function loadPolicies() {
  const response = await apiPost<{ items: any[] }>('/api/manage/v1/policy/query', {
    organizationId: currentOrganizationId.value || currentOrganization.value?.id || '',
  })
  policies.value = response.items
}

async function loadAudit() {
  const response = await apiPost<{ items: any[] }>('/api/manage/v1/audit_log/query', {})
  auditLogs.value = response.items
}

async function loadExternalIDPs() {
  const response = await apiPost<{ items: any[] }>('/api/manage/v1/external_idp/query', {
    organizationId: externalIDPForm.organizationId
  })
  externalIDPs.value = response.items
}

async function createOrganization() {
  await withFeedback(async () => {
    await apiPost('/api/manage/v1/organization/create', organizationForm)
    await loadOrganizations()
  })
}

async function updateOrganization() {
  await withFeedback(async () => {
    await apiPost('/api/manage/v1/organization/update', organizationUpdateForm)
    await loadOrganizations()
  })
}

function addOrganizationMetadataRow() {
  organizationMetadataRows.value.push({
    id: createLocalRowId(),
    key: '',
    value: ''
  })
}

function removeOrganizationMetadataRow(index: number) {
  organizationMetadataRows.value.splice(index, 1)
}

async function saveOrganizationMetadata() {
  if (!organizationUpdateForm.id) {
    return
  }
  await withFeedback(async () => {
    const metadata: Record<string, string> = {}
    for (const item of organizationMetadataRows.value) {
      const key = item.key.trim()
      if (!key) {
        continue
      }
      if (metadata[key] !== undefined) {
        throw new Error(`duplicate metadata key: ${key}`)
      }
      metadata[key] = item.value
    }
    organizationUpdateForm.metadata = metadata
    await apiPost('/api/manage/v1/organization/update', {
      id: organizationUpdateForm.id,
      metadata
    })
    await loadOrganizations()
  })
}

async function saveOrganizationBasicSettings() {
  await withFeedback(async () => {
    await saveOrganizationConsoleSettings({
      name: organizationBasicSettingForm.name.trim()
    })
  })
}

async function saveOrganizationDomainSettings() {
  await withFeedback(async () => {
    await saveOrganizationConsoleSettings()
  })
}

async function saveOrganizationLoginPolicy() {
  await withFeedback(async () => {
    if (!organizationLoginPolicyForm.allowUsername && !organizationLoginPolicyForm.allowEmail && !organizationLoginPolicyForm.allowPhone) {
      throw new Error('当前实现至少需要保留用户名、邮箱、手机其中一种登录方式')
    }
    await saveOrganizationConsoleSettings()
  })
}

async function saveOrganizationPasswordPolicy() {
  await withFeedback(async () => {
    await saveOrganizationConsoleSettings()
  })
}

async function saveOrganizationMFAPolicy() {
  await withFeedback(async () => {
    await saveOrganizationConsoleSettings()
  })
}

async function createProject() {
  await withFeedback(async () => {
    await apiPost('/api/manage/v1/project/create', projectForm)
    await loadProjects()
    await loadOrganizations()
  })
}

async function submitProjectCreatePage() {
  await createProject()
  projectForm.name = ''
  await router.push({ name: 'console-project-list', params: { organizationId: currentOrganizationId.value || currentOrganization.value?.id || '' } })
}

async function updateProject() {
  await withFeedback(async () => {
    await apiPost('/api/manage/v1/project/update', projectUpdateForm)
    await loadProjects()
    await loadOrganizations()
  })
}

async function createApplication() {
  normalizeApplicationProtocolSelection(applicationForm)
  const createProtocolError = validateApplicationProtocolInput(applicationForm)
  if (createProtocolError) {
    toast.error(createProtocolError)
    return ''
  }
  let createdApplicationId = ''
  await withFeedback(async () => {
    const created = await apiPost<any>('/api/manage/v1/application/create', {
      ...applicationForm,
      roles: [...applicationForm.roles],
      accessTokenTTLMinutes: Number(applicationForm.accessTokenTTLMinutes),
      refreshTokenTTLHours: Number(applicationForm.refreshTokenTTLHours),
    })
    createdApplicationId = created.id ?? ''
    applicationForm.publicKey = created.publicKey ?? ''
    if (created.generatedPrivateKey) {
      showApplicationPrivateKey(created.generatedPrivateKey, '应用私钥')
    }
    await loadApplications()
    await loadOrganizations()
  })
  return createdApplicationId
}

async function submitApplicationCreatePage() {
  const createdApplicationId = await createApplication()
  if (!createdApplicationId) {
    return
  }
  applicationForm.name = ''
  applicationForm.roles = []
  selectedApplicationId.value = createdApplicationId
  await router.push({
    name: 'console-application-detail',
    params: {
      organizationId: currentOrganizationId.value || currentOrganization.value?.id || '',
      projectId: selectedProjectId.value || currentProject.value?.id || '',
      applicationId: createdApplicationId
    }
  })
}

async function updateApplication() {
  normalizeApplicationProtocolSelection(applicationUpdateForm)
  const updateProtocolError = validateApplicationProtocolInput(applicationUpdateForm)
  if (updateProtocolError) {
    toast.error(updateProtocolError)
    return
  }
  await withFeedback(async () => {
    const updated = await apiPost<any>('/api/manage/v1/application/update', {
      ...applicationUpdateForm,
      roles: [...applicationUpdateForm.roles],
      accessTokenTTLMinutes: Number(applicationUpdateForm.accessTokenTTLMinutes),
      refreshTokenTTLHours: Number(applicationUpdateForm.refreshTokenTTLHours),
    })
    applicationUpdateForm.publicKey = updated.publicKey ?? applicationUpdateForm.publicKey
    if (updated.generatedPrivateKey) {
      showApplicationPrivateKey(updated.generatedPrivateKey, '应用私钥')
    }
    await loadApplications()
    await loadOrganizations()
  })
}

async function resetApplicationKey() {
  if (!applicationUpdateForm.id) {
    return
  }
  await withFeedback(async () => {
    const result = await apiPost<any>('/api/manage/v1/application/key/reset', {
      applicationId: applicationUpdateForm.id
    })
    applicationUpdateForm.publicKey = result.publicKey ?? ''
    if (result.generatedPrivateKey) {
      showApplicationPrivateKey(result.generatedPrivateKey, '重置后的应用私钥')
    }
    await loadApplications()
    await loadOrganizations()
  })
}

function showApplicationPrivateKey(privateKey: string, title: string) {
  applicationPrivateKeySnapshot.value = privateKey
  applicationKeyModalTitle.value = title
  applicationKeyModalVisible.value = true
}

async function createRole() {
  await withFeedback(async () => {
    await apiPost('/api/manage/v1/role/create', roleForm)
    await Promise.all([loadRoles(), loadPolicies()])
  })
}

async function updateRole() {
  if (!selectedRole.value?.id) {
    return
  }
  await withFeedback(async () => {
    await apiPost('/api/manage/v1/role/update', {
      id: selectedRole.value.id,
      name: roleForm.name,
      type: roleForm.type,
      description: roleForm.description
    })
    await Promise.all([loadRoles(), loadPolicies()])
  })
}

async function createUser() {
  await withFeedback(async () => {
    await apiPost('/api/manage/v1/user/create', {
      ...userForm,
      phoneNumber: composePhoneNumber(userPhoneInput),
      roles: splitRoleLabels(userForm.roleLabels)
    })
    resetPhoneInput(userPhoneInput)
    await loadUsers()
  })
}

async function updateUser() {
  await withFeedback(async () => {
    await apiPost('/api/manage/v1/user/update', {
      ...userUpdateForm,
      phoneNumber: composePhoneNumber(userUpdatePhoneInput),
      roles: userRoleAssignments.value
    })
    await loadUsers()
    await loadUserDetail()
  })
}

async function savePolicy() {
  if (!selectedRole.value?.id) {
    throw new Error('请先选择角色')
  }
  const payload = {
    id: policyForm.id || undefined,
    organizationId: currentOrganizationId.value || currentOrganization.value?.id || '',
    roleId: selectedRole.value.id,
    name: policyForm.name.trim(),
    effect: policyForm.effect,
    priority: Number(policyForm.priority),
    apiRules: JSON.parse(policyForm.apiRulesText || '[]')
  }
  await withFeedback(async () => {
    if (policyForm.id) {
      await apiPost('/api/manage/v1/policy/update', payload)
    } else {
      await apiPost('/api/manage/v1/policy/create', payload)
    }
    resetPolicyForm()
    await loadPolicies()
  })
}

function editPolicy(policy: any) {
  policyForm.id = policy.id ?? ''
  policyForm.organizationId = policy.organizationId ?? currentOrganizationId.value
  policyForm.roleId = policy.roleId ?? selectedRole.value?.id ?? ''
  policyForm.name = policy.name ?? ''
  policyForm.effect = policy.effect ?? 'allow'
  policyForm.priority = Number(policy.priority ?? 10)
  policyForm.apiRulesText = JSON.stringify(policy.apiRules ?? [], null, 2)
}

function resetPolicyForm() {
  policyForm.id = ''
  policyForm.organizationId = currentOrganizationId.value || currentOrganization.value?.id || ''
  policyForm.roleId = selectedRole.value?.id ?? ''
  policyForm.name = ''
  policyForm.effect = 'allow'
  policyForm.priority = 10
  policyForm.apiRulesText = '[\n  {\n    "method": "POST",\n    "path": "/api/manage/v1/example/query"\n  }\n]'
}

async function deletePolicy(policyId: string) {
  await withFeedback(async () => {
    await apiPost('/api/manage/v1/policy/delete', { policyId })
    if (policyForm.id === policyId) {
      resetPolicyForm()
    }
    await loadPolicies()
  })
}

async function evaluatePolicyCheck() {
  await withFeedback(async () => {
    decisionResult.value = await apiPost('/api/authz/v1/policy/check', {
      subjectType: policyCheckForm.subjectType,
      subjectId: policyCheckForm.subjectId.trim(),
      method: policyCheckForm.method.trim() || 'POST',
      path: policyCheckForm.path.trim()
    })
  })
}

async function createExternalIDP() {
  await withFeedback(async () => {
    await apiPost('/api/manage/v1/external_idp/create', {
      organizationId: currentOrganizationId.value || currentOrganization.value?.id || externalIDPForm.organizationId,
      protocol: externalIDPForm.protocol,
      name: externalIDPForm.name,
      issuer: externalIDPForm.issuer,
      clientId: externalIDPForm.clientId,
      clientSecret: externalIDPForm.clientSecret,
      scopes: externalIDPForm.scopes,
      authorizationUrl: externalIDPForm.authorizationUrl,
      tokenUrl: externalIDPForm.tokenUrl,
      userInfoUrl: externalIDPForm.userInfoUrl,
      jwksUrl: externalIDPForm.jwksUrl,
      metadata: {
        providerKind: currentExternalIDPKind.value
      }
    })
    await loadExternalIDPs()
  })
}

async function updateExternalIDP() {
  await withFeedback(async () => {
    await apiPost('/api/manage/v1/external_idp/update', {
      id: externalIDPForm.id,
      organizationId: currentOrganizationId.value || currentOrganization.value?.id || externalIDPForm.organizationId,
      protocol: externalIDPForm.protocol,
      name: externalIDPForm.name,
      issuer: externalIDPForm.issuer,
      clientId: externalIDPForm.clientId,
      clientSecret: externalIDPForm.clientSecret,
      scopes: externalIDPForm.scopes,
      authorizationUrl: externalIDPForm.authorizationUrl,
      tokenUrl: externalIDPForm.tokenUrl,
      userInfoUrl: externalIDPForm.userInfoUrl,
      jwksUrl: externalIDPForm.jwksUrl,
      metadata: {
        providerKind: currentExternalIDPKind.value
      }
    })
    await loadExternalIDPs()
  })
}

async function submitExternalIDPConfig() {
  if (externalIDPForm.id) {
    await updateExternalIDP()
  } else {
    await createExternalIDP()
  }
  externalIDPConfigModalVisible.value = false
}

async function createExternalBinding() {
  await withFeedback(async () => {
    if (currentView.value === 'my') {
      await apiPost('/api/user/v1/external_identity_binding/create', {
        externalIdpId: externalBindingForm.externalIdpId,
        issuer: externalBindingForm.issuer,
        subject: externalBindingForm.subject
      })
      externalBindingForm.subject = ''
      await loadCurrentUserDetail()
      return
    }
    externalBindingForm.userId = externalBindingForm.userId || selectedUserId.value
    await apiPost('/api/manage/v1/external_identity_binding/create', externalBindingForm)
    await loadUserDetail()
  })
}

async function registerSecureKey(purpose: 'webauthn' | 'u2f' = 'webauthn') {
  const userId = selectedUserId.value
  if (currentView.value !== 'my' && !userId) {
    return
  }
  await withFeedback(async () => {
    const begin = await apiPost<{ challengeId: string; options: any }>(
      currentView.value === 'my' ? '/api/user/v1/securekey/register/begin' : '/api/manage/v1/user/securekey/register/begin',
      currentView.value === 'my' ? { purpose } : { userId, purpose }
    )
    const credential = await navigator.credentials.create({
      publicKey: normalizeCreationOptions(begin.options)
    })
    if (!credential) {
      throw new Error('Secure key registration was cancelled')
    }
    await apiPost(currentView.value === 'my' ? '/api/user/v1/securekey/register/finish' : '/api/manage/v1/user/securekey/register/finish', {
      challengeId: begin.challengeId,
      response: serializeCredential(credential as PublicKeyCredential)
    })
    if (currentView.value === 'my') {
      await loadCurrentUserDetail()
      return
    }
    await loadUserDetail()
  })
}

async function enrollTotp() {
  const userId = selectedUserId.value
  if (currentView.value !== 'my' && !userId) {
    return
  }
  await withFeedback(async () => {
    totpSetup.value = await apiPost(
      currentView.value === 'my' ? '/api/user/v1/totp/enroll' : '/api/manage/v1/user/totp/enroll',
      currentView.value === 'my'
        ? { applicationId: consoleApplicationId }
        : { userId, applicationId: consoleApplicationId }
    )
    totpVerifyForm.enrollmentId = (totpSetup.value as { enrollmentId?: string })?.enrollmentId ?? ''
    if (currentView.value === 'my') {
      await loadCurrentUserDetail()
      return
    }
    await loadUserDetail()
  })
}

async function verifyTotpEnrollment() {
  const userId = selectedUserId.value
  if (currentView.value !== 'my' && !userId) {
    return
  }
  await withFeedback(async () => {
    await apiPost(
      currentView.value === 'my' ? '/api/user/v1/totp/verify' : '/api/manage/v1/user/totp/verify',
      currentView.value === 'my'
        ? {
            enrollmentId: pendingTotpEnrollmentId.value,
            code: totpVerifyForm.code
          }
        : {
            userId,
            enrollmentId: pendingTotpEnrollmentId.value,
            code: totpVerifyForm.code
          }
    )
    totpSetup.value = null
    totpVerifyForm.enrollmentId = ''
    totpVerifyForm.code = ''
    if (currentView.value === 'my') {
      await loadCurrentUserDetail()
      return
    }
    await loadUserDetail()
  })
}

async function generateRecoveryCodes() {
  const userId = selectedUserId.value
  if (currentView.value !== 'my' && !userId) {
    return
  }
  await withFeedback(async () => {
    recoveryCodes.value = await apiPost(
      currentView.value === 'my' ? '/api/user/v1/recovery_code/generate' : '/api/manage/v1/user/recovery_code/generate',
      currentView.value === 'my' ? {} : { userId }
    )
    if (currentView.value === 'my') {
      await loadCurrentUserDetail()
      return
    }
    await loadUserDetail()
  })
}

async function saveMFAEmailSetting() {
  if (!selectedUserId.value) {
    return
  }
  await withFeedback(async () => {
    await apiPost('/api/manage/v1/user/mfa_method/update', {
      userId: selectedUserId.value,
      method: 'email_code',
      enabled: mfaSettingForm.emailEnabled === 'active'
    })
    await loadUserDetail()
    mfaConfigModalVisible.value = false
  })
}

async function saveMFASMSSetting() {
  if (!selectedUserId.value) {
    return
  }
  await withFeedback(async () => {
    await apiPost('/api/manage/v1/user/mfa_method/update', {
      userId: selectedUserId.value,
      method: 'sms_code',
      enabled: mfaSettingForm.smsEnabled === 'active'
    })
    await loadUserDetail()
    mfaConfigModalVisible.value = false
  })
}

async function toggleInlineMFAMethod(method: MFAMethod, enabled: boolean) {
  if (method !== 'email_code' && method !== 'sms_code') {
    return
  }
  if (currentView.value !== 'my' && !selectedUserId.value) {
    return
  }
  await withFeedback(async () => {
    await apiPost(currentView.value === 'my' ? '/api/user/v1/mfa_method/update' : '/api/manage/v1/user/mfa_method/update', currentView.value === 'my'
      ? {
          method,
          enabled
        }
      : {
          userId: selectedUserId.value,
          method,
          enabled
        })
    if (currentView.value === 'my') {
      await loadCurrentUserDetail()
      return
    }
    await loadUserDetail()
  })
}

async function handleInlineMFAMethodAction(item: { id: MFAMethod; enabled: boolean; disabled?: boolean }) {
  if (item.id !== 'email_code' && item.id !== 'sms_code') {
    return
  }
  if (item.disabled) {
    message.value = item.id === 'email_code' ? '请先在基本信息中配置邮箱' : '请先在基本信息中配置手机'
    messageVariant.value = 'danger'
    return
  }
  await toggleInlineMFAMethod(item.id, !item.enabled)
}

async function submitCurrentMFAModal() {
  if (currentMFAMethod.value === 'totp') {
    await verifyTotpEnrollment()
    return
  }
  if (currentMFAMethod.value === 'email_code') {
    await saveMFAEmailSetting()
    return
  }
  if (currentMFAMethod.value === 'sms_code') {
    await saveMFASMSSetting()
    return
  }
  if (currentMFAMethod.value === 'u2f') {
    await registerSecureKey('u2f')
    return
  }
  if (currentMFAMethod.value === 'recovery_code') {
    await generateRecoveryCodes()
  }
}

async function deleteTotpEnrollments() {
  if (currentView.value !== 'my' && !selectedUserId.value) {
    return
  }
  await withFeedback(async () => {
    await apiPost(currentView.value === 'my' ? '/api/user/v1/mfa_enrollment/delete' : '/api/manage/v1/user/mfa_enrollment/delete', currentView.value === 'my'
      ? {
          method: 'totp'
        }
      : {
          userId: selectedUserId.value,
          method: 'totp'
        })
    totpSetup.value = null
    totpVerifyForm.enrollmentId = ''
    totpVerifyForm.code = ''
    if (currentView.value === 'my') {
      await loadCurrentUserDetail()
    } else {
      await loadUserDetail()
    }
    mfaConfigModalVisible.value = false
  })
}

async function deleteSecureKey(credentialId: string) {
  if (currentView.value !== 'my' && !selectedUserId.value) {
    return
  }
  await withFeedback(async () => {
    await apiPost(currentView.value === 'my' ? '/api/user/v1/securekey/delete' : '/api/manage/v1/user/securekey/delete', currentView.value === 'my'
      ? {
          credentialId
        }
      : {
          userId: selectedUserId.value,
          credentialId
        })
    if (currentView.value === 'my') {
      await loadCurrentUserDetail()
      return
    }
    await loadUserDetail()
  })
}

async function toggleWebAuthnLogin(enabled: boolean) {
  if (currentView.value !== 'my' && !selectedUserId.value) {
    return
  }
  await withFeedback(async () => {
    await apiPost(currentView.value === 'my' ? '/api/user/v1/mfa_method/update' : '/api/manage/v1/user/mfa_method/update', currentView.value === 'my'
      ? {
          method: 'webauthn',
          enabled
        }
      : {
          userId: selectedUserId.value,
          method: 'webauthn',
          enabled
        })
    if (currentView.value === 'my') {
      await loadCurrentUserDetail()
      return
    }
    await loadUserDetail()
  }, enabled ? '已启用通行密钥登录' : '已关闭通行密钥登录')
}

async function resetUserPassword() {
  if (!selectedUserId.value) {
    return
  }
  await withFeedback(async () => {
    userAdminResult.value = await apiPost('/api/manage/v1/user/reset_password', {
      userId: selectedUserId.value,
      password: userAdminForm.password
    })
    userAdminForm.password = ''
  })
}

async function resetUserUkid() {
  if (!selectedUserId.value) {
    return
  }
  await withFeedback(async () => {
    userAdminResult.value = await apiPost('/api/manage/v1/user/reset_ukid', {
      userId: selectedUserId.value
    })
    await loadUserDetail()
  })
}

async function disableUser() {
  if (!selectedUserId.value) {
    return
  }
  await withFeedback(async () => {
    userAdminResult.value = await apiPost('/api/manage/v1/user/disable', {
      userId: selectedUserId.value
    })
    await loadUsers()
    await loadUserDetail()
  })
}

async function enableUser() {
  if (!selectedUserId.value) {
    return
  }
  await withFeedback(async () => {
    userAdminResult.value = await apiPost('/api/manage/v1/user/enable', {
      userId: selectedUserId.value
    })
    await loadUsers()
    await loadUserDetail()
  })
}

async function deleteExternalBinding(bindingId: string) {
  if (currentView.value !== 'my' && !selectedUserId.value) {
    return
  }
  await withFeedback(async () => {
    await apiPost(currentView.value === 'my' ? '/api/user/v1/external_identity_binding/delete' : '/api/manage/v1/external_identity_binding/delete', currentView.value === 'my'
      ? {
          bindingId
        }
      : {
          userId: selectedUserId.value,
          bindingId
        })
    if (currentView.value === 'my') {
      await loadCurrentUserDetail()
      return
    }
    await loadUserDetail()
  })
}

async function untrustManagedDevice(deviceId: string) {
  if (!selectedUserId.value) {
    return
  }
  await withFeedback(async () => {
    await apiPost('/api/manage/v1/user/device/untrust', {
      userId: selectedUserId.value,
      deviceId
    })
    await loadUserDetail()
  })
}

async function revokeAllUserSessions() {
  if (!selectedUserId.value) {
    return
  }
  await withFeedback(async () => {
    userAdminResult.value = await apiPost('/api/manage/v1/user/session/revoke_all', {
      userId: selectedUserId.value
    })
    await loadUserDetail()
  })
}

async function rotateUserToken() {
  if (!selectedUserId.value) {
    return
  }
  await withFeedback(async () => {
    userAdminResult.value = await apiPost('/api/manage/v1/user/token/rotate', {
      userId: selectedUserId.value
    })
    await loadUserDetail()
  })
}

function toggleUserRole(roleName: string, checked: boolean) {
  if (checked) {
    if (!userRoleAssignments.value.includes(roleName)) {
      userRoleAssignments.value = [...userRoleAssignments.value, roleName]
    }
    return
  }
  userRoleAssignments.value = userRoleAssignments.value.filter((item) => item !== roleName)
}

function toggleProjectUserAssignment(userId: string, checked: boolean) {
  if (checked) {
    if (!projectAssignedUserIds.value.includes(userId)) {
      projectAssignedUserIds.value = [...projectAssignedUserIds.value, userId]
    }
    return
  }
  projectAssignedUserIds.value = projectAssignedUserIds.value.filter((item) => item !== userId)
}

function openProjectUserAssignmentModal() {
  projectAssignmentDraftUserIds.value = [...projectAssignedUserIds.value]
  projectUserAssignmentModalVisible.value = true
}

function toggleProjectAssignmentDraftUser(userId: string, checked: boolean) {
  if (checked) {
    if (!projectAssignmentDraftUserIds.value.includes(userId)) {
      projectAssignmentDraftUserIds.value = [...projectAssignmentDraftUserIds.value, userId]
    }
    return
  }
  projectAssignmentDraftUserIds.value = projectAssignmentDraftUserIds.value.filter((item) => item !== userId)
}

function selectAllProjectAssignmentUsers() {
  projectAssignmentDraftUserIds.value = users.value.map((item: any) => item.id)
}

function invertProjectAssignmentUsers() {
  const selectedSet = new Set(projectAssignmentDraftUserIds.value)
  projectAssignmentDraftUserIds.value = users.value
    .map((item: any) => item.id)
    .filter((id: string) => !selectedSet.has(id))
}

function clearProjectAssignmentUsers() {
  projectAssignmentDraftUserIds.value = []
}

function confirmProjectUserAssignmentModal() {
  projectAssignedUserIds.value = [...projectAssignmentDraftUserIds.value]
  projectUserAssignmentModalVisible.value = false
}

function removeProjectAssignedUser(userId: string) {
  projectAssignedUserIds.value = projectAssignedUserIds.value.filter((item) => item !== userId)
}

async function saveProjectUserAssignments() {
  if (!selectedProjectId.value) {
    return
  }
  await withFeedback(async () => {
    const response = await apiPost<{ userIds: string[] }>('/api/manage/v1/project/user_assignment/update', {
      projectId: selectedProjectId.value,
      userIds: projectAssignedUserIds.value
    })
    projectAssignedUserIds.value = [...(response.userIds ?? [])]
    await loadProjects()
  }, '用户分配已保存')
}

function syncExternalBindingIssuer() {
  const provider = userDetail.value?.externalIdps?.find((item: any) => item.id === externalBindingForm.externalIdpId)
  if (provider?.issuer) {
    externalBindingForm.issuer = provider.issuer
  }
}

async function openMFAModal(method: MFAMethod) {
  currentMFAMethod.value = method
  mfaSettingForm.emailEnabled = emailCodeEnrollment.value?.status === 'active' ? 'active' : 'disabled'
  mfaSettingForm.smsEnabled = smsCodeEnrollment.value?.status === 'active' ? 'active' : 'disabled'
  mfaConfigModalVisible.value = true
  if (method === 'totp' && activeTOTPEnrollments.value.length === 0 && !pendingTotpProvisioningUri.value) {
    await enrollTotp()
  }
}

async function withFeedback(fn: () => Promise<void>, successMessage = '操作成功') {
  try {
    await fn()
    message.value = successMessage
    messageVariant.value = 'success'
  } catch (error) {
    message.value = String(error)
    messageVariant.value = 'danger'
  }
}

function splitRoleLabels(value: string) {
  return value
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
}

function syncOrganizationMetadataRows(organization?: any) {
  const entries = Object.entries(normalizeMetadataMap(organization?.metadata))
  organizationMetadataRows.value = entries.map(([key, value]) => ({
    id: createLocalRowId(),
    key,
    value
  }))
}

function parseOrganizationConsoleSettings(organization?: any): OrganizationConsoleSettings {
  const defaults: OrganizationConsoleSettings = {
    tosUrl: '',
    privacyPolicyUrl: '',
    supportEmail: '',
    logoUrl: '',
    domains: [],
    loginPolicy: {
      passwordLoginEnabled: true,
      webauthnLoginEnabled: true,
      allowUsername: true,
      allowEmail: true,
      allowPhone: true,
      usernameMode: 'optional',
      emailMode: 'required',
      phoneMode: 'optional'
    },
    passwordPolicy: {
      minLength: 12,
      requireUppercase: true,
      requireLowercase: true,
      requireNumber: true,
      requireSymbol: false,
      passwordExpires: false,
      expiryDays: 90
    },
    mfaPolicy: {
      requireForAllUsers: false,
      allowWebauthn: true,
      allowTotp: true,
      allowEmailCode: true,
      allowSmsCode: false,
      allowU2f: true,
      allowRecoveryCode: true,
      emailChannel: {
        enabled: false,
        from: '',
        host: '',
        port: 587,
        username: '',
        password: ''
      }
    }
  }
  const parsed = organization?.consoleSettings
  if (!parsed || typeof parsed !== 'object') {
    return defaults
  }
  return {
    ...defaults,
    ...parsed,
    loginPolicy: { ...defaults.loginPolicy, ...(parsed.loginPolicy || {}) },
    passwordPolicy: { ...defaults.passwordPolicy, ...(parsed.passwordPolicy || {}) },
    mfaPolicy: {
      ...defaults.mfaPolicy,
      ...(parsed.mfaPolicy || {}),
      emailChannel: {
        ...defaults.mfaPolicy.emailChannel,
        ...((parsed.mfaPolicy && parsed.mfaPolicy.emailChannel) || {})
      }
    },
    domains: Array.isArray(parsed.domains)
      ? parsed.domains.map((item: any) => ({
          host: String(item.host || ''),
          verified: Boolean(item.verified)
        })).filter((item: OrganizationConsoleSettings['domains'][number]) => item.host)
      : []
  }
}

function syncOrganizationSettingForms(organization?: any) {
  const settings = parseOrganizationConsoleSettings(organization)
  externalIDPForm.organizationId = organization?.id || externalIDPForm.organizationId
  organizationBasicSettingForm.name = organization?.name || ''
  organizationBasicSettingForm.tosUrl = settings.tosUrl
  organizationBasicSettingForm.privacyPolicyUrl = settings.privacyPolicyUrl
  organizationBasicSettingForm.supportEmail = settings.supportEmail
  organizationBasicSettingForm.logoUrl = settings.logoUrl
  organizationLoginPolicyForm.passwordLoginEnabled = settings.loginPolicy.passwordLoginEnabled
  organizationLoginPolicyForm.webauthnLoginEnabled = settings.loginPolicy.webauthnLoginEnabled
  organizationLoginPolicyForm.allowUsername = settings.loginPolicy.allowUsername
  organizationLoginPolicyForm.allowEmail = settings.loginPolicy.allowEmail
  organizationLoginPolicyForm.allowPhone = settings.loginPolicy.allowPhone
  organizationLoginPolicyForm.usernameMode = settings.loginPolicy.usernameMode
  organizationLoginPolicyForm.emailMode = settings.loginPolicy.emailMode
  organizationLoginPolicyForm.phoneMode = settings.loginPolicy.phoneMode
  organizationPasswordPolicyForm.minLength = settings.passwordPolicy.minLength
  organizationPasswordPolicyForm.requireUppercase = settings.passwordPolicy.requireUppercase
  organizationPasswordPolicyForm.requireLowercase = settings.passwordPolicy.requireLowercase
  organizationPasswordPolicyForm.requireNumber = settings.passwordPolicy.requireNumber
  organizationPasswordPolicyForm.requireSymbol = settings.passwordPolicy.requireSymbol
  organizationPasswordPolicyForm.passwordExpires = settings.passwordPolicy.passwordExpires
  organizationPasswordPolicyForm.expiryDays = settings.passwordPolicy.expiryDays
  organizationMFAPolicyForm.requireForAllUsers = settings.mfaPolicy.requireForAllUsers
  organizationMFAPolicyForm.allowWebauthn = settings.mfaPolicy.allowWebauthn
  organizationMFAPolicyForm.allowTotp = settings.mfaPolicy.allowTotp
  organizationMFAPolicyForm.allowEmailCode = settings.mfaPolicy.allowEmailCode
  organizationMFAPolicyForm.allowSmsCode = settings.mfaPolicy.allowSmsCode
  organizationMFAPolicyForm.allowU2f = settings.mfaPolicy.allowU2f
  organizationMFAPolicyForm.allowRecoveryCode = settings.mfaPolicy.allowRecoveryCode
  organizationMFAPolicyForm.emailChannelEnabled = settings.mfaPolicy.emailChannel.enabled
  organizationMFAPolicyForm.emailChannelFrom = settings.mfaPolicy.emailChannel.from
  organizationMFAPolicyForm.emailChannelHost = settings.mfaPolicy.emailChannel.host
  organizationMFAPolicyForm.emailChannelPort = settings.mfaPolicy.emailChannel.port
  organizationMFAPolicyForm.emailChannelUsername = settings.mfaPolicy.emailChannel.username
  organizationMFAPolicyForm.emailChannelPassword = settings.mfaPolicy.emailChannel.password
  organizationDomainRows.value = settings.domains.map((item) => ({
    id: createLocalRowId(),
    host: item.host,
    verified: item.verified
  }))
}

function normalizeMetadataMap(value: unknown) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return {} as Record<string, string>
  }
  return Object.fromEntries(
    Object.entries(value as Record<string, unknown>).map(([key, entryValue]) => [key, String(entryValue ?? '')])
  )
}

function createLocalRowId() {
  return `${Date.now()}-${Math.random().toString(16).slice(2)}`
}

function formatRoleLabels(value?: string[]) {
  if (!value || value.length === 0) {
    return 'none'
  }
  return value.join(', ')
}

function syncPhoneInput(target: PhoneInputState, value?: string) {
  const normalized = String(value || '').trim()
  if (!normalized) {
    target.countryCode = '+86'
    target.localNumber = ''
    return
  }
  const matched = phoneCountryOptions
    .map((item) => item.value)
    .sort((left, right) => right.length - left.length)
    .find((code) => normalized.startsWith(code))
  if (matched) {
    target.countryCode = matched
    target.localNumber = normalized.slice(matched.length).replace(/^[\\s-]+/, '')
    return
  }
  target.countryCode = '+86'
  target.localNumber = normalized.replace(/^\\+?86[\\s-]*/, '')
}

function composePhoneNumber(source: PhoneInputState) {
  const localNumber = source.localNumber.trim()
  if (!localNumber) {
    return ''
  }
  return `${source.countryCode}${localNumber}`
}

function resetPhoneInput(target: PhoneInputState) {
  target.countryCode = '+86'
  target.localNumber = ''
}

function buildOrganizationConsoleSettings(): OrganizationConsoleSettings {
  return {
    tosUrl: organizationBasicSettingForm.tosUrl.trim(),
    privacyPolicyUrl: organizationBasicSettingForm.privacyPolicyUrl.trim(),
    supportEmail: organizationBasicSettingForm.supportEmail.trim(),
    logoUrl: organizationBasicSettingForm.logoUrl.trim(),
    domains: organizationDomainRows.value
      .map((item) => ({
        host: item.host.trim(),
        verified: item.verified
      }))
      .filter((item) => item.host),
    loginPolicy: {
      passwordLoginEnabled: organizationLoginPolicyForm.passwordLoginEnabled,
      webauthnLoginEnabled: organizationLoginPolicyForm.webauthnLoginEnabled,
      allowUsername: organizationLoginPolicyForm.allowUsername,
      allowEmail: organizationLoginPolicyForm.allowEmail,
      allowPhone: organizationLoginPolicyForm.allowPhone,
      usernameMode: organizationLoginPolicyForm.usernameMode,
      emailMode: organizationLoginPolicyForm.emailMode,
      phoneMode: organizationLoginPolicyForm.phoneMode
    },
    passwordPolicy: {
      minLength: Number(organizationPasswordPolicyForm.minLength),
      requireUppercase: organizationPasswordPolicyForm.requireUppercase,
      requireLowercase: organizationPasswordPolicyForm.requireLowercase,
      requireNumber: organizationPasswordPolicyForm.requireNumber,
      requireSymbol: organizationPasswordPolicyForm.requireSymbol,
      passwordExpires: organizationPasswordPolicyForm.passwordExpires,
      expiryDays: Number(organizationPasswordPolicyForm.expiryDays)
    },
    mfaPolicy: {
      requireForAllUsers: organizationMFAPolicyForm.requireForAllUsers,
      allowWebauthn: organizationMFAPolicyForm.allowWebauthn,
      allowTotp: organizationMFAPolicyForm.allowTotp,
      allowEmailCode: organizationMFAPolicyForm.allowEmailCode,
      allowSmsCode: organizationMFAPolicyForm.allowSmsCode,
      allowU2f: organizationMFAPolicyForm.allowU2f,
      allowRecoveryCode: organizationMFAPolicyForm.allowRecoveryCode,
      emailChannel: {
        enabled: organizationMFAPolicyForm.emailChannelEnabled,
        from: organizationMFAPolicyForm.emailChannelFrom.trim(),
        host: organizationMFAPolicyForm.emailChannelHost.trim(),
        port: Number(organizationMFAPolicyForm.emailChannelPort),
        username: organizationMFAPolicyForm.emailChannelUsername.trim(),
        password: organizationMFAPolicyForm.emailChannelPassword
      }
    }
  }
}

async function saveOrganizationConsoleSettings(options: { name?: string } = {}) {
  if (!currentOrganization.value?.id) {
    return
  }
  await apiPost('/api/manage/v1/organization/update', {
    id: currentOrganization.value.id,
    name: options.name ?? '',
    consoleSettings: buildOrganizationConsoleSettings()
  })
  await loadOrganizations()
}

function addOrganizationDomainRow() {
  organizationDomainRows.value.push({
    id: createLocalRowId(),
    host: '',
    verified: false
  })
}

function removeOrganizationDomainRow(index: number) {
  organizationDomainRows.value.splice(index, 1)
}

function verifyOrganizationDomain(index: number) {
  const item = organizationDomainRows.value[index]
  if (!item || !item.host.trim()) {
    message.value = '请先填写域名'
    messageVariant.value = 'danger'
    return
  }
  item.verified = true
  message.value = '域名已标记为已验证'
  messageVariant.value = 'success'
}

function providerPreset(kind: 'google' | 'github' | 'apple' | 'qq' | 'weibo' | 'custom_oauth' | 'custom_oidc') {
  if (kind === 'google') {
    return {
      protocol: 'oidc',
      name: 'Google',
      issuer: 'https://accounts.google.com',
      scopes: 'openid profile email',
      authorizationUrl: 'https://accounts.google.com/o/oauth2/v2/auth',
      tokenUrl: 'https://oauth2.googleapis.com/token',
      userInfoUrl: 'https://openidconnect.googleapis.com/v1/userinfo',
      jwksUrl: 'https://www.googleapis.com/oauth2/v3/certs'
    }
  }
  if (kind === 'github') {
    return {
      protocol: 'oauth',
      name: 'GitHub',
      issuer: 'https://github.com',
      scopes: 'read:user user:email',
      authorizationUrl: 'https://github.com/login/oauth/authorize',
      tokenUrl: 'https://github.com/login/oauth/access_token',
      userInfoUrl: 'https://api.github.com/user',
      jwksUrl: ''
    }
  }
  if (kind === 'apple') {
    return {
      protocol: 'oidc',
      name: 'Apple',
      issuer: 'https://appleid.apple.com',
      scopes: 'name email',
      authorizationUrl: 'https://appleid.apple.com/auth/authorize',
      tokenUrl: 'https://appleid.apple.com/auth/token',
      userInfoUrl: '',
      jwksUrl: 'https://appleid.apple.com/auth/keys'
    }
  }
  if (kind === 'qq') {
    return {
      protocol: 'oauth',
      name: 'QQ',
      issuer: 'https://graph.qq.com',
      scopes: 'get_user_info',
      authorizationUrl: 'https://graph.qq.com/oauth2.0/authorize',
      tokenUrl: 'https://graph.qq.com/oauth2.0/token',
      userInfoUrl: 'https://graph.qq.com/user/get_user_info',
      jwksUrl: ''
    }
  }
  if (kind === 'weibo') {
    return {
      protocol: 'oauth',
      name: 'Weibo',
      issuer: 'https://api.weibo.com',
      scopes: 'email',
      authorizationUrl: 'https://api.weibo.com/oauth2/authorize',
      tokenUrl: 'https://api.weibo.com/oauth2/access_token',
      userInfoUrl: 'https://api.weibo.com/2/users/show.json',
      jwksUrl: ''
    }
  }
  if (kind === 'custom_oauth') {
    return {
      protocol: 'oauth',
      name: 'Custom OAuth',
      issuer: '',
      scopes: '',
      authorizationUrl: '',
      tokenUrl: '',
      userInfoUrl: '',
      jwksUrl: ''
    }
  }
  return {
    protocol: 'oidc',
    name: 'Custom OIDC',
    issuer: '',
    scopes: 'openid profile email',
    authorizationUrl: '',
    tokenUrl: '',
    userInfoUrl: '',
    jwksUrl: ''
  }
}

function providerKindName(kind: 'google' | 'github' | 'apple' | 'qq' | 'weibo' | 'custom_oauth' | 'custom_oidc') {
  if (kind === 'google') return 'google'
  if (kind === 'github') return 'github'
  if (kind === 'apple') return 'apple'
  if (kind === 'qq') return 'qq'
  if (kind === 'weibo') return 'weibo'
  if (kind === 'custom_oauth') return 'custom oauth'
  return 'custom oidc'
}

function isCustomExternalIDPKind(kind: 'google' | 'github' | 'apple' | 'qq' | 'weibo' | 'custom_oauth' | 'custom_oidc') {
  return kind === 'custom_oauth' || kind === 'custom_oidc'
}

function normalizeProviderKind(item: any) {
  const metadataKind = normalizeProviderName(item?.metadata?.providerKind)
  if (metadataKind) {
    return metadataKind
  }
  const normalizedName = normalizeProviderName(item?.name)
  if (normalizedName === 'google') return 'google'
  if (normalizedName === 'github') return 'github'
  if (normalizedName === 'apple') return 'apple'
  if (normalizedName === 'qq') return 'qq'
  if (normalizedName === 'weibo' || normalizedName === '新浪微博') return 'weibo'
  if (item?.protocol === 'oidc') return 'custom_oidc'
  return 'custom_oauth'
}

function findExistingExternalIDP(kind: 'google' | 'github' | 'apple' | 'qq' | 'weibo' | 'custom_oauth' | 'custom_oidc') {
  if (isCustomExternalIDPKind(kind)) {
    return null
  }
  const expectedName = providerKindName(kind)
  return externalIDPs.value.find((item: any) => normalizeProviderKind(item) === expectedName || normalizeProviderName(item.name) === expectedName) || null
}

function openExternalIDPModal(kind: 'google' | 'github' | 'apple' | 'qq' | 'weibo' | 'custom_oauth' | 'custom_oidc') {
  currentExternalIDPKind.value = kind
  const existing = findExistingExternalIDP(kind)
  const preset = providerPreset(kind)
  externalIDPForm.id = existing?.id || ''
  externalIDPForm.organizationId = currentOrganizationId.value || currentOrganization.value?.id || ''
  externalIDPForm.protocol = existing?.protocol || preset.protocol
  externalIDPForm.name = existing?.name || preset.name
  externalIDPForm.issuer = existing?.issuer || preset.issuer
  externalIDPForm.clientId = existing?.clientId || ''
  externalIDPForm.clientSecret = ''
  externalIDPForm.scopes = existing?.scopes || preset.scopes
  externalIDPForm.authorizationUrl = existing?.authorizationUrl || preset.authorizationUrl
  externalIDPForm.tokenUrl = existing?.tokenUrl || preset.tokenUrl
  externalIDPForm.userInfoUrl = existing?.userInfoUrl || preset.userInfoUrl
  externalIDPForm.jwksUrl = existing?.jwksUrl || preset.jwksUrl
  externalIDPConfigModalVisible.value = true
}

function openExistingExternalIDP(item: any) {
  const kind = normalizeProviderKind(item) as 'google' | 'github' | 'apple' | 'qq' | 'weibo' | 'custom_oauth' | 'custom_oidc'
  currentExternalIDPKind.value = kind
  externalIDPForm.id = item.id || ''
  externalIDPForm.organizationId = currentOrganizationId.value || currentOrganization.value?.id || ''
  externalIDPForm.protocol = item.protocol || providerPreset(kind).protocol
  externalIDPForm.name = item.name || ''
  externalIDPForm.issuer = item.issuer || ''
  externalIDPForm.clientId = item.clientId || ''
  externalIDPForm.clientSecret = ''
  externalIDPForm.scopes = item.scopes || ''
  externalIDPForm.authorizationUrl = item.authorizationUrl || ''
  externalIDPForm.tokenUrl = item.tokenUrl || ''
  externalIDPForm.userInfoUrl = item.userInfoUrl || ''
  externalIDPForm.jwksUrl = item.jwksUrl || ''
  externalIDPConfigModalVisible.value = true
}

function normalizeProviderName(value?: string) {
  return String(value || '').trim().toLowerCase()
}

async function showProjectDisableNotice() {
  if (!selectedProjectId.value) {
    return
  }
  await withFeedback(async () => {
    await apiPost('/api/manage/v1/project/disable', { projectId: selectedProjectId.value })
    await Promise.all([loadProjects(), loadApplications(), loadOrganizations()])
  }, '项目已停用')
}

async function showOrganizationDisableNotice() {
  if (!currentOrganization.value?.id) {
    return
  }
  await withFeedback(async () => {
    await apiPost('/api/manage/v1/organization/disable', { organizationId: currentOrganization.value.id })
    await Promise.all([loadOrganizations(), loadProjects(), loadApplications()])
  }, '组织已停用')
}

async function showOrganizationDeleteNotice() {
  if (!currentOrganization.value?.id) {
    return
  }
  const deletedOrganizationId = currentOrganization.value.id
  await withFeedback(async () => {
    await apiPost('/api/manage/v1/organization/delete', { organizationId: deletedOrganizationId })
    await Promise.all([loadOrganizations(), loadUsers(), loadRoles(), loadPolicies(), loadExternalIDPs(), loadAudit()])
    const fallbackOrganization = organizations.value.find((item: any) => item.id !== deletedOrganizationId) || organizations.value[0]
    currentOrganizationId.value = fallbackOrganization?.id ?? ''
    organizationSwitcher.value = currentOrganizationId.value
    projectQuery.organizationId = currentOrganizationId.value
    userQuery.organizationId = currentOrganizationId.value
    roleQuery.organizationId = currentOrganizationId.value
    policyForm.organizationId = currentOrganizationId.value
    roleForm.organizationId = currentOrganizationId.value
    userForm.organizationId = currentOrganizationId.value
    externalIDPForm.organizationId = currentOrganizationId.value
    externalBindingForm.organizationId = currentOrganizationId.value
    if (currentOrganizationId.value) {
      await Promise.all([loadProjects(), loadApplications()])
      await router.push({ name: 'console-organization', params: { organizationId: currentOrganizationId.value } })
      return
    }
    projects.value = []
    applications.value = []
    selectedProjectId.value = ''
    selectedApplicationId.value = ''
  }, '组织已删除')
}

async function showProjectDeleteNotice() {
  if (!selectedProjectId.value) {
    return
  }
  await withFeedback(async () => {
    await apiPost('/api/manage/v1/project/delete', { projectId: selectedProjectId.value })
    selectedProjectId.value = ''
    selectedApplicationId.value = ''
    await Promise.all([loadProjects(), loadApplications(), loadOrganizations()])
    backToProjectList()
  }, '项目已删除')
}

async function showApplicationDisableNotice() {
  if (!selectedApplicationId.value) {
    return
  }
  await withFeedback(async () => {
    await apiPost('/api/manage/v1/application/disable', { applicationId: selectedApplicationId.value })
    await loadApplications()
    await loadOrganizations()
  }, '应用已停用')
}

async function showApplicationDeleteNotice() {
  if (!selectedApplicationId.value) {
    return
  }
  await withFeedback(async () => {
    await apiPost('/api/manage/v1/application/delete', { applicationId: selectedApplicationId.value })
    selectedApplicationId.value = ''
    await loadApplications()
    await loadOrganizations()
    await backToProjectDetail()
  }, '应用已删除')
}

function formatDateTime(value?: string) {
  if (!value) {
    return '-'
  }
  return new Date(value).toLocaleString()
}

function isTabActive(targetTab: 'dashboard' | 'organization' | 'project' | 'user' | 'role' | 'audit' | 'setting') {
  if (currentView.value === 'my' || currentView.value === 'organization-manage') {
    return false
  }
  return tab.value === targetTab
}

async function copyMetricValue(value: string) {
  if (!value || value === '-') {
    return
  }
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(value)
    } else {
      const textarea = document.createElement('textarea')
      textarea.value = value
      textarea.setAttribute('readonly', 'true')
      textarea.style.position = 'absolute'
      textarea.style.left = '-9999px'
      document.body.appendChild(textarea)
      textarea.select()
      document.execCommand('copy')
      document.body.removeChild(textarea)
    }
    message.value = '已复制到剪贴板'
    messageVariant.value = 'success'
  } catch (error) {
    message.value = String(error)
    messageVariant.value = 'danger'
  }
}

function formatApplicationType(value?: string) {
  if (value === 'native') return 'Native'
  if (value === 'api') return 'API'
  return 'Web'
}

function formatApplicationTokenType(value?: string | string[]) {
  const values = Array.isArray(value) ? value : value ? [value] : []
  if (!values.length) {
    return '-'
  }
  return values.join(' + ')
}

function formatApplicationGrantType(value?: string | string[]) {
  const values = Array.isArray(value) ? value : value ? [value] : []
  if (!values.length) {
    return '-'
  }
  return values.join(' + ')
}

function formatApplicationClientAuthenticationType(value?: string) {
  switch (value) {
    case 'client_secret_basic':
      return 'client_secret_basic'
    case 'client_secret_post':
      return 'client_secret_post'
    case 'client_secret_jwt':
      return 'client_secret_jwt'
    case 'private_key_jwt':
      return 'private_key_jwt'
    case 'tls_client_auth':
      return 'tls_client_auth'
    case 'self_signed_tls_client_auth':
      return 'self_signed_tls_client_auth'
    case 'none':
      return 'none'
    default:
      return '-'
  }
}

function formatPolicyRules(rules?: Array<{ method?: string; path?: string }>) {
  if (!rules?.length) {
    return '-'
  }
  return rules.map((item) => `${item.method || 'POST'} ${item.path || '-'}`).join(' | ')
}

function formatIPLine(ipAddress?: string, ipLocation?: string) {
  const ip = String(ipAddress || '').trim() || '-'
  const location = String(ipLocation || '').trim()
  return location ? `${ip} (${location})` : ip
}

function toggleRoleName(target: string[], value: string, checked: boolean) {
  const index = target.indexOf(value)
  if (checked && index < 0) {
    target.push(value)
    target.sort()
    return
  }
  if (!checked && index >= 0) {
    target.splice(index, 1)
  }
}

function formatAdminResult(value: unknown) {
  if (!value) {
    return '无'
  }
  if (typeof value === 'string') {
    return value
  }
  return JSON.stringify(value)
}

function inferDeviceName(userAgent?: string) {
  const source = String(userAgent || '').trim()
  if (!source) {
    return '未知设备'
  }

  const browser = source.includes('Edg/') ? 'Edge'
    : source.includes('Chrome/') && !source.includes('Edg/') ? 'Chrome'
    : source.includes('Firefox/') ? 'Firefox'
    : source.includes('Safari/') && !source.includes('Chrome/') ? 'Safari'
    : source.includes('MSIE') || source.includes('Trident/') ? 'Internet Explorer'
    : ''

  const os = source.includes('Windows NT') ? 'Windows'
    : source.includes('Mac OS X') || source.includes('Macintosh') ? 'macOS'
    : source.includes('Android') ? 'Android'
    : source.includes('iPhone') || source.includes('iPad') || source.includes('iOS') ? 'iOS'
    : source.includes('Linux') ? 'Linux'
    : ''

  if (browser || os) {
    if (browser && os) {
      return `${browser} (${os})`
    }
    return browser || os
  }
  return source
}
</script>
