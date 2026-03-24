# Web Auth

## Overview

`web/auth` 当前是一个由 `ppvt-auth` 承载的单入口授权前端。

- 前端入口固定为 `GET /auth/authorize`
- 登录、选账号、确认授权、MFA 都是同一页面内的内部状态
- device code 网页验证入口也统一到 `GET /auth/authorize?type=device_code`
- 前端只调用 `ppvt-auth` 暴露的 `/auth/api/*`
- `ppvt-auth` 再按需调用 core `/api/authn/v1/*`
- OAuth / OIDC 协议端点继续由 `ppvt-auth` 负责
- 旧 `/auth/headless/*` 已移除

## Deployment Boundary

- `web/auth` 不是独立 server 容器
- `ppvt-auth` 就是 `web/auth` 的 server 容器
- `ppvt-auth` 同时提供：
  - `/auth/authorize`
  - `/auth/authorize?type=device_code`
  - `/auth/api/*`
  - `/auth/token`、`/auth/userinfo`、`/auth/end_session` 等协议接口
  - `web/auth/dist` 下的静态资源

相关代码：

- [main.go](/Users/Wenxin/VSCodeProjects/pass-pivot/cmd/ppvt-auth/main.go)
- [bootstrap.go](/Users/Wenxin/VSCodeProjects/pass-pivot/internal/server/auth/bootstrap/bootstrap.go)
- [router.go](/Users/Wenxin/VSCodeProjects/pass-pivot/internal/server/auth/router/router.go)
- [static_asset_handler.go](/Users/Wenxin/VSCodeProjects/pass-pivot/internal/server/auth/handler/static_asset_handler.go)

## Frontend Structure

```text
web/auth/src/
  App.vue
  main.ts
  router/
    index.ts
  stores/
    auth.ts
  api/
    auth.ts
  pages/
    MainPage.vue
  layout/
    AuthHeader.vue
    AuthFooter.vue
  components/
    LoginStep.vue
    AccountStep.vue
    ConfirmationStep.vue
    MfaStep.vue
    DoneStep.vue
    captcha/
      DefaultCaptcha.vue
      CloudflareCaptcha.vue
      GoogleCaptcha.vue
  i18n/
    locale.ts
 utils/
    auth-error.ts
```

## Frontend Responsibility

### Router

router 只保留单入口页面：

- `/` -> `/auth/authorize`
- `/auth/authorize`

不再暴露：

- `/auth/authorize/login`
- `/auth/authorize/account`
- `/auth/authorize/confirmation`
- `/auth/authorize/mfa`

当前认证阶段由 store 状态决定，不由 URL 决定。

相关代码：

- [index.ts](/Users/Wenxin/VSCodeProjects/pass-pivot/web/auth/src/router/index.ts)

### Store

`stores/auth.ts` 现在是单一 store，统一负责：

- auth context / flowType / stage
- locale、message、captcha、selectedMethod
- stage title / hint 等展示派生
- `context/query`
- `session/create`
- `session/account/switch`
- `session/confirm`
- `session/verify_mfa`
- `session/mfa_challenge/create`
- webauthn / u2f 交互
- captcha refresh

相关代码：

- [auth.ts](/Users/Wenxin/VSCodeProjects/pass-pivot/web/auth/src/stores/auth.ts)

### Pages And Layout

`MainPage.vue` 是唯一 page：

- 负责初始化 `auth.initialize()`
- 负责 header / footer / toast
- 根据 `auth.stage` 切换步骤组件

相关代码：

- [MainPage.vue](/Users/Wenxin/VSCodeProjects/pass-pivot/web/auth/src/pages/MainPage.vue)

步骤组件按授权流程拆分：

- [LoginStep.vue](/Users/Wenxin/VSCodeProjects/pass-pivot/web/auth/src/components/LoginStep.vue)
- [AccountStep.vue](/Users/Wenxin/VSCodeProjects/pass-pivot/web/auth/src/components/AccountStep.vue)
- [ConfirmationStep.vue](/Users/Wenxin/VSCodeProjects/pass-pivot/web/auth/src/components/ConfirmationStep.vue)
- [MfaStep.vue](/Users/Wenxin/VSCodeProjects/pass-pivot/web/auth/src/components/MfaStep.vue)
- [DoneStep.vue](/Users/Wenxin/VSCodeProjects/pass-pivot/web/auth/src/components/DoneStep.vue)

`App.vue` 现在是单一前端入口：

- 仅渲染授权流程 `RouterView`
- `?type=device_code` 通过同一套 `auth/authn` store 进入 device code 模式
- 不再存在独立的 `DeviceApp.vue`、`device.ts`、`device.html`、`device.js`

相关代码：

- [App.vue](/Users/Wenxin/VSCodeProjects/pass-pivot/web/auth/src/App.vue)
- [DoneStep.vue](/Users/Wenxin/VSCodeProjects/pass-pivot/web/auth/src/components/DoneStep.vue)

### API Modules

前端 API 层遵循 `web/console` 风格：

- 只描述路径
- 直接 `requestPost(...)`
- 不从外部传入 API base URL
- 不承担状态编排

相关代码：

- [auth.ts](/Users/Wenxin/VSCodeProjects/pass-pivot/web/auth/src/api/auth.ts)

## API Layers

### Core APIs

core 对认证域暴露稳定版本接口：

- `POST /api/authn/v1/login_target/query`
- `POST /api/authn/v1/external_idp/query`
- `POST /api/authn/v1/authorize/interaction/query`
- `POST /api/authn/v1/session/create`
- `POST /api/authn/v1/session/confirm`
- `POST /api/authn/v1/session/mfa_challenge/create`
- `POST /api/authn/v1/session/verify_mfa`
- `POST /api/authn/v1/webauthn/login/begin`
- `POST /api/authn/v1/webauthn/login/finish`
- `POST /api/authn/v1/session/u2f/begin`
- `POST /api/authn/v1/session/u2f/finish`
- `POST /api/authn/v1/recovery_code/query`

协议与令牌接口也继续保留在 core `/api/authn/v1/*` 下。

相关代码：

- [authn_router.go](/Users/Wenxin/VSCodeProjects/pass-pivot/internal/server/core/api/authn/authn_router.go)

### Auth APIs

`ppvt-auth` 对 `web/auth` 暴露无版本内部接口：

- `POST /auth/api/context/query`
- `POST /auth/api/device/complete`
- `POST /auth/api/session/create`
- `POST /auth/api/session/account/switch`
- `POST /auth/api/session/confirm`
- `POST /auth/api/session/verify_mfa`
- `POST /auth/api/session/mfa_challenge/create`
- `POST /auth/api/webauthn/login/begin`
- `POST /auth/api/webauthn/login/finish`
- `POST /auth/api/session/u2f/begin`
- `POST /auth/api/session/u2f/finish`
- `POST /auth/api/captcha/refresh`

同时保留 OAuth / OIDC 协议接口：

- `GET /.well-known/openid-configuration`
- `GET /auth/keys`
- `GET /auth/authorize`
- `POST /auth/device/code`
- `POST /auth/token`
- `GET /auth/userinfo`
- `POST /auth/revoke`
- `POST /auth/introspect`
- `GET /auth/end_session`

device code 网页验证相关接口：

- `GET /auth/authorize?type=device_code`
- `GET /auth/device`
  - 兼容跳转到 `/auth/authorize?type=device_code`

相关代码：

- [router.go](/Users/Wenxin/VSCodeProjects/pass-pivot/internal/server/auth/router/router.go)
- [oidc_api_handler.go](/Users/Wenxin/VSCodeProjects/pass-pivot/internal/server/auth/handler/oidc_api_handler.go)

### Frontend APIs

`web/auth` 当前实际只调用这些接口：

- `POST /auth/api/context/query`
- `POST /auth/api/device/complete`
- `POST /auth/api/session/create`
- `POST /auth/api/session/account/switch`
- `POST /auth/api/session/confirm`
- `POST /auth/api/session/verify_mfa`
- `POST /auth/api/session/mfa_challenge/create`
- `POST /auth/api/webauthn/login/begin`
- `POST /auth/api/webauthn/login/finish`
- `POST /auth/api/session/u2f/begin`
- `POST /auth/api/session/u2f/finish`
- `POST /auth/api/captcha/refresh`

## Auth To Core Mapping

- `/auth/api/context/query`
  -> `/api/authn/v1/authorize/interaction/query`
- `/auth/api/device/complete`
  -> auth 本地审批 device authorization，不经过 core
- `/auth/api/session/create`
  -> `/api/authn/v1/session/create`
- `/auth/api/session/confirm`
  -> `/api/authn/v1/session/confirm`
- `/auth/api/session/verify_mfa`
  -> `/api/authn/v1/session/verify_mfa`
- `/auth/api/session/mfa_challenge/create`
  -> `/api/authn/v1/session/mfa_challenge/create`
- `/auth/api/webauthn/login/begin`
  -> `/api/authn/v1/webauthn/login/begin`
- `/auth/api/webauthn/login/finish`
  -> `/api/authn/v1/webauthn/login/finish`
- `/auth/api/session/u2f/begin`
  -> `/api/authn/v1/session/u2f/begin`
- `/auth/api/session/u2f/finish`
  -> `/api/authn/v1/session/u2f/finish`
- `/auth/api/captcha/refresh`
  -> auth 本地生成 captcha challenge，不经过 core
- `/auth/api/session/account/switch`
  -> auth 本地清理 pending login / portal session cookie，不经过 core

## Current Decisions

- 使用 `pinia`
- 保留 `router`，但只作为单入口壳，不承载流程步骤
- `AuthHeader.vue`、`AuthFooter.vue` 放在 `layout/`
- captcha 组件放在 `components/captcha/`
- `constants/` 已收敛为 `i18n/`
- `utils/` 已收敛为 utils/`
- 不使用 `useCaptcha.ts`
- 不单独维护 `types/` 目录
- `bootstrap` 不再作为 auth SPA 的架构基础
- 旧 `headless` 接口已删除
- OIDC metadata 中的 `device_authorization_endpoint` 为 `/auth/device/code`
- device code 的 `verification_uri` 为 `/auth/authorize?type=device_code`
- device code 与普通网页登录共用同一套页面、store 和交互逻辑

## Validation

最近一次已验证：

- `cd web && npm run build:auth`
- `GOCACHE=$(pwd)/.gocache GOMODCACHE=$(pwd)/.gomodcache go test ./internal/server/auth/...`
