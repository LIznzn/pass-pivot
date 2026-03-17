package fido

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	"github.com/go-webauthn/webauthn/protocol"

	"pass-pivot/internal/model"
)

func (s *Service) BeginRegistration(ctx context.Context, userID, purpose string) (string, any, error) {
	user, webUser, err := s.loadWebAuthnUser(ctx, userID, "all")
	if err != nil {
		return "", nil, err
	}
	wa, err := s.webAuthnForOrganization(ctx, user.OrganizationID)
	if err != nil {
		return "", nil, err
	}
	options, sessionData, err := wa.BeginRegistration(webUser)
	if err != nil {
		return "", nil, err
	}
	if purpose == "" {
		purpose = "webauthn"
	}
	return s.storeWebAuthnSession(ctx, user.OrganizationID, user.ID, "", "registration:"+purpose, sessionData, options)
}

func (s *Service) BeginRegistrationForSession(ctx context.Context, sessionID, purpose string) (string, any, error) {
	if sessionID == "" {
		return "", nil, errors.New("sessionId is required")
	}
	var session model.Session
	if err := s.db.WithContext(ctx).First(&session, "id = ?", sessionID).Error; err != nil {
		return "", nil, err
	}
	return s.BeginRegistration(ctx, session.UserID, purpose)
}

func (s *Service) FinishRegistration(ctx context.Context, challengeID string, payload json.RawMessage) error {
	record, sessionData, user, webUser, err := s.loadWebAuthnSessionByPrefix(ctx, challengeID, "registration:")
	if err != nil {
		return err
	}
	wa, err := s.webAuthnForOrganization(ctx, record.OrganizationID)
	if err != nil {
		return err
	}
	parsedResponse, err := protocol.ParseCredentialCreationResponseBytes(payload)
	if err != nil {
		return err
	}
	credential, err := wa.CreateCredential(webUser, *sessionData, parsedResponse)
	if err != nil {
		return err
	}
	registrationPurpose := strings.TrimPrefix(record.FlowType, "registration:")
	entry := model.SecureKey{
		OrganizationID: user.OrganizationID,
		UserID:         user.ID,
		Type:           "securekey",
		Identifier:     defaultLoginIdentifier(user),
		PublicKey:      base64.RawURLEncoding.EncodeToString(credential.PublicKey),
		PublicKeyID:    base64.RawURLEncoding.EncodeToString(credential.ID),
		SignCount:      credential.Authenticator.SignCount,
		WebAuthnEnable: registrationPurpose == "webauthn",
		U2FEnable:      registrationPurpose == "webauthn" || registrationPurpose == "u2f",
		BackupEligible: credential.Flags.BackupEligible,
		BackupState:    credential.Flags.BackupState,
		Transports:     transportString(credential.Transport),
	}
	if err := s.db.WithContext(ctx).Create(&entry).Error; err != nil {
		return err
	}
	if err := SyncCredentialEnrollments(ctx, s.db, user); err != nil {
		return err
	}
	s.deleteWebAuthnSession(record.ChallengeID)
	if s.recordRegistrationAudit != nil {
		if err := s.recordRegistrationAudit(ctx, RegistrationAuditRecord{
			OrganizationID: user.OrganizationID,
			UserID:         user.ID,
			CredentialID:   entry.ID,
			Purpose:        registrationPurpose,
		}); err != nil {
			return err
		}
	}
	return nil
}
