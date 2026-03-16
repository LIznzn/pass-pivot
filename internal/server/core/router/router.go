package router

import (
	"net/http"

	authhandler "pass-pivot/internal/server/auth/handler"
	apiauthn "pass-pivot/internal/server/core/api/authn"
	apiauthz "pass-pivot/internal/server/core/api/authz"
	apimanage "pass-pivot/internal/server/core/api/manage"
	apisystem "pass-pivot/internal/server/core/api/system"
	apiuser "pass-pivot/internal/server/core/api/user"
	"pass-pivot/internal/server/core/middleware"
)

func NewCoreRouter(system *apisystem.Handler, manage *apimanage.Handler, user *apiuser.Handler, authn *apiauthn.Handler, authz *apiauthz.Handler, oidc *authhandler.OIDCHandler, federation *authhandler.FederationHandler, passkey *authhandler.PasskeyHandler, cors func(http.Handler) http.Handler) http.Handler {
	mux := http.NewServeMux()
	apisystem.RegisterRoutes(mux, system)
	apiauthn.RegisterRoutes(mux, system, authn, oidc, federation, passkey)
	apiauthz.RegisterRoutes(mux, authz)
	apimanage.RegisterRoutes(mux, manage, authn, authz, passkey)
	apiuser.RegisterRoutes(mux, user, authn, passkey)
	return cors(middleware.APIPolicyAuthorization(authz.Service(), middleware.APIClientAuthentication(system.Service(), oidc, mux)))
}
