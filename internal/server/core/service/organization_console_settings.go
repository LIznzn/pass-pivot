package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strings"

	"gorm.io/gorm"

	"pass-pivot/internal/model"
)

const (
	OrganizationMetadataDisplayName        = "displayName"
	OrganizationMetadataDisplayNameEN      = "displayName.en"
	OrganizationMetadataDisplayNameJA      = "displayName.ja"
	OrganizationMetadataDisplayNameCHS     = "displayName.chs"
	OrganizationMetadataDisplayNameCHT     = "displayName.cht"
	OrganizationMetadataWebsiteURL         = "websiteUrl"
	OrganizationMetadataTermsOfServiceURL  = "termsOfServiceUrl"
	OrganizationMetadataPrivacyPolicyURL   = "privacyPolicyUrl"
	OrganizationDomainVerificationHTTPFile = "http_file"
	OrganizationDomainVerificationDNSTXT   = "dns_txt"
	OrganizationDomainVerificationTXTName  = "_ppvt-domain-verification"
	OrganizationDomainVerificationFilePath = "/.well-known/ppvt-domain-verification.txt"
)

func defaultOrganizationConsoleSettings() model.OrganizationSetting {
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
		},
		Mail: model.OrganizationMailSettings{
			Provider: "disabled",
			SMTPPort: 587,
		},
		Captcha: model.OrganizationCaptchaSettings{
			Provider: "disabled",
		},
	}
}

func normalizeOrganizationConsoleSettings(input *model.OrganizationSetting) model.OrganizationSetting {
	settings := defaultOrganizationConsoleSettings()
	defaults := settings
	if input != nil {
		settings = *input
		if settings.Mail.SMTPPort == 0 {
			settings.Mail.SMTPPort = 587
		}
		if strings.TrimSpace(settings.Captcha.Provider) == "" {
			settings.Captcha.Provider = defaults.Captcha.Provider
		}
	}
	settings.Domains = normalizeOrganizationDomains(settings.Domains)
	settings.Mail = normalizeOrganizationMailSettings(settings.Mail)
	settings.Captcha = normalizeOrganizationCaptchaSettings(settings.Captcha)
	return settings
}

func normalizeOrganizationDomains(input []model.OrganizationDomain) []model.OrganizationDomain {
	if input == nil {
		return []model.OrganizationDomain{}
	}
	result := make([]model.OrganizationDomain, 0, len(input))
	for _, item := range input {
		normalized, ok := normalizeOrganizationDomain(item)
		if !ok {
			continue
		}
		result = append(result, normalized)
	}
	return result
}

func normalizeOrganizationDomain(input model.OrganizationDomain) (model.OrganizationDomain, bool) {
	host := normalizeOrganizationDomainHost(input.Host)
	if host == "" {
		return model.OrganizationDomain{}, false
	}
	method := strings.ToLower(strings.TrimSpace(input.VerificationMethod))
	if method == "" {
		method = OrganizationDomainVerificationHTTPFile
	}
	domain := model.OrganizationDomain{
		Host:               host,
		Verified:           input.Verified,
		VerificationMethod: method,
		VerificationToken:  strings.TrimSpace(input.VerificationToken),
		VerifiedAt:         input.VerifiedAt,
	}
	if !domain.Verified {
		domain.VerifiedAt = nil
	}
	if domain.VerifiedAt != nil {
		domain.Verified = true
	}
	return domain, true
}

func normalizeOrganizationDomainHost(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return ""
	}
	if strings.Contains(value, "://") {
		if parsed, err := url.Parse(value); err == nil {
			value = strings.ToLower(strings.TrimSpace(parsed.Host))
		}
	}
	value = strings.TrimSuffix(value, ".")
	if strings.ContainsAny(value, "/?#") {
		return ""
	}
	if host, port, err := net.SplitHostPort(value); err == nil {
		if strings.TrimSpace(host) == "" || strings.TrimSpace(port) == "" {
			return ""
		}
		return strings.ToLower(host) + ":" + strings.TrimSpace(port)
	}
	return value
}

func ValidateOrganizationDomains(input []model.OrganizationDomain) error {
	seen := map[string]struct{}{}
	for _, item := range normalizeOrganizationDomains(input) {
		if _, exists := seen[item.Host]; exists {
			return fmt.Errorf("duplicate domain: %s", item.Host)
		}
		seen[item.Host] = struct{}{}
		if IsPrivateOrganizationDomainHost(item.Host) {
			return fmt.Errorf("IP or local network addresses are not allowed: %s", item.Host)
		}
		switch item.VerificationMethod {
		case OrganizationDomainVerificationHTTPFile:
		case OrganizationDomainVerificationDNSTXT:
			if strings.Contains(item.Host, ":") {
				return fmt.Errorf("dns_txt verification does not support ports: %s", item.Host)
			}
		default:
			return fmt.Errorf("invalid domain verification method: %s", item.VerificationMethod)
		}
	}
	return nil
}

func IsPrivateOrganizationDomainHost(host string) bool {
	hostname := strings.TrimSpace(host)
	if hostname == "" {
		return false
	}
	if parsedHost, _, err := net.SplitHostPort(hostname); err == nil {
		hostname = parsedHost
	}
	hostname = strings.Trim(strings.ToLower(strings.TrimSpace(hostname)), "[]")
	if hostname == "" {
		return false
	}
	if hostname == "localhost" || strings.HasSuffix(hostname, ".localhost") {
		return true
	}
	ip := net.ParseIP(hostname)
	if ip != nil {
		return true
	}
	return false
}

func isPrivateOrLocalIPAddress(ip net.IP) bool {
	return ip.IsPrivate() ||
		ip.IsLoopback() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified()
}

func DomainVerificationTXTRecordName(host string) string {
	normalized := normalizeOrganizationDomainHost(host)
	if normalized == "" || strings.Contains(normalized, ":") {
		return ""
	}
	return OrganizationDomainVerificationTXTName + "." + normalized
}

func DomainVerificationFileURL(host string, insecure bool) string {
	normalized := normalizeOrganizationDomainHost(host)
	if normalized == "" {
		return ""
	}
	scheme := "https"
	if insecure {
		scheme = "http"
	}
	return scheme + "://" + normalized + OrganizationDomainVerificationFilePath
}

func normalizeOrganizationCaptchaSettings(input model.OrganizationCaptchaSettings) model.OrganizationCaptchaSettings {
	settings := model.OrganizationCaptchaSettings{
		Provider:     strings.ToLower(strings.TrimSpace(input.Provider)),
		ClientKey:    strings.TrimSpace(input.ClientKey),
		ClientSecret: strings.TrimSpace(input.ClientSecret),
	}
	if settings.Provider == "disabled" {
		settings.ClientKey = ""
		settings.ClientSecret = ""
	}
	return settings
}

func normalizeOrganizationMailSettings(input model.OrganizationMailSettings) model.OrganizationMailSettings {
	settings := model.OrganizationMailSettings{
		Provider:       strings.ToLower(strings.TrimSpace(input.Provider)),
		From:           strings.TrimSpace(input.From),
		SMTPHost:       strings.TrimSpace(input.SMTPHost),
		SMTPPort:       input.SMTPPort,
		SMTPUser:       strings.TrimSpace(input.SMTPUser),
		SMTPPass:       input.SMTPPass,
		MailgunDomain:  strings.TrimSpace(input.MailgunDomain),
		MailgunAPIKey:  strings.TrimSpace(input.MailgunAPIKey),
		MailgunAPIBase: strings.TrimRight(strings.TrimSpace(input.MailgunAPIBase), "/"),
		SendGridAPIKey: strings.TrimSpace(input.SendGridAPIKey),
	}
	if settings.Provider == "" {
		settings.Provider = "disabled"
	}
	if settings.SMTPPort == 0 {
		settings.SMTPPort = 587
	}
	if settings.Provider == "disabled" {
		settings.From = ""
		settings.SMTPHost = ""
		settings.SMTPPort = 587
		settings.SMTPUser = ""
		settings.SMTPPass = ""
		settings.MailgunDomain = ""
		settings.MailgunAPIKey = ""
		settings.MailgunAPIBase = ""
		settings.SendGridAPIKey = ""
	}
	if settings.Provider == "mailgun" && settings.MailgunAPIBase == "" {
		settings.MailgunAPIBase = "https://api.mailgun.net"
	}
	return settings
}

func ValidateOrganizationMailSettings(input model.OrganizationMailSettings) error {
	settings := normalizeOrganizationMailSettings(input)
	switch settings.Provider {
	case "disabled":
		return nil
	case "smtp":
		if settings.From == "" {
			return fmt.Errorf("mail from is required when provider is smtp")
		}
		if settings.SMTPHost == "" {
			return fmt.Errorf("mail smtpHost is required when provider is smtp")
		}
		if settings.SMTPPort <= 0 {
			return fmt.Errorf("mail smtpPort is required when provider is smtp")
		}
		return nil
	case "mailgun":
		if settings.From == "" {
			return fmt.Errorf("mail from is required when provider is mailgun")
		}
		if settings.MailgunDomain == "" {
			return fmt.Errorf("mail mailgunDomain is required when provider is mailgun")
		}
		if settings.MailgunAPIKey == "" {
			return fmt.Errorf("mail mailgunApiKey is required when provider is mailgun")
		}
		return nil
	case "sendgrid":
		if settings.From == "" {
			return fmt.Errorf("mail from is required when provider is sendgrid")
		}
		if settings.SendGridAPIKey == "" {
			return fmt.Errorf("mail sendgridApiKey is required when provider is sendgrid")
		}
		return nil
	default:
		return fmt.Errorf("invalid mail provider: %s", settings.Provider)
	}
}

func OrganizationMailSettingsReady(input model.OrganizationMailSettings) bool {
	settings := normalizeOrganizationMailSettings(input)
	return ValidateOrganizationMailSettings(settings) == nil && settings.Provider != "disabled"
}

func ValidateOrganizationCaptchaSettings(input model.OrganizationCaptchaSettings) error {
	settings := normalizeOrganizationCaptchaSettings(input)
	switch settings.Provider {
	case "disabled":
		return nil
	case "default":
		return nil
	case "google", "cloudflare":
		if settings.ClientKey == "" {
			return fmt.Errorf("captcha clientKey is required when provider is %s", settings.Provider)
		}
		if settings.ClientSecret == "" {
			return fmt.Errorf("captcha clientSecret is required when provider is %s", settings.Provider)
		}
		return nil
	default:
		return fmt.Errorf("invalid captcha provider: %s", settings.Provider)
	}
}

func defaultOrganizationMetadata() map[string]string {
	return map[string]string{
		OrganizationMetadataDisplayName:       "",
		OrganizationMetadataDisplayNameEN:     "",
		OrganizationMetadataDisplayNameJA:     "",
		OrganizationMetadataDisplayNameCHS:    "",
		OrganizationMetadataDisplayNameCHT:    "",
		OrganizationMetadataWebsiteURL:        "http://example.com",
		OrganizationMetadataTermsOfServiceURL: "http://example.com/terms-of-service",
		OrganizationMetadataPrivacyPolicyURL:  "http://example.com/privacy-policy",
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

func BuildOrganizationDisplayNameMap(metadata map[string]string) map[string]string {
	normalized := NormalizeOrganizationMetadata(metadata, nil)
	return map[string]string{
		"default": OrganizationDisplayNameForLocale(normalized, "", ""),
		"en":      OrganizationDisplayNameForLocale(normalized, "en", ""),
		"ja":      OrganizationDisplayNameForLocale(normalized, "ja", ""),
		"chs":     OrganizationDisplayNameForLocale(normalized, "chs", ""),
		"cht":     OrganizationDisplayNameForLocale(normalized, "cht", ""),
	}
}

func OrganizationDisplayNameForLocale(metadata map[string]string, locale string, fallback string) string {
	normalized := NormalizeOrganizationMetadata(metadata, nil)
	switch strings.TrimSpace(locale) {
	case "en", "en-US":
		if value := strings.TrimSpace(normalized[OrganizationMetadataDisplayNameEN]); value != "" {
			return value
		}
	case "ja", "ja-JP":
		if value := strings.TrimSpace(normalized[OrganizationMetadataDisplayNameJA]); value != "" {
			return value
		}
	case "chs":
		if value := strings.TrimSpace(normalized[OrganizationMetadataDisplayNameCHS]); value != "" {
			return value
		}
	case "cht":
		if value := strings.TrimSpace(normalized[OrganizationMetadataDisplayNameCHT]); value != "" {
			return value
		}
	}
	if value := strings.TrimSpace(normalized[OrganizationMetadataDisplayName]); value != "" {
		return value
	}
	return strings.TrimSpace(fallback)
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
		SupportEmail:   organization.SupportEmail,
		LogoURL:        organization.LogoURL,
		Domains:        organization.Domains,
		LoginPolicy:    organization.LoginPolicy,
		PasswordPolicy: organization.PasswordPolicy,
		MFAPolicy:      organization.MFAPolicy,
		Mail:           organization.Mail,
		Captcha:        organization.Captcha,
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
