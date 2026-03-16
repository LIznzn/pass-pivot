package authn

import (
	"net/http"

	authhandler "pass-pivot/internal/server/auth/handler"
)

func RegisterRoutes(mux *http.ServeMux, system systemHandler, authn *Handler, oidc *authhandler.OIDCHandler, federation *authhandler.FederationHandler, passkey *authhandler.PasskeyHandler) {
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
	mux.HandleFunc("POST /api/authn/v1/federation/start", federation.StartLogin)
	mux.HandleFunc("POST /api/authn/v1/federation/callback", federation.CompleteLogin)
	mux.HandleFunc("POST /api/authn/v1/passkey/login/begin", passkey.BeginLogin)
	mux.HandleFunc("POST /api/authn/v1/passkey/login/finish", passkey.FinishLogin)
	mux.HandleFunc("POST /api/authn/v1/session/u2f/begin", passkey.BeginSessionMFA)
	mux.HandleFunc("POST /api/authn/v1/session/u2f/finish", passkey.FinishSessionMFA)
}

type systemHandler interface {
	GetLoginTarget(http.ResponseWriter, *http.Request)
	ListPublicExternalIDPs(http.ResponseWriter, *http.Request)
	IntrospectToken(http.ResponseWriter, *http.Request)
}
