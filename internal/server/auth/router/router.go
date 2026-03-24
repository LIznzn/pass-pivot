package router

import (
	"net/http"
	"strings"

	authhandler "pass-pivot/internal/server/auth/handler"
)

func NewAuthRouter(oidc *authhandler.OIDCHandler, staticAssetHandler func(string) http.HandlerFunc, cors func(http.Handler) http.Handler) http.Handler {
	mux := http.NewServeMux()
	oauth := authhandler.NewOAuthHandler(oidc.Config(), oidc.Service())
	mux.HandleFunc("GET /auth/authorize/app.js", staticAssetHandler("auth.js"))
	mux.HandleFunc("GET /auth/authorize/app.css", staticAssetHandler("auth.css"))
	mux.HandleFunc("GET /auth/authorize/assets/", authhandler.StaticAssetPrefixHandler("/auth/authorize/assets/", "assets"))
	mux.HandleFunc("GET /auth/authorize/", func(w http.ResponseWriter, r *http.Request) {
		target := "/auth/authorize"
		if rawQuery := strings.TrimSpace(r.URL.RawQuery); rawQuery != "" {
			target += "?" + rawQuery
		}
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	})
	mux.HandleFunc("GET /.well-known/openid-configuration", oidc.Metadata)
	mux.HandleFunc("GET /auth/keys", oidc.JWKS)
	mux.HandleFunc("GET /auth/authorize", oidc.Authorize)
	mux.HandleFunc("GET /auth/device", oidc.DeviceVerificationRedirect)
	mux.HandleFunc("POST /auth/device/code", oidc.DeviceAuthorization)
	mux.HandleFunc("POST /auth/api/context/query", oidc.QueryAuthorizeContextAPI)
	mux.HandleFunc("POST /auth/api/session/create", oidc.CreateAuthorizeSessionAPI)
	mux.HandleFunc("POST /auth/api/device/complete", oidc.CompleteDeviceAuthorizationAPI)
	mux.HandleFunc("POST /auth/api/session/confirm", oidc.ConfirmAuthorizeSessionAPI)
	mux.HandleFunc("POST /auth/api/session/verify_mfa", oidc.VerifyAuthorizeMFAAPI)
	mux.HandleFunc("POST /auth/api/session/mfa_challenge/create", oidc.AuthorizeChallenge)
	mux.HandleFunc("POST /auth/api/webauthn/login/begin", oidc.AuthorizeWebAuthnLoginBegin)
	mux.HandleFunc("POST /auth/api/webauthn/login/finish", oidc.AuthorizeWebAuthnLoginFinish)
	mux.HandleFunc("POST /auth/api/session/u2f/begin", oidc.AuthorizeSessionU2FBegin)
	mux.HandleFunc("POST /auth/api/session/u2f/finish", oidc.AuthorizeSessionU2FFinish)
	mux.HandleFunc("POST /auth/api/captcha/refresh", oidc.AuthorizeCaptchaRefresh)
	mux.HandleFunc("POST /auth/token", oidc.Token)
	mux.HandleFunc("GET /auth/userinfo", oidc.UserInfo)
	mux.HandleFunc("POST /auth/revoke", oauth.Revoke)
	mux.HandleFunc("POST /auth/introspect", oauth.Introspect)
	mux.HandleFunc("GET /auth/end_session", oidc.EndSession)
	return cors(mux)
}
