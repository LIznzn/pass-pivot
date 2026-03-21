# Authn API Error Codes

本文件说明 `/api/authn/v1/*` 使用的错误码约定。

响应格式：

```json
{
  "code": "authn.invalid_credentials",
  "message": "invalid credentials"
}
```

约定：

- `code`：前端应基于该字段做稳定分支判断和多语言映射。
- `message`：英文调试信息，便于开发和测试，不应作为最终用户文案来源。

## 通用请求类

| code | 含义 | 常见接口 |
| --- | --- | --- |
| `authn.invalid_json_body` | 请求体不是合法 JSON，或请求结构无法解析 | 所有 JSON POST 接口 |
| `authn.forbidden` | 当前调用方没有执行该操作的权限 | 用户管理类认证接口 |
| `authn.resource_not_found` | 后端依赖资源不存在，通常是数据库记录不存在 | 通用兜底 |
| `authn.internal_error` | 未被细分映射的服务端错误 | 通用兜底 |

## 登录与会话

| code | 含义 | 常见 message |
| --- | --- | --- |
| `authn.invalid_credentials` | 用户名/邮箱/手机号与密码不匹配 | `invalid credentials` |
| `authn.user_inactive` | 用户状态不是可登录状态 | `user is not active` |
| `authn.organization_disabled` | 用户所在组织被禁用 | `organization is disabled` |
| `authn.application_disabled` | 目标应用被禁用 | `application is disabled` |
| `authn.application_access_denied` | 用户未被授权访问目标项目/应用 | `user is not assigned to the target project` |
| `authn.confirmation_rejected` | 用户在确认步骤主动拒绝继续 | `confirmation rejected` |
| `authn.session_required` | OIDC 授权流程要求提供 session，但当前没有 | `session is required` |
| `authn.session_id_required` | 某些 MFA/U2F 流程缺少 `sessionId` | `sessionId is required` |
| `authn.session_not_authenticated` | session 存在，但尚未达到已认证状态 | `session is not authenticated` |
| `authn.session_state_invalid` | 当前 session 状态不允许执行该 MFA 动作 | `session is not awaiting mfa` |

## MFA

| code | 含义 | 常见 message |
| --- | --- | --- |
| `authn.mfa_method_unsupported` | MFA 方法不支持或当前接口不接受该方法 | `unsupported MFA method` / `unsupported delivery method` |
| `authn.mfa_target_unreachable` | 该用户没有当前方法可用的接收目标 | `no reachable target for selected method` |
| `authn.mfa_email_not_configured` | 组织未配置邮箱验证码发送能力 | `email mfa is not configured for this organization` |
| `authn.mfa_challenge_not_found` | 邮件/SMS 验证码挑战不存在 | `mfa challenge not found` |
| `authn.mfa_challenge_expired` | 邮件/SMS 验证码挑战已过期 | `MFA challenge expired` |
| `authn.mfa_code_invalid` | TOTP、邮箱码、短信码、恢复码校验失败 | `invalid TOTP code` / `invalid challenge code` / `invalid recovery code` |
| `authn.totp_enrollment_not_found` | TOTP 注册会话不存在或已过期 | `TOTP enrollment expired or not found` |

## WebAuthn / U2F / FIDO

| code | 含义 | 常见 message |
| --- | --- | --- |
| `authn.webauthn_challenge_not_found` | WebAuthn/U2F challenge 不存在 | `webauthn challenge not found` |
| `authn.webauthn_challenge_expired` | WebAuthn/U2F challenge 已过期 | `webauthn challenge expired` |
| `authn.webauthn_login_disabled` | 用户未启用 WebAuthn 登录 | `webauthn login is disabled` |
| `authn.webauthn_usage_unsupported` | WebAuthn 断言用途不支持 | `unsupported assertion usage` |
| `authn.webauthn_use_completion_endpoint` | 当前 MFA 校验不应直接提交 code，而应走 WebAuthn 完成端点 | `use WebAuthn completion endpoint for webauthn/u2f verification` |
| `authn.fido_not_configured` | 服务端未注入 FIDO 运行时 | `fido service is not configured` |
| `authn.runtime_not_configured` | 相关 WebAuthn MFA runtime 未配置 | `webauthn mfa runtime is not configured` |
| `authn.fido_usage_mismatch` | FIDO 返回的 usage 与当前流程预期不一致 | `fido assertion usage mismatch` |

## External IdP

| code | 含义 | 常见 message |
| --- | --- | --- |
| `authn.external_idp_state_not_found` | 外部登录回调 state 未找到 | `external idp state not found` |
| `authn.external_idp_state_expired` | 外部登录回调 state 已过期 | `external idp state expired` |
| `authn.external_idp_identity_unbound` | 外部身份尚未绑定本地账号 | `external identity is not bound to an existing user` |
| `authn.external_idp_missing_id_token` | 外部 OIDC provider 未返回 `id_token` | `missing id_token from provider` |
| `authn.external_idp_missing_subject` | 外部 OIDC provider 返回的用户声明缺少 `sub` | `missing subject from provider` |

## OAuth / OIDC Client 与 Token

| code | 含义 | 常见 message |
| --- | --- | --- |
| `authn.grant_type_unsupported` | 当前 `grant_type` 不支持，或调用方式不符合约定 | `unsupported grant_type` 等 |
| `authn.grant_type_disabled` | 当前应用未启用该授权类型 | `authorization_code grant is not enabled for this application` 等 |
| `authn.client_authentication_invalid` | client secret / client auth method 校验失败 | `invalid client credentials` / `invalid client` |
| `authn.client_assertion_invalid` | `client_assertion` 相关字段缺失、无效、过期或校验失败 | `invalid client_assertion_type` 等 |
| `authn.client_id_invalid` | `client_id` 非法 | `invalid client_id` |
| `authn.redirect_uri_invalid` | `redirect_uri` 非法或不匹配 | `invalid redirect_uri` / `redirect_uri mismatch` |
| `authn.pkce_required` | 当前流程要求 PKCE，但未提供 | `pkce is required` |
| `authn.pkce_verifier_mismatch` | `code_verifier` 与 challenge 不匹配，或 challenge method 不支持 | `pkce verifier mismatch` |
| `authn.authorization_code_invalid` | 授权码失效或不可再用 | `code is no longer valid` |
| `authn.refresh_token_invalid` | refresh token 非法 | `invalid refresh token` |
| `authn.refresh_token_expired` | refresh token 已失效或不再可用 | `refresh token is no longer valid` |
| `authn.access_token_invalid` | access token 不存在、已过期、已撤销或不可用 | `token not found` / `token expired or revoked` |
| `authn.missing_bearer_token` | `userinfo` 调用缺少 Bearer token | `missing bearer token` |

## 前端建议

- 前端只基于 `code` 决定展示哪条本地化文案。
- `message` 只用于开发态提示、日志或调试面板。
- 对未知 `code` 应回退到通用错误文案，避免直接把英文 `message` 暴露给最终用户。
