package service

import (
	"context"
	"encoding/json"
	"strings"

	"gorm.io/gorm"

	"pass-pivot/internal/model"
)

const (
	OrganizationMetadataDisplayName      = "displayName"
	OrganizationMetadataDisplayNameEN    = "displayName.en"
	OrganizationMetadataDisplayNameJA    = "displayName.ja"
	OrganizationMetadataDisplayNameZHS   = "displayName.zhs"
	OrganizationMetadataDisplayNameZHT   = "displayName.zht"
	OrganizationMetadataWebsiteURL       = "websiteUrl"
	OrganizationMetadataTermsOfServiceURL = "termsOfServiceUrl"
	OrganizationMetadataPrivacyPolicyURL = "privacyPolicyUrl"
)

func defaultOrganizationConsoleSettings() model.OrganizationSetting {
	return model.OrganizationSetting{
		SupportEmail:     "",
		LogoURL:          "",
		Domains:          []model.OrganizationDomain{},
		LoginPolicy: model.OrganizationLoginPolicy{
			PasswordLoginEnabled: true,
			WebAuthnLoginEnabled: true,
			AllowUsername:        true,
			AllowEmail:           true,
			AllowPhone:           true,
			UsernameMode:         "optional",
			EmailMode:            "required",
			PhoneMode:            "optional",
		},
		PasswordPolicy: model.OrganizationPasswordPolicy{
			MinLength:        12,
			RequireUppercase: true,
			RequireLowercase: true,
			RequireNumber:    true,
			RequireSymbol:    false,
			PasswordExpires:  false,
			ExpiryDays:       90,
		},
		MFAPolicy: model.OrganizationMFAPolicy{
			RequireForAllUsers: false,
			AllowWebAuthn:      true,
			AllowTotp:          true,
			AllowEmailCode:     true,
			AllowSmsCode:       false,
			AllowU2F:           true,
			AllowRecoveryCode:  true,
			EmailChannel: model.OrganizationEmailChannel{
				Port: 587,
			},
		},
	}
}

func normalizeOrganizationConsoleSettings(input *model.OrganizationSetting) model.OrganizationSetting {
	settings := defaultOrganizationConsoleSettings()
	if input != nil {
		settings = *input
		if settings.Domains == nil {
			settings.Domains = []model.OrganizationDomain{}
		}
		if settings.MFAPolicy.EmailChannel.Port == 0 {
			settings.MFAPolicy.EmailChannel.Port = 587
		}
	}
	return settings
}

func defaultOrganizationMetadata() map[string]string {
	return map[string]string{
		OrganizationMetadataDisplayName:      "",
		OrganizationMetadataDisplayNameEN:    "",
		OrganizationMetadataDisplayNameJA:    "",
		OrganizationMetadataDisplayNameZHS:   "",
		OrganizationMetadataDisplayNameZHT:   "",
		OrganizationMetadataWebsiteURL:       "http://example.com",
		OrganizationMetadataTermsOfServiceURL: "http://example.com/terms-of-service",
		OrganizationMetadataPrivacyPolicyURL: "http://example.com/privacy-policy",
	}
}

func NormalizeOrganizationMetadata(candidate map[string]string, fallback map[string]string) map[string]string {
	result := defaultOrganizationMetadata()
	for key, value := range fallback {
		if strings.TrimSpace(key) == "" {
			continue
		}
		result[key] = value
	}
	for key, value := range candidate {
		if strings.TrimSpace(key) == "" {
			continue
		}
		result[key] = value
	}
	return result
}

func parseLegacyOrganizationConsoleSettings(organization model.Organization) *model.OrganizationSetting {
	if organization.Metadata == nil {
		return nil
	}
	raw := organization.Metadata["console_settings"]
	if raw == "" {
		return nil
	}
	settings := defaultOrganizationConsoleSettings()
	if err := json.Unmarshal([]byte(raw), &settings); err != nil {
		return nil
	}
	return &settings
}

func loadOrganizationConsoleSettings(ctx context.Context, db *gorm.DB, organizationID string) (model.Organization, model.OrganizationSetting, error) {
	var organization model.Organization
	if err := db.WithContext(ctx).First(&organization, "id = ?", organizationID).Error; err != nil {
		return model.Organization{}, model.OrganizationSetting{}, err
	}
	organization.Metadata = NormalizeOrganizationMetadata(organization.Metadata, nil)
	legacy := parseLegacyOrganizationConsoleSettings(organization)
	if legacy != nil {
		settings := normalizeOrganizationConsoleSettings(legacy)
		return organization, settings, nil
	}
	settings := normalizeOrganizationConsoleSettings(&model.OrganizationSetting{
		SupportEmail:     organization.SupportEmail,
		LogoURL:          organization.LogoURL,
		Domains:          organization.Domains,
		LoginPolicy:      organization.LoginPolicy,
		PasswordPolicy:   organization.PasswordPolicy,
		MFAPolicy:        organization.MFAPolicy,
	})
	return organization, settings, nil
}

func LoadOrganizationConsoleSettings(ctx context.Context, db *gorm.DB, organizationID string) (model.Organization, model.OrganizationSetting, error) {
	return loadOrganizationConsoleSettings(ctx, db, organizationID)
}

func NormalizeOrganizationConsoleSettings(input *model.OrganizationSetting) model.OrganizationSetting {
	return normalizeOrganizationConsoleSettings(input)
}

func ParseLegacyOrganizationConsoleSettings(organization model.Organization) *model.OrganizationSetting {
	return parseLegacyOrganizationConsoleSettings(organization)
}
