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
	"pass-pivot/util"

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

	ensurePublicKey := func(seed string, publicKeyKey string) error {
		publicKey, err := util.DeriveEd25519PublicKey(seed)
		if err != nil {
			return err
		}
		rootValues[publicKeyKey] = publicKey
		return nil
	}
	if err := ensurePublicKey(cfg.APIManagePrivateSeed, "PPVT_MANAGE_API_PUBLIC_KEY"); err != nil {
		return err
	}
	if err := ensurePublicKey(cfg.APIUserPrivateSeed, "PPVT_USER_API_PUBLIC_KEY"); err != nil {
		return err
	}
	if err := ensurePublicKey(cfg.APIAuthnPrivateSeed, "PPVT_AUTHN_API_PUBLIC_KEY"); err != nil {
		return err
	}
	if err := ensurePublicKey(cfg.APIAuthzPrivateSeed, "PPVT_AUTHZ_API_PUBLIC_KEY"); err != nil {
		return err
	}
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
		&model.Application{},
		&model.User{},
		&model.SecureKey{},
		&model.MFAEnrollment{},
		&model.MFARecoveryCode{},
		&model.Session{},
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
	passwordHash, err := util.HashSecret("ChangeMe123!")
	if err != nil {
		return err
	}
	return database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		organization := model.Organization{
			BaseModel:         model.BaseModel{ID: cfg.InternalOrganizationID},
			Name:              "internal",
			Metadata:          map[string]string{},
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
		}
		if err := upsertByID(tx, &organization); err != nil {
			return err
		}

		project := model.Project{
			BaseModel:      model.BaseModel{ID: cfg.SystemProjectID},
			OrganizationID: cfg.InternalOrganizationID,
			Name:           "ppvt",
			Description:    "PPVT system project",
		}
		if err := upsertByID(tx, &project); err != nil {
			return err
		}

		applications := []model.Application{
			{
				BaseModel:                model.BaseModel{ID: cfg.ManageAPIApplicationID},
				ProjectID:                cfg.SystemProjectID,
				Name:                     "manage-api",
				Description:              "System manage API application",
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
				Description:              "System current-user API application",
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
				Description:              "System authentication API application",
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
				Description:              "System authorization decision API application",
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
				Description:              "Official PPVT console frontend",
				RedirectURIs:             "http://localhost:8093/console/callback",
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
				Description:              "Official PPVT portal frontend",
				RedirectURIs:             "http://localhost:8092/portal/callback",
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
			if applications[i].ClientAuthenticationType != "private_key_jwt" {
				continue
			}
			seedByID := map[string]string{
				cfg.ManageAPIApplicationID: cfg.APIManagePrivateSeed,
				cfg.UserAPIApplicationID:   cfg.APIUserPrivateSeed,
				cfg.AuthnAPIApplicationID:  cfg.APIAuthnPrivateSeed,
				cfg.AuthzAPIApplicationID:  cfg.APIAuthzPrivateSeed,
			}
			publicKey, err := util.DeriveEd25519PublicKey(seedByID[applications[i].ID])
			if err != nil {
				return err
			}
			applications[i].PublicKey = publicKey
		}
		for i := range applications {
			if err := upsertByID(tx, &applications[i]); err != nil {
				return err
			}
		}

		roles := []model.Role{
			{
				BaseModel:      model.BaseModel{ID: cfg.ConsoleAdminRoleID},
				OrganizationID: cfg.InternalOrganizationID,
				Name:           "console:admin",
				Type:           "user",
				Description:    "Built-in console administrator role label",
			},
			{
				BaseModel:      model.BaseModel{ID: stableUUID(cfg.InternalOrganizationID + ":user:self:all")},
				OrganizationID: cfg.InternalOrganizationID,
				Name:           "user:self:all",
				Type:           "user",
				Description:    "Built-in self-service user role",
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
			newPolicy(cfg.InternalOrganizationID, roles[1].ID, "user:self:all", "allow", 10, []model.PolicyAPIRule{{Method: "POST", Path: "/api/user/v1/*"}}),
			newPolicy(cfg.InternalOrganizationID, roles[2].ID, "manage:all", "allow", 10, []model.PolicyAPIRule{{Method: "POST", Path: "/api/manage/v1/*"}}),
			newPolicy(cfg.InternalOrganizationID, roles[3].ID, "user:self:all", "allow", 10, []model.PolicyAPIRule{{Method: "POST", Path: "/api/user/v1/*"}}),
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
			Roles:          []string{"console:admin", "user:self:all"},
			Status:         "active",
			PasswordHash:   passwordHash,
			CurrentUKID:    "ukid-" + cfg.AdminUserID,
		}
		if err := upsertByID(tx, &user); err != nil {
			return err
		}
		return nil
	})
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
		TOSURL:           "",
		PrivacyPolicyURL: "",
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
				Enabled:  false,
				From:     "",
				Host:     "",
				Port:     587,
				Username: "",
				Password: "",
			},
		},
	}
}
