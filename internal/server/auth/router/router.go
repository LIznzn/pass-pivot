package router

import (
	"net/http"
	"strings"

	authhandler "pass-pivot/internal/server/auth/handler"
)

func NewAuthRouter(oidc *authhandler.OIDCHandler, oauthUIAssetHandler func(string) http.HandlerFunc, cors func(http.Handler) http.Handler) http.Handler {
	mux := http.NewServeMux()
	oauth := authhandler.NewOAuthHandler(oidc.Config(), oidc.Service())
	mux.HandleFunc("GET /auth/authorize/app.js", oauthUIAssetHandler("auth.js"))
	mux.HandleFunc("GET /auth/authorize/app.css", oauthUIAssetHandler("auth.css"))
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
	mux.HandleFunc("POST /auth/headless/login", oidc.AuthorizeLogin)
	mux.HandleFunc("POST /auth/headless/confirm", oidc.AuthorizeConfirm)
	mux.HandleFunc("POST /auth/headless/mfa", oidc.AuthorizeMFA)
	mux.HandleFunc("POST /auth/headless/mfa/challenge/generator", oidc.AuthorizeChallenge)
	mux.HandleFunc("POST /auth/headless/login/passkey/begin", oidc.AuthorizePasskeyLoginBegin)
	mux.HandleFunc("POST /auth/headless/login/passkey/finish", oidc.AuthorizePasskeyLoginFinish)
	mux.HandleFunc("POST /auth/headless/mfa/u2f/begin", oidc.AuthorizeSessionPasskeyBegin)
	mux.HandleFunc("POST /auth/headless/mfa/u2f/finish", oidc.AuthorizeSessionPasskeyFinish)
	mux.HandleFunc("POST /auth/token", oidc.Token)
	mux.HandleFunc("GET /auth/userinfo", oidc.UserInfo)
	mux.HandleFunc("POST /auth/revoke", oauth.Revoke)
	mux.HandleFunc("POST /auth/introspect", oauth.Introspect)
	mux.HandleFunc("GET /auth/end_session", oidc.EndSession)
	return cors(mux)
}
