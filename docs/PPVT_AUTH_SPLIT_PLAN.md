# PPVT Auth Split Plan

本文档定义当前仓库中 `ppvt-auth` 与 `ppvt-core` 的拆分方案。

目标不是引入内部特权链路，而是把协议层从能力层拆出来，同时保持系统自用链路与第三方接入链路完全自洽。

## 当前状态

本方案在当前代码中已大部分落地：

- `cmd/ppvt-auth` 已存在，并承载 `/auth/*` 与 `/.well-known/*`
- `cmd/ppvt-core` 已存在，并承载 `/api/*`
- `ppvt-auth -> ppvt-core` 已通过正式 `/api/authn/v1/*`、`/api/authz/v1/*` 调用
- 内置 application 仍按正式客户端模型工作，没有隐藏 service 后门

仍待继续完善的部分：

- `/auth/*` 的协议完整度和标准兼容性
- introspection / revoke / end_session 的更多兼容场景
- 外部 IdP 与更复杂认证编排的协议细节

## 1. 目标

拆分为两个后端二进制：

- `ppvt-auth`
- `ppvt-core`

## 2. 职责边界

### 2.1 ppvt-auth

只负责协议层：

- `GET /.well-known/openid-configuration`
- `GET /auth/keys`
- `GET /auth/authorize`
- `POST /auth/headless/*`
- `POST /auth/token`
- `GET /auth/userinfo`
- `POST /auth/revoke`
- `POST /auth/introspect`
- `GET /auth/end_session`

特点：

- 对外表现为 OAuth2 / OIDC Provider
- 直接承载授权页登录交互
- 不持有 `/api/*` 控制面或能力面路由
- 通过正式 `/api/authn/v1/*`、`/api/authz/v1/*` 调用 `ppvt-core`

### 2.2 ppvt-core

负责所有能力面与控制面：

- `/api/system/v1/*`
- `/api/authn/v1/*`
- `/api/authz/v1/*`
- `/api/user/v1/*`
- `/api/manage/v1/*`

特点：

- 是真实能力中心
- 对第三方开放的就是正式 `/api/*`
- `ppvt-auth` 也只是 `ppvt-core` 的一个合法客户端

## 3. 自洽原则

必须满足：

- `ppvt-auth` 不依赖隐藏内部 service 特权
- `ppvt-auth` 调 `ppvt-core` 走正式 `/api/authn/v1/*`、`/api/authz/v1/*`
- 使用内置 application：
  - `authn-api`
  - `authz-api`
- 客户端认证方式仍为正式 `private_key_jwt`

也就是说：

- 内置客户端不是后门
- 只是系统官方客户端
- 与第三方模型一致

## 4. 默认端口

当前开发默认端口：

- `ppvt-core`: `:8090`
- `ppvt-auth`: `:8091`
- `portal`: `:8092`
- `console`: `:8093`

说明：

- 当前已按该端口模型运行
- 运行时默认监听地址已显式改为 IPv4 `0.0.0.0:*`

## 5. 配置约定

新增：

- `PPVT_CORE_URL`

语义：

- `PPVT_AUTH_URL`：协议面对外地址，给 `ppvt-auth` 使用
- `PPVT_CORE_URL`：能力面地址，给 `ppvt-auth` 调 `ppvt-core` 使用

## 6. Web 侧约定

前端分别使用各自命名空间环境变量：

- console: `PPVT_CONSOLE_*`
- portal: `PPVT_PORTAL_*`

其中：

- `PPVT_CONSOLE_AUTH_BASE_URL` / `PPVT_PORTAL_AUTH_BASE_URL` 指向 `ppvt-auth`
- `PPVT_CONSOLE_API_BASE_URL` / `PPVT_PORTAL_API_BASE_URL` 指向 `ppvt-core`

## 7. 迁移原则

### 已允许保留在 ppvt-auth 的本地能力

- 协议页渲染
- OIDC metadata 生成
- JWKS 输出
- 内置客户端 assertion 构造

### 必须留在 ppvt-core 的能力

- 登录、确认、MFA、Passkey、联邦
- Policy Check
- 用户/应用/组织/角色/策略/审计 CRUD
- token introspection 的真实数据查询

## 8. 已完成项与后续关注点

已完成：

- 已拆分 `auth router` 和 `core router`
- 已提供 `cmd/ppvt-auth`
- 已提供 `cmd/ppvt-core`
- `ppvt-auth -> ppvt-core` 已通过 `PPVT_CORE_URL` 调用正式 `/api/*`
- web 侧 `/auth` 与 `/api` 地址已拆开
- console 登录、portal 用户中心、`/auth/introspect`、`/auth/authorize` 已具备基础可用链路

后续关注：

- 继续补齐 OAuth2 / OIDC 标准兼容细节
- 继续完善 `revoke`、`introspect`、`end_session` 的边界行为
- 继续补齐外部 IdP 联邦与更复杂认证编排

## 9. 非目标

本次不做：

- 数据库拆分
- auth/core 各自独立数据模型
- gRPC
- 服务注册发现
- 内部专用隐藏接口
