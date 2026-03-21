package bootstrap

import (
	"pass-pivot/internal/config"
	authhandler "pass-pivot/internal/server/auth/handler"
	authrouter "pass-pivot/internal/server/auth/router"
	sharedbootstrap "pass-pivot/internal/server/shared/bootstrap"
	sharedweb "pass-pivot/internal/server/shared/web"
)

type App = sharedbootstrap.App

func NewApp(cfg config.Config) (*App, error) {
	app, deps, err := sharedbootstrap.OpenBase(cfg)
	if err != nil {
		return nil, err
	}
	app.Router = authrouter.NewAuthRouter(
		authhandler.NewOIDCHandler(cfg, deps.OIDC, deps.Authn, deps.System),
		authhandler.StaticAssetHandler,
		sharedweb.NewCORS(app.DB, cfg),
	)
	return app, nil
}
