# Pass Pivot 安全审计整理版

**整理日期**: 2026-03-25  
**说明**: 本版基于当前代码重新核对，删除了已失效、暂不处理、或属于明确产品取舍的条目，只保留建议继续修改的问题。

---

## 1. 结论

当前仍建议处理的问题主要有 4 类：

1. 登录侧缺少独立的限流/锁定机制，当前主要依赖验证码抬高自动化攻击成本。
2. Recovery code 采用明文可回显设计，这是明确的产品取舍，但也带来数据库泄露后的直接可用风险。
3. `web/console` 与 `web/portal` 将 token 持久化到 `localStorage`，一旦出现 XSS，token 可被直接窃取。
4. 若使用默认 `PPVT_SECRET` 启动，当前没有显式 warning，容易让弱配置进入运行环境。

另有 1 项低到中风险的稳健性问题：

5. 多处 `io.ReadAll` 对响应体没有大小限制，异常上游可能放大内存占用。

---

## 2. 保留问题

### 2.1 登录缺少独立限流机制

- 📍 Location: `internal/server/core/api/authn/authn_service.go`
- 🔍 现状:
  - 登录流程有验证码校验。
  - 生产环境若启用验证码，确实可以提高无头浏览器和批量撞库成本。
  - 但当前仍没有账号级锁定、IP级速率限制、失败退避等独立机制。
- 评估:
  - 该问题不应再按“完全无防护”描述。
  - 如果验证码在生产默认开启，严重度应低于旧版报告。
  - 但验证码不能完全替代限流，尤其是在验证码被复用、绕过、或人工打码的情况下。
- 建议:
  - 至少补一层轻量的 IP + 标识符限流。
  - 如果暂不实现，可在文档中明确“当前主要依赖验证码作为第一道防线”。

### 2.2 Recovery Code 明文可回显

- 📍 Location:
  - `internal/server/auth/service/mfa_service.go`
  - `internal/server/core/api/manage/manage_service.go`
- 🔍 现状:
  - Recovery code 以明文形式生成、存储、查询和返回。
  - 当前产品允许用户或管理端再次查看 recovery code。
- 评估:
  - 这不是单纯实现错误，而是产品能力与安全性的明确 tradeoff。
  - 只要保留“可再次查看”能力，服务端就必须持有可恢复明文，或等价的可逆数据。
  - 风险在于数据库泄露、备份泄露、越权读取时，攻击者可直接拿到有效 recovery code。
- 风险边界:
  - 该设计默认信任数据库主存、数据库备份、具备读权限的运维与管理员侧链路不会泄露 recovery code 明文。
  - 一旦上述任一边界失守，recovery code 将被视为可直接使用的认证绕过材料，而不是仅供审计的低敏感数据。
  - 因此该能力应被归类为“高敏感恢复凭据可回显”，其保护级别应接近密码重置材料或一次性恢复口令。
  - 该风险不是通过“前端不展示”规避的，因为核心暴露面在服务端存储与查询能力本身。
- 建议:
  - 如果业务必须支持“随时查看”，则应把该风险写入安全假设，而不是继续按普通 bug 表述。
  - 如果未来允许调整产品设计，更稳妥的方案仍是“仅首次展示，之后只能重新生成”。

### 2.3 Console / Portal token 持久化到 localStorage

- 📍 Location:
  - `web/console/src/api/auth.ts`
  - `web/portal/src/stores/auth.ts`
- 🔍 现状:
  - `web/auth` 当前不持久化 access token，这部分实现没有问题。
  - 但 `web/console` 和 `web/portal` 会把 `access_token`、`refresh_token`、`id_token` 存入 `localStorage`。
- 评估:
  - 对 `ppvt-auth` 这种认证前端，不持有 token 是更合理的。
  - 对真正的 SPA 业务前端，把 token 放入 `localStorage` 是常见做法，但代价是只要发生 XSS，token 就可被直接读取并外传。
- 建议:
  - 如果继续保留当前模式，应把该项视为前端 XSS 后果放大项。
  - 后续可评估是否改为更短寿命 token、后端会话、BFF，或减少 `localStorage` 中的高价值令牌。

### 2.4 默认密钥启动缺少显式 warning

- 📍 Location: `internal/config/config.go`
- 🔍 现状:
  - `PPVT_SECRET` 默认值仍为 `ppvt-dev-secret`。
  - 若部署时未显式覆盖，系统会在弱默认密钥下运行。
- 建议:
  - 使用默认值启动时打印高可见 warning。
  - 更严格的做法是在非开发环境直接拒绝启动。

### 2.5 `io.ReadAll` 无大小限制

- 📍 Location:
  - `internal/server/auth/handler/oidc_authorize_interaction_handler.go`
  - `internal/server/auth/handler/oauth_protocol_handler.go`
- 🔍 现状:
  - 多处直接对响应体执行 `io.ReadAll`，没有上限。
- 风险:
  - 若上游返回异常大的 body，会放大单请求内存占用。
  - 在并发场景下可能造成 GC 压力上升、延迟抖动，严重时触发 OOM。
- 评估:
  - 这更偏可用性和稳健性问题，不是高危数据安全问题。
  - 对内部 API 风险较低，对外部 OAuth 上游风险更真实一些。
- 建议:
  - 用 `io.LimitReader` 增加一个保守上限即可。

---

## 3. 已删除条目

以下条目已从本版移除，不再作为当前待修改问题：

- MFA 验证码写入审计日志：当前代码已不再写入 `demoCode`。
- Auth -> Core 使用 `http.DefaultClient`：当前已改为带超时和连接池的专用 HTTP client。
- MFA challenge 无尝试次数限制：当前已存在尝试次数限制。
- `buildAuthorizeAppShell` title 未转义：当前实现已无该 title 拼接逻辑。
- Device user code 存在 modulo bias：旧报告判断错误，当前 alphabet 长度为 32，不存在该偏差。
- TOTP 使用 SHA-1：属于兼容性优先的设计选择，暂不作为待修改问题。
- `web/auth` token 存 `localStorage`：当前 `web/auth` 并未这样实现。
- CORS 每次查库无缓存：已知架构问题，后续若接 Redis/缓存层再统一处理。
- TransientStore 进程内存存储：已知架构问题，后续若接 Redis 再统一处理。

---

## 4. 建议优先级

### P1

- 为默认 `PPVT_SECRET` 增加启动 warning。
- 为登录流程补充轻量限流，降低对验证码的单点依赖。

### P2

- 评估 `web/console` / `web/portal` 的 token 存储策略是否需要收紧。
- 为外部与内部上游响应增加 body 大小限制。
