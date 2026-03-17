# 未完成的重构工作

生成时间：2026-03-17

## 高优先级

### 1. 拆分 `manage.Service`

当前问题：

- `internal/server/core/api/manage/manage_service.go` 仍然过胖。
- `internal/server/core/api/manage/manage_handler.go` 仍然承载过多入口。

建议方向：

- 按资源域继续拆分 service。
- `manage` 只保留 API 聚合层，不继续作为平台总汇式 service。

建议拆分维度：

- organization 管理
- project / application 管理
- user / securekey 管理
- external_idp / binding 管理
- session / device / token 管理

### 2. 收敛 `authn_service.go` 的职责面

当前问题：

- `internal/server/core/api/authn/authn_service.go` 体量偏大。
- 同时承担登录、WebAuthn 登录完成、会话确认、MFA 验证、token 下发、device upsert、session 编排。

建议方向：

- 继续保留认证语义归属。
- 在 authn 域内拆分更细的 orchestration / issuing / persistence helper。

## 中优先级

### 3. 评估 `shared/bootstrap` 是否需要继续拆分

当前问题：

- `internal/server/shared/bootstrap/bootstrap.go` 仍集中承担依赖组装。

建议方向：

- 如果后续依赖继续增加，再按 auth / core / shared 组装阶段拆分。
- 当前可以暂不处理，但要避免继续膨胀。

## 已确认不是问题的事项

以下事项不再视为未完成工作：

- `core/api/user` 作为 `manage user` 的 self-scope 切片是合理设计，不需要强行独立成另一套 user service。
