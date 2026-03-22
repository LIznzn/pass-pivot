package manage

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"strings"
	"time"

	"pass-pivot/internal/config"
	"pass-pivot/internal/model"
	authservice "pass-pivot/internal/server/auth/service"
	coreservice "pass-pivot/internal/server/core/service"
	sharedauthn "pass-pivot/internal/server/shared/authn"
	sharedfido "pass-pivot/internal/server/shared/fido"
	sharedhandler "pass-pivot/internal/server/shared/handler"
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
	organizationNamePattern = regexp.MustCompile(`^[A-Za-z0-9-]+$`)
	passwordNumberPattern   = regexp.MustCompile(`[0-9]`)
	passwordUpperPattern    = regexp.MustCompile(`[A-Z]`)
	passwordLowerPattern    = regexp.MustCompile(`[a-z]`)
	passwordSymbolPattern   = regexp.MustCompile(`[^A-Za-z0-9]`)
	applicationTypeOptions  = map[string]bool{
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

func validatePasswordAgainstPolicy(password string, policy model.OrganizationPasswordPolicy) error {
	if strings.TrimSpace(password) == "" {
		return errors.New("password is required")
	}
	minLength := policy.MinLength
	if minLength <= 0 {
		minLength = 8
	}
	if len(password) < minLength {
		return errors.New("password does not meet minimum length requirement")
	}
	if policy.RequireUppercase && !passwordUpperPattern.MatchString(password) {
		return errors.New("password must include an uppercase letter")
	}
	if policy.RequireLowercase && !passwordLowerPattern.MatchString(password) {
		return errors.New("password must include a lowercase letter")
	}
	if policy.RequireNumber && !passwordNumberPattern.MatchString(password) {
		return errors.New("password must include a number")
	}
	if policy.RequireSymbol && !passwordSymbolPattern.MatchString(password) {
		return errors.New("password must include a symbol")
	}
	return nil
}

func NewService(db *gorm.DB, cfg config.Config, audit *coreservice.AuditService) *Service {
	return &Service{
		db:    db,
		cfg:   cfg,
		audit: audit,
		geoip: coreservice.NewGeoIPService("provider/geoip/resource/GeoLite2-City.mmdb"),
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

func currentUserRolesFromContext(ctx context.Context) []string {
	identity, ok := sharedhandler.AccessTokenIdentityFromContext(ctx)
	if !ok || identity.User == nil {
		return nil
	}
	return identity.User.Roles
}

func currentUserIDFromContext(ctx context.Context) string {
	identity, ok := sharedhandler.AccessTokenIdentityFromContext(ctx)
	if !ok || identity.User == nil {
		return ""
	}
	return identity.User.ID
}

func (s *Service) ensureCanManageOrganization(ctx context.Context, organizationID string) error {
	roles := currentUserRolesFromContext(ctx)
	if len(roles) == 0 {
		return nil
	}
	if sharedhandler.RolesContainOrganizationManagementRole(roles, organizationID) {
		return nil
	}
	return errors.New("organization management role is required")
}

func (s *Service) ensureCanDeleteOrganization(ctx context.Context, organizationID string) error {
	roles := currentUserRolesFromContext(ctx)
	if len(roles) == 0 {
		return nil
	}
	if sharedhandler.RolesContainOrganizationOwnerRole(roles, organizationID) {
		return nil
	}
	return errors.New("organization owner role is required")
}

func (s *Service) ensureCanManageInternalOrganization(ctx context.Context) error {
	return s.ensureCanManageOrganization(ctx, s.cfg.InternalOrganizationID)
}

func (s *Service) managedOrganizationIDs(ctx context.Context) []string {
	return sharedhandler.RolesManagedOrganizationIDs(currentUserRolesFromContext(ctx))
}

func (s *Service) loadProjectForManagement(ctx context.Context, projectID string) (*model.Project, error) {
	var project model.Project
	if err := s.db.WithContext(ctx).First(&project, "id = ?", projectID).Error; err != nil {
		return nil, err
	}
	if err := s.ensureCanManageOrganization(ctx, project.OrganizationID); err != nil {
		return nil, err
	}
	return &project, nil
}

func (s *Service) loadApplicationForManagement(ctx context.Context, applicationID string) (*model.Application, *model.Project, error) {
	var app model.Application
	if err := s.db.WithContext(ctx).First(&app, "id = ?", applicationID).Error; err != nil {
		return nil, nil, err
	}
	project, err := s.loadProjectForManagement(ctx, app.ProjectID)
	if err != nil {
		return nil, nil, err
	}
	return &app, project, nil
}

func (s *Service) loadUserForManagement(ctx context.Context, userID string) (*model.User, error) {
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}
	if err := s.ensureCanManageOrganization(ctx, user.OrganizationID); err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Service) ListOrganizations(ctx context.Context) ([]model.Organization, error) {
	var items []model.Organization
	if err := s.db.WithContext(ctx).Preload("Projects.Applications").Preload("Users").Preload("Roles").Find(&items).Error; err != nil {
		return nil, err
	}
	managedOrganizationIDs := s.managedOrganizationIDs(ctx)
	if len(managedOrganizationIDs) > 0 {
		allowed := make(map[string]struct{}, len(managedOrganizationIDs))
		for _, organizationID := range managedOrganizationIDs {
			allowed[organizationID] = struct{}{}
		}
		filtered := make([]model.Organization, 0, len(items))
		for _, item := range items {
			if _, ok := allowed[item.ID]; ok {
				filtered = append(filtered, item)
			}
		}
		items = filtered
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
	application.Metadata = coreservice.NormalizeApplicationMetadata(application.Metadata, nil)
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
	organization.Metadata = coreservice.NormalizeOrganizationMetadata(organization.Metadata, nil)
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
		OrganizationID:           organization.ID,
		OrganizationName:         organization.Name,
		DisplayName:              organization.Metadata[coreservice.OrganizationMetadataDisplayName],
		OrganizationDisplayNames: coreservice.BuildOrganizationDisplayNameMap(organization.Metadata),
		WebsiteURL:               organization.Metadata[coreservice.OrganizationMetadataWebsiteURL],
		TermsOfServiceURL:        organization.Metadata[coreservice.OrganizationMetadataTermsOfServiceURL],
		PrivacyPolicyURL:         organization.Metadata[coreservice.OrganizationMetadataPrivacyPolicyURL],
		ProjectID:                project.ID,
		ProjectName:              project.Name,
		ApplicationID:            application.ID,
		ApplicationName:          application.Name,
		ApplicationDisplayNames:  coreservice.BuildApplicationDisplayNameMap(application.Metadata),
		ExternalIDPs:             publicProviders,
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
	org.Name = strings.TrimSpace(org.Name)
	org.Description = strings.TrimSpace(org.Description)
	if org.Name == "" {
		return nil, errors.New("name is required")
	}
	if !organizationNamePattern.MatchString(org.Name) {
		return nil, errors.New("name must contain only letters, numbers, and hyphens")
	}
	if err := s.ensureCanManageInternalOrganization(ctx); err != nil {
		return nil, err
	}
	if strings.TrimSpace(org.ID) == "" {
		org.ID = uuid.NewString()
	}
	if strings.TrimSpace(org.Status) == "" {
		org.Status = "active"
	}
	org.Metadata = coreservice.NormalizeOrganizationMetadata(org.Metadata, nil)
	ownerRoleName := sharedhandler.OrganizationOwnerRoleName(org.ID)
	adminRoleName := sharedhandler.OrganizationAdminRoleName(org.ID)
	creatorUserID := currentUserIDFromContext(ctx)
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&org).Error; err != nil {
			return err
		}
		if err := s.createOrganizationSigningKey(tx, org.ID); err != nil {
			return err
		}
		organizationOwnerRole := model.Role{
			OrganizationID: s.cfg.InternalOrganizationID,
			Name:           ownerRoleName,
			Type:           "user",
			Description:    "Organization owner role for " + org.ID,
		}
		if err := tx.Where("organization_id = ? AND name = ?", s.cfg.InternalOrganizationID, ownerRoleName).
			FirstOrCreate(&organizationOwnerRole).Error; err != nil {
			return err
		}
		organizationAdminRole := model.Role{
			OrganizationID: s.cfg.InternalOrganizationID,
			Name:           adminRoleName,
			Type:           "user",
			Description:    "Organization admin role for " + org.ID,
		}
		if err := tx.Where("organization_id = ? AND name = ?", s.cfg.InternalOrganizationID, adminRoleName).
			FirstOrCreate(&organizationAdminRole).Error; err != nil {
			return err
		}
		if creatorUserID != "" {
			var creator model.User
			if err := tx.First(&creator, "id = ?", creatorUserID).Error; err != nil {
				return err
			}
			if !containsString(creator.Roles, ownerRoleName) {
				creator.Roles = append(creator.Roles, ownerRoleName)
				creator.Roles = normalizeRoleNames(creator.Roles)
				if err := tx.Model(&creator).Updates(model.User{Roles: creator.Roles}).Error; err != nil {
					return err
				}
			}
		}
		if org.ConsoleSettings == nil {
			return nil
		}
		settings := coreservice.NormalizeOrganizationConsoleSettings(org.ConsoleSettings)
		return tx.Model(&org).
			Select("SupportEmail", "LogoURL", "Domains", "LoginPolicy", "PasswordPolicy", "MFAPolicy").
			Updates(model.Organization{
				SupportEmail:   settings.SupportEmail,
				LogoURL:        settings.LogoURL,
				Domains:        settings.Domains,
				LoginPolicy:    settings.LoginPolicy,
				PasswordPolicy: settings.PasswordPolicy,
				MFAPolicy:      settings.MFAPolicy,
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
	org.Name = strings.TrimSpace(org.Name)
	org.Description = strings.TrimSpace(org.Description)
	if err := s.ensureCanManageOrganization(ctx, org.ID); err != nil {
		return nil, err
	}
	var existing model.Organization
	if err := s.db.WithContext(ctx).First(&existing, "id = ?", org.ID).Error; err != nil {
		return nil, err
	}
	metadata := normalizeMetadata(org.Metadata, existing.Metadata)
	metadata = coreservice.NormalizeOrganizationMetadata(metadata, nil)
	delete(metadata, "console_settings")
	updateModel := model.Organization{
		Name:              coalesceString(org.Name, existing.Name),
		Description:       coalesceString(org.Description, existing.Description),
		Metadata:          metadata,
		AllowJWTAccess:    org.AllowJWTAccess,
		AllowBasicAccess:  org.AllowBasicAccess,
		AllowNoAuthAccess: org.AllowNoAuthAccess,
		AllowRefreshToken: org.AllowRefreshToken,
		AllowAuthCode:     org.AllowAuthCode,
		AllowPKCE:         org.AllowPKCE,
	}
	selectedFields := []string{
		"Name",
		"Description",
		"Metadata",
		"AllowJWTAccess",
		"AllowBasicAccess",
		"AllowNoAuthAccess",
		"AllowRefreshToken",
		"AllowAuthCode",
		"AllowPKCE",
	}
	if org.ConsoleSettings != nil {
		settings := coreservice.NormalizeOrganizationConsoleSettings(org.ConsoleSettings)
		updateModel.SupportEmail = settings.SupportEmail
		updateModel.LogoURL = settings.LogoURL
		updateModel.Domains = settings.Domains
		updateModel.LoginPolicy = settings.LoginPolicy
		updateModel.PasswordPolicy = settings.PasswordPolicy
		updateModel.MFAPolicy = settings.MFAPolicy
		selectedFields = append(selectedFields,
			"SupportEmail",
			"LogoURL",
			"Domains",
			"LoginPolicy",
			"PasswordPolicy",
			"MFAPolicy",
		)
	}
	if !organizationNamePattern.MatchString(updateModel.Name) {
		return nil, errors.New("name must contain only letters, numbers, and hyphens")
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Model(&existing).Select(selectedFields).Updates(updateModel).Error
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
	if err := s.ensureCanManageOrganization(ctx, organizationID); err != nil {
		return err
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
	if err := s.ensureCanDeleteOrganization(ctx, organizationID); err != nil {
		return err
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
		if len(appIDs) > 0 {
			if err := tx.Where("application_id IN ?", appIDs).Delete(&model.AuthorizationCode{}).Error; err != nil {
				return err
			}
			if err := tx.Where("application_id IN ?", appIDs).Delete(&model.Token{}).Error; err != nil {
				return err
			}
			if err := tx.Where("application_id IN ?", appIDs).Delete(&model.ApplicationKey{}).Error; err != nil {
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
		if err := tx.Where("organization_id = ?", organization.ID).Delete(&model.OrganizationSigningKey{}).Error; err != nil {
			return err
		}
		organizationRoleNames := []string{
			sharedhandler.OrganizationOwnerRoleName(organization.ID),
			sharedhandler.OrganizationAdminRoleName(organization.ID),
		}
		var internalRoles []model.Role
		if err := tx.Where("organization_id = ? AND name IN ?", s.cfg.InternalOrganizationID, organizationRoleNames).Find(&internalRoles).Error; err != nil {
			return err
		}
		if len(internalRoles) > 0 {
			roleIDs := make([]string, 0, len(internalRoles))
			for _, role := range internalRoles {
				roleIDs = append(roleIDs, role.ID)
			}
			if err := tx.Where("role_id IN ?", roleIDs).Delete(&model.Policy{}).Error; err != nil {
				return err
			}
			if err := tx.Where("id IN ?", roleIDs).Delete(&model.Role{}).Error; err != nil {
				return err
			}
		}
		var allUsers []model.User
		if err := tx.Find(&allUsers).Error; err != nil {
			return err
		}
		for _, user := range allUsers {
			nextRoles := make([]string, 0, len(user.Roles))
			changed := false
			for _, roleName := range user.Roles {
				if containsString(organizationRoleNames, roleName) {
					changed = true
					continue
				}
				nextRoles = append(nextRoles, roleName)
			}
			if changed {
				if err := tx.Model(&user).Updates(model.User{Roles: normalizeRoleNames(nextRoles)}).Error; err != nil {
					return err
				}
			}
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
		organizations[index].Metadata = coreservice.NormalizeOrganizationMetadata(current.Metadata, nil)
		if legacy := coreservice.ParseLegacyOrganizationConsoleSettings(current); legacy != nil {
			settings := coreservice.NormalizeOrganizationConsoleSettings(legacy)
			organizations[index].ConsoleSettings = &settings
			continue
		}
		settings := coreservice.NormalizeOrganizationConsoleSettings(&model.OrganizationSetting{
			SupportEmail:   current.SupportEmail,
			LogoURL:        current.LogoURL,
			Domains:        current.Domains,
			LoginPolicy:    current.LoginPolicy,
			PasswordPolicy: current.PasswordPolicy,
			MFAPolicy:      current.MFAPolicy,
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
	user, err := s.loadUserForManagement(ctx, userID)
	if err != nil {
		return err
	}
	now := time.Now()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(user).Update("status", "disabled").Error; err != nil {
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
	user, err := s.loadUserForManagement(ctx, userID)
	if err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Model(user).Update("status", "active").Error; err != nil {
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
	user, err := s.loadUserForManagement(ctx, userID)
	if err != nil {
		return err
	}
	hash, err := util.HashSecret(password)
	if err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Model(user).Update("password_hash", hash).Error; err != nil {
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
	user, err := s.loadUserForManagement(ctx, userID)
	if err != nil {
		return "", err
	}
	newUKID, err := util.RandomToken(18)
	if err != nil {
		return "", err
	}
	now := time.Now()
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(user).Update("current_ukid", newUKID).Error; err != nil {
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
	if err := s.ensureCanManageOrganization(ctx, project.OrganizationID); err != nil {
		return nil, err
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
	existing, err := s.loadProjectForManagement(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{
		"name":             coalesceString(project.Name, existing.Name),
		"description":      coalesceString(project.Description, existing.Description),
		"user_acl_enabled": project.UserACLEnabled,
	}
	if err := s.db.WithContext(ctx).Model(existing).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).First(existing, "id = ?", project.ID).Error; err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *Service) DisableProject(ctx context.Context, projectID string) error {
	project, err := s.loadProjectForManagement(ctx, projectID)
	if err != nil {
		return err
	}
	now := time.Now()
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(project).Update("status", "disabled").Error; err != nil {
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
		project, err := s.loadProjectForManagement(ctx, projectID)
		if err != nil {
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
		if err := tx.Delete(project).Error; err != nil {
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
	project, err := s.loadProjectForManagement(ctx, app.ProjectID)
	if err != nil {
		return nil, err
	}
	if app.ID == "" {
		app.ID = uuid.NewString()
	}
	app.Metadata = coreservice.NormalizeApplicationMetadata(app.Metadata, nil)
	applyApplicationDefaults(&app)
	if err := validateApplicationProtocol(app); err != nil {
		return nil, err
	}
	if app.AccessTokenTTLMinutes <= 0 || app.RefreshTokenTTLHours <= 0 {
		return nil, errors.New("accessTokenTTLMinutes and refreshTokenTTLHours are required")
	}
	validatedRoles, err := s.validateRoleAssignments(ctx, project.OrganizationID, app.Roles, "application")
	if err != nil {
		return nil, err
	}
	app.Roles = validatedRoles
	if strings.TrimSpace(app.Status) == "" {
		app.Status = "active"
	}
	generatedPrivateKey := ""
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if app.ClientAuthenticationType == "private_key_jwt" {
			publicKey, privateKey, err := s.rotateApplicationClientKey(tx, app.ID)
			if err != nil {
				return err
			}
			app.PublicKey = publicKey
			generatedPrivateKey = privateKey
		} else {
			app.PublicKey = ""
		}
		return tx.Create(&app).Error
	}); err != nil {
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
	existing, project, err := s.loadApplicationForManagement(ctx, app.ID)
	if err != nil {
		return nil, err
	}
	updateModel := model.Application{
		Name:                     coalesceString(app.Name, existing.Name),
		Metadata:                 coreservice.NormalizeApplicationMetadata(app.Metadata, existing.Metadata),
		Description:              coalesceString(app.Description, existing.Description),
		RedirectURIs:             coalesceString(app.RedirectURIs, existing.RedirectURIs),
		ApplicationType:          coalesceString(app.ApplicationType, existing.ApplicationType),
		GrantType:                coalesceGrantTypes(app.GrantType, existing.GrantType),
		EnableRefreshToken:       app.EnableRefreshToken,
		ClientAuthenticationType: coalesceString(app.ClientAuthenticationType, existing.ClientAuthenticationType),
		TokenType:                coalesceTokenTypes(app.TokenType, existing.TokenType),
		Roles:                    coalesceApplicationRoles(app.Roles, existing.Roles),
		AccessTokenTTLMinutes:    coalesceInt(app.AccessTokenTTLMinutes, existing.AccessTokenTTLMinutes),
		RefreshTokenTTLHours:     coalesceInt(app.RefreshTokenTTLHours, existing.RefreshTokenTTLHours),
	}
	candidate := *existing
	candidate.Name = updateModel.Name
	candidate.Metadata = updateModel.Metadata
	candidate.Description = updateModel.Description
	candidate.RedirectURIs = updateModel.RedirectURIs
	candidate.ApplicationType = updateModel.ApplicationType
	candidate.GrantType = updateModel.GrantType
	candidate.EnableRefreshToken = updateModel.EnableRefreshToken
	candidate.ClientAuthenticationType = updateModel.ClientAuthenticationType
	candidate.TokenType = updateModel.TokenType
	candidate.Roles = updateModel.Roles
	applyApplicationDefaults(&candidate)
	validatedRoles, err := s.validateRoleAssignments(ctx, project.OrganizationID, candidate.Roles, "application")
	if err != nil {
		return nil, err
	}
	candidate.Roles = validatedRoles
	if err := validateApplicationProtocol(candidate); err != nil {
		return nil, err
	}
	generatedPrivateKey := ""
	updateModel.ApplicationType = candidate.ApplicationType
	updateModel.GrantType = candidate.GrantType
	updateModel.EnableRefreshToken = candidate.EnableRefreshToken
	updateModel.ClientAuthenticationType = candidate.ClientAuthenticationType
	updateModel.TokenType = candidate.TokenType
	updateModel.Roles = candidate.Roles
	selectedFields := []string{
		"Name",
		"Metadata",
		"Description",
		"RedirectURIs",
		"ApplicationType",
		"GrantType",
		"EnableRefreshToken",
		"ClientAuthenticationType",
		"TokenType",
		"Roles",
		"AccessTokenTTLMinutes",
		"RefreshTokenTTLHours",
	}
	if candidate.ClientAuthenticationType == "private_key_jwt" || strings.TrimSpace(existing.PublicKey) != "" {
		selectedFields = append(selectedFields, "PublicKey")
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if candidate.ClientAuthenticationType == "private_key_jwt" && strings.TrimSpace(existing.PublicKey) == "" {
			publicKey, privateKey, err := s.rotateApplicationClientKey(tx, existing.ID)
			if err != nil {
				return err
			}
			updateModel.PublicKey = publicKey
			generatedPrivateKey = privateKey
		} else if candidate.ClientAuthenticationType != "private_key_jwt" {
			if err := s.deactivateApplicationClientKeys(tx, existing.ID); err != nil {
				return err
			}
			updateModel.PublicKey = ""
		}
		if err := tx.Model(existing).Select(selectedFields).Updates(updateModel).Error; err != nil {
			return err
		}
		return tx.First(existing, "id = ?", app.ID).Error
	}); err != nil {
		return nil, err
	}
	return &coreservice.ApplicationMutationResult{
		Application:         *existing,
		GeneratedPrivateKey: generatedPrivateKey,
	}, nil
}

func (s *Service) DisableApplication(ctx context.Context, applicationID string) error {
	app, _, err := s.loadApplicationForManagement(ctx, applicationID)
	if err != nil {
		return err
	}
	now := time.Now()
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(app).Update("status", "disabled").Error; err != nil {
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
		app, _, err := s.loadApplicationForManagement(ctx, applicationID)
		if err != nil {
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
		if err := tx.Where("application_id = ?", app.ID).Delete(&model.ApplicationKey{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(app).Error; err != nil {
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
	privateKey := ""
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		publicKey, generatedPrivateKey, err := s.rotateApplicationClientKey(tx, app.ID)
		if err != nil {
			return err
		}
		privateKey = generatedPrivateKey
		if err := tx.Model(&app).Updates(map[string]any{
			"public_key": publicKey,
		}).Error; err != nil {
			return err
		}
		return tx.First(&app, "id = ?", applicationID).Error
	}); err != nil {
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

func (s *Service) createOrganizationSigningKey(tx *gorm.DB, organizationID string) error {
	record, err := authservice.NewOrganizationSigningKey(organizationID)
	if err != nil {
		return err
	}
	return tx.Create(record).Error
}

func (s *Service) deactivateApplicationClientKeys(tx *gorm.DB, applicationID string) error {
	return tx.Model(&model.ApplicationKey{}).
		Where("application_id = ? AND status = ?", applicationID, "active").
		Update("status", "inactive").Error
}

func (s *Service) rotateApplicationClientKey(tx *gorm.DB, applicationID string) (string, string, error) {
	if err := s.deactivateApplicationClientKeys(tx, applicationID); err != nil {
		return "", "", err
	}
	record, generatedPrivateKey, err := authservice.NewApplicationClientKey(applicationID)
	if err != nil {
		return "", "", err
	}
	if err := tx.Create(record).Error; err != nil {
		return "", "", err
	}
	return record.PublicKeyBase64, generatedPrivateKey, nil
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
			if grantType != "authorization_code_pkce" && grantType != "device_code" && grantType != "password" && grantType != "implicit" {
				return errors.New("clientAuthenticationType=none is only allowed for authorization_code_pkce, device_code, password, or implicit")
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
		if err := s.ensureCanManageOrganization(ctx, organizationID); err != nil {
			return nil, err
		}
		query = query.Where("organization_id = ?", organizationID)
	} else if managedOrganizationIDs := s.managedOrganizationIDs(ctx); len(managedOrganizationIDs) > 0 {
		query = query.Where("organization_id IN ?", managedOrganizationIDs)
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
	project, err := s.loadProjectForManagement(ctx, projectID)
	if err != nil {
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
		project, err := s.loadProjectForManagement(ctx, projectID)
		if err != nil {
			return nil, err
		}
		query = query.Where("project_id = ?", project.ID)
	} else if managedOrganizationIDs := s.managedOrganizationIDs(ctx); len(managedOrganizationIDs) > 0 {
		var projectIDs []string
		if err := s.db.WithContext(ctx).Model(&model.Project{}).Where("organization_id IN ?", managedOrganizationIDs).Pluck("id", &projectIDs).Error; err != nil {
			return nil, err
		}
		if len(projectIDs) == 0 {
			return []model.Application{}, nil
		}
		query = query.Where("project_id IN ?", projectIDs)
	}
	err := query.Find(&items).Error
	for index := range items {
		items[index].Metadata = coreservice.NormalizeApplicationMetadata(items[index].Metadata, nil)
	}
	return items, err
}

func (s *Service) ListUsers(ctx context.Context, organizationID string) ([]model.User, error) {
	var items []model.User
	query := s.db.WithContext(ctx)
	if organizationID != "" {
		if err := s.ensureCanManageOrganization(ctx, organizationID); err != nil {
			return nil, err
		}
		query = query.Where("organization_id = ?", organizationID)
	} else if managedOrganizationIDs := s.managedOrganizationIDs(ctx); len(managedOrganizationIDs) > 0 {
		query = query.Where("organization_id IN ?", managedOrganizationIDs)
	}
	err := query.Find(&items).Error
	return items, err
}

func (s *Service) GetUserDetail(ctx context.Context, userID string) (*coreservice.UserDetailData, error) {
	if userID == "" {
		return nil, errors.New("userId is required")
	}
	user, err := s.loadUserForManagement(ctx, userID)
	if err != nil {
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
		User:               *user,
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
	if err := s.ensureCanManageOrganization(ctx, user.OrganizationID); err != nil {
		return nil, err
	}
	if user.Email == "" && user.PhoneNumber == "" {
		return nil, errors.New("email or phoneNumber is required")
	}
	if password == "" {
		return nil, errors.New("password is required")
	}
	var organization model.Organization
	if err := s.db.WithContext(ctx).Select("id", "password_policy").First(&organization, "id = ?", user.OrganizationID).Error; err != nil {
		return nil, err
	}
	if err := validatePasswordAgainstPolicy(password, organization.PasswordPolicy); err != nil {
		return nil, err
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
	existing, err := s.loadUserForManagement(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	nextRoles := coalesceRoles(user.Roles, existing.Roles)
	validatedRoles, err := s.validateRoleAssignments(ctx, existing.OrganizationID, nextRoles, "user")
	if err != nil {
		return nil, err
	}
	updateModel := model.User{
		Username:    coalesceString(user.Username, existing.Username),
		Name:        coalesceString(user.Name, existing.Name),
		Email:       coalesceString(user.Email, existing.Email),
		PhoneNumber: coalesceString(user.PhoneNumber, existing.PhoneNumber),
		Roles:       validatedRoles,
		Status:      coalesceString(user.Status, existing.Status),
	}
	if err := s.db.WithContext(ctx).
		Model(existing).
		Select("Username", "Name", "Email", "PhoneNumber", "Roles", "Status").
		Updates(updateModel).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).First(existing, "id = ?", user.ID).Error; err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *Service) SetUserMFAMethod(ctx context.Context, userID, method string, enabled bool) error {
	if userID == "" {
		return errors.New("userId is required")
	}
	if method != "email_code" && method != "sms_code" && method != "webauthn" && method != "u2f" && method != "mfa" {
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
			return errors.New("passkey credential is required before enabling webauthn")
		}
	}
	if method == "u2f" {
		label = "安全密钥"
		target = ""
		var count int64
		if err := s.db.WithContext(ctx).
			Model(&model.SecureKey{}).
			Where("user_id = ? AND u2f_enable = ? AND deleted_at IS NULL", user.ID, true).
			Count(&count).Error; err != nil {
			return err
		}
		if enabled && count == 0 {
			return errors.New("securekey is required before enabling u2f")
		}
	}
	if method == "mfa" {
		label = "两步验证"
		target = ""
	}
	if enabled && target == "" {
		if method != "webauthn" && method != "u2f" && method != "mfa" {
			return errors.New("target is required before enabling method")
		}
	}

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var enrollment model.MFAEnrollment
		err := tx.
			Where("user_id = ? AND method = ?", user.ID, method).
			Order("created_at desc").
			First(&enrollment).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status := "disabled"
			if enabled {
				status = "active"
			}
			enrollment = model.MFAEnrollment{
				OrganizationID: user.OrganizationID,
				UserID:         user.ID,
				Method:         method,
				Label:          label,
				Target:         target,
				Status:         status,
			}
			if err := tx.Create(&enrollment).Error; err != nil {
				return err
			}
		} else {
			status := "disabled"
			if enabled {
				status = "active"
			}
			if err := tx.Model(&enrollment).Updates(map[string]any{
				"method": method,
				"label":  label,
				"target": target,
				"status": status,
			}).Error; err != nil {
				return err
			}
		}
		if method == "mfa" && enabled {
			return s.ensureUserRecoveryCodesWithDB(ctx, tx, user)
		}
		if method == "mfa" && !enabled {
			if err := tx.
				Where("user_id = ? AND method IN ?", user.ID, []string{"mfa", "totp", "email_code", "sms_code", "u2f"}).
				Delete(&model.MFAEnrollment{}).Error; err != nil {
				return err
			}
			if err := tx.
				Where("user_id = ?", user.ID).
				Delete(&model.MFARecoveryCode{}).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
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

func (s *Service) ensureUserRecoveryCodes(ctx context.Context, user model.User) error {
	return s.ensureUserRecoveryCodesWithDB(ctx, s.db, user)
}

func (s *Service) ensureUserRecoveryCodesWithDB(ctx context.Context, db *gorm.DB, user model.User) error {
	var available int64
	if err := db.WithContext(ctx).Model(&model.MFARecoveryCode{}).
		Where("user_id = ? AND consumed_at IS NULL AND deleted_at IS NULL AND code <> ''", user.ID).
		Count(&available).Error; err != nil {
		return err
	}
	if available > 0 {
		return nil
	}
	codes := sharedauthn.RecoveryCodes()
	_ = db.WithContext(ctx).Where("user_id = ?", user.ID).Delete(&model.MFARecoveryCode{}).Error
	for _, code := range codes {
		entry := model.MFARecoveryCode{
			UserID:         user.ID,
			OrganizationID: user.OrganizationID,
			Code:           code,
		}
		if err := db.WithContext(ctx).Create(&entry).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) QueryUserRecoveryCodes(ctx context.Context, userID string) ([]string, error) {
	if userID == "" {
		return nil, errors.New("userId is required")
	}
	if _, err := s.loadUserForManagement(ctx, userID); err != nil {
		return nil, err
	}
	var codes []model.MFARecoveryCode
	if err := s.db.WithContext(ctx).
		Where("user_id = ? AND consumed_at IS NULL AND deleted_at IS NULL", userID).
		Order("created_at asc").
		Find(&codes).Error; err != nil {
		return nil, err
	}
	items := make([]string, 0, len(codes))
	for _, item := range codes {
		if strings.TrimSpace(item.Code) == "" {
			continue
		}
		items = append(items, item.Code)
	}
	return items, nil
}

func (s *Service) DeleteUserMFAEnrollments(ctx context.Context, userID, method string) error {
	if userID == "" || method == "" {
		return errors.New("userId and method are required")
	}
	if _, err := s.loadUserForManagement(ctx, userID); err != nil {
		return err
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
	user, err := s.loadUserForManagement(ctx, userID)
	if err != nil {
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
		return sharedfido.SyncCredentialEnrollments(ctx, tx, *user)
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

func (s *Service) updateSecureKeyIdentifier(ctx context.Context, userID, credentialID, identifier string) (string, error) {
	if userID == "" || credentialID == "" {
		return "", errors.New("userId and credentialId are required")
	}
	trimmedIdentifier := strings.TrimSpace(identifier)
	if trimmedIdentifier == "" {
		return "", errors.New("identifier is required")
	}
	if len(trimmedIdentifier) > 128 {
		return "", errors.New("identifier is too long")
	}
	result := s.db.WithContext(ctx).
		Model(&model.SecureKey{}).
		Where("id = ? AND user_id = ?", credentialID, userID).
		Update("identifier", trimmedIdentifier)
	if result.Error != nil {
		return "", result.Error
	}
	if result.RowsAffected == 0 {
		return "", errors.New("securekey not found")
	}
	return trimmedIdentifier, nil
}

func (s *Service) UpdateUserSecureKey(ctx context.Context, userID, credentialID, identifier string) error {
	user, err := s.loadUserForManagement(ctx, userID)
	if err != nil {
		return err
	}
	trimmedIdentifier, err := s.updateSecureKeyIdentifier(ctx, userID, credentialID, identifier)
	if err != nil {
		return err
	}
	return s.audit.Record(ctx, coreservice.AuditEvent{
		OrganizationID: user.OrganizationID,
		ActorType:      "admin",
		EventType:      "user.securekey.updated",
		Result:         "success",
		TargetType:     "credential",
		TargetID:       credentialID,
		Detail: map[string]any{
			"userId":     userID,
			"identifier": trimmedIdentifier,
		},
	})
}

func (s *Service) DeleteUserRecoveryCodes(ctx context.Context, userID string) error {
	if userID == "" {
		return errors.New("userId is required")
	}
	if _, err := s.loadUserForManagement(ctx, userID); err != nil {
		return err
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
		for _, user := range users {
			if err := s.ensureCanManageOrganization(ctx, user.OrganizationID); err != nil {
				return err
			}
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
	if sharedhandler.RolesContainAnyOrganizationManagementRole(identity.User.Roles) {
		return identity, nil
	}
	return nil, errors.New("organization management role is required")
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

func (s *Service) UpdateCurrentUserSecureKey(ctx context.Context, sessionID, credentialID, identifier string) error {
	session, user, err := s.loadSessionUser(ctx, sessionID)
	if err != nil {
		return err
	}
	trimmedIdentifier, err := s.updateSecureKeyIdentifier(ctx, user.ID, credentialID, identifier)
	if err != nil {
		return err
	}
	return s.audit.Record(ctx, coreservice.AuditEvent{
		OrganizationID: user.OrganizationID,
		ApplicationID:  session.ApplicationID,
		ActorType:      "user",
		EventType:      "user.securekey.updated",
		Result:         "success",
		TargetType:     "credential",
		TargetID:       credentialID,
		Detail: map[string]any{
			"userId":     user.ID,
			"identifier": trimmedIdentifier,
		},
	})
}

func (s *Service) QueryCurrentUserRecoveryCodes(ctx context.Context, sessionID string) ([]string, error) {
	_, user, err := s.loadSessionUser(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return s.QueryUserRecoveryCodes(ctx, user.ID)
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
	var organization model.Organization
	if err := s.db.WithContext(ctx).Select("id", "password_policy").First(&organization, "id = ?", user.OrganizationID).Error; err != nil {
		return err
	}
	if err := validatePasswordAgainstPolicy(newPassword, organization.PasswordPolicy); err != nil {
		return err
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
		if err := s.ensureCanManageOrganization(ctx, organizationID); err != nil {
			return nil, err
		}
		query = query.Where("organization_id = ?", organizationID)
	} else if managedOrganizationIDs := s.managedOrganizationIDs(ctx); len(managedOrganizationIDs) > 0 {
		query = query.Where("organization_id IN ?", managedOrganizationIDs)
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
		if err := s.ensureCanManageOrganization(ctx, organizationID); err != nil {
			return nil, err
		}
		query = query.Where("organization_id = ?", organizationID)
	} else if managedOrganizationIDs := s.managedOrganizationIDs(ctx); len(managedOrganizationIDs) > 0 {
		query = query.Where("organization_id IN ?", managedOrganizationIDs)
	}
	err := query.Find(&items).Error
	return items, err
}

func (s *Service) CreateExternalIDP(ctx context.Context, provider model.ExternalIDP) (*model.ExternalIDP, error) {
	if provider.OrganizationID == "" || provider.Name == "" {
		return nil, errors.New("organizationId and name are required")
	}
	if err := s.ensureCanManageOrganization(ctx, provider.OrganizationID); err != nil {
		return nil, err
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
	if err := s.ensureCanManageOrganization(ctx, existing.OrganizationID); err != nil {
		return nil, err
	}
	updateModel := model.ExternalIDP{
		Protocol:         coalesceString(provider.Protocol, existing.Protocol),
		Name:             coalesceString(provider.Name, existing.Name),
		Issuer:           coalesceString(provider.Issuer, existing.Issuer),
		ClientID:         coalesceString(provider.ClientID, existing.ClientID),
		AuthorizationURL: coalesceString(provider.AuthorizationURL, existing.AuthorizationURL),
		TokenURL:         coalesceString(provider.TokenURL, existing.TokenURL),
		UserInfoURL:      coalesceString(provider.UserInfoURL, existing.UserInfoURL),
		JWKSURL:          coalesceString(provider.JWKSURL, existing.JWKSURL),
		Scopes:           coalesceString(provider.Scopes, existing.Scopes),
		Metadata:         normalizeMetadata(provider.Metadata, existing.Metadata),
	}
	selectedFields := []string{
		"Protocol",
		"Name",
		"Issuer",
		"ClientID",
		"AuthorizationURL",
		"TokenURL",
		"UserInfoURL",
		"JWKSURL",
		"Scopes",
		"Metadata",
	}
	if provider.ClientSecret != "" {
		updateModel.ClientSecret = provider.ClientSecret
		selectedFields = append(selectedFields, "ClientSecret")
	}
	if err := s.db.WithContext(ctx).Model(&existing).Select(selectedFields).Updates(updateModel).Error; err != nil {
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
	if err := s.ensureCanManageOrganization(ctx, binding.OrganizationID); err != nil {
		return nil, err
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
	if _, err := s.loadUserForManagement(ctx, userID); err != nil {
		return err
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
	if _, err := s.loadUserForManagement(ctx, userID); err != nil {
		return err
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
	if _, err := s.loadUserForManagement(ctx, userID); err != nil {
		return err
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
	if _, err := s.loadUserForManagement(ctx, userID); err != nil {
		return err
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
