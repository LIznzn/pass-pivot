package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	HTTPAddr         string
	AuthURL          string
	CoreURL          string
	DatabaseDriver   string
	DatabaseHost     string
	DatabasePort     int
	DatabaseUsername string
	DatabasePassword string
	DatabaseSchema   string
	RedisEnabled     bool
	RedisHost        string
	RedisPort        int
	RedisPassword    string
	RedisDB          int
	LogLevel         string
	Secret           string

	InternalOrganizationID string
	SystemProjectID        string
	ManageAPIApplicationID string
	UserAPIApplicationID   string
	AuthnAPIApplicationID  string
	AuthzAPIApplicationID  string
	ConsoleApplicationID   string
	PortalApplicationID    string
	ConsoleAdminRoleID     string
	AdminUserID            string
	APIManagePrivateSeed   string
	APIUserPrivateSeed     string
	APIAuthnPrivateSeed    string
	APIAuthzPrivateSeed    string
}

func Load() Config {
	loadDotEnv(".env")
	loadDotEnv(".init")
	return loadFromEnv()
}

func LoadInit() Config {
	loadDotEnv(".env")
	loadDotEnv(".init")
	return loadFromEnv()
}

func loadFromEnv() Config {
	return Config{
		HTTPAddr:               getenv("PPVT_HTTP_ADDR", "0.0.0.0:8090"),
		AuthURL:                getenv("PPVT_AUTH_URL", "http://localhost:8091"),
		CoreURL:                getenv("PPVT_CORE_URL", "http://localhost:8090"),
		DatabaseDriver:         getenv("PPVT_DATABASE_DRIVER", "mysql"),
		DatabaseHost:           getenv("PPVT_DATABASE_HOST", "127.0.0.1"),
		DatabasePort:           getenvInt("PPVT_DATABASE_PORT", 3306),
		DatabaseUsername:       getenv("PPVT_DATABASE_USERNAME", "root"),
		DatabasePassword:       getenv("PPVT_DATABASE_PASSWORD", "root"),
		DatabaseSchema:         getenv("PPVT_DATABASE_SCHEMA", "ppvt"),
		RedisEnabled:           getenvBool("PPVT_REDIS_ENABLED", false),
		RedisHost:              getenv("PPVT_REDIS_HOST", "127.0.0.1"),
		RedisPort:              getenvInt("PPVT_REDIS_PORT", 6379),
		RedisPassword:          getenv("PPVT_REDIS_PASSWORD", ""),
		RedisDB:                getenvInt("PPVT_REDIS_DB", 0),
		LogLevel:               getenv("PPVT_LOG_LEVEL", "INFO"),
		Secret:                 getenv("PPVT_SECRET", "ppvt-dev-secret"),
		InternalOrganizationID: getenv("PPVT_INTERNAL_ORGANIZATION_ID", ""),
		SystemProjectID:        getenv("PPVT_SYSTEM_PROJECT_ID", ""),
		ManageAPIApplicationID: getenv("PPVT_MANAGE_API_APPLICATION_ID", ""),
		UserAPIApplicationID:   getenv("PPVT_USER_API_APPLICATION_ID", ""),
		AuthnAPIApplicationID:  getenv("PPVT_AUTHN_API_APPLICATION_ID", ""),
		AuthzAPIApplicationID:  getenv("PPVT_AUTHZ_API_APPLICATION_ID", ""),
		ConsoleApplicationID:   getenv("PPVT_CONSOLE_APPLICATION_ID", ""),
		PortalApplicationID:    getenv("PPVT_PORTAL_APPLICATION_ID", ""),
		ConsoleAdminRoleID:     getenv("PPVT_CONSOLE_ADMIN_ROLE_ID", ""),
		AdminUserID:            getenv("PPVT_ADMIN_USER_ID", ""),
		APIManagePrivateSeed:   getenv("PPVT_API_MANAGE_PRIVATE_SEED", ""),
		APIUserPrivateSeed:     getenv("PPVT_API_USER_PRIVATE_SEED", ""),
		APIAuthnPrivateSeed:    getenv("PPVT_API_AUTHN_PRIVATE_SEED", ""),
		APIAuthzPrivateSeed:    getenv("PPVT_API_AUTHZ_PRIVATE_SEED", ""),
	}
}

func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		if key == "" || os.Getenv(key) != "" {
			continue
		}
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, `"'`)
		_ = os.Setenv(key, value)
	}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func getenvBool(key string, fallback bool) bool {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	switch raw {
	case "1", "true", "TRUE", "True", "yes", "YES", "on", "ON":
		return true
	case "0", "false", "FALSE", "False", "no", "NO", "off", "OFF":
		return false
	default:
		return fallback
	}
}
