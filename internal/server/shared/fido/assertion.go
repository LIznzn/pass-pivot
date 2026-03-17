package fido

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	"github.com/go-webauthn/webauthn/protocol"
	"gorm.io/gorm"

	"pass-pivot/internal/model"
)

func (s *Service) BeginAssertionForIdentifier(ctx context.Context, identifier, usage string) (string, any, error) {
	if !isSupportedAssertionUsage(usage) {
		return "", nil, errors.New("unsupported assertion usage")
	}
	var user model.User
	if err := s.db.WithContext(ctx).
		Where("username = ? OR email = ? OR phone_number = ?", identifier, identifier, identifier).
		First(&user).Error; err != nil {
		return "", nil, err
	}
	return s.beginAssertionForUser(ctx, user.ID, "", usage)
}

func (s *Service) BeginAssertionForSession(ctx context.Context, sessionID, usage string) (string, any, error) {
	if sessionID == "" {
		return "", nil, errors.New("sessionId is required")
	}
	if !isSupportedAssertionUsage(usage) {
		return "", nil, errors.New("unsupported assertion usage")
	}
	var session model.Session
	if err := s.db.WithContext(ctx).First(&session, "id = ?", sessionID).Error; err != nil {
		return "", nil, err
	}
	return s.beginAssertionForUser(ctx, session.UserID, session.ID, usage)
}

func (s *Service) beginAssertionForUser(ctx context.Context, userID, sessionID, usage string) (string, any, error) {
	if usage == "webauthn" {
		var enrollment model.MFAEnrollment
		if err := s.db.WithContext(ctx).
			Where("user_id = ? AND method = ?", userID, "webauthn").
			Order("created_at desc").
			First(&enrollment).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return "", nil, errors.New("webauthn login is disabled")
			}
			return "", nil, err
		}
		if enrollment.Status != "active" {
			return "", nil, errors.New("webauthn login is disabled")
		}
	}
	user, webUser, err := s.loadWebAuthnUser(ctx, userID, usage)
	if err != nil {
		return "", nil, err
	}
	wa, err := s.webAuthnForOrganization(ctx, user.OrganizationID)
	if err != nil {
		return "", nil, err
	}
	options, sessionData, err := wa.BeginLogin(webUser)
	if err != nil {
		return "", nil, err
	}
	return s.storeWebAuthnSession(ctx, user.OrganizationID, user.ID, sessionID, "assertion:"+usage, sessionData, options)
}

func (s *Service) FinishAssertion(ctx context.Context, challengeID string, payload json.RawMessage) (*AssertionResult, error) {
	record, sessionData, user, webUser, err := s.loadWebAuthnSessionByPrefix(ctx, challengeID, "assertion:")
	if err != nil {
		return nil, err
	}
	wa, err := s.webAuthnForOrganization(ctx, record.OrganizationID)
	if err != nil {
		return nil, err
	}
	parsedResponse, err := protocol.ParseCredentialRequestResponseBytes(payload)
	if err != nil {
		return nil, err
	}
	credential, err := wa.ValidateLogin(webUser, *sessionData, parsedResponse)
	if err != nil {
		if strings.Contains(err.Error(), "Backup Eligible flag inconsistency detected during login validation") {
			if retryErr := s.reconcileCredentialFlagsFromAssertion(ctx, user.ID, payload); retryErr == nil {
				parsedResponse, err = protocol.ParseCredentialRequestResponseBytes(payload)
				if err == nil {
					credential, err = wa.ValidateLogin(webUser, *sessionData, parsedResponse)
				}
			}
		}
	}
	if err != nil {
		return nil, err
	}
	usage := strings.TrimPrefix(record.FlowType, "assertion:")
	credentialID := base64.RawURLEncoding.EncodeToString(credential.ID)
	_ = s.db.WithContext(ctx).
		Model(&model.SecureKey{}).
		Where("user_id = ? AND public_key_id = ?", user.ID, credentialID).
		Updates(map[string]any{
			"sign_count":      credential.Authenticator.SignCount,
			"backup_eligible": credential.Flags.BackupEligible,
			"backup_state":    credential.Flags.BackupState,
		}).Error
	s.deleteWebAuthnSession(record.ChallengeID)
	return &AssertionResult{
		OrganizationID: record.OrganizationID,
		UserID:         record.UserID,
		SessionID:      record.SessionID,
		CredentialID:   credentialID,
		Usage:          usage,
	}, nil
}

func isSupportedAssertionUsage(usage string) bool {
	switch usage {
	case "webauthn", "u2f":
		return true
	default:
		return false
	}
}
