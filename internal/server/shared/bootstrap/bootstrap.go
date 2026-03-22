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
	sharedfido "pass-pivot/internal/server/shared/fido"
)

type App struct {
	Config config.Config
	Logger *slog.Logger
	DB     *gorm.DB
	Redis  *redis.Client
	Router http.Handler
}

type Dependencies struct {
	Audit       *coreservice.AuditService
	Manage      *apimanage.Service
	System      *apisystem.Service
	User        *apiuser.Service
	MFA         *authservice.MFAService
	Authn       *apiauthn.AuthnService
	Authz       *apiauthz.AuthzService
	OIDC        *authservice.OIDCService
	ExternalIDP *authservice.ExternalIDPService
	FIDO        *sharedfido.Service
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
	manageService.SetAuthStateCleanup(authservice.DeleteAuthorizationCodesByUser, authservice.DeleteMFAChallengesByUser)
	systemService := apisystem.NewService(manageService)
	userService := apiuser.NewService(manageService)
	keyStore := authservice.NewProviderKeyStore(database)
	oidcService := authservice.NewOIDCService(database, cfg, authAuditService, authService, keyStore)
	externalIDPService := authservice.NewExternalIDPService(database, cfg, authAuditService, authService)
	fidoService, err := sharedfido.NewService(database, cfg, func(ctx context.Context, record sharedfido.RegistrationAuditRecord) error {
		return auditService.Record(ctx, coreservice.AuditEvent{
			OrganizationID: record.OrganizationID,
			ActorType:      "user",
			ActorID:        record.UserID,
			EventType:      "user.securekey.registered",
			Result:         "success",
			TargetType:     "credential",
			TargetID:       record.CredentialID,
			Detail:         map[string]any{"purpose": record.Purpose},
		})
	})
	if err != nil {
		return nil, nil, err
	}
	authService.SetFIDOService(fidoService)
	mfaService.SetFIDOService(fidoService)
	mfaService.SetWebAuthnMFARuntime(authService)
	manageService.SetFIDOService(fidoService)

	return &App{
			Config: cfg,
			Logger: log,
			DB:     database,
			Redis:  redisClient,
		}, &Dependencies{
			Audit:       auditService,
			MFA:         mfaService,
			Authn:       authService,
			Authz:       authzService,
			OIDC:        oidcService,
			ExternalIDP: externalIDPService,
			FIDO:        fidoService,
			Manage:      manageService,
			System:      systemService,
			User:        userService,
		}, nil
}
