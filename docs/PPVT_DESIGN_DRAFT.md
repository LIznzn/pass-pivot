# Pass Pivot（PPVT）IAM 系统设计稿（初版）

## 1. 项目概述

- 项目名称：Pass Pivot 通枢
- 项目缩写：PPVT
- 项目定位：统一身份认证与授权管理平台（IAM）
- 目标：为一个或多个业务系统提供统一的认证与授权能力，降低多系统接入成本并提升安全与可维护性。
- 定位说明：PPVT 是完整 IAM 平台，OIDC Provider 仅是其中一项协议能力，不代表系统全部功能。

### 1.1 规范术语（唯一用词）

为避免歧义，PPVT 采用固定分级模型（参考 ZITADEL）：

层级关系：`Instance > Organization > Project > Application`

1. Instance（实例）
   - 表示 PPVT 的一个运行实例（Runtime）。
   - 可在一台机器部署一个实例，也可在一台机器部署多个实例。
   - Instance 之间默认不共享配置与数据。

2. Organization（组织）
   - 用于多租户隔离（Tenant Boundary）。
   - 每个 Organization 的身份、权限、应用与数据相互独立。
   - 在 SaaS 场景中，一个 Instance 可承载多个 Organization。
   - 常规部署可仅使用一个业务 Organization。
   - 系统内置一个 Internal Organization，用于 PPVT 自身后台和内部组件鉴权。

3. Project（项目）
   - Organization 下的业务系统边界。
   - 用于区分不同业务系统，例如“支付系统”是一个独立 Project。
   - 权限域、资源域、审计域可按 Project 管理。

4. Application（应用）
   - Project 下的访问主体定义单元（客户端/调用方）。
   - 可表示同一 Project 的不同端或不同调用形态，例如 Web、iOS、API。
   - Application 是令牌签发、授权同意、凭据管理的直接对象。

术语约束：

- “用户凭据”统一称为 User Credential。
- “应用凭据（系统调用凭据）”统一称为 Client Credential。
- 文档中不再使用其他主体称呼。

主体授权边界：

- User Credential：仅用于 Project 内部授权（用户访问业务系统内资源）
- Client Credential：仅用于外部 API 访问授权（系统到系统调用）
- 两类凭据的权限集合与令牌用途必须隔离，禁止混用

### 1.2 跨组织访问规则（组织即安全边界）

Organization 是硬隔离边界，默认不支持跨组织直接授权。

规则定义：

1. A 组织用户访问 B 组织项目时，必须走“外部身份提供商联邦”方式
2. A 组织需要先开启对外 OAuth/OIDC 提供能力
3. B 组织将 A 组织作为外部 IdP 接入，再完成身份映射与授权
4. 整体行为等价于“B 组织接入 Auth0/第三方 IdP”，不做组织内直通授权
5. 外部 IdP 登录必须绑定到 B 组织内“已存在用户”，不支持通过 IdP 首次登录自动创建用户

设计结论：

- 跨组织关系在模型上视同“两个独立实例之间的联邦连接”
- A 组织在 B 组织看来就是一个标准外部身份源（与 Auth0 类产品等价）

### 1.3 典型业务场景（商城 + 支付）

在同一 Organization 下，存在两个 Project：

- Project A：商城系统
- Project B：支付系统

场景一：系统间 API 调用（机器主体）

1. 商城系统的 API Application 向 PPVT 申请机器凭据（API Key / client_secret）
2. PPVT 统一托管 API 密钥生命周期（签发、轮换、吊销、审计）
3. 商城系统调用支付系统接口时，先向 PPVT 获取访问令牌，再携带令牌访问支付 API
4. 支付系统通过 PPVT 的令牌校验与权限校验结果执行付款逻辑

场景二：用户跨系统访问（人类主体）

1. 用户在商城系统完成登录后建立 PPVT 会话
2. 用户进入支付流程时，在同一 Organization 内可无感访问支付系统（SSO）
3. 支付系统按 PPVT 下发的用户身份与权限完成授权判断

## 2. 核心能力

PPVT 由两大核心模块组成：

1. 认证（Authentication）
2. 授权（Authorization）

### 2.1 认证（Authentication）

认证用于验证用户或系统的访问凭据，确保“访问者是谁”。

当前规划支持的认证方式包括：

- 用户名/密码登录
- WebAuthn（前端可显示为通行密钥 / Passkey，用于无密码登录）
- OAuth2 / OIDC 授权码登录
- 多因素认证（MFA）

当前已实现或已落地到系统配置层的认证相关能力：

- Organization 级登录策略：可分别控制 `username`、`email`、`phone` 的启用与必填/选填/隐藏
- Organization 级密码策略：最小长度、大小写、数字、特殊字符、过期时间
- Organization 级 MFA 策略：是否强制全员启用，以及允许的 MFA 方式
- Organization 级验证码策略：`disabled`、`default`、`google`、`cloudflare`

说明：

- “二次确认流程（敏感操作再验证）”当前仍属于设计方向，尚未形成稳定实现与后台策略配置，不应视为现有能力。

MFA 支持方式：

- U2F（安全密钥）
- 邮箱验证码
- 短信验证码
- TOTP（基于时间的一次性口令）
- 救援密钥（Recovery Codes）

建议安全优先级：

1. U2F / WebAuthn
2. TOTP
3. 邮箱验证码
4. 短信验证码（可作为兜底，不建议作为高安全默认项）

恢复与降级策略：

- 用户至少绑定 2 种 MFA 方式（如 U2F + TOTP）
- 首次启用 MFA 时生成一次性救援密钥，用户可离线保存
- 当主 MFA 不可用时，允许通过救援密钥完成一次恢复登录并强制重绑

可信设备策略：

- 已验证且被标记为可信的设备，可在有效期内跳过 MFA
- 可信状态需绑定设备指纹 + 浏览器/系统特征，且仅对当前账号生效
- 建议设置可信有效期（如 30 天）并支持用户主动撤销全部可信设备
- 以下场景必须强制 MFA：异地登录、设备指纹显著变化、敏感操作、风险评分过高

### 2.2 凭据与令牌策略（Credential & Token）

PPVT 当前按 Application 粒度控制令牌与授权方式，而不是独立的“令牌模板系统”。

统一要求：

- 验证方式：Access Token / Refresh Token
- 鉴权方式：Authorization Code / Authorization Code + PKCE
- 访问方式：JWT / Basic / None
- 按场景选择不同 Token 生命周期与组合策略
- 三类方式支持自由组合，组合粒度为 Application 级配置

术语映射说明：

- 上述为 PPVT 内部产品命名；对接 OAuth2/OIDC 时按标准参数映射实现

当前实现状态：

- Application 直接配置：
  - `grantType`
  - `tokenType`
  - `enableRefreshToken`
  - `clientAuthenticationType`
  - `accessTokenTTLMinutes`
  - `refreshTokenTTLHours`
- 系统默认值目前接近：
  - Access Token：10 分钟
  - Refresh Token：7 天
- “Long-lived Access Token：永久”目前不是已实现能力，文档不应按现状能力描述。

### 2.2.1 用户密钥对与全量令牌吊销

PPVT 为每个用户维护一组“用户密钥标识”（`ukid`）和对应公钥，用于令牌绑定与全量吊销控制。

设计原则：

- 系统侧仅保存公钥与 `ukid`
- 私钥仅在用户侧持有（securekey / WebAuthn / U2F 设备），PPVT 不保存私钥
- 未启用 WebAuthn 的用户，不保存私钥，仅保留公钥记录与密钥版本

令牌绑定规则：

- Access Token / Refresh Token / Long-lived Access Token 均携带 `ukid`（或等价的 `key_version`）
- 令牌校验时，必须与用户当前有效 `ukid` 匹配

全量吊销机制：

1. 用户触发“重置密钥对”
2. PPVT 生成新 `ukid` 并更新当前生效公钥
3. 旧 `ukid` 对应的全部 Token 立即判定失效
4. 强制旧会话重新认证并重新签发新令牌

用途边界：

- 公钥主要用于身份绑定、挑战验证和令牌版本锚定
- 不直接把“公钥本体”作为业务数据加密主键，建议使用 `ukid` 或公钥指纹作为索引键

### 2.3 授权（Authorization）

授权用于定义“认证通过后可以做什么”，并在系统中统一管理权限边界。

- 授权模型基于 RBAC（Role-Based Access Control）
- 支持对一个或多个接入系统进行统一权限控制
- 通过中心化权限模型管理可访问范围与操作级权限

## 3. 权限模型设计方向（RBAC）

建议采用标准 RBAC 结构：

- User（用户）
- Role（角色）
- Permission（权限）
- Resource（资源）
- Policy/Scope（策略/范围，可按系统逐步扩展）

基础关系建议：

- 用户与角色：多对多
- 角色与权限：多对多
- 权限与资源：可按资源类型和操作维度建模（如 `user:read`、`user:write`）

### 3.1 权限定义归属

设计方向上，PPVT 期望采用“接入系统定义权限、IAM 统一治理”的模式：

- 具体权限项由第三方业务系统声明
- PPVT 负责权限校验、版本管理、审批发布、分配与审计
- PPVT 不强行发明业务权限，只做标准化管理与策略控制

当前状态说明：

- 这部分仍以方向性设计为主。

建议将权限分为两类：

- Client Credential 权限（Client/API Permissions）：用于系统到系统调用
- User Credential 权限（User Permissions）：用于用户访问 Project 内部资源

## 4. UI/UX 设计参考

### 4.1 用户端（认证相关）

以下页面与交互应参考 GitHub 当前登录体系风格：

- 登录
- 注册
- 授权确认
- 二次确认
- MFA 验证流程

重点强调：

- 清晰、简洁、高信任感的认证流程
- 对异常状态（失败、风控、锁定）有明确反馈
- 用户在关键授权动作前看到清晰权限说明

### 4.2 管理后台（运营与管理）

后台管理界面应参考 Stripe 后台管理风格：

- 信息密度高但结构清晰
- 模块化导航与统一操作体验
- 支持配置、审计、策略管理等高频后台场景

## 5. 协议与生态扩展能力

PPVT 需要预留可拓展架构，覆盖内部统一认证和外部身份接入。
其中 OIDC Provider 角色仅用于协议对接，不等同于 PPVT 的整体能力边界。

### 5.1 对内授权/认证协议（作为 IAM 核心能力）

- OIDC
- OAuth2
- JWT

### 5.2 对外身份接入（作为第三方身份桥接能力）

当前已收敛支持的外部 IdP Provider：

- Google
- GitHub
- Apple

说明：

- QQ、新浪微博、自定义 OAuth、自定义 OIDC 已不在当前产品范围内。
- 文档中凡是“外部身份桥接能力”的描述，当前都应以上述三种预置 Provider 为准。

### 5.3 OIDC 用户资料与 Claims 规则

PPVT 对 OIDC 用户资料采用“最小必需、其余可选”策略。

必需标识：

- `iss + sub`：外部身份唯一键（用于绑定已有用户）

资料字段策略：

- `name`：推荐字段，采用单字段设计，不强制拆分名/姓
- `email`：可选
- `phone_number`：可选
- `username`：可选（映射 OIDC 标准 claim：`preferred_username`）
- `picture`：可选

联系方式要求：

- 当前实现不是 `email_or_phone / email_only / phone_only` 这种单枚举模型。
- 当前 Organization 级登录策略采用三个独立维度配置：
  - `username`: `hidden / optional / required`
  - `email`: `hidden / optional / required`
  - `phone`: `hidden / optional / required`
- 系统同时支持按策略开启或关闭 `username`、`email`、`phone` 登录入口。
- 因此本节若继续保留联系方式策略，应按上述真实配置模型重写。

scope 建议：

- 必选：`openid`
- 按需可选：`profile`、`email`、`phone`

目标链路示例：

1. 用户通过 GitHub（或 Google）完成外部身份认证
2. PPVT 完成外部身份映射与本地身份统一
3. PPVT 将用户会话/令牌下发给目标业务系统
4. 目标业务系统基于 PPVT 结果完成登录与权限校验

即：`第三方身份提供商 -> PPVT IAM -> 业务系统`

## 6. 域名接入与验证

当前已实现且与域名接入直接相关的能力：

- Organization 级域名设置
- 域名所有权验证
  - `http_file`
  - `dns_txt`
- 域名验证通过后，可作为控制台与鉴权相关域名白名单依据

## 7. 审计能力

系统需提供完整审计能力，用于安全追踪、合规检查与故障回溯。

建议覆盖：

- 登录/登出记录
- 认证失败记录（含失败原因与来源）
- 权限变更记录（用户、角色、权限关系）
- 管理员关键操作日志
- 授权同意与撤销记录
- API Token 生成、使用、吊销记录
- 用户密钥对重置与全量 Token 吊销记录

建议新增审计项：

- 域名验证记录（成功/失败、失败原因）
- 配置变更记录（谁在何时修改组织级登录/密码/MFA/验证码配置）

当前状态说明：

- 审计能力已存在，但本节列出的覆盖范围应理解为目标范围，而非逐项均已完成。

## 8. 技术选型与实现方向

- 开发语言：Go
- ORM：GORM
- 数据库策略：通过 GORM 兼容多种数据库与部署环境
- 前端框架：Vue 3
- 前端语言：TypeScript
- 前端样式库：bootstrap-vue-next

选型理由：

- Go 具备较高执行效率，适合认证授权等高并发基础能力服务
- GORM 提供较好的数据库抽象与工程效率，便于适配多架构/多环境/多数据库
- Vue 3 + TypeScript 兼顾工程化与可维护性，适合 IAM 控制台与认证页面长期迭代
- bootstrap-vue-next 可快速落地一致化后台与表单交互组件

## 9. 系统边界与定位总结

PPVT 的核心定位是“认证与授权中台”：

- 向上对接第三方身份提供商（GitHub、Google 等）
- 向下服务多个业务系统（统一登录与统一权限）
- 对内沉淀统一的认证协议能力、授权模型与审计能力

通过标准化协议与可扩展架构，PPVT 可逐步演进为企业级 IAM 平台。

### 9.1 网关接入与中间件边界

PPVT 不包含网关实现，也不包含业务系统鉴权中间件实现；两者均属于接入方系统工程范围。

接入模式 A：业务系统接入网关（推荐）

- 网关作为第一执行层，向 PPVT 获取会话/令牌校验与权限判定结果
- 业务系统内中间件仅需校验“请求是否来自可信网关”
- 此模式下，业务系统内中间件不负责认证、鉴权、权限控制逻辑

接入模式 B：业务系统不接入网关

- 业务系统内中间件需承担完整执行职责：
  - 认证（Authentication）
  - 鉴权（Authorization Flow Validation）
  - 权限控制（Permission Enforcement）

统一要求：

- 两种模式都必须对接 PPVT 的身份与权限能力
- 接入方可按系统能力选择模式 A 或模式 B

## 10. 下一步设计建议（用于进入开发阶段）

1. 输出领域模型与 ER 图（用户、角色、权限、租户/系统、审计）
2. 统一整理“已实现能力”和“规划能力”的边界，避免设计稿继续混写
3. 定义统一权限命名规范（资源+动作）
4. 补齐权限治理、审批与审计的最小闭环
5. 继续扩展 MFA、WebAuthn、OIDC/OAuth2、第三方登录桥接

## 10.1 当前实现对齐说明

截至当前代码状态，以下能力已明确存在：

- 分层模型：`Organization > Project > Application`
- OIDC / OAuth2 基础协议能力
- Application 级授权类型、Token 类型、TTL 配置
- Organization 级登录策略、密码策略、MFA 策略
- Organization 级验证码配置：`default/google/cloudflare`
- Organization 级域名设置与域名所有权验证
- 外部 IdP 接入：Google / GitHub / Apple

以下能力仍应视为规划项，而非当前能力：

- 通用外部 IdP 模板体系
- 永久 Long-lived Access Token 模板化能力
- 更完整的权限治理与审批发布闭环
- 完整的“二次确认流程”策略体系

## 11. 未来里程碑（当前不实现）

为支持企业级基础设施统一身份治理，PPVT 未来规划接入：

- Active Directory（AD）
- LDAP
- RADIUS

目标能力：

- 将 PPVT 身份体系延伸至企业终端与网络接入场景
- 实现 PC、Linux、WiFi 的集中认证与统一授权策略
- 打通业务系统身份与企业基础设施身份，形成统一身份中台

范围说明：

- 本里程碑仅作为长期路线图，当前版本不进入开发范围
