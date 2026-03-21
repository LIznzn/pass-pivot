package fido

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
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

func (s *Service) BeginDiscoverableAssertionForApplication(ctx context.Context, applicationID, usage string) (string, any, error) {
	if !isSupportedAssertionUsage(usage) {
		return "", nil, errors.New("unsupported assertion usage")
	}
	if strings.TrimSpace(applicationID) == "" {
		return "", nil, errors.New("applicationId is required")
	}
	var application model.Application
	if err := s.db.WithContext(ctx).First(&application, "id = ?", applicationID).Error; err != nil {
		return "", nil, err
	}
	var project model.Project
	if err := s.db.WithContext(ctx).First(&project, "id = ?", application.ProjectID).Error; err != nil {
		return "", nil, err
	}
	wa, err := s.webAuthnForOrganization(ctx, project.OrganizationID)
	if err != nil {
		return "", nil, err
	}
	options, sessionData, err := wa.BeginDiscoverableLogin()
	if err != nil {
		return "", nil, err
	}
	return s.storeWebAuthnSession(ctx, project.OrganizationID, "", "", "assertion:"+usage, sessionData, options)
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
	record, ok := s.getWebAuthnSession(challengeID)
	if !ok || !strings.HasPrefix(record.FlowType, "assertion:") {
		return nil, errors.New("webauthn challenge not found")
	}
	if record.ExpiresAt.Before(time.Now()) {
		s.deleteWebAuthnSession(challengeID)
		return nil, errors.New("webauthn challenge expired")
	}
	var sessionData webauthn.SessionData
	if err := json.Unmarshal([]byte(record.Challenge), &sessionData); err != nil {
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
	usage := strings.TrimPrefix(record.FlowType, "assertion:")
	if strings.TrimSpace(record.UserID) == "" {
		return s.finishDiscoverableAssertion(ctx, wa, record, sessionData, parsedResponse, payload, usage)
	}
	user, webUser, err := s.loadWebAuthnUser(ctx, record.UserID, "all")
	if err != nil {
		return nil, err
	}
	credential, err := wa.ValidateLogin(webUser, sessionData, parsedResponse)
	if err != nil {
		if strings.Contains(err.Error(), "Backup Eligible flag inconsistency detected during login validation") {
			if retryErr := s.reconcileCredentialFlagsFromAssertion(ctx, user.ID, payload); retryErr == nil {
				parsedResponse, err = protocol.ParseCredentialRequestResponseBytes(payload)
				if err == nil {
					credential, err = wa.ValidateLogin(webUser, sessionData, parsedResponse)
				}
			}
		}
	}
	if err != nil {
		return nil, err
	}
	return s.buildAssertionResult(ctx, record, user.ID, credential, usage), nil
}

func (s *Service) finishDiscoverableAssertion(ctx context.Context, wa *webauthn.WebAuthn, record webauthnChallengeRecord, sessionData webauthn.SessionData, parsedResponse *protocol.ParsedCredentialAssertionData, payload json.RawMessage, usage string) (*AssertionResult, error) {
	handler := func(rawID, userHandle []byte) (webauthn.User, error) {
		userID := strings.TrimSpace(string(userHandle))
		if userID == "" {
			return nil, errors.New("blank user handle")
		}
		var enrollment model.MFAEnrollment
		if err := s.db.WithContext(ctx).
			Where("user_id = ? AND method = ?", userID, "webauthn").
			Order("created_at desc").
			First(&enrollment).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New("webauthn login is disabled")
			}
			return nil, err
		}
		if enrollment.Status != "active" {
			return nil, errors.New("webauthn login is disabled")
		}
		var user model.User
		if err := s.db.WithContext(ctx).Where("id = ? AND organization_id = ?", userID, record.OrganizationID).First(&user).Error; err != nil {
			return nil, err
		}
		_, webUser, err := s.loadWebAuthnUser(ctx, user.ID, usage)
		if err != nil {
			return nil, err
		}
		return webUser, nil
	}
	userHandle := strings.TrimSpace(string(parsedResponse.Response.UserHandle))
	user, credential, err := wa.ValidatePasskeyLogin(handler, sessionData, parsedResponse)
	if err != nil {
		if strings.Contains(err.Error(), "Backup Eligible flag inconsistency detected during login validation") && userHandle != "" {
			if retryErr := s.reconcileCredentialFlagsFromAssertion(ctx, userHandle, payload); retryErr == nil {
				parsedRetry, parseErr := protocol.ParseCredentialRequestResponseBytes(payload)
				if parseErr == nil {
					user, credential, err = wa.ValidatePasskeyLogin(handler, sessionData, parsedRetry)
				}
			}
		}
	}
	if err != nil {
		return nil, err
	}
	return s.buildAssertionResult(ctx, record, string(user.WebAuthnID()), credential, usage), nil
}

func (s *Service) buildAssertionResult(ctx context.Context, record webauthnChallengeRecord, userID string, credential *webauthn.Credential, usage string) *AssertionResult {
	credentialID := base64.RawURLEncoding.EncodeToString(credential.ID)
	_ = s.db.WithContext(ctx).
		Model(&model.SecureKey{}).
		Where("user_id = ? AND public_key_id = ?", userID, credentialID).
		Updates(map[string]any{
			"sign_count":      credential.Authenticator.SignCount,
			"backup_eligible": credential.Flags.BackupEligible,
			"backup_state":    credential.Flags.BackupState,
		}).Error
	s.deleteWebAuthnSession(record.ChallengeID)
	return &AssertionResult{
		OrganizationID: record.OrganizationID,
		UserID:         userID,
		SessionID:      record.SessionID,
		CredentialID:   credentialID,
		Usage:          usage,
	}
}

func isSupportedAssertionUsage(usage string) bool {
	switch usage {
	case "webauthn", "u2f":
		return true
	default:
		return false
	}
}
