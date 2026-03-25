# Pass Pivot 第5轮安全审计报告

**审计日期**: 2025-03-25  
**审计范围**: 30个后端核心文件 (9762行) + 12个前端核心文件 (1906行) + go.mod  
**审计版本**: v5 (Auth前端重构 + Device Code Handler重构)  

---

## 1. 执行摘要

**总体评分: 5.5 / 10** (上一轮: 5.0 / 10，小幅改善)

本轮重大变更（Auth前端组件化、Device Code重构）在架构层面有明显改善，消除了服务端HTML渲染的XSS风险面，但**前4轮遗留的2个CRITICAL问题仍未修复**，且引入了**3个新的中高严重度问题**。

| 严重度 | 数量 | 说明 |
|--------|------|------|
| 🔴 CRITICAL | 2 | 暴力破解无防护（第5轮未修）、MFA验证码明文写入审计日志（新发现）|
| 🟠 HIGH | 3 | `http.DefaultClient` 无超时/SSRF、Recovery Code 明文存储、MFA挑战无尝试次数限制 |
| 🟡 MEDIUM | 4 | TransientStore内存存储、CORS无缓存、TOTP SHA-1、`buildAuthorizeAppShell` title未转义 |
| 🟢 LOW | 5 | 前端token存localStorage、日志无脱敏、无Dockerfile、零测试、默认Secret弱 |

---

## 2. 前次遗留问题追踪

### 2.1 🔴 ❌ 登录暴力破解防护 — 第5轮仍未修复

- 📍 Location: `internal/server/core/api/authn/authn_service.go:3363` (`LoginWithUserCredential`)
- 🔍 Issue: `LoginWithUserCredential` 方法在密码验证失败时仅写审计日志，**没有任何速率限制、账号锁定、或IP级别的限流**。攻击者可以无限次尝试密码。
- 💥 Impact: 在线暴力破解攻击可以无限制地枚举用户密码。对于弱密码账户，攻击者可以在短时间内破解。
- 📎 Ref: CWE-307 (Improper Restriction of Excessive Authentication Attempts)
- ✅ Fix: 实现以下至少一项：
  1. **账号级锁定**: 连续N次失败后锁定账号15-30分钟
  2. **IP级Rate Limit**: 使用令牌桶/滑动窗口限制单IP登录频率
  3. **渐进式延迟**: 每次失败后指数增长的响应延迟

```go
// 建议添加在 LoginWithUserCredential 开头
if blocked, retryAfter := s.rateLimiter.Check(in.IPAddress, in.Identifier); blocked {
    return nil, fmt.Errorf("too many login attempts, retry after %v", retryAfter)
}
// 密码失败后
s.rateLimiter.RecordFailure(in.IPAddress, in.Identifier)
```

**严重度: 🔴 CRITICAL — 这是第5轮仍未修复的问题，必须立即处理。**

---

### 2.2 🟡 ⚠️ TransientStore 内存存储 — 第5轮仍未修复

- 📍 Location: `internal/server/auth/service/transient_store.go:1179-1328`
- 🔍 Issue: Authorization Code、MFA Challenge、External Auth State 全部存储在进程内存中的全局 `transientStore` 变量中。
- 💥 Impact:
  - **服务重启丢失所有活跃的授权码和MFA挑战**
  - **多实例部署不可行**（每个实例有独立的内存存储）
  - **内存泄漏风险**：虽然有 `cleanupExpiredTransientState()` 在每次操作时调用，但在高并发下这个同步清理会成为瓶颈
- ✅ Fix: 将这些状态迁移到 Redis（已在 go.mod 中引入 `go-redis/v9`，但 `config.go` 中 `RedisEnabled` 默认为 `false`）

**状态: ⚠️ 有Redis依赖但未启用，本轮无变化。**

---

### 2.3 🟡 ⚠️ CORS 每次查DB无缓存 — 第5轮仍未修复

- 📍 Location: 未在本次源文件范围中看到CORS中间件的完整实现，但根据前次报告和 `router.go:841` 中 `cors(mux)` 的调用，每次请求都会查询DB验证域名。
- 💥 Impact: 高频请求下对数据库造成不必要的压力。
- ✅ Fix: 使用内存缓存（如 sync.Map + TTL）缓存已验证的域名，TTL 5分钟。

**状态: ⚠️ 无变化。**

---

### 2.4 🟢 ⚠️ 前端 token 存 localStorage — 无变化

- 📍 Location: `web/auth/src/stores/auth.ts` — 当前前端代码未显式存储token到localStorage，token通过API响应返回后立即触发 `window.location.assign(redirectTarget)` 重定向。
- 🔍 分析: 前端重构后token实际上是通过HTTP redirect传递的（authorization code flow），**前端不直接持有access token**。但 `sessionStorage` 被用于存储 device review 确认状态（`ppvt_device_review_confirmed`），这是安全的。
- ✅ 评估: **风险较低**。如果后续有SPA直接使用implicit flow，则需要注意。

**状态: ⚠️ 风险已降低，但implicit flow支持仍返回token到fragment。**

---

### 2.5 🟢 ⚠️ 日志无脱敏 — 无变化

- 📍 Location: 全局
- 🔍 Issue: 审计日志中记录了 `identifier`（用户名/邮箱）但没有脱敏处理。更严重的是审计日志中记录了 MFA demo code（见新发现#1）。

**状态: ⚠️ 无变化，且发现新的更严重的日志泄露问题。**

---

### 2.6 🟢 ⚠️ 无 Dockerfile — 无变化

**状态: ⚠️ 无变化。**

---

### 2.7 🟢 ⚠️ 零测试 — 无变化

**状态: ⚠️ 无变化，上一轮删除了所有测试文件。**

---

### 2.8 🔴 ✅ Bootstrap JSON 注入 XSS — 已通过架构变更消除

- 📍 Location: `oidc_authorize_interaction_handler.go:472-487` (`buildAuthorizeAppShell`)
- 🔍 分析: 本轮重构中，服务端不再将 JSON 数据注入到 `<script>` 标签中。`buildAuthorizeAppShell` 现在只生成一个静态的HTML shell，加载 Vue SPA 的 JS/CSS 资源。所有数据通过 API 调用获取。
- ✅ **这是一个优秀的架构改进**，从根本上消除了服务端HTML模板注入的风险面。

**状态: ✅ 已通过架构重构修复。**

---

## 3. 新发现的安全问题

### 3.1 🔴 CRITICAL — MFA 验证码明文写入审计日志

- 📍 Location: `internal/server/auth/service/mfa_service.go:2483` (`CreateDeliveryChallenge`)
- 🔍 Issue: MFA 一次性验证码的明文被直接写入审计日志的 `detail` 字段：
```go
_ = s.audit.Record(ctx, AuditEvent{
    // ...
    Detail: map[string]any{
        "method":   method,
        "target":   maskTarget(method, target),
        "demoCode": code,  // ← 明文验证码写入审计日志！
    },
})
```
- 💥 Impact:
  - **任何有审计日志读取权限的管理员都可以看到用户的MFA验证码**
  - 如果审计日志被泄露（数据库泄露、日志备份被窃取），攻击者可以绕过MFA
  - 违反了MFA验证码的保密性原则
- ✅ Fix: 立即移除 `demoCode` 字段，或将其设为仅在开发环境启用：
```go
detail := map[string]any{
    "method": method,
    "target": maskTarget(method, target),
}
// 仅在开发环境返回demoCode（且不写入审计日志）
// 生产环境绝不能记录验证码
_ = s.audit.Record(ctx, AuditEvent{
    // ...
    Detail: detail,
})
```
- 📎 Ref: CWE-532 (Insertion of Sensitive Information into Log File)

**注意**: 该 `demoCode` 还通过 `sendMFAChallenge` API 返回给前端（`auth.ts:480-482`），前端会显示 `challengeSentWithDemoCode`。这在开发环境可以理解，但**生产环境必须移除**。

---

### 3.2 🟠 HIGH — `callAuthnAPIWithHeaders` 使用 `http.DefaultClient` — 无超时且绕过SSRF防护

- 📍 Location: `oidc_authorize_interaction_handler.go:141` (`callAuthnAPIWithHeaders`)
- 🔍 Issue:
```go
resp, err := http.DefaultClient.Do(req)
```
`http.DefaultClient` 没有超时设置，且**没有使用SSRF防护的自定义Transport**（项目在域名验证中已实现了SSRF防护的 `DialContext`，但这里没有使用）。
- 💥 Impact:
  1. **超时缺失**: 如果 CoreURL 指向的服务无响应，请求将永久阻塞，导致 goroutine 泄漏
  2. **SSRF风险**: 如果 `h.cfg.CoreURL` 被配置为内网地址（虽然通常是管理员配置），请求不会经过SSRF过滤。更关键的是，如果未来有任何用户可控的URL被传入此路径，将直接绕过SSRF防护。
  3. **无连接池限制**: `http.DefaultClient` 是全局共享的，高并发下可能耗尽连接
- ✅ Fix:
```go
var authnHTTPClient = &http.Client{
    Timeout: 10 * time.Second,
    Transport: &http.Transport{
        MaxIdleConnsPerHost: 20,
        IdleConnTimeout:     30 * time.Second,
    },
}
// 然后使用 authnHTTPClient.Do(req) 代替 http.DefaultClient.Do(req)
```
- 📎 Ref: CWE-400 (Uncontrolled Resource Consumption), CWE-918 (Server-Side Request Forgery)

---

### 3.3 🟠 HIGH — Recovery Code 明文存储在数据库中

- 📍 Location: `internal/server/auth/service/mfa_service.go:2381-2390` (`GenerateRecoveryCodes`)
- 🔍 Issue:
```go
codes := sharedauthn.RecoveryCodes()
for _, code := range codes {
    entry := model.MFARecoveryCode{
        // ...
        Code: code,  // ← 明文存储！
    }
    if err := s.db.WithContext(ctx).Create(&entry).Error; err != nil {
```
Recovery code 以明文形式存储在 `mfa_recovery_code.code` 字段中。虽然 `model.MFARecoveryCode` 定义中也有 `CodeHash` 字段，但代码中并未对 recovery code 进行哈希。

在验证时（`mfa_service.go:2554-2555`），代码也优先比对明文：
```go
if strings.TrimSpace(item.Code) == code || (strings.TrimSpace(item.Code) == "" && utils.CheckSecret(item.CodeHash, code)) {
```
- 💥 Impact: 数据库泄露后，攻击者可以直接使用recovery code绕过MFA
- ✅ Fix: 生成recovery code时应存储哈希而非明文：
```go
hash, err := utils.HashSecret(code)
if err != nil { return nil, err }
entry := model.MFARecoveryCode{
    // ...
    CodeHash: hash,
    // Code: "",  // 不存储明文
}
```
- 📎 Ref: CWE-256 (Plaintext Storage of a Password)

---

### 3.4 🟠 HIGH — MFA Challenge 无尝试次数限制

- 📍 Location: `internal/server/auth/service/mfa_service.go:2507-2564` (`Verify`)
- 🔍 Issue: MFA验证失败时虽然递增了 `challenge.AttemptCount`，但**从未检查该计数是否超过阈值**：
```go
if !utils.CheckSecret(challenge.CodeHash, code) {
    challenge.AttemptCount++
    challenge.UpdatedAt = time.Now()
    updateMFAChallenge(challenge)
    return errors.New("invalid challenge code")
}
```
同样，TOTP验证（`mfa_service.go:2522-2529`）和 recovery code 验证（`mfa_service.go:2550-2559`）也没有任何尝试限制。
- 💥 Impact: 攻击者可以无限次尝试猜测6位数MFA验证码（最多100万种组合），在10分钟过期期内完全可以暴力破解。
- ✅ Fix:
```go
const maxMFAChallengeAttempts = 5

if challenge.AttemptCount >= maxMFAChallengeAttempts {
    now := time.Now()
    challenge.ConsumedAt = &now
    updateMFAChallenge(challenge)
    return errors.New("mfa challenge max attempts exceeded")
}
```
- 📎 Ref: CWE-307 (Improper Restriction of Excessive Authentication Attempts)

---

### 3.5 🟡 MEDIUM — `buildAuthorizeAppShell` title 未 HTML 转义

- 📍 Location: `oidc_authorize_interaction_handler.go:472-487`
- 🔍 Issue:
```go
func buildAuthorizeAppShell(title string) ([]byte, error) {
    html := `<!DOCTYPE html>
<html lang="en">
<head>
  ...
  <title>PPVT ` + title + `</title>
```
`title` 参数直接拼接到HTML中未经转义。
- 🔍 分析: `title` 来自 `authorizePageTitle(response.Stage)`，而 `stage` 的值来自 `switch` 语句中的硬编码中文字符串（"登录"、"输入设备码"等）。**当前没有用户可控输入能到达这里**。
- 💥 Impact: **低风险**，因为输入来源是硬编码的。但如果将来stage值来源改变（例如从数据库读取应用名称），则会产生XSS。
- ✅ Fix: 预防性修复——使用 `html.EscapeString(title)`:
```go
html := `...
  <title>PPVT ` + html.EscapeString(title) + `</title>
```
- 📎 Ref: CWE-79 (Cross-site Scripting)
- **置信度**: 中等 — 当前不可利用，但属于不安全的编码模式。

---

### 3.6 🟡 MEDIUM — TOTP 使用 SHA-1 算法

- 📍 Location: `internal/server/auth/service/mfa_service.go:2302`
- 🔍 Issue:
```go
key, err := totp.Generate(totp.GenerateOpts{
    // ...
    Algorithm: otp.AlgorithmSHA1,
```
TOTP生成使用SHA-1算法。虽然RFC 6238推荐SHA-1为默认算法，且大多数Authenticator App也只支持SHA-1，但SHA-256是更好的选择。
- 💥 Impact: 低 — SHA-1在HMAC-TOTP上下文中仍然安全，但不符合最佳实践。
- ✅ Fix: 如果兼容性允许，升级为 `otp.AlgorithmSHA256`。

**状态: 注意前次SHA-1→SHA-256指的是密码哈希，那个已修复。这里是TOTP特定的问题。**

---

### 3.7 🟡 MEDIUM — `io.ReadAll` 对内部API响应无大小限制

- 📍 Location: `oidc_authorize_interaction_handler.go:149`
- 🔍 Issue:
```go
responseBody, err := io.ReadAll(resp.Body)
```
对来自Core API的响应没有大小限制。虽然这是内部API调用，但如果Core API被入侵或出现bug返回巨大响应，可能导致OOM。
- ✅ Fix: 使用 `io.LimitReader`:
```go
responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // 10MB
```
- 📎 Ref: CWE-770 (Allocation of Resources Without Limits or Throttling)

---

### 3.8 🟡 MEDIUM — User Code 随机数偏差

- 📍 Location: `internal/server/auth/service/device_code.go:1110-1114`
- 🔍 Issue:
```go
const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // 31个字符
// ...
builder.WriteByte(alphabet[int(item)%len(alphabet)])
```
使用 `rand.Read` 生成的字节 `% 31` 存在模偏差（modulo bias）。256不能被31整除，所以某些字符出现的概率比其他字符高约 `(256 mod 31) / 256 ≈ 3%`。
- 💥 Impact: 轻微降低user code的随机性，但对于8字符的临时设备码，实际安全影响很小。
- ✅ Fix: 使用rejection sampling消除偏差。

---

## 4. 代码质量与架构问题

### 4.1 🟢 LOW — 默认Secret值硬编码

- 📍 Location: `internal/config/config.go:9133`
```go
Secret: getenv("PPVT_SECRET", "ppvt-dev-secret"),
```
- 🔍 Issue: 如果部署时忘记设置 `PPVT_SECRET` 环境变量，系统将使用弱默认值。该secret用于设备指纹签名和验证码HMAC。
- ✅ Fix: 启动时检查secret长度，如果是默认值则拒绝启动或至少打印WARNING。

### 4.2 🟢 LOW — DeviceAuthorizationByUserCode N+3 查询

- 📍 Location: `internal/server/auth/service/device_code.go:996-1022`
- 🔍 Issue: 查询设备授权视图需要4次独立的DB查询（device_authorization → application → project → organization），应使用JOIN或Preload。

### 4.3 🟢 LOW — `cleanupExpiredTransientState` 在每次操作时执行全量扫描

- 📍 Location: `internal/server/auth/service/transient_store.go:1309-1328`
- 🔍 Issue: 每次 store/load/consume 操作都会加写锁并遍历所有记录清理过期数据。在高并发下这是性能瓶颈。
- ✅ Fix: 使用定期后台清理（如每30秒一次），而不是在每次操作时同步清理。

### 4.4 架构改善亮点（正面评价）

本轮重构有以下值得肯定的改进：
- ✅ **Auth前端组件化**: 从1100行单文件拆分为7个组件 + Pinia Store + Router + i18n，代码可维护性大幅提升
- ✅ **消除服务端HTML渲染**: Device Code Handler从260行SSR缩减到75行，改为SPA重定向
- ✅ **消除 Bootstrap JSON 注入**: 所有数据通过API获取，不再注入到HTML模板中
- ✅ **URL编码安全**: `DeviceVerification` 正确使用 `url.QueryEscape` 处理 user_code
- ✅ **Static asset handler 路径遍历防护**: `StaticAssetPrefixHandler` 正确拒绝包含 `..` 的路径

---

## 5. 依赖审计

| 依赖 | 版本 | CVE | 严重度 | 状态 |
|------|------|-----|--------|------|
| `github.com/jackc/pgx/v5` | v5.6.0 | CVE-2024-27304 | 🟠 HIGH | ⚠️ 需升级到 ≥v5.5.4（此CVE影响 <v5.5.4，但v5.6.0应已包含修复。需确认完整版本号） |
| `github.com/golang-jwt/jwt/v5` | v5.3.1 | CVE-2024-51744, CVE-2025-30204 | 🟠 HIGH | ⚠️ CVE-2025-30204 需要 ≥v5.2.2（v5.3.1已包含修复 ✅） |
| `golang.org/x/crypto` | v0.49.0 | — | — | ✅ 较新版本 |
| `gorm.io/gorm` | v1.31.0 | — | — | ✅ 较新版本 |
| `github.com/go-jose/go-jose/v4` | v4.1.3 | — | — | ✅ |
| `github.com/go-webauthn/webauthn` | v0.16.1 | — | — | ✅ |
| `github.com/pquerna/otp` | v1.5.0 | — | — | ✅ |
| `github.com/redis/go-redis/v9` | v9.7.1 | — | — | ✅ 但尚未实际使用 |
| `github.com/casbin/casbin/v2` | v2.135.0 | — | — | ✅ |
| `github.com/google/uuid` | v1.6.0 | — | — | ✅ |

**关键发现**: `pgx/v5 v5.6.0` — CVE-2024-27304 修复版本为 v5.5.4。v5.6.0 应已包含修复 [5]。建议通过 `go list -m -json` 确认精确版本。

**`golang-jwt/v5` v5.3.1** — CVE-2025-30204 修复版本为 v5.2.2 [8]，当前 v5.3.1 已包含修复。CVE-2024-51744 为文档问题，不影响安全性 [7]。

---

## 6. 攻击面地图

### 6.1 外部输入点

| 端点 | 方法 | 认证 | 信任边界 | 风险 |
|------|------|------|----------|------|
| `GET /auth/authorize` | GET | 无 | 公开 | 参数经白名单switch过滤 |
| `POST /auth/api/session/create` | POST | Cookie | 公开 | **暴力破解入口点** |
| `POST /auth/api/session/verify_mfa` | POST | Cookie | 公开 | **MFA暴力破解入口点** |
| `POST /auth/api/context/query` | POST | Cookie | 公开 | 数据泄露（返回组织信息） |
| `POST /auth/token` | POST | ClientAuth | 半公开 | Auth code/refresh token交换 |
| `POST /auth/device/code` | POST | ClientAuth | 半公开 | 设备码创建 |
| `GET /auth/device` | GET | 无 | 公开 | 重定向到SPA |
| `POST /auth/api/captcha/refresh` | POST | 无 | 公开 | 验证码刷新 |
| `POST /auth/api/webauthn/login/*` | POST | 无 | 公开 | WebAuthn登录流程 |
| `POST /auth/api/session/u2f/*` | POST | Cookie | 公开 | U2F MFA流程 |
| `GET /auth/userinfo` | GET | Bearer Token | 认证 | 用户信息 |
| `POST /auth/revoke` | POST | ClientAuth | 认证 | Token撤销 |
| `POST /auth/introspect` | POST | ClientAuth | 认证 | Token检查 |
| `GET /auth/end_session` | GET | Cookie | 认证 | 会话注销 |
| `GET /.well-known/openid-configuration` | GET | 无 | 公开 | 元数据（信息泄露） |
| `GET /auth/keys` | GET | 无 | 公开 | JWKS |
| 内部API: `/api/authn/v1/*` | POST | JWT Assertion | 内部 | Auth→Core通信 |

### 6.2 信任边界

```
[浏览器/CLI] → [Auth Server] → [Core Server]
       ↑              ↑              ↑
  用户输入      CORS/Cookie      JWT Client Assertion
  (不可信)    Session Cookie     (X-PPVT-Client-*)
              (半可信)           (受信内部通信)
                                       ↓
                                   [Database]
                                   [Redis(未启用)]
```

**关键信任边界问题**:
1. Auth→Core 使用 `http.DefaultClient`，无超时
2. Auth 转发 `Cookie` 和 `X-PPVT-Original-*` 头给Core
3. Device Code 的 `user_code` 来自用户输入，正确地做了 `url.QueryEscape`

---

## 7. 修复优先级路线图

### P0 — 立即修复（24小时内）

| # | 问题 | 工作量 |
|---|------|--------|
| 1 | 🔴 登录暴力破解防护 — 至少实现IP+账号级的rate limit | 8h |
| 2 | 🔴 移除审计日志中的 `demoCode` 明文验证码 | 0.5h |
| 3 | 🟠 MFA Challenge 添加尝试次数限制（最大5次） | 1h |

### P1 — 短期修复（1周内）

| # | 问题 | 工作量 |
|---|------|--------|
| 4 | 🟠 Recovery Code 哈希存储，停止明文存储 | 2h |
| 5 | 🟠 `callAuthnAPIWithHeaders` 创建带超时的专用HTTP Client | 1h |
| 6 | 🟡 `buildAuthorizeAppShell` title 添加 `html.EscapeString` | 0.5h |
| 7 | 🟡 `io.ReadAll` 添加大小限制 | 0.5h |

### P2 — 中期修复（1个月内）

| # | 问题 | 工作量 |
|---|------|--------|
| 8 | 🟡 TransientStore 迁移到 Redis | 16h |
| 9 | 🟡 CORS 域名查询添加缓存 | 4h |
| 10 | 🟢 默认Secret检测与告警 | 1h |
| 11 | 🟢 DeviceAuthorizationByUserCode 优化为JOIN查询 | 2h |

### P3 — 长期改进

| # | 问题 | 工作量 |
|---|------|--------|
| 12 | 🟢 补充单元测试和集成测试 | 40h+ |
| 13 | 🟢 添加 Dockerfile | 4h |
| 14 | 🟢 审计日志脱敏框架 | 8h |
| 15 | 🟢 `cleanupExpiredTransientState` 改为后台定期清理 | 2h |

---

## 8. 与前次审计的对比总结

| 维度 | 第4轮 | 第5轮 | 变化 |
|------|-------|-------|------|
| CRITICAL 问题数 | 2 | 2 | → (1个修复 + 1个新发现) |
| HIGH 问题数 | 0 | 3 | ↑ 新发现3个 |
| MEDIUM 问题数 | 2 | 4 | ↑ |
| LOW 问题数 | 5 | 5 | → |
| 暴力破解防护 | ❌ | ❌ | **第5轮仍未修复** |
| Bootstrap XSS | ❌ | ✅ | **架构重构消除** |
| 代码架构 | 较差(1100行单文件) | ✅ 优秀 | **大幅改善** |
| 总体评分 | 5.0 | 5.5 | ↑ +0.5 |

---

## 9. 结论

第5轮的前端架构重构是**重大正面改进**，从根本上消除了Bootstrap JSON注入XSS风险面，并大幅提升了代码可维护性。但**登录暴力破解防护已连续5轮未修复**，这是一个不可接受的安全缺陷。新发现的MFA验证码明文日志泄露和recovery code明文存储也需要立即处理。

**在修复P0问题之前，该系统不应在生产环境中暴露到互联网上。**

---

*报告生成: Pass Pivot Security Audit v5 — Reviewer Agent*
