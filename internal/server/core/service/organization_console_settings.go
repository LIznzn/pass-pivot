package service

import (
	"context"
	"encoding/json"

	"gorm.io/gorm"

	"pass-pivot/internal/model"
)

func defaultOrganizationConsoleSettings() model.OrganizationSetting {
	return model.OrganizationSetting{
		TOSURL:           "",
		PrivacyPolicyURL: "",
		SupportEmail:     "",
		LogoURL:          "",
		Domains:          []model.OrganizationDomain{},
		LoginPolicy: model.OrganizationLoginPolicy{
			PasswordLoginEnabled: true,
			PasskeyLoginEnabled:  true,
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
			AllowPasskey:       true,
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
	legacy := parseLegacyOrganizationConsoleSettings(organization)
	if legacy != nil {
		return organization, normalizeOrganizationConsoleSettings(legacy), nil
	}
	settings := normalizeOrganizationConsoleSettings(&model.OrganizationSetting{
		TOSURL:           organization.TOSURL,
		PrivacyPolicyURL: organization.PrivacyPolicyURL,
		SupportEmail:     organization.SupportEmail,
		LogoURL:          organization.LogoURL,
		Domains:          organization.Domains,
		LoginPolicy:      organization.LoginPolicy,
		PasswordPolicy:   organization.PasswordPolicy,
		MFAPolicy:        organization.MFAPolicy,
	})
	return organization, settings, nil
}
