# PPVT (Pass Pivot)

PPVT 是一个控制面导向的 IAM 平台实现。


当前代码已经包含：

- Go + GORM 后端
- Vue 3 + TypeScript + bootstrap-vue-next 前端
- `auth`、`portal` 与 `console` 前端拆分
- User Credential / Client Credential 基础认证
- 二次确认、MFA、Passkey、会话与 `ukid` 吊销
- RBAC + Casbin 决策接口
- OIDC / OAuth2 最小可用能力
- 审计日志
- 外部 OAuth/OIDC Provider 管理配置
- Captcha / GeoIP / 外部 IdP 扩展接入骨架

当前密钥模型：

- OIDC Provider 使用单一 issuer、组织级 RSA signing key
- `/auth/keys` 返回所有 active 组织公钥
- `private_key_jwt` 客户端统一使用应用级 Ed25519 持久化密钥
- 内置 application 与外部 application 走同一套 client key 模型

## 仓库结构

- `cmd/ppvt-auth`: 协议面入口，只承载 `/auth/*` 与 `/.well-known/*`
- `cmd/ppvt-core`: 控制面入口，承载 `/api/*`
- `cmd/ppvt-init`: 初始化数据库与系统内置数据
- `internal/`: 领域模型、服务、路由、中间件与数据库初始化
- `provider/`: Captcha、GeoIP、外部 IdP 等外部能力接入点
- `web/auth`: `/auth/authorize` 直出认证交互前端
- `web/portal`: 用户中心前端
- `web/console`: 控制台前端
- `web/shared`: 共享组件、样式与工具
- `docs/`: 设计文档

## 运行端口

- `ppvt-core`：`8090`
- `ppvt-auth`：`8091`
- `portal`：`8092`
- `console`：`8093`

## 环境变量

运行时后端环境变量读取根目录 [`.env`](.env) 或系统环境。

当前支持：

- `PPVT_HTTP_ADDR`
- `PPVT_AUTH_URL`
- `PPVT_CORE_URL`
- `PPVT_DATABASE_DRIVER`
- `PPVT_DATABASE_HOST`
- `PPVT_DATABASE_PORT`
- `PPVT_DATABASE_USERNAME`
- `PPVT_DATABASE_PASSWORD`
- `PPVT_DATABASE_SCHEMA`
- `PPVT_REDIS_ENABLED`
- `PPVT_REDIS_HOST`
- `PPVT_REDIS_PORT`
- `PPVT_REDIS_PASSWORD`
- `PPVT_REDIS_DB`
- `PPVT_LOG_LEVEL`
- `PPVT_SECRET`

示例见 [`.env.example`](.env.example)。

初始化专用环境变量读取根目录 [`.init`](.init)。

这部分只供 `ppvt-init` 使用，主要包含：

- 内置 organization / project / application / role / user 的固定 ID

`.init` 不再保存内置 application 的 client seed / public key。内置 application 的 `private_key_jwt` 密钥和组织级 OIDC signing key 都由 `ppvt-init` 直接生成并写入数据库。

示例见 [`.init.example`](.init.example)。

前端环境变量：

- portal: 复制 [`web/portal/.env.example`](web/portal/.env.example) 为本地 `web/portal/.env`
- console: 复制 [`web/console/.env.example`](web/console/.env.example) 为本地 `web/console/.env`

## 启动方式

数据库初始化：

```bash
GOCACHE=$(pwd)/.gocache GOMODCACHE=$(pwd)/.gomodcache go run ./cmd/ppvt-init
```

强制重建数据库：

```bash
GOCACHE=$(pwd)/.gocache GOMODCACHE=$(pwd)/.gomodcache go run ./cmd/ppvt-init --force
```

ppvt-core：

```bash
GOCACHE=$(pwd)/.gocache GOMODCACHE=$(pwd)/.gomodcache go run ./cmd/ppvt-core
```

ppvt-auth：

```bash
GOCACHE=$(pwd)/.gocache GOMODCACHE=$(pwd)/.gomodcache go run ./cmd/ppvt-auth
```

portal：

```bash
cd web
npm install
npm run dev:portal
```

console：

```bash
cd web
npm run dev:console
```

## 数据库初始化说明

- 当前支持 `mysql`、`postgres`
- 当前默认使用 MySQL
- SQLite 与 H2 已移除
- 后端启动时不执行建表、不执行种子初始化
- 请先运行 `ppvt-init`
- `ppvt-init` 会同时读取 `.env` 与 `.init`
- `ppvt-init` 默认会先检查目标数据库是否已存在表；如果存在则直接退出，避免覆盖现有数据
- 只有显式传入 `--force` 时，`ppvt-init` 才会删除并重建目标数据库
- 在允许初始化时，`ppvt-init` 会创建表并写入系统内置基础数据
- `ppvt-init` 会为内置 organization 生成 OIDC provider signing key
- `ppvt-init` 会为内置 `manage-api` / `user-api` / `authn-api` / `authz-api` 生成应用级 client key
- `ppvt-init` 不负责历史版本迁移兼容，当前开发流程以重建数据库为准

当前默认本地数据库配置：

- Host: `127.0.0.1`
- Port: `3306`
- Username: `root`
- Password: `root`
- Schema: `ppvt`

`ppvt-init` 会创建：

- organization: `internal`
- project: `ppvt`
- application: `manage-api`
- application: `user-api`
- application: `authn-api`
- application: `authz-api`
- application: `console-web`
- application: `portal-web`
- 默认内置管理员用户：`admin@example.com / ChangeMe123!`
- 默认管理员角色标签：`console:admin`

说明：

- OAuth/OIDC 协议里的 `clientId` 统一使用 `Application ID`
- `/auth/*` 与 `/.well-known/*` 由 `ppvt-auth` 暴露
- `/api/*` 由 `ppvt-core` 暴露
- `/api/system/v1/*` 与 `/auth/*` 默认无需鉴权
- `/api/manage/v1/*` 仅允许 `manage-api` 与 `console-web` 调用；其中 `console-web` 还要求用户具备 `console:admin`
- `/api/user/v1/*` 仅允许 `user-api` 与 `portal-web` 调用，且必须具备当前登录用户上下文
- `/api/authn/v1/*` 仅允许 `authn-api` 调用
- `/api/authz/v1/*` 仅允许 `authz-api` 调用
- `ppvt-auth` 调用 `ppvt-core` 时也走正式 `/api/authn/v1/*`、`/api/authz/v1/*`
- 控制台继续通过标准 `/auth/authorize` + `/auth/token` 流程登录，不使用内部特权绕过
- 本地 `web/console/.env` 中的 `PPVT_CONSOLE_APPLICATION_ID` 需要与 `console-web` 的 `Application ID` 对齐
- `portal` 是用户中心站点，不再作为默认登录页
- `/auth/authorize` 直接返回登录、二次确认和 MFA 交互页
- `/auth/authorize` 的页面资源由 `web/auth` 构建产物提供，后端直接以 `/auth/authorize/app.js` 与 `/auth/authorize/app.css` 加载
- discovery 导入链路当前已从运行代码中移除，仅保留设计草案
- 当前访问控制系统采用 `Role + Policy + Policy Check` 模型

## 当前实现范围

已实现：

- 多组织、多项目、多应用基本管理
- 用户、角色、策略、Policy Check
- portal 用户中心、自助安全设置
- 密码登录、Passkey 登录
- TOTP、邮箱验证码、恢复码、U2F/Passkey 型 MFA
- 可信设备、设备列表、会话吊销
- 外部 IdP 配置管理
- OIDC metadata、JWKS、authorize、token、userinfo
- 审计日志

当前 OIDC / OAuth 密钥行为：

- `id_token` 按所属 organization 的 RSA key 签名
- 所有 organization 的 active RSA 公钥统一通过 `/auth/keys` 暴露
- `private_key_jwt` 校验从数据库读取 application 的 active Ed25519 公钥
- 应用重置 key 会使旧的 application key 记录失活，并写入新的 active key

未完全实现：

- 短信供应商接入
- discovery 运行能力
- 组织级策略对所有登录/用户页的强制执行
- 更完整的外部 IdP 生命周期管理

## 当前开发进度评估

以下评估基于当前仓库实现，不是设计目标覆盖率。

| 模块 | 进度 | 判断 |
| --- | --- | --- |
| 基础工程与部署结构 | `90%` | `ppvt-core` / `ppvt-auth` / `ppvt-init` 已拆分，目录结构已稳定 |
| 控制台与用户中心前端 | `80%` | console 与 portal 已分离，核心页面可用，但仍有细节待打磨 |
| 认证主流程 | `85%` | 密码登录、二次确认、MFA、Passkey、授权页直出已具备 |
| 会话、设备与吊销 | `85%` | device / session / ukid 机制已落地，行为基本闭环 |
| OAuth2 / OIDC Provider | `75%` | 标准主链路已可用，但协议覆盖和兼容性还不是完整实现 |
| 外部 OAuth / OIDC 联邦 | `60%` | 配置面已具备，完整联邦登录编排与生命周期管理仍未完成 |
| 角色、策略、Casbin 裁决 | `80%` | `Role + Policy + Policy Check` 主干已可用，但治理能力仍偏基础 |
| 管理面治理能力 | `70%` | 组织、项目、应用、用户、角色等 CRUD 已成形 |
| 审计 | `70%` | 关键事件落库已完成，查询与治理能力仍可继续增强 |
| 自发现 discovery | `10%` | 当前只保留设计稿，运行代码已移除 |

整体判断：

- 当前系统已从“原型”进入“可持续开发的工程骨架”阶段
- 认证、控制台、自举初始化、协议拆分已经成形
- 真正还缺的是协议完备性、联邦深度、策略治理深度，以及 discovery 的重新设计与落地

## 补充结构说明

当前仓库中与设计稿对应的主要运行单元为：

- `cmd/ppvt-core`：核心控制面与 `/api/*`
- `cmd/ppvt-auth`：标准 `/auth/*` 协议服务
- `cmd/ppvt-init`：初始化数据库与系统内置数据
- `web/auth`：授权页前端
- `web/portal`：用户中心前端
- `web/console`：控制台前端

当前 `provider/` 目录中的扩展点为：

- `provider/captcha`：验证码 provider 工厂与各 provider 独立实现文件
- `provider/geoip`：IP 归属地 provider 工厂，当前接 MaxMind GeoLite
- `provider/idp`：外部 OAuth/OIDC IdP provider 工厂与各实现

## 文档

- `docs/` 目录只保留设计稿、架构草案和拆分方案，不再维护运行时快照类文档
- 运行态清单、一次性提示词、数据库快照、角色绑定导出等文档已移除
- 这类信息应以代码、数据库和根 README 为准，不再在 `docs/` 中长期维护副本
- 总设计稿： [docs/PPVT_DESIGN_DRAFT.md](docs/PPVT_DESIGN_DRAFT.md)
- PLNK 集成草案： [docs/PLNK_INTEGRATION_DRAFT.md](docs/PLNK_INTEGRATION_DRAFT.md)
- 策略系统： [docs/PPVT_POLICY_SYSTEM.md](docs/PPVT_POLICY_SYSTEM.md)
- `ppvt-auth` 拆分方案： [docs/PPVT_AUTH_SPLIT_PLAN.md](docs/PPVT_AUTH_SPLIT_PLAN.md)
