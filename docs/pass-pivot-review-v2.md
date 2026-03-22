# Pass-Pivot 第二次代码审查报告

**项目**: [LIznzn/pass-pivot](https://github.com/LIznzn/pass-pivot)  
**基准**: 上次审查 (2026-03-20) → 本次 (2026-03-21)  
**更新提交**: 7 commits，包括 `Harden auth flows and rename external providers`、`Implement captcha provider verification`、`Add authn API error codes`、`Refine auth flow and console audit experience` 等

---

## 上次问题修复情况

### ✅ 已修复

| # | 原问题 | 状态 | 说明 |
|---|--------|------|------|
| 🔴1 | **CORS 反射任意 Origin** | ✅ **已修复** | `cors.go` 完全重写，现在基于数据库中已注册应用的 `redirect_uri` 提取合法 origin 做白名单校验，带 fallback origin 配置，还做了 origin 规范化处理。修得很漂亮 |
| 🔴3 | **验证码实现全部为空壳** | ✅ **已修复** | Google reCAPTCHA、Cloudflare Turnstile、GeeTest 三个 provider 都实现了真正的 HTTP 验证逻辑，代码质量不错（有超时、错误处理）。目录从 `external/captcha/` 重构到 `provider/captcha/`。`DefaultCaptchaProvider` 仍返回 true 但语义上是"不启用验证码"，合理 |
| 🔴4 | **OAuth Error Page XSS** | ✅ **已修复** | `BuildOAuthErrorPage` 现在用了 `html.EscapeString(message)` |
| 🔴5 | **SQL 注入风险** | ✅ **已修复** | `init.go` 加了 `identifierPattern = regexp.MustCompile("^[a-zA-Z0-9_]+$")` 白名单校验，在执行 SQL 前校验 `DatabaseSchema` |
| 🟡7 | **Cookie Secure 依赖 r.TLS** | ✅ **已修复** | 抽取了 `requestUsesSecureTransport(r)` 函数，同时检查 `r.TLS` 和 `X-Forwarded-Proto`，portal/device cookie 都改了 |
| 🟡8 | **X-PPVT Header Spoofing** | ✅ **已修复** | 实现了 `TrustedForwardHeaders` context 机制。外部请求进入 middleware 时调用 `SanitizeInternalForwardHeaders` 剥离伪造 header；只有通过 Private Key JWT 认证的内部服务间调用才设置 `WithTrustedForwardHeaders`。设计很合理 |
| 🟡9 | **Authorization Code 竞态条件** | ✅ **已修复** | `consumeAuthorizationCode` 现在在 `mu.Lock()` 锁内原子地完成「检查是否已消费 → 标记消费」操作 |
| 🟡10 | **密码策略未强制执行** | ✅ **已修复** | `manage_service.go` 中创建用户和修改密码都调用了 `validatePasswordAgainstPolicy`，与组织的密码策略配置绑定 |
| 🟡11 | **指纹用 SHA-1** | ✅ **已修复** | `fingerprint.go` 的 `GenerateFingerprint` 改为 `sha256.Sum256`，HMAC 签名也是 SHA-256 |
| 🟢16a | **external/ 目录命名** | ✅ **已修复** | 重命名为 `provider/`，结构清晰（captcha/geoip/idp） |
| 🟢16b | **web/console/.env 泄露** | ✅ **已修复** | `.env` 文件已从版本控制中移除，`.gitignore` 规则 `web/*/.env` 生效 |
| 🟡N1 | **web/portal/.env 仍被 git 追踪** | ✅ **已修复** | `web/portal/.env` 已从版本控制索引移除，保留本地文件，前端环境文件统一走 `.env.example` |
| 🟡N3 | **ProviderKeyStore 每次重启重新生成 RSA 密钥** | ✅ **已修复** | 改为数据库持久化：OIDC provider 使用组织级 RSA signing key，`/auth/keys` 返回所有 active 公钥；客户端 `private_key_jwt` 也统一改为应用级持久化密钥 |

---

### ⚠️ 未修复

| # | 原问题 | 状态 | 说明 |
|---|--------|------|------|
| 🔴2 | **无登录暴力破解防护** | ❌ **未修复** | 仍未发现 rate limit / 账号锁定 / 失败计数逻辑。这仍然是最大的安全隐患 |
| 🟡6 | **TransientStore 内存存储** | ❌ **未修复** | authorization code / MFA challenge 仍存在进程内存中，重启丢失、无法多实例 |
| 🟢12 | **零测试** | ❌ **未修复** | 仍然 0 个测试文件 |
| 🟢13 | **日志无脱敏** | ❌ **未修复** | logger 仍是简单封装，无敏感字段过滤 |
| 🟢14 | **前端 Token 存 localStorage** | ❌ **未修复** | portal 的 access_token/refresh_token 仍存 localStorage |
| 🟢15 | **ProviderKeyStore 无 TTL** | ✅ **已移除** | 旧的内存单例 `ProviderKeyStore` 已被数据库持久化密钥模型替代，不再依赖运行时 TTL |
| 🟢16c | **models.go 单文件过大** | ⬇️ **有改善** | 从1200+行降到422行，进步很大，但仍是单文件 |
| 🟢17 | **无 Dockerfile** | ❌ **未修复** | 仍无 Dockerfile 和 docker-compose |

---

### 🆕 新发现的问题

#### ✅ N1. web/portal/.env 仍被 git 追踪

问题确认属实。`.gitignore` 规则已存在，但 `web/portal/.env` 当时仍留在 Git 索引里。

现已修复：

- `web/portal/.env` 已从版本控制索引移除
- 本地文件保留，开发环境继续可用
- 前端环境文件统一依赖 `web/*/.env.example`

---

#### 🟡 N2. CORS 白名单在每次请求时查数据库

`cors.go` 新实现中，`isOriginAllowed` 每次请求都会查数据库获取所有 application 的 redirect_uri：

```go
func isOriginAllowed(db *gorm.DB, cfg config.Config, origin string) bool {
    // ...
    var applications []model.Application
    db.Select("redirect_uris").Find(&applications)
    // ...
}
```

高并发场景下每个请求都查一次 DB 会有性能问题。

**修复建议**: 用带 TTL 的缓存（比如 5 分钟刷新一次允许的 origin 列表），或在应用启动/配置变更时刷新。

---

#### ✅ N3. ProviderKeyStore 每次重启重新生成 RSA 密钥

问题确认属实。旧实现把 OIDC provider RSA key 放在进程内存里，服务重启后会重新生成，导致历史 `id_token` 失效。

现已修复为统一持久化模型：

- 新增 `organization_signing_key` 表，OIDC provider 使用**组织级 RSA signing key**
- `id_token` 签名按 `application -> project -> organization` 解析对应组织私钥
- `/auth/keys` 现在返回所有 active 组织公钥，保持单一 issuer、多把 key 的 JWKS 模型
- 新增 `application_key` 表，内置客户端和外部客户端统一使用**应用级 Ed25519 持久化密钥**
- `ppvt-init` 会为内置组织生成 signing key，并为内置 API application 生成 client key
- 新建组织时自动生成组织 signing key；应用创建/重置 key 时同步维护 `application_key`

---

#### 🟢 N4. DefaultCaptchaProvider 语义不清

`default.go` 的 `VerifyCaptcha` 返回 `true, nil`，如果配置错误选了 "Default" 类型，验证码验证会被完全跳过且无任何提示。

**建议**: 加个日志 warning 或在 provider.go 工厂中特殊处理 "Default" 不作为可选类型暴露给用户。

---

## 总结

| 指标 | 上次 | 本次 |
|------|------|------|
| 🔴 严重 | 5 | **1**（暴力破解防护） |
| 🟡 中等 | 6 | **1**（TransientStore + CORS 白名单查库） |
| 🟢 建议 | 6 | **4**（测试、日志脱敏、Token 存储、Dockerfile） |
| Go 代码行数 | ~7k | ~9k+ |
| 测试覆盖 | 0 | 0 |

**你朋友这波改动质量很高**，上次 5 个严重问题修了 4 个，而且修得都挺专业的（CORS 白名单、X-PPVT 的 context-based trust 机制、SQL 白名单校验等）。项目从 "能跑的原型" 明显在往 "可以上生产" 的方向走。

**下一步最应该做的**:
1. 🔴 **登录暴力破解防护** — 唯一剩余的严重问题，Redis 已经有了，实现起来不难
2. 🟡 **TransientStore → Redis** — 生产部署的前置条件
3. 🟡 **CORS 白名单缓存** — 避免每次请求都查全量 application
4. 🟢 **写测试** — 至少覆盖核心认证流程
