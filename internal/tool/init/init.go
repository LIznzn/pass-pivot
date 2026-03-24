package toolinit

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"pass-pivot/internal/config"
	"pass-pivot/internal/db"
	"pass-pivot/internal/model"
	authservice "pass-pivot/internal/server/auth/service"
	coreservice "pass-pivot/internal/server/core/service"
	sharedhandler "pass-pivot/internal/server/shared/handler"
	"pass-pivot/utils"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var identifierPattern = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

type Options struct {
	Force bool
}

func Run(ctx context.Context, cfg config.Config, opts Options) error {
	if err := ensureSystemBootstrapConfig(&cfg); err != nil {
		return err
	}
	if err := recreateDatabase(cfg, opts); err != nil {
		return err
	}
	database, err := db.Open(
		cfg.DatabaseDriver,
		cfg.DatabaseHost,
		cfg.DatabasePort,
		cfg.DatabaseUsername,
		cfg.DatabasePassword,
		cfg.DatabaseSchema,
	)
	if err != nil {
		return err
	}
	if err := initSchema(database); err != nil {
		return err
	}
	return seed(ctx, database, cfg)
}

func ensureSystemBootstrapConfig(cfg *config.Config) error {
	rootValues, err := readEnvFile(".init")
	if err != nil {
		return err
	}
	ensureUUID := func(key string, target *string) {
		value := strings.TrimSpace(rootValues[key])
		if value == "" {
			value = uuid.NewString()
			rootValues[key] = value
		}
		*target = value
	}
	ensureUUID("PPVT_INTERNAL_ORGANIZATION_ID", &cfg.InternalOrganizationID)
	ensureUUID("PPVT_SYSTEM_PROJECT_ID", &cfg.SystemProjectID)
	ensureUUID("PPVT_MANAGE_API_APPLICATION_ID", &cfg.ManageAPIApplicationID)
	ensureUUID("PPVT_USER_API_APPLICATION_ID", &cfg.UserAPIApplicationID)
	ensureUUID("PPVT_AUTHN_API_APPLICATION_ID", &cfg.AuthnAPIApplicationID)
	ensureUUID("PPVT_AUTHZ_API_APPLICATION_ID", &cfg.AuthzAPIApplicationID)
	ensureUUID("PPVT_CONSOLE_APPLICATION_ID", &cfg.ConsoleApplicationID)
	ensureUUID("PPVT_PORTAL_APPLICATION_ID", &cfg.PortalApplicationID)
	ensureUUID("PPVT_CONSOLE_ADMIN_ROLE_ID", &cfg.ConsoleAdminRoleID)
	ensureUUID("PPVT_ADMIN_USER_ID", &cfg.AdminUserID)
	if err := writeEnvFile(".init", rootValues); err != nil {
		return err
	}

	consoleEnv, err := readEnvFile(filepath.Join("web", "console", ".env"))
	if err != nil {
		return err
	}
	consoleEnv["PPVT_CONSOLE_APPLICATION_ID"] = cfg.ConsoleApplicationID
	if err := writeEnvFile(filepath.Join("web", "console", ".env"), consoleEnv); err != nil {
		return err
	}

	portalEnv, err := readEnvFile(filepath.Join("web", "portal", ".env"))
	if err != nil {
		return err
	}
	portalEnv["PPVT_PORTAL_APPLICATION_ID"] = cfg.PortalApplicationID
	if err := writeEnvFile(filepath.Join("web", "portal", ".env"), portalEnv); err != nil {
		return err
	}
	return nil
}

func readEnvFile(path string) (map[string]string, error) {
	values := map[string]string{}
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return values, nil
		}
		return nil, err
	}
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		values[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return values, nil
}

func writeEnvFile(path string, values map[string]string) error {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var builder strings.Builder
	for _, key := range keys {
		builder.WriteString(key)
		builder.WriteString("=")
		builder.WriteString(values[key])
		builder.WriteString("\n")
	}
	return os.WriteFile(path, []byte(builder.String()), 0o644)
}

func recreateDatabase(cfg config.Config, opts Options) error {
	if !identifierPattern.MatchString(cfg.DatabaseSchema) {
		return fmt.Errorf("invalid database schema: %s", cfg.DatabaseSchema)
	}

	switch cfg.DatabaseDriver {
	case "mysql":
		adminDB, err := gorm.Open(mysql.Open(fmt.Sprintf(
			"%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=True&loc=Local",
			cfg.DatabaseUsername,
			cfg.DatabasePassword,
			cfg.DatabaseHost,
			cfg.DatabasePort,
		)), &gorm.Config{})
		if err != nil {
			return err
		}
		if !opts.Force {
			hasTables, err := mysqlSchemaHasTables(adminDB, cfg.DatabaseSchema)
			if err != nil {
				return err
			}
			if hasTables {
				return fmt.Errorf("database %s already contains tables; rerun ppvt-init with --force to rebuild", cfg.DatabaseSchema)
			}
		}
		if err := adminDB.Exec("DROP DATABASE IF EXISTS `" + cfg.DatabaseSchema + "`").Error; err != nil {
			return err
		}
		return adminDB.Exec("CREATE DATABASE `" + cfg.DatabaseSchema + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci").Error
	case "postgres":
		adminDB, err := gorm.Open(postgres.Open(fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=postgres sslmode=disable",
			cfg.DatabaseHost,
			cfg.DatabasePort,
			cfg.DatabaseUsername,
			cfg.DatabasePassword,
		)), &gorm.Config{})
		if err != nil {
			return err
		}
		if !opts.Force {
			hasTables, err := postgresSchemaHasTables(cfg)
			if err != nil {
				return err
			}
			if hasTables {
				return fmt.Errorf("database %s already contains tables; rerun ppvt-init with --force to rebuild", cfg.DatabaseSchema)
			}
		}
		if err := adminDB.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS "%s" WITH (FORCE)`, cfg.DatabaseSchema)).Error; err != nil {
			return err
		}
		return adminDB.Exec(`CREATE DATABASE "` + cfg.DatabaseSchema + `"`).Error
	default:
		return errors.New("unsupported database driver")
	}
}

func mysqlSchemaHasTables(adminDB *gorm.DB, schema string) (bool, error) {
	var count int64
	if err := adminDB.Raw(
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ?",
		schema,
	).Scan(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func postgresSchemaHasTables(cfg config.Config) (bool, error) {
	var count int64
	adminDB, err := gorm.Open(postgres.Open(fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=postgres sslmode=disable",
		cfg.DatabaseHost,
		cfg.DatabasePort,
		cfg.DatabaseUsername,
		cfg.DatabasePassword,
	)), &gorm.Config{})
	if err != nil {
		return false, err
	}
	if err := adminDB.Raw(
		"SELECT COUNT(*) FROM pg_database WHERE datname = ?",
		cfg.DatabaseSchema,
	).Scan(&count).Error; err != nil {
		return false, err
	}
	if count == 0 {
		return false, nil
	}

	database, err := gorm.Open(postgres.Open(fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.DatabaseHost,
		cfg.DatabasePort,
		cfg.DatabaseUsername,
		cfg.DatabasePassword,
		cfg.DatabaseSchema,
	)), &gorm.Config{})
	if err != nil {
		return false, err
	}
	if err := database.Raw(
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public'",
	).Scan(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func initSchema(database *gorm.DB) error {
	if err := database.AutoMigrate(
		&model.Organization{},
		&model.Project{},
		&model.ProjectUserAssignment{},
		&model.Application{},
		&model.ApplicationKey{},
		&model.OrganizationSigningKey{},
		&model.User{},
		&model.SecureKey{},
		&model.MFAEnrollment{},
		&model.MFARecoveryCode{},
		&model.Session{},
		&model.AuthorizationCode{},
		&model.DeviceAuthorization{},
		&model.Token{},
		&model.Device{},
		&model.Role{},
		&model.Policy{},
		&model.AuditLog{},
		&model.ExternalIDP{},
		&model.ExternalIdentityBinding{},
	); err != nil {
		return err
	}
	return nil
}

func seed(ctx context.Context, database *gorm.DB, cfg config.Config) error {
	passwordHash, err := utils.HashSecret("ChangeMe123!")
	if err != nil {
		return err
	}
	return database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		organization := model.Organization{
			BaseModel:         model.BaseModel{ID: cfg.InternalOrganizationID},
			Name:              "internal",
			Description:       "Internal organization",
			Status:            "active",
			Metadata:          buildInternalOrganizationMetadata("PassPivot", "PassPivot", "PassPivot", "PassPivot", "PassPivot", "http://example.com", "http://example.com/terms-of-service", "http://example.com/privacy-policy"),
			AllowJWTAccess:    true,
			AllowBasicAccess:  true,
			AllowNoAuthAccess: true,
			AllowRefreshToken: true,
			AllowAuthCode:     true,
			AllowPKCE:         true,
			TOSURL:            "",
			PrivacyPolicyURL:  "",
			SupportEmail:      "",
			LogoURL:           "",
			Domains:           []model.OrganizationDomain{},
			LoginPolicy:       defaultConsoleSettings().LoginPolicy,
			PasswordPolicy:    defaultConsoleSettings().PasswordPolicy,
			MFAPolicy:         defaultConsoleSettings().MFAPolicy,
			Captcha:           defaultConsoleSettings().Captcha,
		}
		if err := upsertByID(tx, &organization); err != nil {
			return err
		}
		signingKey, err := authservice.NewOrganizationSigningKey(cfg.InternalOrganizationID)
		if err != nil {
			return err
		}
		if err := tx.Create(signingKey).Error; err != nil {
			return err
		}

		project := model.Project{
			BaseModel:      model.BaseModel{ID: cfg.SystemProjectID},
			OrganizationID: cfg.InternalOrganizationID,
			Name:           "ppvt",
			Description:    "PPVT system project",
			Status:         "active",
		}
		if err := upsertByID(tx, &project); err != nil {
			return err
		}

		applications := []model.Application{
			{
				BaseModel:                model.BaseModel{ID: cfg.ManageAPIApplicationID},
				ProjectID:                cfg.SystemProjectID,
				Name:                     "manage-api",
				Metadata:                 buildInternalApplicationMetadata("Manage API", "Manage API", "Manage API", "管理 API", "管理 API"),
				Description:              "System manage API application",
				Status:                   "active",
				ApplicationType:          "api",
				GrantType:                []string{"client_credentials"},
				EnableRefreshToken:       false,
				ClientAuthenticationType: "private_key_jwt",
				TokenType:                []string{"access_token"},
				Roles:                    []string{"api:manage"},
				AccessTokenTTLMinutes:    10,
				RefreshTokenTTLHours:     168,
			},
			{
				BaseModel:                model.BaseModel{ID: cfg.UserAPIApplicationID},
				ProjectID:                cfg.SystemProjectID,
				Name:                     "user-api",
				Metadata:                 buildInternalApplicationMetadata("User API", "User API", "User API", "用户 API", "使用者 API"),
				Description:              "System current-user API application",
				Status:                   "active",
				ApplicationType:          "api",
				GrantType:                []string{"client_credentials"},
				EnableRefreshToken:       false,
				ClientAuthenticationType: "private_key_jwt",
				TokenType:                []string{"access_token"},
				Roles:                    []string{"api:user"},
				AccessTokenTTLMinutes:    10,
				RefreshTokenTTLHours:     168,
			},
			{
				BaseModel:                model.BaseModel{ID: cfg.AuthnAPIApplicationID},
				ProjectID:                cfg.SystemProjectID,
				Name:                     "authn-api",
				Metadata:                 buildInternalApplicationMetadata("Authentication API", "Authentication API", "Authentication API", "认证 API", "認證 API"),
				Description:              "System authentication API application",
				Status:                   "active",
				ApplicationType:          "api",
				GrantType:                []string{"client_credentials"},
				EnableRefreshToken:       false,
				ClientAuthenticationType: "private_key_jwt",
				TokenType:                []string{"access_token"},
				Roles:                    []string{"api:authn"},
				AccessTokenTTLMinutes:    10,
				RefreshTokenTTLHours:     168,
			},
			{
				BaseModel:                model.BaseModel{ID: cfg.AuthzAPIApplicationID},
				ProjectID:                cfg.SystemProjectID,
				Name:                     "authz-api",
				Metadata:                 buildInternalApplicationMetadata("Authorization API", "Authorization API", "Authorization API", "授权 API", "授權 API"),
				Description:              "System authorization decision API application",
				Status:                   "active",
				ApplicationType:          "api",
				GrantType:                []string{"client_credentials"},
				EnableRefreshToken:       false,
				ClientAuthenticationType: "private_key_jwt",
				TokenType:                []string{"access_token"},
				Roles:                    []string{"api:authz"},
				AccessTokenTTLMinutes:    10,
				RefreshTokenTTLHours:     168,
			},
			{
				BaseModel:                model.BaseModel{ID: cfg.ConsoleApplicationID},
				ProjectID:                cfg.SystemProjectID,
				Name:                     "console-web",
				Metadata:                 buildInternalApplicationMetadata("Console", "Console", "コンソール", "控制台", "控制台"),
				Description:              "Official PPVT console frontend",
				RedirectURIs:             "http://localhost:8093/console/callback",
				Status:                   "active",
				ApplicationType:          "web",
				GrantType:                []string{"authorization_code_pkce"},
				EnableRefreshToken:       false,
				ClientAuthenticationType: "none",
				TokenType:                []string{"access_token"},
				Roles:                    []string{"api:manage", "api:user"},
				AccessTokenTTLMinutes:    10,
				RefreshTokenTTLHours:     168,
			},
			{
				BaseModel:                model.BaseModel{ID: cfg.PortalApplicationID},
				ProjectID:                cfg.SystemProjectID,
				Name:                     "portal-web",
				Metadata:                 buildInternalApplicationMetadata("Portal", "Portal", "ポータル", "用户中心", "使用者中心"),
				Description:              "Official PPVT portal frontend",
				RedirectURIs:             "http://localhost:8092/portal/callback",
				Status:                   "active",
				ApplicationType:          "web",
				GrantType:                []string{"authorization_code_pkce"},
				EnableRefreshToken:       false,
				ClientAuthenticationType: "none",
				TokenType:                []string{"access_token"},
				Roles:                    []string{"api:user"},
				AccessTokenTTLMinutes:    10,
				RefreshTokenTTLHours:     168,
			},
		}
		for i := range applications {
			if applications[i].ClientAuthenticationType == "private_key_jwt" {
				clientKey, _, err := authservice.NewApplicationClientKey(applications[i].ID)
				if err != nil {
					return err
				}
				applications[i].PublicKey = clientKey.PublicKeyBase64
				if err := tx.Create(clientKey).Error; err != nil {
					return err
				}
			}
			if err := upsertByID(tx, &applications[i]); err != nil {
				return err
			}
		}

		roles := []model.Role{
			{
				BaseModel:      model.BaseModel{ID: cfg.ConsoleAdminRoleID},
				OrganizationID: cfg.InternalOrganizationID,
				Name:           sharedhandler.OrganizationOwnerRoleName(cfg.InternalOrganizationID),
				Type:           "user",
				Description:    "Built-in internal organization owner role",
			},
			{
				BaseModel:      model.BaseModel{ID: stableUUID(cfg.InternalOrganizationID + ":" + sharedhandler.OrganizationAdminRoleName(cfg.InternalOrganizationID))},
				OrganizationID: cfg.InternalOrganizationID,
				Name:           sharedhandler.OrganizationAdminRoleName(cfg.InternalOrganizationID),
				Type:           "user",
				Description:    "Built-in internal organization admin role",
			},
			{
				BaseModel:      model.BaseModel{ID: stableUUID(cfg.InternalOrganizationID + ":api:manage")},
				OrganizationID: cfg.InternalOrganizationID,
				Name:           "api:manage",
				Type:           "application",
				Description:    "Built-in manage API application role",
			},
			{
				BaseModel:      model.BaseModel{ID: stableUUID(cfg.InternalOrganizationID + ":api:user")},
				OrganizationID: cfg.InternalOrganizationID,
				Name:           "api:user",
				Type:           "application",
				Description:    "Built-in user API application role",
			},
			{
				BaseModel:      model.BaseModel{ID: stableUUID(cfg.InternalOrganizationID + ":api:authn")},
				OrganizationID: cfg.InternalOrganizationID,
				Name:           "api:authn",
				Type:           "application",
				Description:    "Built-in authn API application role",
			},
			{
				BaseModel:      model.BaseModel{ID: stableUUID(cfg.InternalOrganizationID + ":api:authz")},
				OrganizationID: cfg.InternalOrganizationID,
				Name:           "api:authz",
				Type:           "application",
				Description:    "Built-in authz API application role",
			},
		}
		for i := range roles {
			if err := upsertByID(tx, &roles[i]); err != nil {
				return err
			}
		}
		policies := []model.Policy{
			newPolicy(cfg.InternalOrganizationID, roles[0].ID, "manage:all", "allow", 10, []model.PolicyAPIRule{{Method: "POST", Path: "/api/manage/v1/*"}}),
			newPolicy(cfg.InternalOrganizationID, roles[1].ID, "manage:all", "allow", 10, []model.PolicyAPIRule{{Method: "POST", Path: "/api/manage/v1/*"}}),
			newPolicy(cfg.InternalOrganizationID, roles[2].ID, "manage:all", "allow", 10, []model.PolicyAPIRule{{Method: "POST", Path: "/api/manage/v1/*"}}),
			newPolicy(cfg.InternalOrganizationID, roles[3].ID, "user:portal", "allow", 10, []model.PolicyAPIRule{{Method: "POST", Path: "/api/user/v1/*"}}),
			newPolicy(cfg.InternalOrganizationID, roles[4].ID, "authn:all", "allow", 10, []model.PolicyAPIRule{{Method: "POST", Path: "/api/authn/v1/*"}}),
			newPolicy(cfg.InternalOrganizationID, roles[5].ID, "authz:all", "allow", 10, []model.PolicyAPIRule{{Method: "POST", Path: "/api/authz/v1/*"}}),
		}
		for i := range policies {
			if err := upsertByID(tx, &policies[i]); err != nil {
				return err
			}
		}

		user := model.User{
			BaseModel:      model.BaseModel{ID: cfg.AdminUserID},
			OrganizationID: cfg.InternalOrganizationID,
			Username:       "admin",
			Name:           "Administrator",
			Email:          "admin@example.com",
			PhoneNumber:    "",
			Roles:          []string{sharedhandler.OrganizationOwnerRoleName(cfg.InternalOrganizationID)},
			Status:         "active",
			PasswordHash:   passwordHash,
			CurrentUKID:    "ukid-" + cfg.AdminUserID,
		}
		if err := upsertByID(tx, &user); err != nil {
			return err
		}
		projectAssignment := model.ProjectUserAssignment{
			BaseModel: model.BaseModel{ID: stableUUID(cfg.SystemProjectID + ":" + cfg.AdminUserID)},
			ProjectID: cfg.SystemProjectID,
			UserID:    cfg.AdminUserID,
		}
		if err := upsertByID(tx, &projectAssignment); err != nil {
			return err
		}
		return nil
	})
}

func buildInternalApplicationMetadata(defaultDisplayName, englishDisplayName, japaneseDisplayName, simplifiedChineseDisplayName, traditionalChineseDisplayName string) map[string]string {
	return coreservice.NormalizeApplicationMetadata(map[string]string{
		coreservice.ApplicationMetadataDisplayName:    defaultDisplayName,
		coreservice.ApplicationMetadataDisplayNameEN:  englishDisplayName,
		coreservice.ApplicationMetadataDisplayNameJA:  japaneseDisplayName,
		coreservice.ApplicationMetadataDisplayNameCHS: simplifiedChineseDisplayName,
		coreservice.ApplicationMetadataDisplayNameCHT: traditionalChineseDisplayName,
	}, nil)
}

func buildInternalOrganizationMetadata(displayName, englishDisplayName, japaneseDisplayName, simplifiedChineseDisplayName, traditionalChineseDisplayName, websiteURL, termsOfServiceURL, privacyPolicyURL string) map[string]string {
	return coreservice.NormalizeOrganizationMetadata(map[string]string{
		coreservice.OrganizationMetadataDisplayName:       displayName,
		coreservice.OrganizationMetadataDisplayNameEN:     englishDisplayName,
		coreservice.OrganizationMetadataDisplayNameJA:     japaneseDisplayName,
		coreservice.OrganizationMetadataDisplayNameCHS:    simplifiedChineseDisplayName,
		coreservice.OrganizationMetadataDisplayNameCHT:    traditionalChineseDisplayName,
		coreservice.OrganizationMetadataWebsiteURL:        websiteURL,
		coreservice.OrganizationMetadataTermsOfServiceURL: termsOfServiceURL,
		coreservice.OrganizationMetadataPrivacyPolicyURL:  privacyPolicyURL,
	}, nil)
}

func upsertByID(tx *gorm.DB, value any) error {
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(value).Error
}

func stableUUID(value string) string {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(value)).String()
}

func newPolicy(organizationID, roleID, name, effect string, priority int, rules []model.PolicyAPIRule) model.Policy {
	return model.Policy{
		BaseModel:      model.BaseModel{ID: stableUUID(strings.Join([]string{organizationID, roleID, name, effect, strconv.Itoa(priority)}, ":"))},
		OrganizationID: organizationID,
		RoleID:         roleID,
		Name:           name,
		Effect:         effect,
		Priority:       priority,
		APIRules:       rules,
	}
}

func defaultConsoleSettings() model.OrganizationSetting {
	return model.OrganizationSetting{
		SupportEmail: "",
		LogoURL:      "",
		Domains:      []model.OrganizationDomain{},
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
				Enabled:  false,
				From:     "",
				Host:     "",
				Port:     587,
				Username: "",
				Password: "",
			},
		},
		Captcha: model.OrganizationCaptchaSettings{
			Provider: "disabled",
		},
	}
}
