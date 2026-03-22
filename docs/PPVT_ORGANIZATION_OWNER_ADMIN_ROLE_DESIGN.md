# PPVT 共享 Console 的组织 Owner/Admin 角色方案

## 背景

当前系统的数据模型里：

- `user.organization_id` 表示用户记录从属于哪个组织
- 它不表示该用户是哪个组织的管理员
- `console` 当前希望被多个组织共用，而不是每个组织单独部署一套

这会带来一个核心问题：

- 用户甲可能是 `internal` 组织下的登录账号
- 但甲需要管理另一个业务组织 `A`
- 甲登录共享 `console` 后，应该只能看到并操作组织 `A`
- 甲不应该因为自己属于 `internal`，就默认看到 `internal`

如果继续沿用“用户属于哪个组织，就只能操作哪个组织”的思路，共享 `console` 很难成立。

因此这里提出一个折中且可渐进落地的方案：

- 保持现有 `user.organization_id` 语义不变
- 不新增组织管理员关系表
- 使用用户角色字符串来表达“某个用户对某个组织拥有什么管理权限”

角色格式如下：

- `organization:{organizationId}:owner`
- `organization:{organizationId}:admin`


## 方案目标

该方案的目标不是重构为“全局账号系统”，而是在现有模型上最小改造，实现以下能力：

1. 多个组织共用一套 `console`
2. 登录账号可以放在 `internal` 组织
3. 用户能否看到某个组织，不再由 `user.organization_id` 决定
4. 用户能否管理某个组织，由其角色中是否包含该组织的 `owner/admin` 角色决定
5. 业务组织删除时，不必删除登录账号

同时需要明确系统管理角色的实际语义：

- 共享 `console` 下的系统管理权限
  - 由 `organization:{internalOrganizationId}:owner/admin` 承担

也就是说，共享 `console` 的核心管理角色统一使用组织级 owner/admin 语义，不再额外引入 `console:admin` 这类独立角色。


## 核心设计

### 1. `user.organization_id` 的语义

`user.organization_id` 继续保持为：

- 用户记录所属组织
- 用户主数据、密码、MFA、通行密钥等认证信息的归属组织

它不再承担下面这些语义：

- 组织所有者
- 组织管理员
- 组织可见范围


### 2. 组织管理权限的表达方式

采用用户角色字符串表示组织级管理权限。

命名规则：

- `organization:{organizationId}:owner`
- `organization:{organizationId}:admin`

示例：

- `organization:8f7f8d14-e612-4d87-9c80-9847f5a6ec2d:owner`
- `organization:8f7f8d14-e612-4d87-9c80-9847f5a6ec2d:admin`

语义：

- `owner`
  - 表示组织所有者
  - 拥有该组织的最高管理权限
  - 默认也应包含 `admin` 的所有能力
- `admin`
  - 表示组织管理员
  - 可管理组织下大部分资源
  - 不一定允许做组织所有权变更

这些角色的类型归属需要明确：

- `organization:{organizationId}:owner`
- `organization:{organizationId}:admin`

本质上就是普通的用户角色

它们与 `api:*` 在“角色字符串”这个层面没有本质区别，只是语义上表达的是“该用户对某个组织具有管理权限”。

因此：

- 它们可以由组织管理逻辑自动维护
- 也可以由 `internal` 的 owner/admin 手工赋予或撤销

这里不额外增加“用户本人默认不能修改自己的这类角色”这一特殊约束。

原因是：

- 该方案里，是否允许某个用户修改某个角色，本来就应由其当前权限决定
- 如果一个用户拥有 `organization:{internalOrganizationId}:admin`
  - 那么他已经处于系统内部组织的管理权限范围内
  - 其是否能够修改自己的相关角色，应由整体权限模型自然推导
  - 不需要在本方案里人为补一条额外限制


### 3. 组织可见范围

用户登录共享 `console` 后，前端展示的组织列表不再是“所有组织”。

而应改为：

- 只显示当前登录用户角色中出现过的组织

计算方式：

1. 读取当前登录用户的全部角色
2. 解析出所有满足以下格式的角色：
   - `organization:{id}:owner`
   - `organization:{id}:admin`
3. 提取其中的 `{id}`
4. 用这些组织 ID 过滤系统中的组织列表

结果：

- 用户只有 `organization:A:owner`
  - 则只看到组织 `A`
- 用户有 `organization:A:owner` 和 `organization:B:admin`
  - 则看到组织 `A` 和 `B`
- 用户没有任何 `organization:*:*`
  - 则看不到任何组织，不能进入组织管理视图


### 4. `internal` 组织的定位

在这个方案里，`internal` 不只是“身份承载组织”，它实际上代表系统自身。

即：

- 登录账号可以创建在 `internal`
- 用户资料、密码、MFA、通行密钥等保存在 `internal`
- 但这不代表用户自动有权管理 `internal`

也就是说：

- `internal` 不是普通业务组织
- `internal` 作为程序的内部组织，实际语义就是系统本身
- 围绕 `internal` 的 owner/admin 权限，本质上就是系统管理权限

只有当用户同时拥有：

- `organization:{internalId}:owner`
  或
- `organization:{internalId}:admin`

时，才应该在共享 `console` 里看到 `internal`

这使得下面这种场景成立：

- 用户甲是 `internal` 用户
- 甲被授予 `organization:A:owner`
- 甲登录共享 `console`
- 甲只能看到组织 `A`
- 看不到 `internal`


## 权限语义

### 1. Owner 与 Admin 的关系

建议采取包含关系：

- `owner` >= `admin`

也就是说，在鉴权时：

- 如果用户拥有 `organization:{id}:owner`
  - 则视为同时拥有该组织的 `admin` 权限

这样可以减少分支判断。


### 2. 建议的能力边界

推荐约定如下：

- `organization:{id}:owner`
  - 查看组织
  - 修改组织设置
  - 管理项目、应用、用户、角色、策略、外部 IDP
  - 分配或撤销该组织的管理员
  - 转移组织 owner

- `organization:{id}:admin`
  - 查看组织
  - 修改组织设置
  - 管理项目、应用、用户、角色、策略、外部 IDP
  - 不能变更 owner
  - 不能删除或移交组织

当前阶段如果不实现 owner 转移，也可以先让二者行为一致，只保留语义区分。


## 后端设计

### 1. 当前问题

当前很多接口是按以下思路工作的：

- 根据 `organizationId` 查询数据
- 默认认为调用方有资格访问这个组织

在共享 `console` 下，这不够安全。

因为只要前端把某个 `organizationId` 传进来，后端如果不校验，就可能越权访问其他组织数据。


### 2. 需要新增的后端能力

后端需要补一个统一的组织管理权限判断逻辑。

建议新增辅助函数：

- `hasOrganizationScopedRole(userRoles []string, organizationID string, scope string) bool`
- `canManageOrganization(userRoles []string, organizationID string) bool`
- `canOwnOrganization(userRoles []string, organizationID string) bool`

建议语义：

- `canManageOrganization`
  - 命中 `organization:{id}:admin`
  - 或命中 `organization:{id}:owner`

- `canOwnOrganization`
  - 仅命中 `organization:{id}:owner`


### 3. 推荐增加的接口层校验

所有以 `organizationId` 为作用域的管理接口，都应增加校验：

- 当前登录用户是否拥有该组织的 `owner/admin` 角色

优先级最高的接口包括：

- 组织列表查询
- 项目列表查询
- 应用列表查询
- 用户列表查询
- 角色列表查询
- 策略列表查询
- 外部 IDP 列表查询
- 审计日志查询
- 对应的创建、更新、删除接口


### 4. 组织列表查询的返回策略

当前 `ListOrganizations` 返回所有组织。

共享 `console` 下建议改为：

- 只返回当前登录用户拥有 `organization:{id}:owner/admin` 的组织

这一步非常关键，因为它决定了前端默认组织切换器的数据来源。

这里需要明确一条核心原则：

- 本系统不存在绕过组织作用域的全局特权角色
- 即使是 `organization:{internalOrganizationId}:owner`
  或 `organization:{internalOrganizationId}:admin`
  - 也只能直接管理 `internal`
  - 不能天然访问全部组织

如果某个 `internal` 管理员后来具备了多个组织的管理能力，那也必须是因为他被显式授予了这些组织各自的：

- `organization:{targetOrganizationId}:owner`
  或
- `organization:{targetOrganizationId}:admin`

因此，“类似超级管理员的能力”不是通过系统内建特权获得，而是通过正常的角色分配链路逐步形成。


### 5. 登录态中的角色来源

该方案依赖“后端能拿到当前登录用户的角色”。

因此需要确保：

- access token 所绑定的用户上下文里，包含完整的 `user.roles`
- 管理接口在鉴权时，能够从上下文中读取到这些角色

如果当前某些接口只依赖 `organizationId` 参数而不读取当前用户上下文，就必须补齐。


## 前端设计

### 1. 组织切换器

前端组织切换器不能再展示后端返回的全部组织。

应该只展示：

- 当前用户被授权可管理的组织

如果后端已经过滤过，则前端直接展示即可。
如果后端未过滤，则前端需要按角色再次过滤。

推荐做法：

- 以后端过滤为主
- 前端只做兜底过滤


### 2. 默认进入组织

登录进入 `console` 后：

- 如果可见组织只有一个
  - 自动进入该组织
- 如果可见组织有多个
  - 默认进入第一个
  - 用户可通过顶部切换器切换
- 如果可见组织为空
  - 显示“当前账号未被授予任何组织管理权限”


### 3. 页面行为

所有页面里当前组织上下文都应来自：

- 当前选择的组织 ID

而不应来自：

- `profileForm.organizationId`
- 当前登录用户所属组织

这意味着：

- 项目、应用、用户、角色、策略、外部 IDP、设置页
  都必须绑定到“当前选中的组织”


## 角色分配策略

### 1. 创建组织时

建议默认行为：

- 创建新组织时，给创建者自动加一条角色：
  - `organization:{newOrganizationId}:owner`

这样创建者创建完组织后，立刻就能在共享 `console` 里看到并管理这个组织。


### 2. 增加组织管理员

可以在组织详情页增加“组织管理员”配置入口。

底层实现是给目标用户追加角色：

- `organization:{organizationId}:admin`

如果要转移所有者：

- 先给新所有者加 `organization:{organizationId}:owner`
- 再移除旧所有者的 `organization:{organizationId}:owner`


### 3. 删除组织时的清理

如果未来仍保留组织删除能力，则删除组织后还应清理所有用户上的动态角色：

- `organization:{organizationId}:owner`
- `organization:{organizationId}:admin`

否则会残留无效角色字符串。


## 兼容性与风险

### 1. 这是“动态角色编码”方案，不是严格的关系模型

该方案的优点是改造小、上手快。

但要清楚它的本质：

- 它把“组织管理员关系”编码进了用户角色字符串
- 不是标准的组织成员/管理员关系表

所以它更适合：

- 快速验证共享 `console` 逻辑
- 渐进演进阶段

不一定适合作为最终形态。


### 2. 主要风险

#### 2.1 组织删除后的脏角色

如果删除组织，不同步清理用户角色，会留下：

- `organization:deleted-id:owner`
- `organization:deleted-id:admin`

解决方式：

- 删除组织时批量扫描相关用户，移除对应角色


#### 2.2 角色名膨胀

每创建一个组织，都会产生新的动态角色前缀。

如果组织很多，用户角色数组会越来越长。

短期可接受，长期可能需要关系表替代。


#### 2.3 角色页中的组织角色展示

`organization:{id}:owner/admin` 不应被视为特殊角色或“伪角色”。

它们就是普通的用户角色，只是命名中包含组织作用域。

因此文档在这里明确调整为：

- `organization:{id}:owner`
- `organization:{id}:admin`

应被当作普通用户角色看待。

它们可以：

- 由组织创建流程自动生成或自动赋予
- 由组织管理逻辑自动维护
- 由 `internal` 的 owner/admin 在角色管理或用户分配流程中手工修改

不需要对“是否允许在角色页创建这类角色”做特殊限制。

真正需要限制的不是“角色是不是普通角色”，而是“谁有权给谁分配这种角色”。

也就是说：

- 这些角色本身是普通用户角色
- 但其分配权限应只属于有更高管理能力的管理员


#### 2.4 后端必须做强校验

不能只依赖前端隐藏组织。

否则用户仍可能通过构造请求访问其他组织。

所以必须在后端按当前登录用户角色做组织级权限校验。


## 与关系表方案的比较

### 本方案优点

- 改造小
- 不新增表
- 可快速验证共享 `console`
- 复用现有用户 `roles` 字段

### 本方案缺点

- 角色字符串承载关系语义，不够干净
- 删除组织后要专门清理动态角色
- 不利于后续做复杂组织成员治理

### 关系表方案优点

- 数据结构更清晰
- 易查询、易约束、易扩展
- 更适合长期演进

### 关系表方案缺点

- 改造面更大
- 需要修改更多后端和前端逻辑


## 推荐落地顺序

建议按以下顺序分阶段实施。

### 第一阶段：只做共享 Console 可见范围

1. 约定角色格式
2. 创建组织时自动给创建者赋 `organization:{id}:owner`
3. 组织列表接口按当前用户角色过滤
4. 前端组织切换器只显示被授权组织

目标：

- 用户登录共享 `console` 后，只看到自己能管的组织


### 第二阶段：补组织级接口鉴权

1. 给各管理接口增加组织访问校验
2. 区分 `owner` 与 `admin` 能力边界

目标：

- 不能通过构造请求越权访问其他组织


### 第三阶段：补组织管理员管理能力

1. 增加组织管理员配置 UI
2. 支持给用户添加/移除 `organization:{id}:admin`
3. 可选支持 owner 转移

目标：

- 一个组织可以有多个管理员


### 第四阶段：评估是否升级为关系表

如果后续出现以下需求，再考虑升级为关系表方案：

- 组织管理员数量很多
- 需要审计组织管理员变更历史
- 需要组织成员状态、邀请、审批
- 需要跨组织复杂权限管理


## 总结

`organization:{id}:owner` / `organization:{id}:admin` 方案，是一个适合当前系统阶段的共享 `console` 折中方案。

它保留了现有模型的主体结构：

- 用户仍从属于某个组织
- 登录账号可以继续放在 `internal`

同时又新增了一个更准确的权限维度：

- 用户能管理哪些组织，不由 `user.organization_id` 决定
- 而由其动态组织管理角色决定

这使得共享 `console` 可以成立，并且不需要立即引入全局账号或组织管理员关系表。

同时还要补充一条关键定位：

- 共享 `console` 的系统管理入口
  - 实际由 `organization:{internalOrganizationId}:owner/admin` 控制

但必须明确：

- 这是一种“以角色编码关系”的过渡方案
- 本系统不存在绕过组织作用域的全局特权角色
- 要想安全可用，后端必须补齐组织级权限校验
- 要想长期可维护，后续仍可能需要升级为关系表设计
