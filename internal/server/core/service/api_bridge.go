package service

import (
	"context"

	"gorm.io/gorm"

	"pass-pivot/internal/config"
	"pass-pivot/internal/model"
	authservice "pass-pivot/internal/server/auth/service"
)

func ResolveApplicationSettingsByID(ctx context.Context, db *gorm.DB, cfg config.Config, applicationID string) (authservice.ApplicationSettings, error) {
	return authservice.ResolveApplicationSettingsByID(ctx, db, cfg, applicationID)
}

func LoadOrganizationConsoleSettings(ctx context.Context, db *gorm.DB, organizationID string) (model.Organization, model.OrganizationSetting, error) {
	return loadOrganizationConsoleSettings(ctx, db, organizationID)
}

func TokenTypesContain(values []string, expected string) bool {
	for _, item := range values {
		if item == expected {
			return true
		}
	}
	return false
}

func AppGrantTypesContain(values []string, expected string) bool {
	for _, item := range values {
		if item == expected {
			return true
		}
	}
	return false
}

func AppTokenTypesContain(values []string, expected string) bool {
	return TokenTypesContain(values, expected)
}

func NormalizeOrganizationConsoleSettings(input *model.OrganizationSetting) model.OrganizationSetting {
	return normalizeOrganizationConsoleSettings(input)
}

func ParseLegacyOrganizationConsoleSettings(organization model.Organization) *model.OrganizationSetting {
	return parseLegacyOrganizationConsoleSettings(organization)
}

func SyncWebAuthnMFAEnrollments(ctx context.Context, db *gorm.DB, user model.User) error {
	return authservice.SyncWebAuthnMFAEnrollments(ctx, db, user)
}

func DeleteAuthorizationCodesByUser(userID string) {
	authservice.DeleteAuthorizationCodesByUser(userID)
}

func DeleteMFAChallengesByUser(userID string) {
	authservice.DeleteMFAChallengesByUser(userID)
}
