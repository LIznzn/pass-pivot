# Pass-Pivot 代码审查报告

**项目**: [LIznzn/pass-pivot](https://github.com/LIznzn/pass-pivot)  
**类型**: 身份认证 & 授权平台 (OAuth2/OIDC Provider)  
**技术栈**: Go (后端) + Vue/TypeScript (前端) + MySQL/PostgreSQL + Redis + Casbin  
**审查日期**: 2026-03-20  

---

## 总体评价

项目架构设计思路清晰，模块划分合理（cmd/internal/provider/web 四层结构），OAuth2/OIDC 的核心流程基本成形。代码风格统一，命名规范。但作为一个**身份认证系统**，在安全性方面有几个需要重点关注的问题。

---

## 🔴 严重问题

### 1. CORS 配置 — 反射任意 Origin（最严重）
**文件**: `internal/server/shared/web/cors.go`

```go
origin := r.Header.Get("Origin")
if origin != "" {
    w.Header().Set("Access-Control-Allow-Origin", origin)
    w.Header().Set("Access-Control-Allow-Credentials", "true")
}
```

这是一个典型的 CORS 误配置漏洞。**任何网站**都可以带 Cookie 向你的 API 发请求并读取响应。对于一个认证系统，这等于直接打开了 CSRF + 数据窃取的大门。

**修复建议**: 维护一个允许的 Origin 白名单（可以从已注册应用的 `redirect_uri` 中提取 origin），仅对匹配的 origin 返回 CORS 头。项目中 `fido/service.go` 的 `CollectOrigins` 方法其实已经写了提取 origin 的逻辑，可以复用。

**处理方法**:
- 已改为基于白名单返回 CORS 头，不再反射任意 `Origin`
- 白名单来源包括 `PPVT_AUTH_URL`、`PPVT_CORE_URL` 以及数据库中各应用配置的 `redirect_uri` origin
- 仅当请求源命中白名单时才返回 `Access-Control-Allow-Origin` 和 `Access-Control-Allow-Credentials`

---

### 2. 无登录暴力破解防护
**文件**: `internal/server/core/api/authn/authn_service.go`

全局搜索 `rate`、`limit`、`lockout`、`attempt` 无任何结果。登录接口 `/api/authn/v1/session/create` 没有：
- 速率限制 (Rate Limiting)
- 账号锁定机制
- 失败次数记录
- IP 级别的限流

攻击者可以无限制地尝试密码。审计日志记了 `auth.login.failed`，但只是记录，没有任何阻断逻辑。

**修复建议**: 在 Redis 中维护 `login:fail:{identifier}` 和 `login:fail:{ip}` 计数器，超过阈值后临时封禁。可以用 `golang.org/x/time/rate` 或 Redis 的 sliding window 实现。

---

### 3. 验证码实现全部为空壳
**文件**: `provider/captcha/` 下所有 provider

```go
func (captcha *GoogleCaptchaProvider) VerifyCaptcha(...) (bool, error) {
    return true, nil  // 直接返回 true！
}
```

Google reCAPTCHA、Cloudflare Turnstile、GeeTest 的验证实现**全部直接返回 true**。如果业务代码调用了验证码校验，实际上完全无效。

**修复建议**: 要么实现真正的验证逻辑，要么在代码中明确标记为未实现并拒绝调用，不能悄悄返回 true。

---

### 4. OAuth Error Page 存在反射型 XSS
**文件**: `internal/server/auth/service/oauth_oidc_standard.go:450`

```go
func BuildOAuthErrorPage(message string) []byte {
    return []byte(fmt.Sprintf("<html><body><h1>OAuth Error</h1><p>%s</p></body></html>", message))
}
```

`message` 未经 HTML 转义直接拼接到 HTML 中。如果 message 来自用户输入（如错误的 `redirect_uri` 参数），攻击者可以注入任意 JS 代码。

**修复建议**: 使用 `html.EscapeString(message)` 或 `html/template` 进行转义。

**处理方法**:
- 已在 `BuildOAuthErrorPage` 中对 `message` 使用 `html.EscapeString`
- 保留原有错误页输出方式，但消除了用户输入进入 HTML 时的脚本注入风险

---

### 5. SQL 拼接存在注入风险
**文件**: `internal/tool/init/init.go:192-219`

```go
adminDB.Exec("DROP DATABASE IF EXISTS `" + cfg.DatabaseSchema + "`")
adminDB.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS "%s" WITH (FORCE)`, cfg.DatabaseSchema))
```

`DatabaseSchema` 来自环境变量/配置文件，直接拼接到 SQL 中。虽然这是初始化工具，但如果配置被污染，可能导致任意 SQL 执行。

**修复建议**: 对 `DatabaseSchema` 做白名单校验（仅允许字母数字下划线），或用 `regexp.MustCompile("^[a-zA-Z0-9_]+$")` 验证。

---

## 🟡 中等问题

### 6. TransientStore 内存存储存在扩展性和可靠性问题
**文件**: `internal/server/auth/service/transient_store.go`

Authorization Code、MFA Challenge、External Auth State 全部存在**进程内存的 sync.Map** 中：

```go
var transientStore = &TransientStore{
    authorizationCodes: make(map[string]model.AuthorizationCode),
    mfaChallenges:      make(map[string]model.MFAChallenge),
    externalAuthStates: make(map[string]model.ExternalAuthState),
}
```

问题：
- 服务重启后所有 authorization code 丢失，正在授权的用户流程中断
- 无法水平扩展（多实例部署时 code 在 A 实例生成，token 请求打到 B 实例会失败）
- 清理靠 `time.AfterFunc(10*time.Minute, cleanupExpiredTransientState)`，粒度粗

**修复建议**: 项目已经依赖了 Redis，把这些临时状态存到 Redis 中，用 TTL 自动过期。

---

### 7. Cookie Secure 标记依赖 `r.TLS != nil`
**文件**: `internal/server/shared/handler/portal_session_cookie.go:25`、`device_cookie.go:25`

```go
Secure: r.TLS != nil,
```

在反向代理（Nginx/Caddy）后面，Go 服务通常以 HTTP 运行，`r.TLS` 永远为 nil，导致 Cookie 的 Secure 标记不会设置。Session Cookie 可能在 HTTP 下被传输。

**修复建议**: 通过配置项或检测 `X-Forwarded-Proto: https` 来决定 Secure 标记。

**处理方法**:
- 已新增统一的安全传输判断函数
- Cookie 的 `Secure` 标记不再只依赖 `r.TLS != nil`
- 当请求经过反向代理且 `X-Forwarded-Proto=https` 时，也会正确下发 `Secure` Cookie

---

### 8. X-PPVT-Original-* Header Spoofing
**文件**: `internal/server/shared/handler/request_forward.go`

```go
func OriginalRemoteAddr(r *http.Request) string {
    if forwarded := r.Header.Get("X-PPVT-Original-Remote-Addr"); forwarded != "" {
        return forwarded
    }
    return r.RemoteAddr
}
```

`X-PPVT-Original-Remote-Addr` 是内部服务间通信使用的 header，但没有验证请求是否来自可信的内部服务。外部客户端可以直接设置这个 header 来伪造 IP 地址，影响审计日志和 GeoIP 的准确性。

**修复建议**: 在 Core 服务的入口中间件中剥离外部请求的 `X-PPVT-*` headers，仅信任来自 Auth 服务的内部请求。

**处理方法**:
- 已在 Core API 入口中间件中默认剥离外部请求携带的 `X-PPVT-Original-*` 头
- 仅当请求经过内部 `private_key_jwt` 服务间调用链时，才在上下文中标记为可信并允许读取这些头
- 外部客户端现在无法伪造原始 IP 和 User-Agent 影响审计与 GeoIP

---

### 9. Authorization Code 一次性使用保护不完整
**文件**: `internal/server/auth/service/oidc_service.go:155-200`

Code 被消费时设置了 `ConsumedAt`，但 cleanup 逻辑中：
```go
if record.ExpiresAt.Before(now) || record.ConsumedAt != nil {
    delete(transientStore.authorizationCodes, code)
}
```

已消费的 code 会在下次 cleanup 时被删除。但在 cleanup 运行前（最长10分钟），如果有并发请求使用同一个 code，由于是先查 map 再标记消费，存在 TOCTOU 竞态条件。

**修复建议**: 在 `ConsumeAuthorizationCode` 中加锁并原子地检查 + 标记消费状态。

**处理方法**:
- 已新增 `consumeAuthorizationCode` 原子消费逻辑
- 在同一把锁内完成“读取 code、校验未过期/未消费、标记 `ConsumedAt`”三个步骤
- 避免并发请求在 cleanup 运行前重复消费同一个 Authorization Code

---

### 10. 密码策略已定义但未见强制执行
**文件**: 
- `internal/tool/init/init.go` 中定义了 `PasswordPolicy`（最小12位、要求大小写+数字）
- `internal/server/core/api/authn/authn_service.go` 中未搜索到密码强度校验逻辑

注册/改密码时可能没有实际执行密码策略校验。

**修复建议**: 在用户注册和修改密码的 service 层加入密码策略验证函数。

**处理方法**:
- 已新增密码策略校验函数，支持最小长度、大小写、数字、符号等规则
- 在后台创建用户时强制执行组织级 `PasswordPolicy`
- 在当前用户修改密码时同样执行相同策略，避免初始化策略仅存在于配置但不生效

---

### 11. Fingerprint 使用 SHA-1
**文件**: `util/fingerprint.go`

```go
sum := sha1.Sum(seed)
```

SHA-1 已被证明存在碰撞攻击。虽然这里只是设备指纹（非密码学安全场景），但在安全相关项目中使用 SHA-1 不是好信号。

**修复建议**: 替换为 SHA-256，改动极小。

**处理方法**:
- 已将设备指纹摘要算法从 `SHA-1` 替换为 `SHA-256`
- 仅影响指纹生成实现，不改变现有签名校验的 HMAC-SHA-256 逻辑

---

## 🟢 改进建议

### 12. 零测试覆盖
整个项目 **0 个测试文件**。对于一个认证系统来说非常危险：
- OAuth2/OIDC 有大量边界条件（state 校验、PKCE、token 过期、scope 验证等）
- 密码哈希、JWT 签名这些不写测试很容易引入回归 bug
- Casbin 策略规则复杂，没有测试很难验证正确性

建议至少覆盖：`authn_service`、`oidc_service`、`util/` 工具函数、`authz_service` 的策略判定。

---

### 13. 日志安全
**文件**: `internal/logger/logger.go`

使用了 `slog`，不错。但没有看到对敏感字段（password、token、secret）的脱敏处理。建议在 logger 层加一个 sanitizer，自动过滤包含敏感关键词的字段值。

---

### 14. 前端 Token 存储
**文件**: `web/console/src/auth.ts`、`web/portal/src/auth.ts`

```typescript
localStorage.setItem('ppvt_access_token', token)
```

Access Token 存在 localStorage 中，容易被 XSS 攻击获取。对于认证系统的管理后台，建议使用 HttpOnly Cookie 或 BFF（Backend For Frontend）模式。

**处理说明**:
- 该项当前选择忽略，不按 Cookie/BFF 方案修改
- 原因是系统明确要求兼容多种客户端，并统一采用 `Authorization: Bearer` 鉴权模型
- 浏览器端继续使用 Bearer Token，并保留 `localStorage` 存储，以保持现有客户端模型和刷新后会话恢复能力
- 该风险作为已知设计权衡接受，后续应通过严格的 XSS 防护、CSP、依赖治理和前端输出转义来降低风险

---

### 15. ProviderKeyStore 密钥缓存
**文件**: `internal/server/auth/service/provider_keys.go`

```go
func (s *ProviderKeyStore) Instance() (*ProviderKeySet, error) {
    s.mu.RLock()
    if s.cached != nil { ... return s.cached }
    // 每次 miss 都从 DB 查
}
```

密钥加载后永不过期更新。如果需要密钥轮转，必须重启服务。建议加个 TTL 或 reload 机制。

**处理方法**:
- 已为 `ProviderKeyStore` 增加缓存加载时间记录
- 已提供显式 `Reload()` 能力，可在后续管理接口或轮转任务中主动刷新内存密钥
- 当前尚未引入自动 TTL 轮转策略，但已消除“只能重启服务刷新”的硬限制

---

### 16. 项目结构方面的小建议
- `external/` 和 `internal/` 并列有点反直觉，`external/` 实际上是"外部服务集成"而非 Go 的 `external` 概念，建议改名为 `integration/` 或 `provider/`
- 处理结果：已改名为 `provider/`
- `web/console/.env` 包含硬编码的 Application ID（`65d5b9f6-...`），这个文件不应该提交到仓库
- 处理结果：已从仓库移除 `web/console/.env`，并通过 `.gitignore` 忽略前端环境文件，改为使用 `web/console/.env.example` 生成本地配置
- `internal/model/models.go` 单文件1200+行，建议按领域拆分（user.go、session.go、token.go 等）

---

### 17. Docker / 部署
没有 Dockerfile 和 docker-compose.yml（README 提到了 Docker 但文件不存在）。建议补充：
- 多阶段构建的 Dockerfile
- 带 MySQL + Redis 的 docker-compose.yml
- health check endpoint

---

## 总结

| 类别 | 发现数量 |
|------|---------|
| 🔴 严重 | 5 |
| 🟡 中等 | 6 |
| 🟢 建议 | 6 |

**最优先修复**:
1. CORS 反射任意 Origin（#1）— 这个是最紧急的，相当于门没锁
2. 登录暴力破解防护（#2）— 认证系统的基本要求
3. XSS in OAuth Error Page（#4）— 简单修复，`html.EscapeString` 一行搞定
4. TransientStore 迁移到 Redis（#6）— 影响生产可用性

整体来看，项目的**架构底子不错**，模块划分清晰，OIDC/OAuth2 的协议实现也比较完整。主要问题集中在安全加固层面，补上这些之后就是一个相当不错的认证平台了。
