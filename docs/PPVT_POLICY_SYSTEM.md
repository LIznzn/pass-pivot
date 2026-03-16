# PPVT Policy System

本文档描述 PPVT 当前确认的角色与策略系统需求定义。  
该文档是后续实现的直接依据。

## 1. 系统定位

PPVT 在访问控制层面的职责是“策略数据平台”。

这意味着：

- PPVT 负责存储角色、策略、策略优先级、API 规则
- PPVT 可以提供 `Policy Check` 裁决接口
- 网关或第三方中间件也可以不调用 `Policy Check`
- 外部系统可以直接拉取策略数据并自行执行访问控制判断

所以 PPVT 必须同时满足两件事：

- 自身可以完成官方裁决
- 外部系统可以直接消费同一套策略数据

## 2. 核心模型

### 2.1 Role

角色是策略组。

角色：

- 属于 `Organization`
- 是一组策略的容器
- 可以赋予用户或应用
- 必须区分角色类型

角色类型：

- `user`
- `application`

约束：

- 同一组织内角色名唯一
- 用户只能分配 `type = user` 的角色
- 应用只能分配 `type = application` 的角色

### 2.2 Policy

策略是最小授权单元。

策略：

- 属于 `Organization`
- 挂在角色下
- 定义名称、效果、优先级、API 规则

一个策略包含：

- `name`
- `effect`
- `priority`
- `api_rules`

### 2.3 User

用户表中的角色字段：

- `user.roles`
- JSON 数组
- 内容存角色名，不存角色 ID

### 2.4 Application

应用表中的角色字段：

- `application.roles`
- JSON 数组
- 内容存角色名，不存角色 ID

同时：

- 废弃 `api_capabilities`

## 3. Policy 字段语义

### 3.1 name

`name` 是策略名。

它本质上是普通字符串字段，数据库不做格式限制。  
推荐命名规范可以使用：

- `user:profile:read`
- `user:profile:write`
- `manage:user:read`
- `manage:user:write`

但这只是约定，不是强制规则。

### 3.2 effect

支持：

- `allow`
- `deny`

### 3.3 priority

`priority` 用于裁决顺序。

规则：

- 数值越小，优先级越高

说明：

- 不要求连续
- 不要求必须从 1 开始
- `10 / 20 / 30 / 40` 是推荐习惯，不是系统限制

### 3.4 api_rules

`api_rules` 直接存储在 `policy` 中，不拆分独立表。

建议格式：

```json
[
  { "method": "POST", "path": "/api/user/v1/profile/query" },
  { "method": "POST", "path": "/api/user/v1/profile/update" }
]
```

规则：

- `method` 表示 HTTP 方法
- `path` 表示路径匹配规则
- `path` 需要支持 `keyMatch2`

## 4. 策略语义

### 4.1 应用侧策略

应用侧策略只用于判断：

- 某个应用是否有资格访问某类 API 或另一个应用

应用侧裁决只看：

- `application.roles`

不看：

- 用户侧策略

结论：

- 如果应用侧不允许，即使用户侧允许，也不能访问

### 4.2 用户侧策略

用户侧策略只用于判断：

- 用户是否有资格在某个应用内执行某个动作

用户侧裁决只看：

- `user.roles`

不看：

- 应用侧是否允许某个用户动作

### 4.3 执行顺序

完整访问流程应理解为两层门禁：

1. 先检查应用是否允许进入该 API / 应用能力范围
2. 应用允许后，再检查用户是否允许执行当前动作

任一层不通过，则最终拒绝。

这里不是“应用 + 用户”合成一个统一主体做一次联合裁决，  
而是两个独立裁决按顺序执行。

## 5. Casbin 集成

Casbin 可以作为 PPVT 的官方裁决引擎。

但 PPVT 的核心仍然是：

- 角色和策略的数据平台
- 而不是只能依赖 Casbin 的黑盒系统

### 5.1 Casbin 使用目标

PPVT 内部通过 Casbin 完成：

- 角色关系装载
- 策略规则装载
- 基于优先级的最终 `allow / deny` 裁决

同时保留：

- 角色 / 策略 / API 规则数据可被外部系统直接读取

### 5.2 主体命名

为了避免用户和应用 ID 冲突，统一命名：

- 用户主体：`user:<userID>`
- 应用主体：`application:<applicationID>`

### 5.3 角色关系

Casbin `g` 关系：

- `g(user:<userID>, role:<roleName>)`
- `g(application:<applicationID>, role:<roleName>)`

### 5.4 策略规则展开

每条 policy 中的每个 `api_rule` 展开为一条 Casbin `p` 规则。

例如：

- role = `role:api-user`
- policy.name = `user:profile:read`
- effect = `allow`
- priority = `10`
- api_rules = `POST /api/user/v1/profile/query`

展开为：

- `p(10, role:api-user, /api/user/v1/profile/query, POST, allow, user:profile:read)`

### 5.5 Casbin 模型建议

```conf
[request_definition]
r = sub, obj, act

[policy_definition]
p = priority, sub, obj, act, eft, name

[role_definition]
g = _, _

[policy_effect]
e = priority(p.eft) || deny

[matchers]
m = g(r.sub, p.sub) && keyMatch2(r.obj, p.obj) && regexMatch(r.act, p.act)
```

### 5.6 Casbin 边界

Casbin 在 PPVT 中的职责：

- 对单一主体做最终裁决
- 基于优先级选择生效策略

PPVT 在 Casbin 之外的职责：

- 装载数据库角色与策略
- 区分 `user` / `application`
- 提供策略查询能力
- 对外返回解释性结果

## 6. Policy Check

原“Decision Check”统一更名为：

- `Policy Check`

用途：

- 控制台调试
- 官方裁决接口
- 网关或中间件的可选调用接口

但：

- 不是唯一消费方式

外部系统也可以直接拉取策略后自行裁决。

### 6.1 输入建议

- `subjectType`
  - `user`
  - `application`
- `subjectId`
- `method`
- `path`

### 6.2 输出建议

- `allowed`
- `matchedRole`
- `matchedPolicy`
- `matchedPolicyName`
- `matchedEffect`
- `matchedPriority`
- `reason`

说明：

- 一次 `Policy Check` 只判断一个主体
- 不做用户和应用的联合单次判定

## 7. 控制台需求

### 7.1 角色管理

支持：

- 查询角色列表
- 创建角色
- 编辑角色
- 删除角色
- 指定角色类型：
  - `user`
  - `application`

### 7.2 策略管理

支持：

- 查看角色下策略
- 创建策略
- 编辑策略
- 删除策略

可编辑字段：

- `name`
- `effect`
- `priority`
- `api_rules`

### 7.3 用户角色分配

只能展示并分配：

- `type = user`

### 7.4 应用角色分配

只能展示并分配：

- `type = application`

### 7.5 Policy Check

控制台支持输入主体、方法、路径，查看裁决结果。

## 8. 数据库调整目标

### 8.1 application

新增：

- `roles` JSON 数组

废弃：

- `api_capabilities`

### 8.2 role

新增或保留：

- `type`

### 8.3 policy

最终至少包含：

- `id`
- `organization_id`
- `role_id`
- `name`
- `effect`
- `priority`
- `api_rules`

## 9. 最终约束

- 角色是策略组
- 策略属于角色
- 角色和策略都按组织隔离
- 用户角色与应用角色分离
- 用户只能拥有 `user` 角色
- 应用只能拥有 `application` 角色
- 应用角色存储在 `application.roles`
- 数组中保存角色名，不保存角色 ID
- `api_capabilities` 废弃
- 策略名只是普通名称字段，不做格式强限制
- `priority` 越小越优先
- `api_rules.path` 支持 `keyMatch2`
- PPVT 可以官方裁决，也可以只作为数据平台
