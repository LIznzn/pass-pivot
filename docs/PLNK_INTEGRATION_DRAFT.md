# 付轨（PLNK）接入草稿

## 1. 基本信息

- 系统名称：付轨
- 英文名：PayLink
- 系统缩写：PLNK
- 接入方式：通过 `/.well-known/ppvt-iam.json` 向 PPVT 上报能力与权限

## 2. 权限定义原则

- 权限由 PLNK 业务系统定义
- PPVT 负责校验、审批、分配与审计
- 权限区分为系统级权限与用户级权限

## 3. 权限清单（当前版本）

### 3.1 系统级权限（Client Permissions）

- `order:create`
- `order:cancel`
- `order:query`
- `order:subscribe`

### 3.2 用户级权限（User Permissions）

- `balance:query`
- `order:cancel`
- `order:query`
- `order:subscribe`
- `transaction:query`

## 4. 自发现配置示例

```json
{
  "schema_version": "1.0",
  "system_id": "plnk",
  "name": "付轨",
  "alias": "PayLink",
  "permissions": {
    "client_permissions": [
      "order:create",
      "order:cancel",
      "order:query",
      "order:subscribe"
    ],
    "user_permissions": [
      "balance:query",
      "order:cancel",
      "order:query",
      "order:subscribe",
      "transaction:query"
    ]
  }
}
```

## 5. 后续待完善

1. 各权限的业务语义与边界说明
2. 权限对应 API 列表与方法范围
3. 默认角色映射策略（如运营、客服、财务）
4. 敏感权限的二次确认与 MFA 要求
5. 版本变更策略（新增/废弃权限）

