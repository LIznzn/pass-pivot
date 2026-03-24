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
	mux.HandleFunc("GET /auth/authorize/shared.css", staticAssetHandler("_plugin-vue_export-helper.css"))
	mux.HandleFunc("GET /auth/device/app.js", staticAssetHandler("device.js"))
	mux.HandleFunc("GET /auth/device/app.css", staticAssetHandler("device.css"))
	mux.HandleFunc("GET /auth/device/shared.css", staticAssetHandler("_plugin-vue_export-helper.css"))
	mux.HandleFunc("GET /auth/authorize/assets/", authhandler.StaticAssetPrefixHandler("/auth/authorize/assets/", "assets"))
	mux.HandleFunc("GET /auth/device/assets/", authhandler.StaticAssetPrefixHandler("/auth/device/assets/", "assets"))
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
	mux.HandleFunc("GET /auth/device", oidc.DeviceVerification)
	mux.HandleFunc("POST /auth/device/login", oidc.DeviceVerificationLogin)
	mux.HandleFunc("POST /auth/device/confirm", oidc.DeviceVerificationConfirm)
	mux.HandleFunc("POST /auth/device_authorization", oidc.DeviceAuthorization)
	mux.HandleFunc("POST /auth/headless/login", oidc.AuthorizeLogin)
	mux.HandleFunc("POST /auth/headless/login/captcha", oidc.AuthorizeCaptchaRefresh)
	mux.HandleFunc("POST /auth/headless/account", oidc.AuthorizeAccount)
	mux.HandleFunc("POST /auth/headless/confirm", oidc.AuthorizeConfirm)
	mux.HandleFunc("POST /auth/headless/mfa", oidc.AuthorizeMFA)
	mux.HandleFunc("POST /auth/headless/mfa/challenge/generator", oidc.AuthorizeChallenge)
	mux.HandleFunc("POST /auth/headless/login/webauthn/begin", oidc.AuthorizeWebAuthnLoginBegin)
	mux.HandleFunc("POST /auth/headless/login/webauthn/finish", oidc.AuthorizeWebAuthnLoginFinish)
	mux.HandleFunc("POST /auth/headless/mfa/u2f/begin", oidc.AuthorizeSessionU2FBegin)
	mux.HandleFunc("POST /auth/headless/mfa/u2f/finish", oidc.AuthorizeSessionU2FFinish)
	mux.HandleFunc("POST /auth/token", oidc.Token)
	mux.HandleFunc("GET /auth/userinfo", oidc.UserInfo)
	mux.HandleFunc("POST /auth/revoke", oauth.Revoke)
	mux.HandleFunc("POST /auth/introspect", oauth.Introspect)
	mux.HandleFunc("GET /auth/end_session", oidc.EndSession)
	return cors(mux)
}
