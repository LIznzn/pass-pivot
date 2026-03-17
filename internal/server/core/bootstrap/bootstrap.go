package bootstrap

import (
	"pass-pivot/internal/config"
	authhandler "pass-pivot/internal/server/auth/handler"
	apiauthn "pass-pivot/internal/server/core/api/authn"
	apiauthz "pass-pivot/internal/server/core/api/authz"
	apimanage "pass-pivot/internal/server/core/api/manage"
	apisystem "pass-pivot/internal/server/core/api/system"
	apiuser "pass-pivot/internal/server/core/api/user"
	corerouter "pass-pivot/internal/server/core/router"
	sharedbootstrap "pass-pivot/internal/server/shared/bootstrap"
	sharedweb "pass-pivot/internal/server/shared/web"
)

type App = sharedbootstrap.App

func NewApp(cfg config.Config) (*App, error) {
	app, deps, err := sharedbootstrap.OpenBase(cfg)
	if err != nil {
		return nil, err
	}
	app.Router = corerouter.NewCoreRouter(
		apisystem.NewHandler(deps.System),
		apimanage.NewHandler(deps.Manage),
		apiuser.NewHandler(deps.User),
		apiauthn.NewHandler(deps.Authn),
		apiauthz.NewHandler(deps.Authz),
		authhandler.NewOIDCHandler(cfg, deps.OIDC, deps.Authn, deps.System),
		authhandler.NewExternalIDPHandler(deps.ExternalIDP),
		authhandler.NewWebAuthnHandler(deps.Authn),
		authhandler.NewMFAU2FHandler(deps.MFA),
		sharedweb.WithCORS,
	)
	return app, nil
}
