# PPVT Exec Split Plan

## 背景

当前 `ppvt-core` 在组织域名验证流程中会直接访问外部不可信目标：

- `http_file`：请求用户控制域名上的验证文件
- `dns_txt`：对用户控制域名执行 TXT 记录查询

这类请求会暴露 `ppvt-core` 所在节点的出口 IP，不适合继续由核心控制面二进制直接执行。为降低暴露面，需要将“访问不可信外部服务”的执行能力从 `ppvt-core` 中拆出，收敛到单独部署的 `ppvt-exec` 二进制。

本次规划仅覆盖“不可信目标访问执行器”拆分，不覆盖现有 Google reCAPTCHA 和 Cloudflare Turnstile 校验。这两类验证码目标固定、边界清晰，暂时继续保留在现有 `core/auth` 进程内。

## 当前范围识别

当前确认属于本次拆分范围的逻辑：

- `VerifyOrganizationDomain`
- `verifyOrganizationDomainHTTPFile`
- `verifyOrganizationDomainDNSTXT`
- `lookupTXTRecords`

这些逻辑当前位于：

- `internal/server/core/api/manage/manage_service.go`

当前明确不纳入本次拆分的逻辑：

- `PrepareOrganizationDomainVerification`
- 组织权限校验
- 组织设置读写与 `verified` 状态落库
- Google reCAPTCHA 校验
- Cloudflare Turnstile 校验

## 拆分目标

- `ppvt-core` 不再直接访问用户控制的域名或 DNS
- `ppvt-core` 继续保留业务编排、鉴权、配置持久化和状态更新职责
- 新增 `ppvt-exec`，专门负责访问不可信外部目标
- `ppvt-exec` 部署在独立云节点或独立出口网络中
- 本地开发环境允许保留本地执行 fallback，降低联调成本

## 目标职责边界

### ppvt-core

保留职责：

- API 鉴权与组织权限校验
- 域名 challenge 准备与 token 生成
- 域名配置读取、归一化、校验
- 调用外部执行器
- 在验证成功后更新数据库中的 `Verified` / `VerifiedAt`
- 记录管理面审计日志

移出职责：

- 对用户控制域名发起 HTTP 请求
- 对用户控制域名发起 DNS TXT 查询

### ppvt-exec

仅承担以下职责：

- 执行 `domain_http_file_verify`
- 执行 `domain_dns_txt_verify`
- 向调用方返回结构化执行结果

明确不承担：

- 数据库连接
- 组织权限判断
- 控制台 API 暴露
- challenge 准备
- `verified` 状态落库

## 推荐架构

建议采用“核心面编排 + 外部执行器”模式：

1. 控制台调用 `ppvt-core` 的现有管理 API
2. `ppvt-core` 完成组织权限校验和 domain 配置读取
3. `ppvt-core` 将最小执行参数发送给 `ppvt-exec`
4. `ppvt-exec` 执行外部访问
5. `ppvt-exec` 返回成功或失败原因
6. `ppvt-core` 根据结果决定是否写回 `Verified=true`

## 代码改造建议

### 1. 先抽执行接口

不要先迁移 handler。优先在 `manage.Service` 内引入执行器接口：

```go
type DomainVerificationExecutor interface {
    VerifyOrganizationDomain(ctx context.Context, domain model.OrganizationDomain) error
}
```

然后将当前本地逻辑整理为默认实现。

### 2. 本地实现与远程实现并存

建议提供两个实现：

- `localDomainVerificationExecutor`
- `remoteDomainVerificationExecutor`

其中：

- `localDomainVerificationExecutor` 复用当前 `verifyOrganizationDomainHTTPFile` / `verifyOrganizationDomainDNSTXT`
- `remoteDomainVerificationExecutor` 通过内部 HTTP API 调用 `ppvt-exec`

### 3. Service 侧调用方式

`VerifyOrganizationDomain` 调整为：

1. 校验 `organizationId`
2. 校验当前操作者是否可管理目标组织
3. 读取 organization domain settings
4. 找到目标 domain 和 `verificationToken`
5. 调用 `executor.VerifyOrganizationDomain(...)`
6. 成功后再写入 `Verified=true` 与 `VerifiedAt`

## 新增二进制建议

建议新增如下目录：

- `cmd/ppvt-exec/main.go`
- `internal/server/exec/bootstrap/bootstrap.go`
- `internal/server/exec/router/router.go`
- `internal/server/exec/api/exec/handler.go`
- `internal/server/exec/api/exec/service.go`

`ppvt-exec` 仅暴露内部执行接口，不复用控制台对外管理路由。

## 内部 API 建议

建议提供最小内部接口：

- `POST /internal/exec/v1/domain/verify`

请求体示例：

```json
{
  "host": "example.com",
  "verificationMethod": "http_file",
  "verificationToken": "xxxx"
}
```

成功响应示例：

```json
{
  "ok": true
}
```

失败响应示例：

```json
{
  "ok": false,
  "error": "domain http_file verification failed: verification file not found"
}
```

建议保持请求字段最小化，只传执行所需信息，不把组织配置、用户信息或数据库状态传给 `ppvt-exec`。

## 配置项建议

建议在 `ppvt-core` 增加：

- `PPVT_EXEC_ENABLED`
- `PPVT_EXEC_URL`
- `PPVT_EXEC_SHARED_SECRET`
- `PPVT_EXEC_TIMEOUT`

建议在 `ppvt-exec` 增加：

- `PPVT_HTTP_ADDR`
- `PPVT_EXEC_SHARED_SECRET`
- `PPVT_LOG_LEVEL`

## 调用鉴权建议

`ppvt-exec` 不能作为裸接口暴露。建议最小安全方案：

- `ppvt-core -> ppvt-exec` 使用独立 `shared secret`
- 通过固定请求头携带认证信息，例如 `X-PPVT-Exec-Token`
- `ppvt-exec` 仅监听内网地址或受限安全组

后续若需增强，可在此基础上追加：

- 时间戳
- HMAC 签名
- 重放窗口限制

## 本地开发策略

建议保留本地 fallback：

- `PPVT_EXEC_ENABLED=false` 时，`ppvt-core` 使用本地 executor
- `PPVT_EXEC_ENABLED=true` 时，`ppvt-core` 改为远程调用 `ppvt-exec`

这样可以降低本地开发和单机调试成本，同时为生产环境保留强制拆分部署能力。

## 安全约束建议

由于 `ppvt-exec` 专门承担对不可信目标的访问，建议在拆分时补充以下防护：

- `http_file` 只允许 `http` / `https`
- 对目标解析结果做地址段拦截，拒绝回环、内网、链路本地和保留地址
- 禁止自动跳转到内网地址
- 限制响应体大小，例如 `64KB`
- 控制超时时间，避免被慢连接拖住
- DNS 查询错误信息适度收敛，避免暴露过多内部信息

当前实现里最需要补的是 SSRF 防护。现有逻辑会直接对用户提供的目标发起请求，拆分时应一并收口。

## 审计与可观测性建议

建议为 `ppvt-core` 和 `ppvt-exec` 都补充执行日志或审计字段，至少包含：

- `organizationId`
- `host`
- `verificationMethod`
- `result`
- `error`
- `duration`
- `executorNode`

## 测试建议

至少覆盖以下测试场景：

- `VerifyOrganizationDomain` 在 executor 成功时正确更新数据库状态
- executor 失败时数据库不应误更新 `Verified`
- `remoteDomainVerificationExecutor` 能正确透传错误信息
- `ppvt-exec` 对 `http_file` / `dns_txt` 路径正常工作
- `ppvt-exec` 拒绝内网地址、回环地址和非法 scheme
- `shared secret` 不匹配时拒绝请求

## 实施顺序建议

建议按以下顺序推进：

1. 在 `manage.Service` 中引入 `DomainVerificationExecutor`
2. 将当前逻辑整理为本地 executor，先保证行为不变
3. 新增 `ppvt-exec` 二进制和最小内部路由
4. 实现 `remoteDomainVerificationExecutor`
5. 增加配置开关与运行时切换
6. 增加测试和最小审计
7. 生产环境切换到 `ppvt-exec`

## 阶段性结论

这次拆分的合适粒度不是迁移整个管理面域名验证 API，而是只拆“访问不可信外部目标”的执行能力：

- `ppvt-core` 保留业务编排和数据状态变更
- `ppvt-exec` 承接不可信 HTTP/DNS 访问

这样可以解决 `ppvt-core` 暴露出口 IP 的问题，同时避免把 Google / Cloudflare 验证码这类可信固定目标一并复杂化。

## 当前状态

该文档仅作为未来改造计划记录。

当前仓库暂不执行此拆分，不包含代码实现变更。
