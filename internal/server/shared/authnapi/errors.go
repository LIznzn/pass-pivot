package authnapi

import (
	"errors"
	"net/http"
	"strings"

	sharedweb "pass-pivot/internal/server/shared/web"

	"gorm.io/gorm"
)

const (
	CodeInvalidJSONBody               = "authn.invalid_json_body"
	CodeMissingBearerToken            = "authn.missing_bearer_token"
	CodeInvalidCredentials            = "authn.invalid_credentials"
	CodeUserInactive                  = "authn.user_inactive"
	CodeOrganizationDisabled          = "authn.organization_disabled"
	CodeApplicationDisabled           = "authn.application_disabled"
	CodeApplicationAccessDenied       = "authn.application_access_denied"
	CodeGrantTypeUnsupported          = "authn.grant_type_unsupported"
	CodeGrantTypeDisabled             = "authn.grant_type_disabled"
	CodeClientAuthenticationInvalid   = "authn.client_authentication_invalid"
	CodeClientAssertionInvalid        = "authn.client_assertion_invalid"
	CodeClientIDInvalid               = "authn.client_id_invalid"
	CodeRedirectURIInvalid            = "authn.redirect_uri_invalid"
	CodePKCERequired                  = "authn.pkce_required"
	CodePKCEVerifierMismatch          = "authn.pkce_verifier_mismatch"
	CodeAuthorizationCodeInvalid      = "authn.authorization_code_invalid"
	CodeRefreshTokenInvalid           = "authn.refresh_token_invalid"
	CodeRefreshTokenExpired           = "authn.refresh_token_expired"
	CodeAccessTokenInvalid            = "authn.access_token_invalid"
	CodeSessionRequired               = "authn.session_required"
	CodeSessionNotAuthenticated       = "authn.session_not_authenticated"
	CodeConfirmationRejected          = "authn.confirmation_rejected"
	CodeSessionIDRequired             = "authn.session_id_required"
	CodeSessionNotFound               = "authn.session_not_found"
	CodeSessionStateInvalid           = "authn.session_state_invalid"
	CodeMFAMethodUnsupported          = "authn.mfa_method_unsupported"
	CodeMFATargetUnreachable          = "authn.mfa_target_unreachable"
	CodeMFAEmailNotConfigured         = "authn.mfa_email_not_configured"
	CodeMFAChallengeNotFound          = "authn.mfa_challenge_not_found"
	CodeMFAChallengeExpired           = "authn.mfa_challenge_expired"
	CodeMFAChallengeAttemptsExceeded  = "authn.mfa_challenge_attempts_exceeded"
	CodeMFACodeInvalid                = "authn.mfa_code_invalid"
	CodeTOTPEnrollmentNotFound        = "authn.totp_enrollment_not_found"
	CodeWebAuthnChallengeNotFound     = "authn.webauthn_challenge_not_found"
	CodeWebAuthnChallengeExpired      = "authn.webauthn_challenge_expired"
	CodeWebAuthnLoginDisabled         = "authn.webauthn_login_disabled"
	CodeWebAuthnUsageUnsupported      = "authn.webauthn_usage_unsupported"
	CodeUseWebAuthnCompletionEndpoint = "authn.webauthn_use_completion_endpoint"
	CodeExternalIDPStateNotFound      = "authn.external_idp_state_not_found"
	CodeExternalIDPStateExpired       = "authn.external_idp_state_expired"
	CodeExternalIDPIdentityUnbound    = "authn.external_idp_identity_unbound"
	CodeExternalIDPMissingIDToken     = "authn.external_idp_missing_id_token"
	CodeExternalIDPMissingSubject     = "authn.external_idp_missing_subject"
	CodeFIDONotConfigured             = "authn.fido_not_configured"
	CodeRuntimeNotConfigured          = "authn.runtime_not_configured"
	CodeFIDOUsageMismatch             = "authn.fido_usage_mismatch"
	CodeForbidden                     = "authn.forbidden"
	CodeResourceNotFound              = "authn.resource_not_found"
	CodeInternalError                 = "authn.internal_error"
)

type APIError struct {
	Status  int
	Code    string
	Message string
}

func (e *APIError) Error() string {
	return e.Message
}

func New(status int, code, message string) *APIError {
	return &APIError{Status: status, Code: code, Message: message}
}

func Write(w http.ResponseWriter, status int, code, message string) {
	sharedweb.ErrorWithCode(w, status, code, message)
}

func WriteKnown(w http.ResponseWriter, err error) {
	apiErr := FromError(err)
	sharedweb.ErrorWithCode(w, apiErr.Status, apiErr.Code, apiErr.Message)
}

func FromError(err error) *APIError {
	if err == nil {
		return New(http.StatusInternalServerError, CodeInternalError, "internal error")
	}
	if apiErr, ok := errors.AsType[*APIError](err); ok {
		return apiErr
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return New(http.StatusNotFound, CodeResourceNotFound, "resource not found")
	}

	message := strings.TrimSpace(err.Error())
	switch message {
	case "invalid JSON body":
		return New(http.StatusBadRequest, CodeInvalidJSONBody, message)
	case "missing bearer token":
		return New(http.StatusUnauthorized, CodeMissingBearerToken, message)
	case "invalid credentials":
		return New(http.StatusUnauthorized, CodeInvalidCredentials, message)
	case "user is not active":
		return New(http.StatusForbidden, CodeUserInactive, message)
	case "organization is disabled":
		return New(http.StatusForbidden, CodeOrganizationDisabled, message)
	case "application is disabled":
		return New(http.StatusForbidden, CodeApplicationDisabled, message)
	case "unsupported grant_type", "unsupported grant_type: use authorization_code with code_verifier for PKCE", "unsupported grant_type: OAuth requires grant_type=authorization_code":
		return New(http.StatusBadRequest, CodeGrantTypeUnsupported, message)
	case "authorization_code grant is not enabled for this application", "client_credentials grant is not enabled for this application", "password grant is not enabled for this application":
		return New(http.StatusUnauthorized, CodeGrantTypeDisabled, message)
	case "invalid client credentials", "invalid client", "unsupported client authentication method":
		return New(http.StatusUnauthorized, CodeClientAuthenticationInvalid, message)
	case "invalid client_assertion_type", "client_assertion is required", "client public key is not configured", "unsupported client assertion alg", "invalid client assertion", "invalid client assertion subject", "invalid client assertion audience", "client assertion expired":
		return New(http.StatusUnauthorized, CodeClientAssertionInvalid, message)
	case "invalid client_id":
		return New(http.StatusBadRequest, CodeClientIDInvalid, message)
	case "invalid redirect_uri", "redirect_uri mismatch":
		return New(http.StatusBadRequest, CodeRedirectURIInvalid, message)
	case "pkce is required":
		return New(http.StatusBadRequest, CodePKCERequired, message)
	case "pkce verifier mismatch", "unsupported code challenge method":
		return New(http.StatusBadRequest, CodePKCEVerifierMismatch, message)
	case "code is no longer valid":
		return New(http.StatusBadRequest, CodeAuthorizationCodeInvalid, message)
	case "invalid refresh token":
		return New(http.StatusBadRequest, CodeRefreshTokenInvalid, message)
	case "refresh token is no longer valid":
		return New(http.StatusBadRequest, CodeRefreshTokenExpired, message)
	case "token not found", "token expired or revoked":
		return New(http.StatusUnauthorized, CodeAccessTokenInvalid, message)
	case "session is required":
		return New(http.StatusBadRequest, CodeSessionRequired, message)
	case "session is not authenticated":
		return New(http.StatusUnauthorized, CodeSessionNotAuthenticated, message)
	case "user is not assigned to the target project":
		return New(http.StatusForbidden, CodeApplicationAccessDenied, message)
	case "confirmation rejected":
		return New(http.StatusConflict, CodeConfirmationRejected, message)
	case "sessionId is required":
		return New(http.StatusBadRequest, CodeSessionIDRequired, message)
	case "session is not awaiting mfa":
		return New(http.StatusConflict, CodeSessionStateInvalid, message)
	case "use WebAuthn completion endpoint for webauthn/u2f verification":
		return New(http.StatusBadRequest, CodeUseWebAuthnCompletionEndpoint, message)
	case "unsupported delivery method", "unsupported MFA method":
		return New(http.StatusBadRequest, CodeMFAMethodUnsupported, message)
	case "no reachable target for selected method":
		return New(http.StatusBadRequest, CodeMFATargetUnreachable, message)
	case "email mfa is not configured for this organization":
		return New(http.StatusBadRequest, CodeMFAEmailNotConfigured, message)
	case "TOTP enrollment expired or not found":
		return New(http.StatusNotFound, CodeTOTPEnrollmentNotFound, message)
	case "invalid TOTP code", "invalid challenge code", "invalid recovery code":
		return New(http.StatusBadRequest, CodeMFACodeInvalid, message)
	case "mfa challenge not found":
		return New(http.StatusNotFound, CodeMFAChallengeNotFound, message)
	case "MFA challenge expired":
		return New(http.StatusGone, CodeMFAChallengeExpired, message)
	case "mfa challenge max attempts exceeded":
		return New(http.StatusTooManyRequests, CodeMFAChallengeAttemptsExceeded, message)
	case "webauthn challenge not found":
		return New(http.StatusNotFound, CodeWebAuthnChallengeNotFound, message)
	case "webauthn challenge expired":
		return New(http.StatusGone, CodeWebAuthnChallengeExpired, message)
	case "webauthn login is disabled":
		return New(http.StatusForbidden, CodeWebAuthnLoginDisabled, message)
	case "unsupported assertion usage":
		return New(http.StatusBadRequest, CodeWebAuthnUsageUnsupported, message)
	case "external idp state not found":
		return New(http.StatusNotFound, CodeExternalIDPStateNotFound, message)
	case "external idp state expired":
		return New(http.StatusGone, CodeExternalIDPStateExpired, message)
	case "external identity is not bound to an existing user":
		return New(http.StatusForbidden, CodeExternalIDPIdentityUnbound, message)
	case "missing id_token from provider":
		return New(http.StatusBadGateway, CodeExternalIDPMissingIDToken, message)
	case "missing subject from provider":
		return New(http.StatusBadGateway, CodeExternalIDPMissingSubject, message)
	case "fido service is not configured":
		return New(http.StatusInternalServerError, CodeFIDONotConfigured, message)
	case "webauthn mfa runtime is not configured":
		return New(http.StatusInternalServerError, CodeRuntimeNotConfigured, message)
	case "fido assertion usage mismatch":
		return New(http.StatusBadRequest, CodeFIDOUsageMismatch, message)
	case "organization management role is required", "user context is required":
		return New(http.StatusForbidden, CodeForbidden, message)
	default:
		return New(http.StatusInternalServerError, CodeInternalError, message)
	}
}
