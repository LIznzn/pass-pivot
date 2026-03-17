package fido

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	"pass-pivot/internal/model"
)

type webauthnUser struct {
	user        model.User
	credentials []webauthn.Credential
}

func (u webauthnUser) WebAuthnID() []byte {
	return []byte(u.user.ID)
}

func (u webauthnUser) WebAuthnName() string {
	return defaultLoginIdentifier(u.user)
}

func (u webauthnUser) WebAuthnDisplayName() string {
	return u.user.Name
}

func (u webauthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.credentials
}

func (u webauthnUser) WebAuthnIcon() string {
	return ""
}

func (s *Service) loadWebAuthnUser(ctx context.Context, userID, usage string) (model.User, webauthnUser, error) {
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		return model.User{}, webauthnUser{}, err
	}
	query := s.db.WithContext(ctx).Where("user_id = ?", user.ID)
	switch usage {
	case "webauthn":
		query = query.Where("webauthn_enable = ?", true)
	case "u2f":
		query = query.Where("u2f_enable = ?", true)
	}
	var credentials []model.SecureKey
	if err := query.Find(&credentials).Error; err != nil {
		return model.User{}, webauthnUser{}, err
	}
	items := make([]webauthn.Credential, 0, len(credentials))
	for _, item := range credentials {
		id, _ := base64.RawURLEncoding.DecodeString(item.PublicKeyID)
		publicKey, _ := base64.RawURLEncoding.DecodeString(item.PublicKey)
		items = append(items, webauthn.Credential{
			ID:        id,
			PublicKey: publicKey,
			Flags: webauthn.CredentialFlags{
				BackupEligible: item.BackupEligible,
				BackupState:    item.BackupState,
			},
			Authenticator: webauthn.Authenticator{
				SignCount: item.SignCount,
			},
		})
	}
	return user, webauthnUser{user: user, credentials: items}, nil
}

func (s *Service) reconcileCredentialFlagsFromAssertion(ctx context.Context, userID string, payload json.RawMessage) error {
	parsed, err := protocol.ParseCredentialRequestResponseBytes(payload)
	if err != nil {
		return err
	}
	credentialID := base64.RawURLEncoding.EncodeToString(parsed.RawID)
	flags := parsed.Response.AuthenticatorData.Flags
	return s.db.WithContext(ctx).
		Model(&model.SecureKey{}).
		Where("user_id = ? AND public_key_id = ?", userID, credentialID).
		Updates(map[string]any{
			"backup_eligible": flags.HasBackupEligible(),
			"backup_state":    flags.HasBackupState(),
		}).Error
}

func transportString(transports []protocol.AuthenticatorTransport) string {
	values := make([]string, 0, len(transports))
	for _, item := range transports {
		values = append(values, string(item))
	}
	return strings.Join(values, ",")
}

func (s *Service) webAuthnForOrganization(ctx context.Context, organizationID string) (*webauthn.WebAuthn, error) {
	origin, err := url.Parse(s.cfg.AuthURL)
	if err != nil {
		return nil, err
	}
	var organization model.Organization
	if err := s.db.WithContext(ctx).First(&organization, "id = ?", organizationID).Error; err != nil {
		return nil, err
	}
	rpOrigins, err := s.resolveOrganizationRPOrigins(ctx, organizationID, s.cfg.AuthURL)
	if err != nil {
		return nil, err
	}
	displayName := strings.TrimSpace(organization.Name)
	if displayName == "" {
		displayName = organization.ID
	}
	return webauthn.New(&webauthn.Config{
		RPDisplayName: displayName,
		RPID:          origin.Hostname(),
		RPOrigins:     rpOrigins,
	})
}

func (s *Service) resolveOrganizationRPOrigins(ctx context.Context, organizationID, fallbackOrigin string) ([]string, error) {
	seen := map[string]bool{}
	origins := make([]string, 0, 4)
	appendOrigin := func(raw string) {
		value := strings.TrimSpace(raw)
		if value == "" {
			return
		}
		parsed, err := url.Parse(value)
		if err != nil || strings.TrimSpace(parsed.Scheme) == "" || strings.TrimSpace(parsed.Host) == "" {
			return
		}
		origin := parsed.Scheme + "://" + parsed.Host
		if seen[origin] {
			return
		}
		seen[origin] = true
		origins = append(origins, origin)
	}

	appendOrigin(fallbackOrigin)

	var projects []model.Project
	if err := s.db.WithContext(ctx).Where("organization_id = ?", organizationID).Find(&projects).Error; err != nil {
		return nil, err
	}
	if len(projects) == 0 {
		return origins, nil
	}
	projectIDs := make([]string, 0, len(projects))
	for _, project := range projects {
		projectIDs = append(projectIDs, project.ID)
	}
	var applications []model.Application
	if err := s.db.WithContext(ctx).Where("project_id IN ?", projectIDs).Find(&applications).Error; err != nil {
		return nil, err
	}
	for _, application := range applications {
		for _, item := range splitRedirectURIs(application.RedirectURIs) {
			appendOrigin(item)
		}
	}
	return origins, nil
}

func defaultLoginIdentifier(user model.User) string {
	if strings.TrimSpace(user.Email) != "" {
		return user.Email
	}
	if strings.TrimSpace(user.PhoneNumber) != "" {
		return user.PhoneNumber
	}
	return user.Username
}

func splitRedirectURIs(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == ' '
	})
	items := make([]string, 0, len(fields))
	for _, item := range fields {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			items = append(items, trimmed)
		}
	}
	return items
}
