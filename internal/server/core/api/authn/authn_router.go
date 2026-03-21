package authn

import (
	"net/http"

	authhandler "pass-pivot/internal/server/auth/handler"
)

func RegisterRoutes(mux *http.ServeMux, system systemHandler, authn *Handler, oidc *authhandler.OIDCHandler, externalIDP *authhandler.ExternalIDPHandler, webAuthn *authhandler.WebAuthnHandler, mfaU2F *authhandler.MFAU2FHandler) {
	mux.HandleFunc("POST /api/authn/v1/login_target/query", system.GetLoginTarget)
	mux.HandleFunc("POST /api/authn/v1/external_idp/query", system.ListPublicExternalIDPs)
	mux.HandleFunc("POST /api/authn/v1/authorize/interaction/query", oidc.QueryAuthorizeInteractionAPI)
	mux.HandleFunc("POST /api/authn/v1/client/validate", oidc.ValidateClientAPI)
	mux.HandleFunc("POST /api/authn/v1/token/introspect", system.IntrospectToken)
	mux.HandleFunc("POST /api/authn/v1/token/exchange", oidc.ExchangeTokenAPI)
	mux.HandleFunc("POST /api/authn/v1/token/revoke", oidc.RevokeTokenAPI)
	mux.HandleFunc("POST /api/authn/v1/logout", oidc.LogoutAPI)
	mux.HandleFunc("POST /api/authn/v1/userinfo/query", oidc.QueryUserInfoAPI)
	mux.HandleFunc("POST /api/authn/v1/session/create", authn.Login)
	mux.HandleFunc("POST /api/authn/v1/session/confirm", authn.Confirm)
	mux.HandleFunc("POST /api/authn/v1/session/mfa_challenge/create", authn.CreateMFAChallenge)
	mux.HandleFunc("POST /api/authn/v1/session/verify_mfa", authn.VerifyMFA)
	mux.HandleFunc("POST /api/authn/v1/external_idp/start", externalIDP.StartLogin)
	mux.HandleFunc("POST /api/authn/v1/external_idp/callback", externalIDP.CompleteLogin)
	mux.HandleFunc("POST /api/authn/v1/webauthn/login/begin", webAuthn.BeginLogin)
	mux.HandleFunc("POST /api/authn/v1/webauthn/login/finish", webAuthn.FinishLogin)
	mux.HandleFunc("POST /api/authn/v1/session/u2f/begin", mfaU2F.BeginAssertion)
	mux.HandleFunc("POST /api/authn/v1/session/u2f/finish", mfaU2F.FinishAssertion)
	mux.HandleFunc("POST /api/authn/v1/recovery_code/query", authn.QueryRecoveryCodes)
}

type systemHandler interface {
	GetLoginTarget(http.ResponseWriter, *http.Request)
	ListPublicExternalIDPs(http.ResponseWriter, *http.Request)
	IntrospectToken(http.ResponseWriter, *http.Request)
}
