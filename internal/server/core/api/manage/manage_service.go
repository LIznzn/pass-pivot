package manage

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"pass-pivot/internal/config"
	"pass-pivot/internal/model"
	coreservice "pass-pivot/internal/server/core/service"
	sharedfido "pass-pivot/internal/server/shared/fido"
	"pass-pivot/util"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service struct {
	db                             *gorm.DB
	cfg                            config.Config
	audit                          *coreservice.AuditService
	geoip                          *coreservice.GeoIPService
	fido                           fidoRegistrationService
	deleteAuthorizationCodesByUser func(string)
	deleteMFAChallengesByUser      func(string)
}

type fidoRegistrationService interface {
	BeginRegistration(ctx context.Context, userID, purpose string) (string, any, error)
	BeginRegistrationForSession(ctx context.Context, sessionID, purpose string) (string, any, error)
	FinishRegistration(ctx context.Context, challengeID string, payload json.RawMessage) error
}

func applicationIsDisabled(status string) bool {
	return strings.TrimSpace(status) == "disabled"
}

func projectIsDisabled(status string) bool {
	return strings.TrimSpace(status) == "disabled"
}

func organizationIsDisabled(status string) bool {
	return strings.TrimSpace(status) == "disabled"
}

var (
	applicationTypeOptions = map[string]bool{
		"web":    true,
		"native": true,
		"api":    true,
	}
	grantTypeOptions = map[string]bool{
		// These are stored as internal application policy labels.
		// External OAuth requests still use standard grant_type values.
		"authorization_code":      true,
		"authorization_code_pkce": true,
		"client_credentials":      true,
		"device_code":             true,
		"implicit":                true,
		"password":                true,
	}
	clientAuthenticationTypeOptions = map[string]bool{
		"none":                        true,
		"client_secret_basic":         true,
		"client_secret_post":          true,
		"client_secret_jwt":           true,
		"private_key_jwt":             true,
		"tls_client_auth":             true,
		"self_signed_tls_client_auth": true,
	}
	tokenTypeOptions = map[string]bool{
		"access_token": true,
		"id_token":     true,
	}
	roleTypeOptions = map[string]bool{
		"user":        true,
		"application": true,
	}
)

func NewService(db *gorm.DB, cfg config.Config, audit *coreservice.AuditService) *Service {
	return &Service{
		db:    db,
		cfg:   cfg,
		audit: audit,
		geoip: coreservice.NewGeoIPService("external/ip/GeoLite2-City.mmdb"),
	}
}

func (s *Service) SetFIDOService(fido fidoRegistrationService) {
	s.fido = fido
}

func (s *Service) SetAuthStateCleanup(deleteAuthorizationCodesByUser, deleteMFAChallengesByUser func(string)) {
	s.deleteAuthorizationCodesByUser = deleteAuthorizationCodesByUser
	s.deleteMFAChallengesByUser = deleteMFAChallengesByUser
}

func (s *Service) BeginUserSecureKeyRegistration(ctx context.Context, userID, purpose string) (string, any, error) {
	if s.fido == nil {
		return "", nil, errors.New("fido service is not configured")
	}
	if strings.TrimSpace(userID) == "" {
		return "", nil, errors.New("userId is required")
	}
	return s.fido.BeginRegistration(ctx, userID, purpose)
}

func (s *Service) BeginCurrentUserSecureKeyRegistration(ctx context.Context, sessionID, purpose string) (string, any, error) {
	if s.fido == nil {
		return "", nil, errors.New("fido service is not configured")
	}
	return s.fido.BeginRegistrationForSession(ctx, sessionID, purpose)
}

func (s *Service) FinishSecureKeyRegistration(ctx context.Context, challengeID string, payload json.RawMessage) error {
	if s.fido == nil {
		return errors.New("fido service is not configured")
	}
	if strings.TrimSpace(challengeID) == "" {
		return errors.New("challengeId is required")
	}
	return s.fido.FinishRegistration(ctx, challengeID, payload)
}

func (s *Service) ListOrganizations(ctx context.Context) ([]model.Organization, error) {
	var items []model.Organization
	if err := s.db.WithContext(ctx).Preload("Projects.Applications").Find(&items).Error; err != nil {
		return nil, err
	}
	if err := s.attachOrganizationSettings(ctx, items); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Service) GetLoginTarget(ctx context.Context, applicationID string) (*coreservice.LoginTarget, error) {
	if applicationID == "" {
		return nil, errors.New("applicationId is required")
	}
	var application model.Application
	if err := s.db.WithContext(ctx).First(&application, "id = ?", applicationID).Error; err != nil {
		return nil, err
	}
	if applicationIsDisabled(application.Status) {
		return nil, errors.New("application is disabled")
	}
	var project model.Project
	if err := s.db.WithContext(ctx).First(&project, "id = ?", application.ProjectID).Error; err != nil {
		return nil, err
	}
	if projectIsDisabled(project.Status) {
		return nil, errors.New("project is disabled")
	}
	var organization model.Organization
	if err := s.db.WithContext(ctx).First(&organization, "id = ?", project.OrganizationID).Error; err != nil {
		return nil, err
	}
	if organizationIsDisabled(organization.Status) {
		return nil, errors.New("organization is disabled")
	}
	providers, err := s.ListExternalIDPs(ctx, organization.ID)
	if err != nil {
		return nil, err
	}
	publicProviders := make([]coreservice.PublicExternalIDP, 0, len(providers))
	for _, item := range providers {
		publicProviders = append(publicProviders, coreservice.PublicExternalIDP{
			ID:             item.ID,
			OrganizationID: item.OrganizationID,
			Protocol:       item.Protocol,
			Name:           item.Name,
			Issuer:         item.Issuer,
		})
	}
	return &coreservice.LoginTarget{
		OrganizationID:   organization.ID,
		OrganizationName: organization.Name,
		ProjectID:        project.ID,
		ProjectName:      project.Name,
		ApplicationID:    application.ID,
		ApplicationName:  application.Name,
		ExternalIDPs:     publicProviders,
	}, nil
}

func (s *Service) ListPublicExternalIDPsByApplication(ctx context.Context, applicationID string) ([]model.ExternalIDP, error) {
	if applicationID == "" {
		return nil, errors.New("applicationId is required")
	}
	var application model.Application
	if err := s.db.WithContext(ctx).First(&application, "id = ?", applicationID).Error; err != nil {
		return nil, err
	}
	if applicationIsDisabled(application.Status) {
		return nil, errors.New("application is disabled")
	}
	var project model.Project
	if err := s.db.WithContext(ctx).First(&project, "id = ?", application.ProjectID).Error; err != nil {
		return nil, err
	}
	if projectIsDisabled(project.Status) {
		return nil, errors.New("project is disabled")
	}
	var organization model.Organization
	if err := s.db.WithContext(ctx).First(&organization, "id = ?", project.OrganizationID).Error; err != nil {
		return nil, err
	}
	if organizationIsDisabled(organization.Status) {
		return nil, errors.New("organization is disabled")
	}
	return s.ListExternalIDPs(ctx, project.OrganizationID)
}

func (s *Service) CreateOrganization(ctx context.Context, org model.Organization) (*model.Organization, error) {
	if org.Name == "" {
		return nil, errors.New("name is required")
	}
	if strings.TrimSpace(org.Status) == "" {
		org.Status = "active"
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&org).Error; err != nil {
			return err
		}
		if org.ConsoleSettings == nil {
			return nil
		}
		settings := coreservice.NormalizeOrganizationConsoleSettings(org.ConsoleSettings)
		return tx.Model(&org).Updates(map[string]any{
			"tos_url":            settings.TOSURL,
			"privacy_policy_url": settings.PrivacyPolicyURL,
			"support_email":      settings.SupportEmail,
			"logo_url":           settings.LogoURL,
			"domains":            settings.Domains,
			"login_policy":       settings.LoginPolicy,
			"password_policy":    settings.PasswordPolicy,
			"mfa_policy":         settings.MFAPolicy,
		}).Error
	}); err != nil {
		return nil, err
	}
	settings := coreservice.NormalizeOrganizationConsoleSettings(org.ConsoleSettings)
	org.ConsoleSettings = &settings
	_ = s.audit.Record(ctx, coreservice.AuditEvent{
		OrganizationID: org.ID,
		ActorType:      "admin",
		EventType:      "organization.created",
		Result:         "success",
		TargetType:     "organization",
		TargetID:       org.ID,
		Detail:         map[string]any{"name": org.Name},
	})
	return &org, nil
}

func (s *Service) UpdateOrganization(ctx context.Context, org model.Organization) (*model.Organization, error) {
	if org.ID == "" {
		return nil, errors.New("id is required")
	}
	var existing model.Organization
	if err := s.db.WithContext(ctx).First(&existing, "id = ?", org.ID).Error; err != nil {
		return nil, err
	}
	metadata := normalizeMetadata(org.Metadata, existing.Metadata)
	delete(metadata, "console_settings")
	updates := map[string]any{
		"name":                 coalesceString(org.Name, existing.Name),
		"metadata":             metadata,
		"allow_jwt_access":     org.AllowJWTAccess,
		"allow_basic_access":   org.AllowBasicAccess,
		"allow_no_auth_access": org.AllowNoAuthAccess,
		"allow_refresh_token":  org.AllowRefreshToken,
		"allow_auth_code":      org.AllowAuthCode,
		"allow_pkce":           org.AllowPKCE,
	}
	if org.ConsoleSettings != nil {
		settings := coreservice.NormalizeOrganizationConsoleSettings(org.ConsoleSettings)
		updates["tos_url"] = settings.TOSURL
		updates["privacy_policy_url"] = settings.PrivacyPolicyURL
		updates["support_email"] = settings.SupportEmail
		updates["logo_url"] = settings.LogoURL
		updates["domains"] = settings.Domains
		updates["login_policy"] = settings.LoginPolicy
		updates["password_policy"] = settings.PasswordPolicy
		updates["mfa_policy"] = settings.MFAPolicy
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Model(&existing).Updates(updates).Error
	}); err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).First(&existing, "id = ?", org.ID).Error; err != nil {
		return nil, err
	}
	settings, err := s.getOrganizationSetting(ctx, existing.ID)
	if err != nil {
		return nil, err
	}
	existing.ConsoleSettings = &settings
	return &existing, nil
}

func (s *Service) DisableOrganization(ctx context.Context, organizationID string) error {
	if strings.TrimSpace(organizationID) == "" {
		return errors.New("organizationId is required")
	}
	var organization model.Organization
	if err := s.db.WithContext(ctx).First(&organization, "id = ?", organizationID).Error; err != nil {
		return err
	}
	now := time.Now()
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&organization).Update("status", "disabled").Error; err != nil {
			return err
		}
		var appIDs []string
		if err := tx.Model(&model.Application{}).
			Joins("JOIN project ON project.id = application.project_id").
			Where("project.organization_id = ?", organization.ID).
			Pluck("application.id", &appIDs).Error; err != nil {
			return err
		}
		if len(appIDs) > 0 {
			if err := tx.Model(&model.Token{}).
				Where("application_id IN ? AND revoked_at IS NULL", appIDs).
				Updates(map[string]any{"revoked_at": now, "revocation_note": "organization_disabled"}).Error; err != nil {
				return err
			}
			if err := tx.Where("application_id IN ?", appIDs).Delete(&model.AuthorizationCode{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("organization_id = ?", organization.ID).Delete(&model.Session{}).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return s.audit.Record(ctx, coreservice.AuditEvent{
		OrganizationID: organization.ID,
		ActorType:      "admin",
		EventType:      "organization.disabled",
		Result:         "success",
		TargetType:     "organization",
		TargetID:       organization.ID,
	})
}

func (s *Service) DeleteOrganization(ctx context.Context, organizationID string) error {
	if strings.TrimSpace(organizationID) == "" {
		return errors.New("organizationId is required")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var organization model.Organization
		if err := tx.First(&organization, "id = ?", organizationID).Error; err != nil {
			return err
		}

		var projectIDs []string
		if err := tx.Model(&model.Project{}).Where("organization_id = ?", organization.ID).Pluck("id", &projectIDs).Error; err != nil {
			return err
		}
		var appIDs []string
		if len(projectIDs) > 0 {
			if err := tx.Model(&model.Application{}).Where("project_id IN ?", projectIDs).Pluck("id", &appIDs).Error; err != nil {
				return err
			}
		}
		var userIDs []string
		if err := tx.Model(&model.User{}).Where("organization_id = ?", organization.ID).Pluck("id", &userIDs).Error; err != nil {
			return err
		}
		var externalIDPIDs []string
		if err := tx.Model(&model.ExternalIDP{}).Where("organization_id = ?", organization.ID).Pluck("id", &externalIDPIDs).Error; err != nil {
			return err
		}

		if len(appIDs) > 0 {
			if err := tx.Where("application_id IN ?", appIDs).Delete(&model.AuthorizationCode{}).Error; err != nil {
				return err
			}
			if err := tx.Where("application_id IN ?", appIDs).Delete(&model.Token{}).Error; err != nil {
				return err
			}
		}
		if len(userIDs) > 0 {
			if err := tx.Where("user_id IN ?", userIDs).Delete(&model.AuthorizationCode{}).Error; err != nil {
				return err
			}
			if err := tx.Where("user_id IN ?", userIDs).Delete(&model.Token{}).Error; err != nil {
				return err
			}
			if err := tx.Where("user_id IN ?", userIDs).Delete(&model.Session{}).Error; err != nil {
				return err
			}
			if err := tx.Where("user_id IN ?", userIDs).Delete(&model.SecureKey{}).Error; err != nil {
				return err
			}
			if err := tx.Where("user_id IN ?", userIDs).Delete(&model.MFAEnrollment{}).Error; err != nil {
				return err
			}
			if err := tx.Where("user_id IN ?", userIDs).Delete(&model.MFARecoveryCode{}).Error; err != nil {
				return err
			}
			if err := tx.Where("user_id IN ?", userIDs).Delete(&model.Device{}).Error; err != nil {
				return err
			}
			if err := tx.Where("user_id IN ?", userIDs).Delete(&model.ExternalIdentityBinding{}).Error; err != nil {
				return err
			}
			if err := tx.Where("id IN ?", userIDs).Delete(&model.User{}).Error; err != nil {
				return err
			}
		}
		if len(externalIDPIDs) > 0 {
			if err := tx.Where("provider_id IN ?", externalIDPIDs).Delete(&model.ExternalAuthState{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("organization_id = ?", organization.ID).Delete(&model.ExternalAuthState{}).Error; err != nil {
			return err
		}
		if err := tx.Where("organization_id = ?", organization.ID).Delete(&model.Session{}).Error; err != nil {
			return err
		}
		if len(appIDs) > 0 {
			if err := tx.Where("id IN ?", appIDs).Delete(&model.Application{}).Error; err != nil {
				return err
			}
		}
		if len(projectIDs) > 0 {
			if err := tx.Unscoped().Where("project_id IN ?", projectIDs).Delete(&model.ProjectUserAssignment{}).Error; err != nil {
				return err
			}
			if err := tx.Where("id IN ?", projectIDs).Delete(&model.Project{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("organization_id = ?", organization.ID).Delete(&model.Policy{}).Error; err != nil {
			return err
		}
		if err := tx.Where("organization_id = ?", organization.ID).Delete(&model.Role{}).Error; err != nil {
			return err
		}
		if err := tx.Where("organization_id = ?", organization.ID).Delete(&model.ExternalIDP{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&organization).Error; err != nil {
			return err
		}
		for _, userID := range userIDs {
			if s.deleteAuthorizationCodesByUser != nil {
				s.deleteAuthorizationCodesByUser(userID)
			}
			if s.deleteMFAChallengesByUser != nil {
				s.deleteMFAChallengesByUser(userID)
			}
		}
		return s.audit.Record(ctx, coreservice.AuditEvent{
			OrganizationID: organization.ID,
			ActorType:      "admin",
			EventType:      "organization.deleted",
			Result:         "success",
			TargetType:     "organization",
			TargetID:       organization.ID,
		})
	})
}

func (s *Service) attachOrganizationSettings(ctx context.Context, organizations []model.Organization) error {
	for index := range organizations {
		current := organizations[index]
		if legacy := coreservice.ParseLegacyOrganizationConsoleSettings(current); legacy != nil {
			settings := coreservice.NormalizeOrganizationConsoleSettings(legacy)
			organizations[index].ConsoleSettings = &settings
			continue
		}
		settings := coreservice.NormalizeOrganizationConsoleSettings(&model.OrganizationSetting{
			TOSURL:           current.TOSURL,
			PrivacyPolicyURL: current.PrivacyPolicyURL,
			SupportEmail:     current.SupportEmail,
			LogoURL:          current.LogoURL,
			Domains:          current.Domains,
			LoginPolicy:      current.LoginPolicy,
			PasswordPolicy:   current.PasswordPolicy,
			MFAPolicy:        current.MFAPolicy,
		})
		organizations[index].ConsoleSettings = &settings
	}
	return nil
}

func (s *Service) getOrganizationSetting(ctx context.Context, organizationID string) (model.OrganizationSetting, error) {
	_, settings, err := coreservice.LoadOrganizationConsoleSettings(ctx, s.db, organizationID)
	return settings, err
}

func normalizeMetadata(candidate map[string]string, fallback map[string]string) map[string]string {
	if candidate == nil {
		if fallback == nil {
			return map[string]string{}
		}
		return fallback
	}
	result := make(map[string]string, len(candidate))
	for key, value := range candidate {
		if key == "" {
			continue
		}
		result[key] = value
	}
	return result
}

func (s *Service) DisableUser(ctx context.Context, userID string) error {
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		return err
	}
	now := time.Now()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&user).Update("status", "disabled").Error; err != nil {
			return err
		}
		if err := tx.Model(&model.Token{}).
			Where("user_id = ? AND revoked_at IS NULL", user.ID).
			Updates(map[string]any{"revoked_at": now, "revocation_note": "user_disabled"}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", user.ID).Delete(&model.Session{}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (s *Service) EnableUser(ctx context.Context, userID string) error {
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Model(&user).Update("status", "active").Error; err != nil {
		return err
	}
	return s.audit.Record(ctx, coreservice.AuditEvent{
		OrganizationID: user.OrganizationID,
		ActorType:      "admin",
		EventType:      "user.enabled",
		Result:         "success",
		TargetType:     "user",
		TargetID:       user.ID,
	})
}

func (s *Service) ResetUserPassword(ctx context.Context, userID, password string) error {
	if password == "" {
		return errors.New("password is required")
	}
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		return err
	}
	hash, err := util.HashSecret(password)
	if err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Model(&user).Update("password_hash", hash).Error; err != nil {
		return err
	}
	return s.audit.Record(ctx, coreservice.AuditEvent{
		OrganizationID: user.OrganizationID,
		ActorType:      "admin",
		EventType:      "user.password.reset",
		Result:         "success",
		TargetType:     "user",
		TargetID:       user.ID,
	})
}

func (s *Service) ResetUserUKID(ctx context.Context, userID string) (string, error) {
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		return "", err
	}
	newUKID, err := util.RandomToken(18)
	if err != nil {
		return "", err
	}
	now := time.Now()
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&user).Update("current_ukid", newUKID).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.Token{}).
			Where("user_id = ? AND revoked_at IS NULL", user.ID).
			Updates(map[string]any{"revoked_at": now, "revocation_note": "ukid_reset"}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", user.ID).Delete(&model.Session{}).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return "", err
	}
	return newUKID, s.audit.Record(ctx, coreservice.AuditEvent{
		OrganizationID: user.OrganizationID,
		ActorType:      "admin",
		ActorID:        user.ID,
		EventType:      "user.ukid.reset",
		Result:         "success",
		TargetType:     "user",
		TargetID:       user.ID,
		Detail:         map[string]any{"newUkid": newUKID},
	})
}

func (s *Service) CreateProject(ctx context.Context, project model.Project) (*model.Project, error) {
	if project.OrganizationID == "" || project.Name == "" {
		return nil, errors.New("organizationId and name are required")
	}
	if strings.TrimSpace(project.Status) == "" {
		project.Status = "active"
	}
	if err := s.db.WithContext(ctx).Create(&project).Error; err != nil {
		return nil, err
	}
	_ = s.audit.Record(ctx, coreservice.AuditEvent{
		OrganizationID: project.OrganizationID,
		ProjectID:      project.ID,
		ActorType:      "admin",
		EventType:      "project.created",
		Result:         "success",
		TargetType:     "project",
		TargetID:       project.ID,
		Detail:         map[string]any{"name": project.Name},
	})
	return &project, nil
}

func (s *Service) UpdateProject(ctx context.Context, project model.Project) (*model.Project, error) {
	if project.ID == "" {
		return nil, errors.New("id is required")
	}
	var existing model.Project
	if err := s.db.WithContext(ctx).First(&existing, "id = ?", project.ID).Error; err != nil {
		return nil, err
	}
	updates := map[string]any{
		"name":             coalesceString(project.Name, existing.Name),
		"description":      coalesceString(project.Description, existing.Description),
		"user_acl_enabled": project.UserACLEnabled,
	}
	if err := s.db.WithContext(ctx).Model(&existing).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).First(&existing, "id = ?", project.ID).Error; err != nil {
		return nil, err
	}
	return &existing, nil
}

func (s *Service) DisableProject(ctx context.Context, projectID string) error {
	var project model.Project
	if err := s.db.WithContext(ctx).First(&project, "id = ?", projectID).Error; err != nil {
		return err
	}
	now := time.Now()
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&project).Update("status", "disabled").Error; err != nil {
			return err
		}
		var appIDs []string
		if err := tx.Model(&model.Application{}).Where("project_id = ?", project.ID).Pluck("id", &appIDs).Error; err != nil {
			return err
		}
		if len(appIDs) == 0 {
			return nil
		}
		if err := tx.Model(&model.Token{}).
			Where("application_id IN ? AND revoked_at IS NULL", appIDs).
			Updates(map[string]any{"revoked_at": now, "revocation_note": "project_disabled"}).Error; err != nil {
			return err
		}
		if err := tx.Where("application_id IN ?", appIDs).Delete(&model.AuthorizationCode{}).Error; err != nil {
			return err
		}
		if err := tx.Where("application_id IN ?", appIDs).Delete(&model.Session{}).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return s.audit.Record(ctx, coreservice.AuditEvent{
		OrganizationID: project.OrganizationID,
		ProjectID:      project.ID,
		ActorType:      "admin",
		EventType:      "project.disabled",
		Result:         "success",
		TargetType:     "project",
		TargetID:       project.ID,
	})
}

func (s *Service) DeleteProject(ctx context.Context, projectID string) error {
	if strings.TrimSpace(projectID) == "" {
		return errors.New("projectId is required")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var project model.Project
		if err := tx.First(&project, "id = ?", projectID).Error; err != nil {
			return err
		}
		var appIDs []string
		if err := tx.Model(&model.Application{}).Where("project_id = ?", project.ID).Pluck("id", &appIDs).Error; err != nil {
			return err
		}
		if len(appIDs) > 0 {
			if err := tx.Where("application_id IN ?", appIDs).Delete(&model.AuthorizationCode{}).Error; err != nil {
				return err
			}
			if err := tx.Where("application_id IN ?", appIDs).Delete(&model.Token{}).Error; err != nil {
				return err
			}
			if err := tx.Where("application_id IN ?", appIDs).Delete(&model.Session{}).Error; err != nil {
				return err
			}
			if err := tx.Where("id IN ?", appIDs).Delete(&model.Application{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Unscoped().Where("project_id = ?", project.ID).Delete(&model.ProjectUserAssignment{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&project).Error; err != nil {
			return err
		}
		return s.audit.Record(ctx, coreservice.AuditEvent{
			OrganizationID: project.OrganizationID,
			ProjectID:      project.ID,
			ActorType:      "admin",
			EventType:      "project.deleted",
			Result:         "success",
			TargetType:     "project",
			TargetID:       project.ID,
		})
	})
}

func (s *Service) CreateApplication(ctx context.Context, app model.Application) (*coreservice.ApplicationMutationResult, error) {
	if app.ProjectID == "" || app.Name == "" {
		return nil, errors.New("projectId and name are required")
	}
	if app.ID == "" {
		app.ID = uuid.NewString()
	}
	applyApplicationDefaults(&app)
	if err := validateApplicationProtocol(app); err != nil {
		return nil, err
	}
	if app.AccessTokenTTLMinutes <= 0 || app.RefreshTokenTTLHours <= 0 {
		return nil, errors.New("accessTokenTTLMinutes and refreshTokenTTLHours are required")
	}
	validatedRoles, err := s.validateOrganizationRoleNames(ctx, app.ProjectID, app.Roles, "application")
	if err != nil {
		return nil, err
	}
	app.Roles = validatedRoles
	if strings.TrimSpace(app.Status) == "" {
		app.Status = "active"
	}
	generatedPrivateKey := ""
	if app.ClientAuthenticationType == "private_key_jwt" {
		publicKey, privateKey, err := util.GenerateEd25519KeyMaterial()
		if err != nil {
			return nil, err
		}
		app.PublicKey = publicKey
		generatedPrivateKey = privateKey
	}
	if err := s.db.WithContext(ctx).Create(&app).Error; err != nil {
		return nil, err
	}
	_ = s.audit.Record(ctx, coreservice.AuditEvent{
		ApplicationID: app.ID,
		ProjectID:     app.ProjectID,
		ActorType:     "admin",
		EventType:     "application.created",
		Result:        "success",
		TargetType:    "application",
		TargetID:      app.ID,
		Detail:        map[string]any{"name": app.Name},
	})
	return &coreservice.ApplicationMutationResult{
		Application:         app,
		GeneratedPrivateKey: generatedPrivateKey,
	}, nil
}

func (s *Service) UpdateApplication(ctx context.Context, app model.Application) (*coreservice.ApplicationMutationResult, error) {
	if app.ID == "" {
		return nil, errors.New("id is required")
	}
	var existing model.Application
	if err := s.db.WithContext(ctx).First(&existing, "id = ?", app.ID).Error; err != nil {
		return nil, err
	}
	updates := map[string]any{
		"name":                       coalesceString(app.Name, existing.Name),
		"description":                coalesceString(app.Description, existing.Description),
		"redirect_uris":              coalesceString(app.RedirectURIs, existing.RedirectURIs),
		"application_type":           coalesceString(app.ApplicationType, existing.ApplicationType),
		"grant_type":                 coalesceGrantTypes(app.GrantType, existing.GrantType),
		"enable_refresh_token":       app.EnableRefreshToken,
		"client_authentication_type": coalesceString(app.ClientAuthenticationType, existing.ClientAuthenticationType),
		"token_type":                 coalesceTokenTypes(app.TokenType, existing.TokenType),
		"roles":                      coalesceApplicationRoles(app.Roles, existing.Roles),
		"access_token_ttl_minutes":   coalesceInt(app.AccessTokenTTLMinutes, existing.AccessTokenTTLMinutes),
		"refresh_token_ttl_hours":    coalesceInt(app.RefreshTokenTTLHours, existing.RefreshTokenTTLHours),
	}
	candidate := existing
	candidate.Name = updates["name"].(string)
	candidate.Description = updates["description"].(string)
	candidate.RedirectURIs = updates["redirect_uris"].(string)
	candidate.ApplicationType = updates["application_type"].(string)
	candidate.GrantType = updates["grant_type"].([]string)
	candidate.EnableRefreshToken = updates["enable_refresh_token"].(bool)
	candidate.ClientAuthenticationType = updates["client_authentication_type"].(string)
	candidate.TokenType = updates["token_type"].([]string)
	candidate.Roles = updates["roles"].([]string)
	applyApplicationDefaults(&candidate)
	validatedRoles, err := s.validateOrganizationRoleNames(ctx, candidate.ProjectID, candidate.Roles, "application")
	if err != nil {
		return nil, err
	}
	candidate.Roles = validatedRoles
	if err := validateApplicationProtocol(candidate); err != nil {
		return nil, err
	}
	generatedPrivateKey := ""
	updates["application_type"] = candidate.ApplicationType
	updates["grant_type"] = candidate.GrantType
	updates["enable_refresh_token"] = candidate.EnableRefreshToken
	updates["client_authentication_type"] = candidate.ClientAuthenticationType
	updates["token_type"] = candidate.TokenType
	updates["roles"] = candidate.Roles
	if candidate.ClientAuthenticationType == "private_key_jwt" && strings.TrimSpace(existing.PublicKey) == "" {
		publicKey, privateKey, err := util.GenerateEd25519KeyMaterial()
		if err != nil {
			return nil, err
		}
		updates["public_key"] = publicKey
		generatedPrivateKey = privateKey
	} else if candidate.ClientAuthenticationType != "private_key_jwt" {
		updates["public_key"] = ""
	}
	if err := s.db.WithContext(ctx).Model(&existing).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).First(&existing, "id = ?", app.ID).Error; err != nil {
		return nil, err
	}
	return &coreservice.ApplicationMutationResult{
		Application:         existing,
		GeneratedPrivateKey: generatedPrivateKey,
	}, nil
}

func (s *Service) DisableApplication(ctx context.Context, applicationID string) error {
	var app model.Application
	if err := s.db.WithContext(ctx).First(&app, "id = ?", applicationID).Error; err != nil {
		return err
	}
	now := time.Now()
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&app).Update("status", "disabled").Error; err != nil {
			return err
		}
		if err := tx.Model(&model.Token{}).
			Where("application_id = ? AND revoked_at IS NULL", app.ID).
			Updates(map[string]any{"revoked_at": now, "revocation_note": "application_disabled"}).Error; err != nil {
			return err
		}
		if err := tx.Where("application_id = ?", app.ID).Delete(&model.AuthorizationCode{}).Error; err != nil {
			return err
		}
		if err := tx.Where("application_id = ?", app.ID).Delete(&model.Session{}).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return s.audit.Record(ctx, coreservice.AuditEvent{
		ApplicationID: app.ID,
		ProjectID:     app.ProjectID,
		ActorType:     "admin",
		EventType:     "application.disabled",
		Result:        "success",
		TargetType:    "application",
		TargetID:      app.ID,
	})
}

func (s *Service) DeleteApplication(ctx context.Context, applicationID string) error {
	if strings.TrimSpace(applicationID) == "" {
		return errors.New("applicationId is required")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var app model.Application
		if err := tx.First(&app, "id = ?", applicationID).Error; err != nil {
			return err
		}
		if err := tx.Where("application_id = ?", app.ID).Delete(&model.AuthorizationCode{}).Error; err != nil {
			return err
		}
		if err := tx.Where("application_id = ?", app.ID).Delete(&model.Token{}).Error; err != nil {
			return err
		}
		if err := tx.Where("application_id = ?", app.ID).Delete(&model.Session{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&app).Error; err != nil {
			return err
		}
		return s.audit.Record(ctx, coreservice.AuditEvent{
			ApplicationID: app.ID,
			ProjectID:     app.ProjectID,
			ActorType:     "admin",
			EventType:     "application.deleted",
			Result:        "success",
			TargetType:    "application",
			TargetID:      app.ID,
		})
	})
}

func (s *Service) ResetApplicationKey(ctx context.Context, applicationID string) (*coreservice.ApplicationMutationResult, error) {
	if strings.TrimSpace(applicationID) == "" {
		return nil, errors.New("applicationId is required")
	}
	var app model.Application
	if err := s.db.WithContext(ctx).First(&app, "id = ?", applicationID).Error; err != nil {
		return nil, err
	}
	if app.ClientAuthenticationType != "private_key_jwt" {
		return nil, errors.New("application is not configured for private_key_jwt")
	}
	publicKey, privateKey, err := util.GenerateEd25519KeyMaterial()
	if err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(&app).Updates(map[string]any{
		"public_key": publicKey,
	}).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).First(&app, "id = ?", applicationID).Error; err != nil {
		return nil, err
	}
	_ = s.audit.Record(ctx, coreservice.AuditEvent{
		ApplicationID: app.ID,
		ProjectID:     app.ProjectID,
		ActorType:     "admin",
		EventType:     "application.key.reset",
		Result:        "success",
		TargetType:    "application",
		TargetID:      app.ID,
	})
	return &coreservice.ApplicationMutationResult{
		Application:         app,
		GeneratedPrivateKey: privateKey,
	}, nil
}

func applyApplicationDefaults(app *model.Application) {
	if app.ApplicationType == "" {
		app.ApplicationType = "web"
	}
	if len(app.GrantType) == 0 {
		if app.ApplicationType == "api" {
			app.GrantType = []string{"client_credentials"}
		} else {
			app.GrantType = []string{"authorization_code_pkce"}
		}
	}
	if app.ClientAuthenticationType == "" {
		if app.ApplicationType == "api" {
			app.ClientAuthenticationType = "private_key_jwt"
		} else {
			app.ClientAuthenticationType = "none"
		}
	}
	if len(app.TokenType) == 0 {
		app.TokenType = []string{"access_token"}
	}
	app.GrantType = normalizeGrantTypes(app.GrantType)
	app.TokenType = normalizeTokenTypes(app.TokenType)
	app.Roles = normalizeRoleNames(app.Roles)
}

func validateApplicationProtocol(app model.Application) error {
	if !applicationTypeOptions[app.ApplicationType] {
		return errors.New("invalid applicationType")
	}
	if len(app.GrantType) == 0 {
		return errors.New("grantType is required")
	}
	for _, grantType := range app.GrantType {
		if !grantTypeOptions[grantType] {
			return errors.New("invalid grantType")
		}
	}
	if !clientAuthenticationTypeOptions[app.ClientAuthenticationType] {
		return errors.New("invalid clientAuthenticationType")
	}
	if len(app.TokenType) == 0 {
		return errors.New("tokenType is required")
	}
	for _, tokenType := range app.TokenType {
		if !tokenTypeOptions[tokenType] {
			return errors.New("invalid tokenType")
		}
	}
	if app.EnableRefreshToken && !tokenTypesContain(app.TokenType, "access_token") {
		return errors.New("enableRefreshToken requires tokenType to include access_token")
	}
	if grantTypesContain(app.GrantType, "client_credentials") {
		if !tokenTypesEqual(app.TokenType, []string{"access_token"}) {
			return errors.New("client_credentials only supports tokenType=access_token")
		}
		if app.EnableRefreshToken {
			return errors.New("client_credentials does not support refresh_token")
		}
	}
	if grantTypesContain(app.GrantType, "implicit") {
		for _, tokenType := range app.TokenType {
			if tokenType != "access_token" && tokenType != "id_token" {
				return errors.New("implicit only supports tokenType=access_token and/or id_token")
			}
		}
		if app.EnableRefreshToken {
			return errors.New("implicit does not support refresh_token")
		}
	}
	if app.ClientAuthenticationType == "none" {
		for _, grantType := range app.GrantType {
			if grantType != "authorization_code_pkce" && grantType != "device_code" && grantType != "password" {
				return errors.New("clientAuthenticationType=none is only allowed for authorization_code_pkce, device_code, or password")
			}
		}
	}
	if grantTypesContain(app.GrantType, "client_credentials") && app.ClientAuthenticationType == "none" {
		return errors.New("client_credentials requires client authentication")
	}
	return nil
}

func normalizeGrantTypes(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, item := range values {
		value := strings.TrimSpace(item)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func coalesceGrantTypes(candidate []string, fallback []string) []string {
	if len(candidate) == 0 {
		return fallback
	}
	return normalizeGrantTypes(candidate)
}

func normalizeTokenTypes(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, item := range values {
		value := strings.TrimSpace(item)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func coalesceTokenTypes(candidate []string, fallback []string) []string {
	if len(candidate) == 0 {
		return fallback
	}
	return normalizeTokenTypes(candidate)
}

func tokenTypesContain(values []string, expected string) bool {
	expected = strings.TrimSpace(expected)
	for _, item := range values {
		if strings.TrimSpace(item) == expected {
			return true
		}
	}
	return false
}

func tokenTypesEqual(left []string, right []string) bool {
	a := normalizeTokenTypes(left)
	b := normalizeTokenTypes(right)
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func grantTypesContain(values []string, expected string) bool {
	expected = strings.TrimSpace(expected)
	for _, item := range values {
		if strings.TrimSpace(item) == expected {
			return true
		}
	}
	return false
}

func normalizeRoleNames(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, item := range values {
		value := strings.TrimSpace(item)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func coalesceApplicationRoles(values, fallback []string) []string {
	if len(values) == 0 {
		return normalizeRoleNames(fallback)
	}
	return normalizeRoleNames(values)
}

func (s *Service) validateOrganizationRoleNames(ctx context.Context, projectID string, roleNames []string, expectedType string) ([]string, error) {
	roleNames = normalizeRoleNames(roleNames)
	if len(roleNames) == 0 {
		return []string{}, nil
	}
	var project model.Project
	if err := s.db.WithContext(ctx).First(&project, "id = ?", projectID).Error; err != nil {
		return nil, err
	}
	return s.validateRoleAssignments(ctx, project.OrganizationID, roleNames, expectedType)
}

func (s *Service) validateRoleAssignments(ctx context.Context, organizationID string, roleNames []string, expectedType string) ([]string, error) {
	roleNames = normalizeRoleNames(roleNames)
	if len(roleNames) == 0 {
		return []string{}, nil
	}
	if !roleTypeOptions[expectedType] {
		return nil, errors.New("invalid role type")
	}
	var roles []model.Role
	if err := s.db.WithContext(ctx).
		Where("organization_id = ? AND name IN ?", organizationID, roleNames).
		Find(&roles).Error; err != nil {
		return nil, err
	}
	roleByName := make(map[string]model.Role, len(roles))
	for _, role := range roles {
		roleByName[role.Name] = role
	}
	validated := make([]string, 0, len(roleNames))
	for _, roleName := range roleNames {
		role, ok := roleByName[roleName]
		if !ok {
			return nil, errors.New("role does not exist: " + roleName)
		}
		if role.Type != expectedType {
			return nil, errors.New("role type mismatch: " + roleName)
		}
		validated = append(validated, role.Name)
	}
	return validated, nil
}

func containsString(values []string, expected string) bool {
	expected = strings.TrimSpace(expected)
	for _, item := range values {
		if strings.TrimSpace(item) == expected {
			return true
		}
	}
	return false
}

func normalizeUserIDs(userIDs []string) []string {
	set := make(map[string]struct{}, len(userIDs))
	normalized := make([]string, 0, len(userIDs))
	for _, userID := range userIDs {
		trimmed := strings.TrimSpace(userID)
		if trimmed == "" {
			continue
		}
		if _, exists := set[trimmed]; exists {
			continue
		}
		set[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	sort.Strings(normalized)
	return normalized
}

func (s *Service) attachProjectAssignedUserIDs(ctx context.Context, items []model.Project) error {
	if len(items) == 0 {
		return nil
	}
	projectIDs := make([]string, 0, len(items))
	for _, item := range items {
		projectIDs = append(projectIDs, item.ID)
	}
	var assignments []model.ProjectUserAssignment
	if err := s.db.WithContext(ctx).Where("project_id IN ?", projectIDs).Find(&assignments).Error; err != nil {
		return err
	}
	assignmentMap := make(map[string][]string, len(items))
	for _, assignment := range assignments {
		assignmentMap[assignment.ProjectID] = append(assignmentMap[assignment.ProjectID], assignment.UserID)
	}
	for i := range items {
		items[i].AssignedUserIDs = normalizeUserIDs(assignmentMap[items[i].ID])
	}
	return nil
}

func (s *Service) ListProjects(ctx context.Context, organizationID string) ([]model.Project, error) {
	var items []model.Project
	query := s.db.WithContext(ctx).Preload("Applications")
	if organizationID != "" {
		query = query.Where("organization_id = ?", organizationID)
	}
	if err := query.Find(&items).Error; err != nil {
		return nil, err
	}
	if err := s.attachProjectAssignedUserIDs(ctx, items); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Service) UpdateProjectUserAssignments(ctx context.Context, projectID string, userIDs []string) ([]string, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, errors.New("projectId is required")
	}
	normalizedUserIDs := normalizeUserIDs(userIDs)
	var project model.Project
	if err := s.db.WithContext(ctx).First(&project, "id = ?", projectID).Error; err != nil {
		return nil, err
	}
	if len(normalizedUserIDs) > 0 {
		var users []model.User
		if err := s.db.WithContext(ctx).
			Where("organization_id = ? AND id IN ?", project.OrganizationID, normalizedUserIDs).
			Find(&users).Error; err != nil {
			return nil, err
		}
		if len(users) != len(normalizedUserIDs) {
			return nil, errors.New("some users do not belong to the current organization")
		}
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("project_id = ?", projectID).Delete(&model.ProjectUserAssignment{}).Error; err != nil {
			return err
		}
		if len(normalizedUserIDs) == 0 {
			return nil
		}
		assignments := make([]model.ProjectUserAssignment, 0, len(normalizedUserIDs))
		for _, userID := range normalizedUserIDs {
			assignments = append(assignments, model.ProjectUserAssignment{
				ProjectID: projectID,
				UserID:    userID,
			})
		}
		return tx.Create(&assignments).Error
	}); err != nil {
		return nil, err
	}
	_ = s.audit.Record(ctx, coreservice.AuditEvent{
		OrganizationID: project.OrganizationID,
		ProjectID:      project.ID,
		ActorType:      "admin",
		EventType:      "project.user_assignment.updated",
		Result:         "success",
		TargetType:     "project",
		TargetID:       project.ID,
		Detail:         map[string]any{"userIds": normalizedUserIDs},
	})
	return normalizedUserIDs, nil
}

func (s *Service) ListApplications(ctx context.Context, projectID string) ([]model.Application, error) {
	var items []model.Application
	query := s.db.WithContext(ctx)
	if projectID != "" {
		query = query.Where("project_id = ?", projectID)
	}
	err := query.Find(&items).Error
	return items, err
}

func (s *Service) ListUsers(ctx context.Context, organizationID string) ([]model.User, error) {
	var items []model.User
	query := s.db.WithContext(ctx)
	if organizationID != "" {
		query = query.Where("organization_id = ?", organizationID)
	}
	err := query.Find(&items).Error
	return items, err
}

func (s *Service) GetUserDetail(ctx context.Context, userID string) (*coreservice.UserDetailData, error) {
	if userID == "" {
		return nil, errors.New("userId is required")
	}
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}
	var credentials []model.SecureKey
	if err := s.db.WithContext(ctx).
		Where("user_id = ?", user.ID).
		Order("created_at desc").
		Find(&credentials).Error; err != nil {
		return nil, err
	}
	secureKeys := make([]coreservice.UserDetailSecureKey, 0)
	for _, credential := range credentials {
		secureKeys = append(secureKeys, coreservice.UserDetailSecureKey{
			ID:             credential.ID,
			PublicKeyID:    credential.PublicKeyID,
			Identifier:     credential.Identifier,
			SignCount:      credential.SignCount,
			WebAuthnEnable: credential.WebAuthnEnable,
			U2FEnable:      credential.U2FEnable,
			CreatedAt:      credential.CreatedAt,
			UpdatedAt:      credential.UpdatedAt,
		})
	}

	var providers []model.ExternalIDP
	if err := s.db.WithContext(ctx).
		Where("organization_id = ?", user.OrganizationID).
		Order("created_at asc").
		Find(&providers).Error; err != nil {
		return nil, err
	}
	providerNameMap := make(map[string]string, len(providers))
	for _, provider := range providers {
		providerNameMap[provider.ID] = provider.Name
	}

	var rawBindings []model.ExternalIdentityBinding
	if err := s.db.WithContext(ctx).
		Where("user_id = ?", user.ID).
		Order("created_at desc").
		Find(&rawBindings).Error; err != nil {
		return nil, err
	}
	bindings := make([]coreservice.UserDetailBinding, 0, len(rawBindings))
	for _, binding := range rawBindings {
		bindings = append(bindings, coreservice.UserDetailBinding{
			ID:            binding.ID,
			ExternalIDPID: binding.ExternalIDPID,
			ProviderName:  providerNameMap[binding.ExternalIDPID],
			Issuer:        binding.Issuer,
			Subject:       binding.Subject,
			CreatedAt:     binding.CreatedAt,
			UpdatedAt:     binding.UpdatedAt,
		})
	}

	var enrollments []model.MFAEnrollment
	if err := s.db.WithContext(ctx).
		Where("user_id = ?", user.ID).
		Order("created_at desc").
		Find(&enrollments).Error; err != nil {
		return nil, err
	}

	var devices []model.Device
	if err := s.db.WithContext(ctx).
		Where("user_id = ?", user.ID).
		Order("last_seen_at desc").
		Find(&devices).Error; err != nil {
		return nil, err
	}

	var recentSessions []model.Session
	if err := s.db.WithContext(ctx).
		Where("user_id = ?", user.ID).
		Order("updated_at desc").
		Limit(20).
		Find(&recentSessions).Error; err != nil {
		return nil, err
	}
	userDevices := buildUserDevices(devices, recentSessions)
	for index := range userDevices {
		userDevices[index].IPLocation = s.resolveIPLocation(userDevices[index].LastLoginIP)
	}

	var recoveryCodes []model.MFARecoveryCode
	if err := s.db.WithContext(ctx).
		Where("user_id = ?", user.ID).
		Find(&recoveryCodes).Error; err != nil {
		return nil, err
	}
	recoverySummary := coreservice.UserDetailRecoverySummary{Total: len(recoveryCodes)}
	for _, item := range recoveryCodes {
		if recoverySummary.LastGeneratedAt == nil || item.CreatedAt.After(*recoverySummary.LastGeneratedAt) {
			createdAt := item.CreatedAt
			recoverySummary.LastGeneratedAt = &createdAt
		}
		if item.ConsumedAt == nil {
			recoverySummary.Available++
		} else {
			recoverySummary.Consumed++
		}
	}

	var auditLogs []model.AuditLog
	if err := s.db.WithContext(ctx).
		Where("target_type = ? AND target_id = ?", "user", user.ID).
		Order("created_at desc").
		Limit(10).
		Find(&auditLogs).Error; err != nil {
		return nil, err
	}
	normalizedAuditLogs := s.decorateAuditLogs(auditLogs)

	return &coreservice.UserDetailData{
		User:               user,
		PasswordCredential: strings.TrimSpace(user.PasswordHash) != "",
		SecureKeys:         secureKeys,
		Bindings:           bindings,
		ExternalIDPs:       providers,
		MFAEnrollments:     enrollments,
		Devices:            userDevices,
		RecoverySummary:    recoverySummary,
		RecentAuditLogs:    normalizedAuditLogs,
	}, nil
}

func (s *Service) CreateUser(ctx context.Context, user model.User, identifier, password string, applicationID string) (*model.User, error) {
	if user.OrganizationID == "" {
		return nil, errors.New("organizationId is required")
	}
	if user.Email == "" && user.PhoneNumber == "" {
		return nil, errors.New("email or phoneNumber is required")
	}
	if password == "" {
		return nil, errors.New("password is required")
	}
	if identifier == "" {
		if user.Email != "" {
			identifier = user.Email
		} else {
			identifier = user.PhoneNumber
		}
	}
	if user.Status == "" {
		user.Status = "active"
	}
	if len(user.Roles) == 0 {
		user.Roles = []string{"user:self:all"}
	} else if !containsString(user.Roles, "user:self:all") {
		user.Roles = append(user.Roles, "user:self:all")
	}
	validatedRoles, err := s.validateRoleAssignments(ctx, user.OrganizationID, user.Roles, "user")
	if err != nil {
		return nil, err
	}
	user.Roles = validatedRoles
	hash, err := util.HashSecret(password)
	if err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		user.PasswordHash = hash
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		if user.CurrentUKID == "" {
			user.CurrentUKID = "ukid-" + user.ID
			if err := tx.Model(&user).Update("current_ukid", user.CurrentUKID).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	_ = s.audit.Record(ctx, coreservice.AuditEvent{
		OrganizationID: user.OrganizationID,
		ApplicationID:  applicationID,
		ActorType:      "admin",
		EventType:      "user.created",
		Result:         "success",
		TargetType:     "user",
		TargetID:       user.ID,
		Detail: map[string]any{
			"identifier": identifier,
		},
	})
	return &user, nil
}

func (s *Service) UpdateUser(ctx context.Context, user model.User) (*model.User, error) {
	if user.ID == "" {
		return nil, errors.New("id is required")
	}
	var existing model.User
	if err := s.db.WithContext(ctx).First(&existing, "id = ?", user.ID).Error; err != nil {
		return nil, err
	}
	nextRoles := coalesceRoles(user.Roles, existing.Roles)
	if !containsString(nextRoles, "user:self:all") {
		nextRoles = append(nextRoles, "user:self:all")
	}
	validatedRoles, err := s.validateRoleAssignments(ctx, existing.OrganizationID, nextRoles, "user")
	if err != nil {
		return nil, err
	}
	updates := map[string]any{
		"username":     coalesceString(user.Username, existing.Username),
		"name":         coalesceString(user.Name, existing.Name),
		"email":        coalesceString(user.Email, existing.Email),
		"phone_number": coalesceString(user.PhoneNumber, existing.PhoneNumber),
		"roles":        validatedRoles,
		"status":       coalesceString(user.Status, existing.Status),
	}
	if err := s.db.WithContext(ctx).Model(&existing).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).First(&existing, "id = ?", user.ID).Error; err != nil {
		return nil, err
	}
	return &existing, nil
}

func (s *Service) SetUserMFAMethod(ctx context.Context, userID, method string, enabled bool) error {
	if userID == "" {
		return errors.New("userId is required")
	}
	if method != "email_code" && method != "sms_code" && method != "webauthn" {
		return errors.New("unsupported method")
	}
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		return err
	}
	target := user.Email
	label := "邮箱验证码"
	if method == "sms_code" {
		target = user.PhoneNumber
		label = "手机验证码"
	}
	if method == "webauthn" {
		label = "通行密钥"
		target = ""
		var count int64
		if err := s.db.WithContext(ctx).
			Model(&model.SecureKey{}).
			Where("user_id = ? AND webauthn_enable = ? AND deleted_at IS NULL", user.ID, true).
			Count(&count).Error; err != nil {
			return err
		}
		if enabled && count == 0 {
			return errors.New("securekey is required before enabling webauthn")
		}
	}
	if enabled && target == "" {
		if method != "webauthn" {
			return errors.New("target is required before enabling method")
		}
	}

	var enrollment model.MFAEnrollment
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND method = ?", user.ID, method).
		Order("created_at desc").
		First(&enrollment).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if !enabled {
			return nil
		}
		enrollment = model.MFAEnrollment{
			OrganizationID: user.OrganizationID,
			UserID:         user.ID,
			Method:         method,
			Label:          label,
			Target:         target,
			Status:         "active",
		}
		if err := s.db.WithContext(ctx).Create(&enrollment).Error; err != nil {
			return err
		}
	} else {
		status := "disabled"
		if enabled {
			status = "active"
		}
		if err := s.db.WithContext(ctx).Model(&enrollment).Updates(map[string]any{
			"method": method,
			"label":  label,
			"target": target,
			"status": status,
		}).Error; err != nil {
			return err
		}
	}

	result := "disabled"
	if enabled {
		result = "enabled"
	}
	return s.audit.Record(ctx, coreservice.AuditEvent{
		OrganizationID: user.OrganizationID,
		ActorType:      "admin",
		EventType:      "user.mfa.method.updated",
		Result:         "success",
		TargetType:     "user",
		TargetID:       user.ID,
		Detail: map[string]any{
			"method": method,
			"status": result,
		},
	})
}

func (s *Service) DeleteUserMFAEnrollments(ctx context.Context, userID, method string) error {
	if userID == "" || method == "" {
		return errors.New("userId and method are required")
	}
	if err := s.db.WithContext(ctx).Where("user_id = ? AND method = ?", userID, method).Delete(&model.MFAEnrollment{}).Error; err != nil {
		return err
	}
	return s.audit.Record(ctx, coreservice.AuditEvent{
		ActorType:  "admin",
		EventType:  "user.mfa.enrollment.deleted",
		Result:     "success",
		TargetType: "user",
		TargetID:   userID,
		Detail: map[string]any{
			"method": method,
		},
	})
}

func (s *Service) DeleteUserSecureKey(ctx context.Context, userID, credentialID string) error {
	if userID == "" || credentialID == "" {
		return errors.New("userId and credentialId are required")
	}
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.
			Where("id = ? AND user_id = ?", credentialID, userID).
			Delete(&model.SecureKey{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("securekey not found")
		}
		return sharedfido.SyncCredentialEnrollments(ctx, tx, user)
	}); err != nil {
		return err
	}
	return s.audit.Record(ctx, coreservice.AuditEvent{
		ActorType:  "admin",
		EventType:  "user.securekey.deleted",
		Result:     "success",
		TargetType: "credential",
		TargetID:   credentialID,
		Detail: map[string]any{
			"userId": userID,
		},
	})
}

func (s *Service) DeleteUserRecoveryCodes(ctx context.Context, userID string) error {
	if userID == "" {
		return errors.New("userId is required")
	}
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&model.MFARecoveryCode{}).Error; err != nil {
		return err
	}
	return s.audit.Record(ctx, coreservice.AuditEvent{
		ActorType:  "admin",
		EventType:  "user.recovery_code.deleted",
		Result:     "success",
		TargetType: "user",
		TargetID:   userID,
	})
}

func (s *Service) DeleteUsers(ctx context.Context, userIDs []string) error {
	if len(userIDs) == 0 {
		return errors.New("userIds is required")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var users []model.User
		if err := tx.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
			return err
		}
		if len(users) == 0 {
			return errors.New("user not found")
		}
		if err := tx.Where("user_id IN ?", userIDs).Delete(&model.Token{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id IN ?", userIDs).Delete(&model.Session{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id IN ?", userIDs).Delete(&model.SecureKey{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id IN ?", userIDs).Delete(&model.MFAEnrollment{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id IN ?", userIDs).Delete(&model.MFARecoveryCode{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id IN ?", userIDs).Delete(&model.Device{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id IN ?", userIDs).Delete(&model.ExternalIdentityBinding{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id IN ?", userIDs).Delete(&model.User{}).Error; err != nil {
			return err
		}
		for _, user := range users {
			if s.deleteAuthorizationCodesByUser != nil {
				s.deleteAuthorizationCodesByUser(user.ID)
			}
			if s.deleteMFAChallengesByUser != nil {
				s.deleteMFAChallengesByUser(user.ID)
			}
		}
		for _, user := range users {
			_ = s.audit.Record(ctx, coreservice.AuditEvent{
				OrganizationID: user.OrganizationID,
				ActorType:      "admin",
				EventType:      "user.deleted",
				Result:         "success",
				TargetType:     "user",
				TargetID:       user.ID,
			})
		}
		return nil
	})
}

func (s *Service) GetCurrentUserProfile(ctx context.Context, sessionID string) (*model.User, error) {
	session, user, err := s.loadSessionUser(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.State == "rejected" {
		return nil, errors.New("session is not active")
	}
	return user, nil
}

func (s *Service) AuthenticateAccessToken(ctx context.Context, accessToken string) (*coreservice.AccessTokenIdentity, error) {
	if accessToken == "" {
		return nil, errors.New("access token is required")
	}
	var token model.Token
	if err := s.db.WithContext(ctx).
		Where("token = ? AND type = ?", accessToken, "access_token").
		First(&token).Error; err != nil {
		return nil, errors.New("access token is invalid")
	}
	if token.RevokedAt != nil || token.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("access token is expired or revoked")
	}
	var application model.Application
	if err := s.db.WithContext(ctx).First(&application, "id = ?", token.ApplicationID).Error; err != nil {
		return nil, errors.New("application is not available")
	}
	identity := &coreservice.AccessTokenIdentity{
		Token:       token,
		Application: application,
	}
	if token.SessionID != "" {
		var session model.Session
		if err := s.db.WithContext(ctx).First(&session, "id = ?", token.SessionID).Error; err != nil {
			return nil, errors.New("session is not active")
		}
		if session.State != "authenticated" {
			return nil, errors.New("session is not active")
		}
		identity.Session = &session
	}
	if token.UserID != "" {
		var user model.User
		if err := s.db.WithContext(ctx).First(&user, "id = ?", token.UserID).Error; err != nil {
			return nil, errors.New("user is not available")
		}
		if user.Status != "active" {
			return nil, errors.New("user is not active")
		}
		if token.UKID != "" && token.UKID != "client-credential" && user.CurrentUKID != token.UKID {
			return nil, errors.New("access token is no longer valid")
		}
		identity.User = &user
	}
	return identity, nil
}

func (s *Service) IntrospectToken(ctx context.Context, tokenValue string) (map[string]any, error) {
	if strings.TrimSpace(tokenValue) == "" {
		return map[string]any{"active": false}, nil
	}
	var token model.Token
	if err := s.db.WithContext(ctx).Where("token = ?", tokenValue).First(&token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return map[string]any{"active": false}, nil
		}
		return nil, err
	}
	active := token.RevokedAt == nil && token.ExpiresAt.After(time.Now())
	if !active {
		return map[string]any{"active": false}, nil
	}
	result := map[string]any{
		"active":     true,
		"scope":      token.Scope,
		"client_id":  token.ApplicationID,
		"token_type": token.Type,
		"exp":        token.ExpiresAt.Unix(),
		"sub":        token.UserID,
	}
	if token.SessionID != "" {
		result["sid"] = token.SessionID
	}
	return result, nil
}

func (s *Service) ValidateConsoleAccessToken(ctx context.Context, accessToken string) (*coreservice.AccessTokenIdentity, error) {
	identity, err := s.AuthenticateAccessToken(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	if identity.User == nil {
		return nil, errors.New("user context is required")
	}
	for _, role := range identity.User.Roles {
		if role == "console:admin" {
			return identity, nil
		}
	}
	return nil, errors.New("console:admin role is required")
}

func (s *Service) GetCurrentUserDetail(ctx context.Context, sessionID string) (*coreservice.UserDetailData, error) {
	_, user, err := s.loadSessionUser(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return s.GetUserDetail(ctx, user.ID)
}

func (s *Service) UpdateCurrentUserProfile(ctx context.Context, sessionID string, patch model.User) (*model.User, error) {
	session, user, err := s.loadSessionUser(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{
		"username":     coalesceString(patch.Username, user.Username),
		"name":         coalesceString(patch.Name, user.Name),
		"email":        coalesceString(patch.Email, user.Email),
		"phone_number": coalesceString(patch.PhoneNumber, user.PhoneNumber),
	}
	if coalesceString(patch.Email, user.Email) == "" && coalesceString(patch.PhoneNumber, user.PhoneNumber) == "" {
		return nil, errors.New("email or phoneNumber is required")
	}
	if err := s.db.WithContext(ctx).Model(user).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).First(user, "id = ?", user.ID).Error; err != nil {
		return nil, err
	}
	_ = s.audit.Record(ctx, coreservice.AuditEvent{
		OrganizationID: user.OrganizationID,
		ApplicationID:  session.ApplicationID,
		ActorType:      "user",
		ActorID:        user.ID,
		EventType:      "user.profile.updated",
		Result:         "success",
		TargetType:     "user",
		TargetID:       user.ID,
	})
	return user, nil
}

func (s *Service) GetCurrentUserSetting(ctx context.Context, sessionID string) (map[string]any, error) {
	session, user, err := s.loadSessionUser(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	var devices []model.Device
	if err := s.db.WithContext(ctx).
		Where("user_id = ?", user.ID).
		Order("last_seen_at desc").
		Find(&devices).Error; err != nil {
		return nil, err
	}
	normalizedDevices := make([]map[string]any, 0, len(devices))
	for _, device := range devices {
		normalizedDevices = append(normalizedDevices, map[string]any{
			"id":          device.ID,
			"fingerprint": device.Fingerprint,
			"description": device.Description,
			"userAgent":   device.UserAgent,
			"lastLoginIp": device.LastLoginIP,
			"ipLocation":  s.resolveIPLocation(device.LastLoginIP),
			"firstSeenAt": device.FirstSeenAt,
			"lastSeenAt":  device.LastSeenAt,
			"trusted":     device.Trusted,
		})
	}
	return map[string]any{
		"user": map[string]any{
			"id":             user.ID,
			"organizationId": user.OrganizationID,
			"username":       user.Username,
			"name":           user.Name,
			"email":          user.Email,
			"phoneNumber":    user.PhoneNumber,
		},
		"session": map[string]any{
			"id":                 session.ID,
			"applicationId":      session.ApplicationID,
			"secondFactorMethod": session.SecondFactorMethod,
			"riskLevel":          session.RiskLevel,
		},
		"devices": normalizedDevices,
	}, nil
}

func (s *Service) SetCurrentUserMFAMethod(ctx context.Context, sessionID, method string, enabled bool) error {
	_, user, err := s.loadSessionUser(ctx, sessionID)
	if err != nil {
		return err
	}
	return s.SetUserMFAMethod(ctx, user.ID, method, enabled)
}

func (s *Service) DeleteCurrentUserMFAEnrollments(ctx context.Context, sessionID, method string) error {
	_, user, err := s.loadSessionUser(ctx, sessionID)
	if err != nil {
		return err
	}
	return s.DeleteUserMFAEnrollments(ctx, user.ID, method)
}

func (s *Service) DeleteCurrentUserSecureKey(ctx context.Context, sessionID, credentialID string) error {
	_, user, err := s.loadSessionUser(ctx, sessionID)
	if err != nil {
		return err
	}
	return s.DeleteUserSecureKey(ctx, user.ID, credentialID)
}

func (s *Service) UpdateCurrentUserPassword(ctx context.Context, sessionID, currentPassword, newPassword string) error {
	if currentPassword == "" || newPassword == "" {
		return errors.New("currentPassword and newPassword are required")
	}
	session, user, err := s.loadSessionUser(ctx, sessionID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(user.PasswordHash) == "" || !util.CheckSecret(user.PasswordHash, currentPassword) {
		return errors.New("current password is invalid")
	}
	hash, err := util.HashSecret(newPassword)
	if err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Model(&user).Update("password_hash", hash).Error; err != nil {
		return err
	}
	_ = s.audit.Record(ctx, coreservice.AuditEvent{
		OrganizationID: user.OrganizationID,
		ApplicationID:  session.ApplicationID,
		ActorType:      "user",
		ActorID:        user.ID,
		EventType:      "user.password.updated",
		Result:         "success",
		TargetType:     "user",
		TargetID:       user.ID,
	})
	return nil
}

func (s *Service) UntrustCurrentDevice(ctx context.Context, sessionID, deviceID string) error {
	if deviceID == "" {
		return errors.New("deviceId is required")
	}
	session, user, err := s.loadSessionUser(ctx, sessionID)
	if err != nil {
		return err
	}
	result := s.db.WithContext(ctx).
		Model(&model.Device{}).
		Where("id = ? AND user_id = ?", deviceID, user.ID).
		Updates(map[string]any{
			"trusted": false,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("device not found")
	}
	_ = s.audit.Record(ctx, coreservice.AuditEvent{
		OrganizationID: user.OrganizationID,
		ApplicationID:  session.ApplicationID,
		ActorType:      "user",
		ActorID:        user.ID,
		EventType:      "user.device.untrusted",
		Result:         "success",
		TargetType:     "device",
		TargetID:       deviceID,
	})
	return nil
}

func (s *Service) CreateCurrentExternalIdentityBinding(ctx context.Context, sessionID string, binding model.ExternalIdentityBinding) (*model.ExternalIdentityBinding, error) {
	_, user, err := s.loadSessionUser(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	binding.OrganizationID = user.OrganizationID
	binding.UserID = user.ID
	return s.CreateExternalIdentityBinding(ctx, binding)
}

func (s *Service) DeleteCurrentExternalIdentityBinding(ctx context.Context, sessionID, bindingID string) error {
	_, user, err := s.loadSessionUser(ctx, sessionID)
	if err != nil {
		return err
	}
	return s.DeleteExternalIdentityBinding(ctx, user.ID, bindingID)
}

func (s *Service) ListAuditLogs(ctx context.Context, organizationID string) ([]coreservice.AuditLogView, error) {
	var items []model.AuditLog
	query := s.db.WithContext(ctx).Order("created_at desc").Limit(100)
	if organizationID != "" {
		query = query.Where("organization_id = ?", organizationID)
	}
	if err := query.Find(&items).Error; err != nil {
		return nil, err
	}
	return s.decorateAuditLogs(items), nil
}

func (s *Service) ListExternalIDPs(ctx context.Context, organizationID string) ([]model.ExternalIDP, error) {
	var items []model.ExternalIDP
	query := s.db.WithContext(ctx)
	if organizationID != "" {
		query = query.Where("organization_id = ?", organizationID)
	}
	err := query.Find(&items).Error
	return items, err
}

func (s *Service) CreateExternalIDP(ctx context.Context, provider model.ExternalIDP) (*model.ExternalIDP, error) {
	if provider.OrganizationID == "" || provider.Name == "" {
		return nil, errors.New("organizationId and name are required")
	}
	if provider.Protocol == "" {
		provider.Protocol = "oidc"
	}
	switch provider.Protocol {
	case "oidc", "oauth":
		if provider.Issuer == "" || provider.ClientID == "" {
			return nil, errors.New("issuer and clientId are required for oauth/oidc providers")
		}
	default:
		return nil, errors.New("unsupported external idp protocol")
	}
	if provider.Scopes == "" {
		provider.Scopes = "openid profile email phone"
	}
	if err := s.db.WithContext(ctx).Create(&provider).Error; err != nil {
		return nil, err
	}
	return &provider, nil
}

func (s *Service) UpdateExternalIDP(ctx context.Context, provider model.ExternalIDP) (*model.ExternalIDP, error) {
	if provider.ID == "" {
		return nil, errors.New("id is required")
	}
	var existing model.ExternalIDP
	if err := s.db.WithContext(ctx).First(&existing, "id = ?", provider.ID).Error; err != nil {
		return nil, err
	}
	updates := map[string]any{
		"protocol":          coalesceString(provider.Protocol, existing.Protocol),
		"name":              coalesceString(provider.Name, existing.Name),
		"issuer":            coalesceString(provider.Issuer, existing.Issuer),
		"client_id":         coalesceString(provider.ClientID, existing.ClientID),
		"authorization_url": coalesceString(provider.AuthorizationURL, existing.AuthorizationURL),
		"token_url":         coalesceString(provider.TokenURL, existing.TokenURL),
		"userinfo_url":      coalesceString(provider.UserInfoURL, existing.UserInfoURL),
		"jwks_url":          coalesceString(provider.JWKSURL, existing.JWKSURL),
		"scopes":            coalesceString(provider.Scopes, existing.Scopes),
		"metadata":          normalizeMetadata(provider.Metadata, existing.Metadata),
	}
	if provider.ClientSecret != "" {
		updates["client_secret"] = provider.ClientSecret
	}
	if err := s.db.WithContext(ctx).Model(&existing).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).First(&existing, "id = ?", provider.ID).Error; err != nil {
		return nil, err
	}
	return &existing, nil
}

func (s *Service) CreateExternalIdentityBinding(ctx context.Context, binding model.ExternalIdentityBinding) (*model.ExternalIdentityBinding, error) {
	if binding.OrganizationID == "" || binding.UserID == "" || binding.ExternalIDPID == "" || binding.Issuer == "" || binding.Subject == "" {
		return nil, errors.New("organizationId, userId, externalIdpId, issuer and subject are required")
	}
	if err := s.db.WithContext(ctx).Create(&binding).Error; err != nil {
		return nil, err
	}
	return &binding, nil
}

func (s *Service) DeleteExternalIdentityBinding(ctx context.Context, userID, bindingID string) error {
	if userID == "" || bindingID == "" {
		return errors.New("userId and bindingId are required")
	}
	result := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", bindingID, userID).
		Delete(&model.ExternalIdentityBinding{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("binding not found")
	}
	return s.audit.Record(ctx, coreservice.AuditEvent{
		ActorType:  "admin",
		EventType:  "user.external_identity_binding.deleted",
		Result:     "success",
		TargetType: "external_identity_binding",
		TargetID:   bindingID,
		Detail: map[string]any{
			"userId": userID,
		},
	})
}

func (s *Service) UntrustUserDevice(ctx context.Context, userID, deviceID string) error {
	if userID == "" || deviceID == "" {
		return errors.New("userId and deviceId are required")
	}
	result := s.db.WithContext(ctx).
		Model(&model.Device{}).
		Where("id = ? AND user_id = ?", deviceID, userID).
		Updates(map[string]any{
			"trusted": false,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("device not found")
	}
	return s.audit.Record(ctx, coreservice.AuditEvent{
		ActorType:  "admin",
		EventType:  "user.device.untrusted",
		Result:     "success",
		TargetType: "device",
		TargetID:   deviceID,
		Detail: map[string]any{
			"userId": userID,
		},
	})
}

func (s *Service) RevokeUserSessions(ctx context.Context, userID string) error {
	if userID == "" {
		return errors.New("userId is required")
	}
	now := time.Now()
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Token{}).
			Where("user_id = ? AND revoked_at IS NULL", userID).
			Updates(map[string]any{"revoked_at": now, "revocation_note": "session_revoked_by_admin"}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&model.Session{}).Error; err != nil {
			return err
		}
		if s.deleteAuthorizationCodesByUser != nil {
			s.deleteAuthorizationCodesByUser(userID)
		}
		return nil
	}); err != nil {
		return err
	}
	return s.audit.Record(ctx, coreservice.AuditEvent{
		ActorType:  "admin",
		EventType:  "user.session.revoked_all",
		Result:     "success",
		TargetType: "user",
		TargetID:   userID,
	})
}

func (s *Service) RotateUserToken(ctx context.Context, userID string) error {
	if userID == "" {
		return errors.New("userId is required")
	}
	now := time.Now()
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Token{}).
			Where("user_id = ? AND revoked_at IS NULL", userID).
			Updates(map[string]any{"revoked_at": now, "revocation_note": "token_rotated_by_admin"}).Error; err != nil {
			return err
		}
		if s.deleteAuthorizationCodesByUser != nil {
			s.deleteAuthorizationCodesByUser(userID)
		}
		return nil
	}); err != nil {
		return err
	}
	return s.audit.Record(ctx, coreservice.AuditEvent{
		ActorType:  "admin",
		EventType:  "user.token.rotated",
		Result:     "success",
		TargetType: "user",
		TargetID:   userID,
	})
}

func coalesceString(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func coalesceInt(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func (s *Service) loadSessionUser(ctx context.Context, sessionID string) (*model.Session, *model.User, error) {
	if sessionID == "" {
		return nil, nil, errors.New("sessionId is required")
	}
	var session model.Session
	if err := s.db.WithContext(ctx).First(&session, "id = ?", sessionID).Error; err != nil {
		return nil, nil, err
	}
	if session.UserID == "" {
		return nil, nil, errors.New("session has no bound user")
	}
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", session.UserID).Error; err != nil {
		return nil, nil, err
	}
	return &session, &user, nil
}

func coalesceRoles(value, fallback []string) []string {
	if value != nil {
		return value
	}
	return fallback
}

func buildUserDevices(devices []model.Device, recentSessions []model.Session) []coreservice.UserDetailDevice {
	deviceMap := map[string]*coreservice.UserDetailDevice{}
	order := make([]string, 0)
	now := time.Now()

	for _, device := range devices {
		key := device.ID
		entry := &coreservice.UserDetailDevice{
			ID:                device.ID,
			DeviceFingerprint: device.Fingerprint,
			UserAgent:         device.UserAgent,
			Online:            false,
			Trusted:           device.Trusted,
			LastLoginIP:       device.LastLoginIP,
		}
		entry.Trusted = device.Trusted
		if device.FirstSeenAt != nil {
			first := *device.FirstSeenAt
			entry.FirstLoginAt = &first
		} else {
			first := device.CreatedAt
			entry.FirstLoginAt = &first
		}
		if !device.LastSeenAt.IsZero() {
			last := device.LastSeenAt
			entry.LastLoginAt = &last
		}
		deviceMap[key] = entry
		order = append(order, key)
	}

	for _, session := range recentSessions {
		key := session.DeviceID
		if key == "" {
			key = "session:" + session.ID
		}
		entry, ok := deviceMap[key]
		if !ok {
			entry = &coreservice.UserDetailDevice{
				ID:          key,
				UserAgent:   session.UserAgent,
				Online:      false,
				Trusted:     false,
				LastLoginIP: session.IPAddress,
			}
			deviceMap[key] = entry
			order = append(order, key)
		}
		sessionCreatedAt := session.CreatedAt
		sessionUpdatedAt := session.UpdatedAt
		if entry.FirstLoginAt == nil || sessionCreatedAt.Before(*entry.FirstLoginAt) {
			first := sessionCreatedAt
			entry.FirstLoginAt = &first
		}
		if entry.LastLoginAt == nil || sessionUpdatedAt.After(*entry.LastLoginAt) {
			last := sessionUpdatedAt
			entry.LastLoginAt = &last
			entry.LastLoginIP = session.IPAddress
			entry.UserAgent = session.UserAgent
			entry.Online = now.Sub(sessionUpdatedAt) <= 5*time.Minute
		}
	}

	result := make([]coreservice.UserDetailDevice, 0, len(order))
	for _, key := range order {
		result = append(result, *deviceMap[key])
	}
	sort.SliceStable(result, func(i, j int) bool {
		left := time.Time{}
		right := time.Time{}
		if result[i].LastLoginAt != nil {
			left = *result[i].LastLoginAt
		}
		if result[j].LastLoginAt != nil {
			right = *result[j].LastLoginAt
		}
		return left.After(right)
	})
	return result
}

func (s *Service) resolveIPLocation(ipAddress string) string {
	if s.geoip == nil {
		return ""
	}
	return s.geoip.Resolve(ipAddress)
}

func (s *Service) decorateAuditLogs(items []model.AuditLog) []coreservice.AuditLogView {
	result := make([]coreservice.AuditLogView, 0, len(items))
	for _, item := range items {
		result = append(result, coreservice.AuditLogView{
			AuditLog:   item,
			IPLocation: s.resolveIPLocation(item.IPAddress),
		})
	}
	return result
}
