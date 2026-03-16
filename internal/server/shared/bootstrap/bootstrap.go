package bootstrap

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"pass-pivot/internal/config"
	"pass-pivot/internal/db"
	"pass-pivot/internal/logger"
	authservice "pass-pivot/internal/server/auth/service"
	apiauthn "pass-pivot/internal/server/core/api/authn"
	apiauthz "pass-pivot/internal/server/core/api/authz"
	apimanage "pass-pivot/internal/server/core/api/manage"
	apisystem "pass-pivot/internal/server/core/api/system"
	apiuser "pass-pivot/internal/server/core/api/user"
	coreservice "pass-pivot/internal/server/core/service"
)

type App struct {
	Config config.Config
	Logger *slog.Logger
	DB     *gorm.DB
	Redis  *redis.Client
	Router http.Handler
}

type Dependencies struct {
	Audit      *coreservice.AuditService
	Manage     *apimanage.Service
	System     *apisystem.Service
	User       *apiuser.Service
	MFA        *authservice.MFAService
	Authn      *apiauthn.AuthnService
	Authz      *apiauthz.AuthzService
	OIDC       *authservice.OIDCService
	Federation *authservice.FederationService
	Passkey    *authservice.PasskeyService
}

func OpenBase(cfg config.Config) (*App, *Dependencies, error) {
	log := logger.New(cfg.LogLevel)
	database, err := db.Open(
		cfg.DatabaseDriver,
		cfg.DatabaseHost,
		cfg.DatabasePort,
		cfg.DatabaseUsername,
		cfg.DatabasePassword,
		cfg.DatabaseSchema,
	)
	if err != nil {
		return nil, nil, err
	}
	redisClient, err := db.OpenRedis(context.Background(), cfg)
	if err != nil {
		return nil, nil, err
	}

	auditService := coreservice.NewAuditService(database)
	authAuditService := authservice.NewAuditService(database)
	mfaService := authservice.NewMFAService(database, cfg, authAuditService)
	authService := apiauthn.NewAuthnService(database, cfg, auditService, mfaService)
	authzService := apiauthz.NewAuthzService(database, auditService)
	manageService := apimanage.NewService(database, cfg, auditService)
	systemService := apisystem.NewService(manageService)
	userService := apiuser.NewService(manageService)
	keyStore := authservice.NewProviderKeyStore(map[string]string{
		cfg.ManageAPIApplicationID: cfg.APIManagePrivateSeed,
		cfg.UserAPIApplicationID:   cfg.APIUserPrivateSeed,
		cfg.AuthnAPIApplicationID:  cfg.APIAuthnPrivateSeed,
		cfg.AuthzAPIApplicationID:  cfg.APIAuthzPrivateSeed,
	})
	oidcService := authservice.NewOIDCService(database, cfg, authAuditService, authService, keyStore)
	federationService := authservice.NewFederationService(database, cfg, authAuditService, authService)
	passkeyService, err := authservice.NewPasskeyService(database, cfg, authAuditService, authService)
	if err != nil {
		return nil, nil, err
	}

	return &App{
			Config: cfg,
			Logger: log,
			DB:     database,
			Redis:  redisClient,
		}, &Dependencies{
			Audit:      auditService,
			MFA:        mfaService,
			Authn:      authService,
			Authz:      authzService,
			OIDC:       oidcService,
			Federation: federationService,
			Passkey:    passkeyService,
			Manage:     manageService,
			System:     systemService,
			User:       userService,
		}, nil
}
