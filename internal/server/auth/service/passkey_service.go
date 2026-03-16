package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"gorm.io/gorm"

	"pass-pivot/internal/config"
	"pass-pivot/internal/model"
	sharedauthn "pass-pivot/internal/server/shared/authn"
	"pass-pivot/util"
)

type PasskeyService struct {
	db               *gorm.DB
	cfg              config.Config
	audit            *AuditService
	auth             passkeyAuthService
	webauthnSessions map[string]webauthnChallengeRecord
	webauthnMu       sync.RWMutex
}

type passkeyAuthService interface {
	ParseFingerprint(signedFingerprint string) string
	UpsertDevice(ctx context.Context, user model.User, fingerprint, userAgent, ipAddress string, trusted bool) (*model.Device, error)
	IssueTokenPair(ctx context.Context, user model.User, session model.Session, scope string) (*sharedauthn.TokenPair, error)
	FingerprintForDevice(device *model.Device) (string, error)
	CompleteWebAuthnMFA(ctx context.Context, sessionID, method string, trustDevice bool) (*sharedauthn.LoginResult, error)
}

type webauthnUser struct {
	user        model.User
	credentials []webauthn.Credential
}

type webauthnChallengeRecord struct {
	UserID         string
	OrganizationID string
	SessionID      string
	FlowType       string
	ChallengeID    string
	Challenge      string
	ExpiresAt      time.Time
}

func NewPasskeyService(db *gorm.DB, cfg config.Config, audit *AuditService, auth passkeyAuthService) (*PasskeyService, error) {
	if _, err := url.Parse(cfg.AuthURL); err != nil {
		return nil, err
	}
	return &PasskeyService{
		db:               db,
		cfg:              cfg,
		audit:            audit,
		auth:             auth,
		webauthnSessions: map[string]webauthnChallengeRecord{},
	}, nil
}

func (s *PasskeyService) ParseFingerprint(signedFingerprint string) string {
	return s.auth.ParseFingerprint(signedFingerprint)
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

func (s *PasskeyService) BeginRegistration(ctx context.Context, userID, purpose string) (string, any, error) {
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
		purpose = "passkey"
	}
	return s.storeWebAuthnSession(ctx, user.OrganizationID, user.ID, "", "registration:"+purpose, sessionData, options)
}

func (s *PasskeyService) BeginRegistrationForSession(ctx context.Context, sessionID, purpose string) (string, any, error) {
	if sessionID == "" {
		return "", nil, errors.New("sessionId is required")
	}
	var session model.Session
	if err := s.db.WithContext(ctx).First(&session, "id = ?", sessionID).Error; err != nil {
		return "", nil, err
	}
	return s.BeginRegistration(ctx, session.UserID, purpose)
}

func (s *PasskeyService) FinishRegistration(ctx context.Context, challengeID string, payload json.RawMessage) error {
	record, sessionData, user, webUser, err := s.loadWebAuthnSessionByPrefix(ctx, challengeID, "registration:")
	if err != nil {
		return err
	}
	wa, err := s.webAuthnForOrganization(ctx, record.OrganizationID)
	if err != nil {
		return err
	}
	req := httptest.NewRequest("POST", "/api/user/v1/passkey/register/finish", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	credential, err := wa.FinishRegistration(webUser, *sessionData, req)
	if err != nil {
		return err
	}
	registrationPurpose := strings.TrimPrefix(record.FlowType, "registration:")
	entry := model.MFAPasskey{
		OrganizationID: user.OrganizationID,
		UserID:         user.ID,
		Type:           "passkey",
		Identifier:     defaultLoginIdentifier(user),
		PublicKey:      base64.RawURLEncoding.EncodeToString(credential.PublicKey),
		PublicKeyID:    base64.RawURLEncoding.EncodeToString(credential.ID),
		SignCount:      credential.Authenticator.SignCount,
		IsPasskey:      registrationPurpose == "passkey",
		IsU2f:          registrationPurpose == "passkey" || registrationPurpose == "u2f",
		BackupEligible: credential.Flags.BackupEligible,
		BackupState:    credential.Flags.BackupState,
		Transports:     transportString(credential.Transport),
	}
	if err := s.db.WithContext(ctx).Create(&entry).Error; err != nil {
		return err
	}
	if err := syncWebAuthnMFAEnrollments(ctx, s.db, user); err != nil {
		return err
	}
	s.deleteWebAuthnSession(record.ChallengeID)
	_ = s.audit.Record(ctx, AuditEvent{
		OrganizationID: user.OrganizationID,
		ActorType:      "user",
		ActorID:        user.ID,
		EventType:      "auth.passkey.registered",
		Result:         "success",
		TargetType:     "credential",
		TargetID:       entry.ID,
	})
	return nil
}

func (s *PasskeyService) BeginLogin(ctx context.Context, identifier string) (string, any, error) {
	var user model.User
	if err := s.db.WithContext(ctx).
		Where("username = ? OR email = ? OR phone_number = ?", identifier, identifier, identifier).
		First(&user).Error; err != nil {
		return "", nil, err
	}
	var enrollment model.MFAEnrollment
	if err := s.db.WithContext(ctx).
		Where("user_id = ? AND method = ?", user.ID, "passkey").
		Order("created_at desc").
		First(&enrollment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, errors.New("passkey login is disabled")
		}
		return "", nil, err
	}
	if enrollment.Status != "active" {
		return "", nil, errors.New("passkey login is disabled")
	}
	user, webUser, err := s.loadWebAuthnUser(ctx, user.ID, "passkey")
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
	return s.storeWebAuthnSession(ctx, user.OrganizationID, user.ID, "", "login", sessionData, options)
}

func (s *PasskeyService) BeginMFA(ctx context.Context, sessionID string) (string, any, error) {
	var session model.Session
	if err := s.db.WithContext(ctx).First(&session, "id = ?", sessionID).Error; err != nil {
		return "", nil, err
	}
	user, webUser, err := s.loadWebAuthnUser(ctx, session.UserID, "u2f")
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
	return s.storeWebAuthnSession(ctx, user.OrganizationID, user.ID, session.ID, "mfa", sessionData, options)
}

func (s *PasskeyService) FinishLogin(ctx context.Context, challengeID string, payload json.RawMessage, applicationID, deviceKey string) (*sharedauthn.LoginResult, error) {
	record, sessionData, user, webUser, err := s.loadWebAuthnSession(ctx, challengeID, "login")
	if err != nil {
		return nil, err
	}
	wa, err := s.webAuthnForOrganization(ctx, record.OrganizationID)
	if err != nil {
		return nil, err
	}
	req := httptest.NewRequest("POST", "/api/authn/v1/passkey/login/finish", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	credential, err := wa.FinishLogin(webUser, *sessionData, req)
	if err != nil {
		if strings.Contains(err.Error(), "Backup Eligible flag inconsistency detected during login validation") {
			if retryErr := s.reconcileCredentialFlagsFromAssertion(ctx, user.ID, payload); retryErr == nil {
				req = httptest.NewRequest("POST", "/api/authn/v1/passkey/login/finish", bytes.NewReader(payload))
				req.Header.Set("Content-Type", "application/json")
				credential, err = wa.FinishLogin(webUser, *sessionData, req)
			}
		}
	}
	if err != nil {
		return nil, err
	}
	credentialID := base64.RawURLEncoding.EncodeToString(credential.ID)
	_ = s.db.WithContext(ctx).
		Model(&model.MFAPasskey{}).
		Where("user_id = ? AND public_key_id = ?", user.ID, credentialID).
		Updates(map[string]any{
			"sign_count":      credential.Authenticator.SignCount,
			"backup_eligible": credential.Flags.BackupEligible,
			"backup_state":    credential.Flags.BackupState,
		}).Error
	var device *model.Device
	device, err = s.auth.UpsertDevice(ctx, user, deviceKey, "", "", false)
	if err != nil {
		return nil, err
	}
	session := model.Session{
		OrganizationID:        user.OrganizationID,
		UserID:                user.ID,
		ApplicationID:         applicationID,
		TrustedDeviceEligible: true,
		State:                 "authenticated",
		RiskLevel:             "low",
		SecondFactorMethod:    "",
	}
	if device != nil {
		session.DeviceID = device.ID
	}
	if err := s.db.WithContext(ctx).Create(&session).Error; err != nil {
		return nil, err
	}
	pair, err := s.auth.IssueTokenPair(ctx, user, session, "openid profile email phone")
	if err != nil {
		return nil, err
	}
	fingerprint, err := s.auth.FingerprintForDevice(device)
	if err != nil {
		return nil, err
	}
	s.deleteWebAuthnSession(record.ChallengeID)
	return &sharedauthn.LoginResult{Session: session, NextStep: "done", Tokens: sharedauthn.CompactTokens(pair), Fingerprint: fingerprint}, nil
}

func (s *PasskeyService) FinishMFA(ctx context.Context, challengeID string, payload json.RawMessage, trustDevice bool) (*sharedauthn.LoginResult, error) {
	record, sessionData, _, webUser, err := s.loadWebAuthnSession(ctx, challengeID, "mfa")
	if err != nil {
		return nil, err
	}
	wa, err := s.webAuthnForOrganization(ctx, record.OrganizationID)
	if err != nil {
		return nil, err
	}
	req := httptest.NewRequest("POST", "/api/authn/v1/session/u2f/finish", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	credential, err := wa.FinishLogin(webUser, *sessionData, req)
	if err != nil {
		return nil, err
	}
	credentialID := base64.RawURLEncoding.EncodeToString(credential.ID)
	_ = s.db.WithContext(ctx).
		Model(&model.MFAPasskey{}).
		Where("user_id = ? AND public_key_id = ?", record.UserID, credentialID).
		Updates(map[string]any{
			"sign_count":      credential.Authenticator.SignCount,
			"backup_eligible": credential.Flags.BackupEligible,
			"backup_state":    credential.Flags.BackupState,
		}).Error
	result, err := s.auth.CompleteWebAuthnMFA(ctx, record.SessionID, "u2f", trustDevice)
	if err != nil {
		return nil, err
	}
	s.deleteWebAuthnSession(record.ChallengeID)
	return result, nil
}

func (s *PasskeyService) loadWebAuthnUser(ctx context.Context, userID, usage string) (model.User, webauthnUser, error) {
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		return model.User{}, webauthnUser{}, err
	}
	query := s.db.WithContext(ctx).Where("user_id = ? AND type = ?", user.ID, "passkey")
	switch usage {
	case "passkey":
		query = query.Where("is_passkey = ?", true)
	case "u2f":
		query = query.Where("is_u2f = ?", true)
	}
	var credentials []model.MFAPasskey
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

func (s *PasskeyService) reconcileCredentialFlagsFromAssertion(ctx context.Context, userID string, payload json.RawMessage) error {
	parsed, err := protocol.ParseCredentialRequestResponseBytes(payload)
	if err != nil {
		return err
	}
	credentialID := base64.RawURLEncoding.EncodeToString(parsed.RawID)
	flags := parsed.Response.AuthenticatorData.Flags
	return s.db.WithContext(ctx).
		Model(&model.MFAPasskey{}).
		Where("user_id = ? AND public_key_id = ?", userID, credentialID).
		Updates(map[string]any{
			"backup_eligible": flags.HasBackupEligible(),
			"backup_state":    flags.HasBackupState(),
		}).Error
}

func (s *PasskeyService) storeWebAuthnSession(_ context.Context, organizationID, userID, sessionID, flow string, sessionData *webauthn.SessionData, options any) (string, any, error) {
	raw, err := json.Marshal(sessionData)
	if err != nil {
		return "", nil, err
	}
	challengeID, err := util.RandomToken(20)
	if err != nil {
		return "", nil, err
	}
	record := webauthnChallengeRecord{
		UserID:         userID,
		OrganizationID: organizationID,
		SessionID:      sessionID,
		FlowType:       flow,
		ChallengeID:    challengeID,
		Challenge:      string(raw),
		ExpiresAt:      time.Now().Add(10 * time.Minute),
	}
	s.storeInMemoryWebAuthnSession(record)
	return challengeID, options, nil
}

func (s *PasskeyService) loadWebAuthnSession(ctx context.Context, challengeID, flow string) (webauthnChallengeRecord, *webauthn.SessionData, model.User, webauthnUser, error) {
	record, ok := s.getWebAuthnSession(challengeID)
	if !ok || record.FlowType != flow {
		return webauthnChallengeRecord{}, nil, model.User{}, webauthnUser{}, errors.New("webauthn challenge not found")
	}
	if record.ExpiresAt.Before(time.Now()) {
		s.deleteWebAuthnSession(challengeID)
		return webauthnChallengeRecord{}, nil, model.User{}, webauthnUser{}, errors.New("webauthn challenge expired")
	}
	var sessionData webauthn.SessionData
	if err := json.Unmarshal([]byte(record.Challenge), &sessionData); err != nil {
		return webauthnChallengeRecord{}, nil, model.User{}, webauthnUser{}, err
	}
	user, webUser, err := s.loadWebAuthnUser(ctx, record.UserID, webauthnUsageForFlow(record.FlowType))
	if err != nil {
		return webauthnChallengeRecord{}, nil, model.User{}, webauthnUser{}, err
	}
	return record, &sessionData, user, webUser, nil
}

func (s *PasskeyService) loadWebAuthnSessionByPrefix(ctx context.Context, challengeID, flowPrefix string) (webauthnChallengeRecord, *webauthn.SessionData, model.User, webauthnUser, error) {
	record, ok := s.getWebAuthnSession(challengeID)
	if !ok || !strings.HasPrefix(record.FlowType, flowPrefix) {
		return webauthnChallengeRecord{}, nil, model.User{}, webauthnUser{}, errors.New("webauthn challenge not found")
	}
	if record.ExpiresAt.Before(time.Now()) {
		s.deleteWebAuthnSession(challengeID)
		return webauthnChallengeRecord{}, nil, model.User{}, webauthnUser{}, errors.New("webauthn challenge expired")
	}
	var sessionData webauthn.SessionData
	if err := json.Unmarshal([]byte(record.Challenge), &sessionData); err != nil {
		return webauthnChallengeRecord{}, nil, model.User{}, webauthnUser{}, err
	}
	user, webUser, err := s.loadWebAuthnUser(ctx, record.UserID, "all")
	if err != nil {
		return webauthnChallengeRecord{}, nil, model.User{}, webauthnUser{}, err
	}
	return record, &sessionData, user, webUser, nil
}

func webauthnUsageForFlow(flow string) string {
	switch {
	case strings.HasPrefix(flow, "registration:"):
		return "all"
	case flow == "login":
		return "passkey"
	case flow == "mfa":
		return "u2f"
	default:
		return "all"
	}
}

func (s *PasskeyService) storeInMemoryWebAuthnSession(record webauthnChallengeRecord) {
	s.cleanupExpiredWebAuthnSessions()
	s.webauthnMu.Lock()
	defer s.webauthnMu.Unlock()
	s.webauthnSessions[record.ChallengeID] = record
}

func (s *PasskeyService) getWebAuthnSession(challengeID string) (webauthnChallengeRecord, bool) {
	s.cleanupExpiredWebAuthnSessions()
	s.webauthnMu.RLock()
	defer s.webauthnMu.RUnlock()
	record, ok := s.webauthnSessions[challengeID]
	return record, ok
}

func (s *PasskeyService) deleteWebAuthnSession(challengeID string) {
	s.webauthnMu.Lock()
	defer s.webauthnMu.Unlock()
	delete(s.webauthnSessions, challengeID)
}

func (s *PasskeyService) cleanupExpiredWebAuthnSessions() {
	now := time.Now()
	s.webauthnMu.Lock()
	defer s.webauthnMu.Unlock()
	for challengeID, record := range s.webauthnSessions {
		if record.ExpiresAt.Before(now) {
			delete(s.webauthnSessions, challengeID)
		}
	}
}

func transportString(transports []protocol.AuthenticatorTransport) string {
	values := make([]string, 0, len(transports))
	for _, item := range transports {
		values = append(values, string(item))
	}
	return strings.Join(values, ",")
}

func (s *PasskeyService) webAuthnForOrganization(ctx context.Context, organizationID string) (*webauthn.WebAuthn, error) {
	origin, err := url.Parse(s.cfg.AuthURL)
	if err != nil {
		return nil, err
	}
	organization, _, err := loadOrganizationConsoleSettings(ctx, s.db, organizationID)
	if err != nil {
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

func (s *PasskeyService) resolveOrganizationRPOrigins(ctx context.Context, organizationID, fallbackOrigin string) ([]string, error) {
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
