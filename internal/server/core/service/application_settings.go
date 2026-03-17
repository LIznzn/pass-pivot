package service

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"pass-pivot/internal/config"
	"pass-pivot/internal/model"
)

const (
	defaultApplicationTokenIssuer           = "http://localhost:8091"
	defaultApplicationAccessTokenTTLMinutes = 10
	defaultApplicationRefreshTokenTTLHours  = 24 * 7
)

type ApplicationSettings struct {
	ApplicationID         string
	TokenIssuer           string
	AccessTokenTTLMinutes int
	RefreshTokenTTLHours  int
}

func resolveApplicationSettingsByID(ctx context.Context, db *gorm.DB, cfg config.Config, applicationID string) (ApplicationSettings, error) {
	settings := ApplicationSettings{
		ApplicationID:         applicationID,
		TokenIssuer:           cfg.AuthURL,
		AccessTokenTTLMinutes: defaultApplicationAccessTokenTTLMinutes,
		RefreshTokenTTLHours:  defaultApplicationRefreshTokenTTLHours,
	}
	if applicationID == "" {
		return settings, errors.New("applicationId is required")
	}
	var app model.Application
	if err := db.WithContext(ctx).First(&app, "id = ?", applicationID).Error; err != nil {
		return settings, err
	}
	return mergeApplicationSettings(app, settings), nil
}

func ResolveApplicationSettingsByID(ctx context.Context, db *gorm.DB, cfg config.Config, applicationID string) (ApplicationSettings, error) {
	return resolveApplicationSettingsByID(ctx, db, cfg, applicationID)
}

func resolveApplicationSettingsByClientID(ctx context.Context, db *gorm.DB, cfg config.Config, clientID string) (model.Application, ApplicationSettings, error) {
	settings := ApplicationSettings{
		TokenIssuer:           cfg.AuthURL,
		AccessTokenTTLMinutes: defaultApplicationAccessTokenTTLMinutes,
		RefreshTokenTTLHours:  defaultApplicationRefreshTokenTTLHours,
	}
	var app model.Application
	if err := db.WithContext(ctx).Where("id = ?", clientID).First(&app).Error; err != nil {
		return app, settings, err
	}
	return app, mergeApplicationSettings(app, settings), nil
}

func ResolveApplicationSettingsByClientID(ctx context.Context, db *gorm.DB, cfg config.Config, clientID string) (model.Application, ApplicationSettings, error) {
	return resolveApplicationSettingsByClientID(ctx, db, cfg, clientID)
}

func mergeApplicationSettings(app model.Application, fallback ApplicationSettings) ApplicationSettings {
	settings := fallback
	settings.ApplicationID = app.ID
	if app.AccessTokenTTLMinutes > 0 {
		settings.AccessTokenTTLMinutes = app.AccessTokenTTLMinutes
	}
	if app.RefreshTokenTTLHours > 0 {
		settings.RefreshTokenTTLHours = app.RefreshTokenTTLHours
	}
	return settings
}
