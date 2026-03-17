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

func NewCoreRouter(system *apisystem.Handler, manage *apimanage.Handler, user *apiuser.Handler, authn *apiauthn.Handler, authz *apiauthz.Handler, oidc *authhandler.OIDCHandler, externalIDP *authhandler.ExternalIDPHandler, webAuthn *authhandler.WebAuthnHandler, mfaU2F *authhandler.MFAU2FHandler, cors func(http.Handler) http.Handler) http.Handler {
	mux := http.NewServeMux()
	apisystem.RegisterRoutes(mux, system)
	apiauthn.RegisterRoutes(mux, system, authn, oidc, externalIDP, webAuthn, mfaU2F)
	apiauthz.RegisterRoutes(mux, authz)
	apimanage.RegisterRoutes(mux, manage, authn, authz)
	apiuser.RegisterRoutes(mux, user, authn)
	return cors(middleware.APIClientAuthentication(system.Service(), oidc, middleware.APIPolicyAuthorization(authz.Service(), mux)))
}
