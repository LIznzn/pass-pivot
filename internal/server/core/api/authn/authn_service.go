package authn

import (
	"context"
	"encoding/json"
	"errors"
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

	"gorm.io/gorm"
)

type AuthnService struct {
	db    *gorm.DB
	cfg   config.Config
	audit *coreservice.AuditService
	mfa   *authservice.MFAService
	fido  webAuthnFIDOService
}

type webAuthnFIDOService interface {
	BeginAssertionForIdentifier(ctx context.Context, identifier, usage string) (string, any, error)
	BeginDiscoverableAssertionForApplication(ctx context.Context, applicationID, usage string) (string, any, error)
	FinishAssertion(ctx context.Context, challengeID string, payload json.RawMessage) (*sharedfido.AssertionResult, error)
}

func applicationIsActive(app model.Application) bool {
	return strings.TrimSpace(app.Status) != "disabled"
}

func organizationIsActive(org model.Organization) bool {
	return strings.TrimSpace(org.Status) != "disabled"
}

func NewAuthnService(db *gorm.DB, cfg config.Config, audit *coreservice.AuditService, mfa *authservice.MFAService) *AuthnService {
	return &AuthnService{db: db, cfg: cfg, audit: audit, mfa: mfa}
}

func (s *AuthnService) SetFIDOService(fido webAuthnFIDOService) {
	s.fido = fido
}

func (s *AuthnService) BeginWebAuthnLogin(ctx context.Context, identifier, applicationID string) (string, any, error) {
	if s.fido == nil {
		return "", nil, errors.New("fido service is not configured")
	}
	if strings.TrimSpace(identifier) == "" {
		return s.fido.BeginDiscoverableAssertionForApplication(ctx, applicationID, "webauthn")
	}
	return s.fido.BeginAssertionForIdentifier(ctx, identifier, "webauthn")
}

func (s *AuthnService) FinishWebAuthnLogin(ctx context.Context, challengeID string, payload json.RawMessage, applicationID, deviceKey string) (*sharedauthn.LoginResult, error) {
	if s.fido == nil {
		return nil, errors.New("fido service is not configured")
	}
	assertion, err := s.fido.FinishAssertion(ctx, challengeID, payload)
	if err != nil {
		return nil, err
	}
	if assertion.Usage != "webauthn" {
		return nil, errors.New("fido assertion usage mismatch")
	}
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", assertion.UserID).Error; err != nil {
		return nil, err
	}
	var organization model.Organization
	if err := s.db.WithContext(ctx).First(&organization, "id = ?", user.OrganizationID).Error; err != nil {
		return nil, err
	}
	if !organizationIsActive(organization) {
		return nil, errors.New("organization is disabled")
	}
	if strings.TrimSpace(applicationID) != "" {
		var app model.Application
		if err := s.db.WithContext(ctx).First(&app, "id = ?", applicationID).Error; err != nil {
			return nil, err
		}
		if !applicationIsActive(app) {
			return nil, errors.New("application is disabled")
		}
	}
	allowed, err := s.isUserAllowedForApplication(ctx, applicationID, user.ID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, errors.New("user is not assigned to the target project")
	}
	device, err := s.upsertDevice(ctx, user, deviceKey, "", "", false)
	if err != nil {
		return nil, err
	}
	session := model.Session{
		OrganizationID:        assertion.OrganizationID,
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
	s.recordLoginSucceeded(ctx, session, user.ID, "", "", map[string]any{
		"method": "webauthn",
	})
	tokens, err := s.issueTokens(ctx, user, session, "openid profile email phone")
	if err != nil {
		return nil, err
	}
	fingerprint, err := s.fingerprintForDevice(device)
	if err != nil {
		return nil, err
	}
	return &sharedauthn.LoginResult{Session: session, NextStep: "done", Tokens: tokens, Fingerprint: fingerprint}, nil
}

func (s *AuthnService) LoginWithUserCredential(ctx context.Context, in sharedauthn.LoginInput) (*sharedauthn.LoginResult, error) {
	if strings.TrimSpace(in.ApplicationID) != "" {
		var app model.Application
		if err := s.db.WithContext(ctx).First(&app, "id = ?", in.ApplicationID).Error; err != nil {
			return nil, err
		}
		if !applicationIsActive(app) {
			return nil, errors.New("application is disabled")
		}
	}
	user, err := s.findUserByIdentifier(ctx, in.OrganizationID, in.Identifier)
	if err != nil {
		_ = s.audit.Record(ctx, coreservice.AuditEvent{
			OrganizationID: in.OrganizationID,
			ApplicationID:  in.ApplicationID,
			ActorType:      "anonymous",
			EventType:      "auth.login.failed",
			Result:         "denied",
			TargetType:     "user",
			IPAddress:      in.IPAddress,
			UserAgent:      in.UserAgent,
			Detail:         map[string]any{"identifier": in.Identifier, "reason": "user_not_found"},
		})
		return nil, errors.New("invalid credentials")
	}
	if strings.TrimSpace(user.PasswordHash) == "" || !util.CheckSecret(user.PasswordHash, in.Secret) {
		_ = s.audit.Record(ctx, coreservice.AuditEvent{
			OrganizationID: in.OrganizationID,
			ApplicationID:  in.ApplicationID,
			ActorType:      "anonymous",
			EventType:      "auth.login.failed",
			Result:         "denied",
			TargetType:     "user",
			TargetID:       user.ID,
			IPAddress:      in.IPAddress,
			UserAgent:      in.UserAgent,
			Detail:         map[string]any{"identifier": in.Identifier, "reason": "secret_mismatch"},
		})
		return nil, errors.New("invalid credentials")
	}
	if user.Status != "active" {
		return nil, errors.New("user is not active")
	}
	var organization model.Organization
	if err := s.db.WithContext(ctx).First(&organization, "id = ?", user.OrganizationID).Error; err != nil {
		return nil, err
	}
	if !organizationIsActive(organization) {
		return nil, errors.New("organization is disabled")
	}
	allowed, err := s.isUserAllowedForApplication(ctx, in.ApplicationID, user.ID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, errors.New("user is not assigned to the target project")
	}

	var device *model.Device
	trusted := false
	if in.DeviceKey != "" {
		device, err = s.upsertDevice(ctx, user, in.DeviceKey, in.UserAgent, in.IPAddress, false)
		if err != nil {
			return nil, err
		}
		trusted = device.Trusted
	} else {
		device, err = s.upsertDevice(ctx, user, "", in.UserAgent, in.IPAddress, false)
		if err != nil {
			return nil, err
		}
	}
	requiresMFA, preferredMFAMethod, err := s.evaluateMFALoginRequirement(ctx, user)
	if err != nil {
		return nil, err
	}

	session := model.Session{
		OrganizationID:        in.OrganizationID,
		UserID:                user.ID,
		ApplicationID:         in.ApplicationID,
		TrustedDeviceEligible: device != nil,
		State:                 "authenticated",
		RequiresConfirmation:  !trusted && requiresMFA && device != nil,
		RequiresMFA:           !trusted && requiresMFA,
		SecondFactorMethod:    preferredMFAMethod,
		IPAddress:             in.IPAddress,
		UserAgent:             in.UserAgent,
		RiskLevel:             "medium",
	}
	if device != nil {
		session.DeviceID = device.ID
	}
	if trusted {
		session.State = "authenticated"
		session.RequiresConfirmation = false
		session.RequiresMFA = false
	} else if session.RequiresMFA {
		session.State = "mfa_required"
	} else {
		session.State = "authenticated"
	}
	if in.RequireAnnouncement {
		session.State = "confirmation_required"
		session.RequiresConfirmation = true
	}
	session.LoginChallenge, _ = util.RandomToken(18)
	if err := s.db.WithContext(ctx).Create(&session).Error; err != nil {
		return nil, err
	}

	if trusted {
		s.recordLoginSucceeded(ctx, session, user.ID, in.IPAddress, in.UserAgent, map[string]any{
			"trustedDevice": trusted,
		})
		tokens, err := s.issueTokens(ctx, user, session, "openid profile email phone")
		if err != nil {
			return nil, err
		}
		fingerprint, err := s.fingerprintForDevice(device)
		if err != nil {
			return nil, err
		}
		return &sharedauthn.LoginResult{Session: session, NextStep: "done", Tokens: tokens, Fingerprint: fingerprint}, nil
	}
	fingerprint, err := s.fingerprintForDevice(device)
	if err != nil {
		return nil, err
	}
	if session.RequiresMFA {
		return &sharedauthn.LoginResult{Session: session, NextStep: "mfa", Fingerprint: fingerprint}, nil
	}
	if session.RequiresConfirmation {
		return &sharedauthn.LoginResult{Session: session, NextStep: "confirmation", Fingerprint: fingerprint}, nil
	}
	s.recordLoginSucceeded(ctx, session, user.ID, in.IPAddress, in.UserAgent, map[string]any{
		"trustedDevice": trusted,
	})
	return &sharedauthn.LoginResult{Session: session, NextStep: "done", Fingerprint: fingerprint}, nil
}

func (s *AuthnService) evaluateMFALoginRequirement(ctx context.Context, user model.User) (bool, string, error) {
	_, settings, err := coreservice.LoadOrganizationConsoleSettings(ctx, s.db, user.OrganizationID)
	if err != nil {
		return false, "", err
	}
	methods := make([]string, 0, 4)

	if settings.MFAPolicy.AllowU2F {
		var u2fCount int64
		if err := s.db.WithContext(ctx).Model(&model.SecureKey{}).
			Where("user_id = ? AND u2f_enable = ? AND deleted_at IS NULL", user.ID, true).
			Count(&u2fCount).Error; err != nil {
			return false, "", err
		}
		var u2fEnrollmentCount int64
		if err := s.db.WithContext(ctx).Model(&model.MFAEnrollment{}).
			Where("user_id = ? AND method = ? AND status = ? AND deleted_at IS NULL", user.ID, "u2f", "active").
			Count(&u2fEnrollmentCount).Error; err != nil {
			return false, "", err
		}
		if u2fCount > 0 && u2fEnrollmentCount > 0 {
			methods = append(methods, "u2f")
		}
	}

	if settings.MFAPolicy.AllowWebAuthn {
		var webauthnEnrollmentCount int64
		if err := s.db.WithContext(ctx).Model(&model.MFAEnrollment{}).
			Where("user_id = ? AND method = ? AND status = ? AND deleted_at IS NULL", user.ID, "webauthn", "active").
			Count(&webauthnEnrollmentCount).Error; err != nil {
			return false, "", err
		}
		if webauthnEnrollmentCount > 0 {
			methods = append(methods, "webauthn")
		}
	}

	if settings.MFAPolicy.AllowTotp {
		var totpCount int64
		if err := s.db.WithContext(ctx).Model(&model.MFAEnrollment{}).
			Where("user_id = ? AND method = ? AND status = ? AND deleted_at IS NULL", user.ID, "totp", "active").
			Count(&totpCount).Error; err != nil {
			return false, "", err
		}
		if totpCount > 0 {
			methods = append(methods, "totp")
		}
	}

	if settings.MFAPolicy.AllowRecoveryCode {
		var recoveryCount int64
		if err := s.db.WithContext(ctx).Model(&model.MFARecoveryCode{}).
			Where("user_id = ? AND consumed_at IS NULL AND deleted_at IS NULL", user.ID).
			Count(&recoveryCount).Error; err != nil {
			return false, "", err
		}
		if recoveryCount > 0 {
			methods = append(methods, "recovery_code")
		}
	}

	if settings.MFAPolicy.AllowEmailCode &&
		strings.TrimSpace(user.Email) != "" &&
		settings.MFAPolicy.EmailChannel.Enabled &&
		strings.TrimSpace(settings.MFAPolicy.EmailChannel.From) != "" &&
		strings.TrimSpace(settings.MFAPolicy.EmailChannel.Host) != "" &&
		settings.MFAPolicy.EmailChannel.Port > 0 {
		methods = append(methods, "email_code")
	}

	if settings.MFAPolicy.AllowSmsCode && strings.TrimSpace(user.PhoneNumber) != "" {
		methods = append(methods, "sms_code")
	}

	if len(methods) == 0 {
		return false, "", nil
	}
	return true, methods[0], nil
}

func (s *AuthnService) ConfirmSession(ctx context.Context, sessionID string, accept bool, trustDevice bool) (*sharedauthn.LoginResult, error) {
	var session model.Session
	if err := s.db.WithContext(ctx).First(&session, "id = ?", sessionID).Error; err != nil {
		return nil, err
	}
	if !accept {
		session.State = "rejected"
		if err := s.db.WithContext(ctx).Save(&session).Error; err != nil {
			return nil, err
		}
		return nil, errors.New("confirmation rejected")
	}
	session.RequiresConfirmation = false
	if trustDevice && session.DeviceID != "" {
		if err := s.db.WithContext(ctx).Model(&model.Device{}).Where("id = ?", session.DeviceID).Update("trusted", true).Error; err != nil {
			return nil, err
		}
	}
	if session.RequiresMFA {
		session.State = "mfa_required"
	} else {
		session.State = "authenticated"
	}
	if err := s.db.WithContext(ctx).Save(&session).Error; err != nil {
		return nil, err
	}
	if session.RequiresMFA {
		return &sharedauthn.LoginResult{Session: session, NextStep: "mfa"}, nil
	}
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", session.UserID).Error; err != nil {
		return nil, err
	}
	s.recordLoginSucceeded(ctx, session, user.ID, session.IPAddress, session.UserAgent, map[string]any{
		"trustedDevice": trustDevice,
	})
	tokens, err := s.issueTokens(ctx, user, session, "openid profile email phone")
	if err != nil {
		return nil, err
	}
	return &sharedauthn.LoginResult{Session: session, NextStep: "done", Tokens: tokens}, nil
}

func (s *AuthnService) VerifyMFA(ctx context.Context, sessionID, method, code string, trustDevice bool) (*sharedauthn.LoginResult, error) {
	var session model.Session
	if err := s.db.WithContext(ctx).First(&session, "id = ?", sessionID).Error; err != nil {
		return nil, err
	}
	if session.State != "mfa_required" && session.State != "confirmation_required" {
		return nil, errors.New("session is not awaiting mfa")
	}
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", session.UserID).Error; err != nil {
		return nil, err
	}
	switch method {
	case "u2f", "webauthn":
		return nil, errors.New("use WebAuthn completion endpoint for webauthn/u2f verification")
	default:
		if err := s.mfa.Verify(ctx, sessionID, method, code); err != nil {
			return nil, err
		}
	}
	return s.completeMFASession(ctx, user, &session, method, trustDevice)
}

func (s *AuthnService) CompleteWebAuthnMFA(ctx context.Context, sessionID, method string, trustDevice bool) (*sharedauthn.LoginResult, error) {
	var session model.Session
	if err := s.db.WithContext(ctx).First(&session, "id = ?", sessionID).Error; err != nil {
		return nil, err
	}
	if session.State != "mfa_required" && session.State != "confirmation_required" {
		return nil, errors.New("session is not awaiting mfa")
	}
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", session.UserID).Error; err != nil {
		return nil, err
	}
	return s.completeMFASession(ctx, user, &session, method, trustDevice)
}

func (s *AuthnService) completeMFASession(ctx context.Context, user model.User, session *model.Session, method string, trustDevice bool) (*sharedauthn.LoginResult, error) {
	session.RequiresMFA = false
	session.SecondFactorMethod = method
	if session.RequiresConfirmation {
		session.State = "confirmation_required"
		if err := s.db.WithContext(ctx).Save(session).Error; err != nil {
			return nil, err
		}
		_ = s.audit.Record(ctx, coreservice.AuditEvent{
			OrganizationID: session.OrganizationID,
			ApplicationID:  session.ApplicationID,
			ActorType:      "user",
			ActorID:        session.UserID,
			EventType:      "auth.mfa.verified",
			Result:         "success",
			TargetType:     "session",
			TargetID:       session.ID,
			Detail:         map[string]any{"method": method},
		})
		return &sharedauthn.LoginResult{Session: *session, NextStep: "confirmation"}, nil
	}
	session.State = "authenticated"
	if trustDevice && session.DeviceID != "" {
		if err := s.db.WithContext(ctx).Model(&model.Device{}).Where("id = ? AND user_id = ?", session.DeviceID, user.ID).Update("trusted", true).Error; err != nil {
			return nil, err
		}
	}
	if err := s.db.WithContext(ctx).Save(session).Error; err != nil {
		return nil, err
	}
	s.recordLoginSucceeded(ctx, *session, user.ID, session.IPAddress, session.UserAgent, map[string]any{
		"trustedDevice": trustDevice,
	})
	tokens, err := s.issueTokens(ctx, user, *session, "openid profile email phone")
	if err != nil {
		return nil, err
	}
	_ = s.audit.Record(ctx, coreservice.AuditEvent{
		OrganizationID: session.OrganizationID,
		ApplicationID:  session.ApplicationID,
		ActorType:      "user",
		ActorID:        session.UserID,
		EventType:      "auth.mfa.verified",
		Result:         "success",
		TargetType:     "session",
		TargetID:       session.ID,
		Detail:         map[string]any{"method": method},
	})
	return &sharedauthn.LoginResult{Session: *session, NextStep: "done", Tokens: tokens}, nil
}

func (s *AuthnService) recordLoginSucceeded(ctx context.Context, session model.Session, userID, ipAddress, userAgent string, detail map[string]any) {
	_ = s.audit.Record(ctx, coreservice.AuditEvent{
		OrganizationID: session.OrganizationID,
		ApplicationID:  session.ApplicationID,
		ActorType:      "user",
		ActorID:        userID,
		EventType:      "auth.login.succeeded",
		Result:         "success",
		TargetType:     "session",
		TargetID:       session.ID,
		IPAddress:      ipAddress,
		UserAgent:      userAgent,
		Detail:         detail,
	})
}

func (s *AuthnService) upsertDevice(ctx context.Context, user model.User, fingerprint, userAgent, ipAddress string, trusted bool) (*model.Device, error) {
	if fingerprint == "" {
		var err error
		fingerprint, err = util.GenerateFingerprint()
		if err != nil {
			return nil, err
		}
	}
	var device model.Device
	err := s.db.WithContext(ctx).Where("user_id = ? AND fingerprint = ?", user.ID, fingerprint).First(&device).Error
	now := time.Now()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		device = model.Device{
			UserID:         user.ID,
			OrganizationID: user.OrganizationID,
			Fingerprint:    fingerprint,
			Description:    "browser device",
			UserAgent:      userAgent,
			LastLoginIP:    ipAddress,
			FirstSeenAt:    &now,
			LastSeenAt:     now,
			Trusted:        trusted,
		}
		if err := s.db.WithContext(ctx).Create(&device).Error; err != nil {
			return nil, err
		}
		return &device, nil
	}
	if err != nil {
		return nil, err
	}
	updates := map[string]any{
		"user_agent":    userAgent,
		"last_login_ip": ipAddress,
		"last_seen_at":  now,
	}
	if device.FirstSeenAt == nil {
		updates["first_seen_at"] = &now
	}
	if trusted {
		updates["trusted"] = true
	}
	if err := s.db.WithContext(ctx).Model(&device).Updates(updates).Error; err != nil {
		return nil, err
	}
	if userAgent != "" {
		device.UserAgent = userAgent
	}
	if ipAddress != "" {
		device.LastLoginIP = ipAddress
	}
	device.LastSeenAt = now
	if trusted {
		device.Trusted = true
	}
	if device.FirstSeenAt == nil {
		device.FirstSeenAt = &now
	}
	return &device, nil
}

func (s *AuthnService) ParseFingerprint(signedFingerprint string) string {
	deviceKey, ok := util.VerifyFingerprint(signedFingerprint, s.cfg.Secret)
	if !ok {
		return ""
	}
	return deviceKey
}

func (s *AuthnService) fingerprintForDevice(device *model.Device) (string, error) {
	if device == nil {
		return "", nil
	}
	return util.SignFingerprint(device.Fingerprint, s.cfg.Secret)
}

func (s *AuthnService) RequestMFAChallenge(ctx context.Context, sessionID, method string) (*model.MFAChallenge, string, error) {
	return s.mfa.CreateDeliveryChallenge(ctx, sessionID, method)
}

func (s *AuthnService) userIDFromSession(ctx context.Context, sessionID string) (string, error) {
	if sessionID == "" {
		return "", errors.New("sessionId is required")
	}
	var session model.Session
	if err := s.db.WithContext(ctx).First(&session, "id = ?", sessionID).Error; err != nil {
		return "", err
	}
	return session.UserID, nil
}

func (s *AuthnService) EnrollTOTP(ctx context.Context, userID, applicationID string) (*authservice.TOTPEnrollmentResult, error) {
	return s.mfa.EnrollTOTPForApplication(ctx, userID, applicationID)
}

func (s *AuthnService) EnrollCurrentUserTOTP(ctx context.Context, sessionID, applicationID string) (*authservice.TOTPEnrollmentResult, error) {
	userID, err := s.userIDFromSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return s.EnrollTOTP(ctx, userID, applicationID)
}

func (s *AuthnService) VerifyTOTPEnrollment(ctx context.Context, userID, enrollmentID, code string) error {
	return s.mfa.VerifyTOTPEnrollment(ctx, userID, enrollmentID, code)
}

func (s *AuthnService) VerifyCurrentUserTOTPEnrollment(ctx context.Context, sessionID, enrollmentID, code string) error {
	userID, err := s.userIDFromSession(ctx, sessionID)
	if err != nil {
		return err
	}
	return s.VerifyTOTPEnrollment(ctx, userID, enrollmentID, code)
}

func (s *AuthnService) GenerateRecoveryCodes(ctx context.Context, userID string) ([]string, error) {
	return s.mfa.GenerateRecoveryCodes(ctx, userID)
}

func (s *AuthnService) GenerateCurrentUserRecoveryCodes(ctx context.Context, sessionID string) ([]string, error) {
	userID, err := s.userIDFromSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return s.GenerateRecoveryCodes(ctx, userID)
}

func (s *AuthnService) CanManageUser(ctx context.Context, roleNames []string, userID string) (bool, error) {
	if strings.TrimSpace(userID) == "" {
		return false, errors.New("userId is required")
	}
	var user model.User
	if err := s.db.WithContext(ctx).Select("id", "organization_id").First(&user, "id = ?", userID).Error; err != nil {
		return false, err
	}
	return sharedhandler.RolesContainOrganizationManagementRole(roleNames, user.OrganizationID), nil
}

func (s *AuthnService) IssueClientCredentialToken(ctx context.Context, clientID, clientSecret, scope string) ([]model.Token, error) {
	var app model.Application
	if err := s.db.WithContext(ctx).Where("id = ?", clientID).First(&app).Error; err != nil {
		return nil, errors.New("invalid client credentials")
	}
	if app.ClientSecretHash == "" || !util.CheckSecret(app.ClientSecretHash, clientSecret) {
		return nil, errors.New("invalid client credentials")
	}
	return s.IssueClientCredentialTokenForApplication(ctx, app, scope)
}

func (s *AuthnService) IssueClientCredentialTokenForApplication(ctx context.Context, app model.Application, scope string) ([]model.Token, error) {
	settings, err := coreservice.ResolveApplicationSettingsByID(ctx, s.db, s.cfg, app.ID)
	if err != nil {
		return nil, err
	}
	session := model.Session{
		ApplicationID: app.ID,
		State:         "authenticated",
		RiskLevel:     "low",
	}
	if err := s.db.WithContext(ctx).Create(&session).Error; err != nil {
		return nil, err
	}
	tokenValue, err := util.RandomToken(32)
	if err != nil {
		return nil, err
	}
	accessToken := &model.Token{
		SessionID:     session.ID,
		ApplicationID: app.ID,
		Type:          "access_token",
		Token:         tokenValue,
		Scope:         scope,
		UKID:          "client-credential",
		ExpiresAt:     time.Now().Add(time.Duration(settings.AccessTokenTTLMinutes) * time.Minute),
	}
	if err := s.db.WithContext(ctx).Create(accessToken).Error; err != nil {
		return nil, err
	}
	_ = s.audit.Record(ctx, coreservice.AuditEvent{
		ApplicationID: app.ID,
		ActorType:     "client",
		ActorID:       app.ID,
		EventType:     "token.issued",
		Result:        "success",
		TargetType:    "token",
		TargetID:      accessToken.ID,
		Detail:        map[string]any{"grant": "client_credentials"},
	})
	return []model.Token{*accessToken}, nil
}

func (s *AuthnService) IssuePasswordGrantTokenForApplication(ctx context.Context, app model.Application, identifier, password, scope, ipAddress, userAgent string) ([]model.Token, *model.User, *model.Session, error) {
	if strings.TrimSpace(identifier) == "" || strings.TrimSpace(password) == "" {
		return nil, nil, nil, errors.New("username and password are required")
	}
	if !applicationIsActive(app) {
		return nil, nil, nil, errors.New("application is disabled")
	}
	var project model.Project
	if err := s.db.WithContext(ctx).First(&project, "id = ?", app.ProjectID).Error; err != nil {
		return nil, nil, nil, errors.New("application project not found")
	}
	var organization model.Organization
	if err := s.db.WithContext(ctx).First(&organization, "id = ?", project.OrganizationID).Error; err != nil {
		return nil, nil, nil, errors.New("organization not found")
	}
	if !organizationIsActive(organization) {
		return nil, nil, nil, errors.New("organization is disabled")
	}

	user, err := s.findUserByIdentifier(ctx, project.OrganizationID, identifier)
	if err != nil {
		return nil, nil, nil, errors.New("invalid credentials")
	}
	if strings.TrimSpace(user.PasswordHash) == "" || !util.CheckSecret(user.PasswordHash, password) {
		return nil, nil, nil, errors.New("invalid credentials")
	}
	if user.Status != "active" {
		return nil, nil, nil, errors.New("user is not active")
	}
	allowed, err := s.isUserAllowedForProject(ctx, project.ID, user.ID)
	if err != nil {
		return nil, nil, nil, err
	}
	if !allowed {
		return nil, nil, nil, errors.New("user is not assigned to the target project")
	}

	session := model.Session{
		OrganizationID: project.OrganizationID,
		UserID:         user.ID,
		ApplicationID:  app.ID,
		State:          "authenticated",
		IPAddress:      ipAddress,
		UserAgent:      userAgent,
		RiskLevel:      "low",
	}
	if err := s.db.WithContext(ctx).Create(&session).Error; err != nil {
		return nil, nil, nil, err
	}
	tokens, err := s.issueTokens(ctx, user, session, scope)
	if err != nil {
		return nil, nil, nil, err
	}
	_ = s.audit.Record(ctx, coreservice.AuditEvent{
		OrganizationID: project.OrganizationID,
		ApplicationID:  app.ID,
		ActorType:      "user",
		ActorID:        user.ID,
		EventType:      "token.issued",
		Result:         "success",
		TargetType:     "session",
		TargetID:       session.ID,
		IPAddress:      ipAddress,
		UserAgent:      userAgent,
		Detail:         map[string]any{"grant": "password"},
	})
	return tokens, &user, &session, nil
}

func (s *AuthnService) findUserByIdentifier(ctx context.Context, organizationID, identifier string) (model.User, error) {
	var user model.User
	err := s.db.WithContext(ctx).
		Where("organization_id = ? AND (username = ? OR email = ? OR phone_number = ?)", organizationID, identifier, identifier, identifier).
		First(&user).Error
	return user, err
}

func (s *AuthnService) isUserAllowedForApplication(ctx context.Context, applicationID, userID string) (bool, error) {
	if strings.TrimSpace(applicationID) == "" || strings.TrimSpace(userID) == "" {
		return true, nil
	}
	var app model.Application
	if err := s.db.WithContext(ctx).Select("id", "project_id").First(&app, "id = ?", applicationID).Error; err != nil {
		return false, err
	}
	return s.isUserAllowedForProject(ctx, app.ProjectID, userID)
}

func (s *AuthnService) isUserAllowedForProject(ctx context.Context, projectID, userID string) (bool, error) {
	if strings.TrimSpace(projectID) == "" || strings.TrimSpace(userID) == "" {
		return true, nil
	}
	var project model.Project
	if err := s.db.WithContext(ctx).Select("id", "status", "user_acl_enabled").First(&project, "id = ?", projectID).Error; err != nil {
		return false, err
	}
	if strings.TrimSpace(project.Status) == "disabled" {
		return false, nil
	}
	if !project.UserACLEnabled {
		return true, nil
	}
	var assignment model.ProjectUserAssignment
	err := s.db.WithContext(ctx).Where("project_id = ? AND user_id = ?", projectID, userID).First(&assignment).Error
	if err == nil {
		return true, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return false, err
}

func (s *AuthnService) RevokeToken(ctx context.Context, tokenValue, reason string) error {
	var token model.Token
	if err := s.db.WithContext(ctx).Where("token = ?", tokenValue).First(&token).Error; err != nil {
		return err
	}
	now := time.Now()
	token.RevokedAt = &now
	token.RevocationNote = reason
	if err := s.db.WithContext(ctx).Save(&token).Error; err != nil {
		return err
	}
	return s.audit.Record(ctx, coreservice.AuditEvent{
		ApplicationID: token.ApplicationID,
		ActorType:     "admin",
		EventType:     "token.revoked",
		Result:        "success",
		TargetType:    "token",
		TargetID:      token.ID,
		Detail:        map[string]any{"reason": reason},
	})
}

func (s *AuthnService) ResetUserUKID(ctx context.Context, userID string) (string, error) {
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		return "", err
	}
	newUKID, err := util.RandomToken(18)
	if err != nil {
		return "", err
	}
	if err := s.db.WithContext(ctx).Model(&user).Update("current_ukid", newUKID).Error; err != nil {
		return "", err
	}
	now := time.Now()
	if err := s.db.WithContext(ctx).
		Model(&model.Token{}).
		Where("user_id = ? AND revoked_at IS NULL", user.ID).
		Updates(map[string]any{"revoked_at": now, "revocation_note": "ukid_reset"}).Error; err != nil {
		return "", err
	}
	if err := s.db.WithContext(ctx).Where("user_id = ?", user.ID).Delete(&model.Session{}).Error; err != nil {
		return "", err
	}
	return newUKID, s.audit.Record(ctx, coreservice.AuditEvent{
		OrganizationID: user.OrganizationID,
		ActorType:      "user",
		ActorID:        user.ID,
		EventType:      "user.ukid.reset",
		Result:         "success",
		TargetType:     "user",
		TargetID:       user.ID,
		Detail:         map[string]any{"newUkid": newUKID},
	})
}

func (s *AuthnService) issueTokens(ctx context.Context, user model.User, session model.Session, scope string) ([]model.Token, error) {
	return s.issueTokensForApplication(ctx, user, session, session.ApplicationID, scope)
}

func (s *AuthnService) issueTokensForApplication(ctx context.Context, user model.User, session model.Session, applicationID, scope string) ([]model.Token, error) {
	settings, err := coreservice.ResolveApplicationSettingsByID(ctx, s.db, s.cfg, applicationID)
	if err != nil {
		return nil, err
	}
	var app model.Application
	if err := s.db.WithContext(ctx).First(&app, "id = ?", applicationID).Error; err != nil {
		return nil, err
	}
	if !applicationIsActive(app) {
		return nil, errors.New("application is disabled")
	}
	var organization model.Organization
	if err := s.db.WithContext(ctx).First(&organization, "id = ?", user.OrganizationID).Error; err != nil {
		return nil, err
	}
	if !organizationIsActive(organization) {
		return nil, errors.New("organization is disabled")
	}
	tokens := make([]model.Token, 0, 2)
	if applicationIssuesAccessToken(app.TokenType) {
		accessValue, err := util.RandomToken(32)
		if err != nil {
			return nil, err
		}
		access := &model.Token{
			SessionID:     session.ID,
			UserID:        user.ID,
			ApplicationID: applicationID,
			Type:          "access_token",
			Token:         accessValue,
			Scope:         scope,
			UKID:          user.CurrentUKID,
			ExpiresAt:     time.Now().Add(time.Duration(settings.AccessTokenTTLMinutes) * time.Minute),
		}
		if err := s.db.WithContext(ctx).Create(access).Error; err != nil {
			return nil, err
		}
		tokens = append(tokens, *access)
	}
	if scope != "" && app.EnableRefreshToken && applicationIssuesAccessToken(app.TokenType) {
		refreshValue, err := util.RandomToken(32)
		if err != nil {
			return nil, err
		}
		refresh := &model.Token{
			SessionID:     session.ID,
			UserID:        user.ID,
			ApplicationID: applicationID,
			Type:          "refresh_token",
			Token:         refreshValue,
			Scope:         scope,
			UKID:          user.CurrentUKID,
			ExpiresAt:     time.Now().Add(time.Duration(settings.RefreshTokenTTLHours) * time.Hour),
		}
		if err := s.db.WithContext(ctx).Create(refresh).Error; err != nil {
			return nil, err
		}
		tokens = append(tokens, *refresh)
	}
	_ = s.audit.Record(ctx, coreservice.AuditEvent{
		OrganizationID: user.OrganizationID,
		ApplicationID:  applicationID,
		ActorType:      "user",
		ActorID:        user.ID,
		EventType:      "token.issued",
		Result:         "success",
		TargetType:     "session",
		TargetID:       session.ID,
		Detail:         map[string]any{"scope": scope},
	})
	return tokens, nil
}

func applicationIssuesAccessToken(tokenType []string) bool {
	return coreservice.TokenTypesContain(tokenType, "access_token")
}

func (s *AuthnService) IssueTokens(ctx context.Context, user model.User, session model.Session, scope string) ([]model.Token, error) {
	return s.issueTokens(ctx, user, session, scope)
}

func (s *AuthnService) IssueTokensForApplication(ctx context.Context, user model.User, session model.Session, applicationID, scope string) ([]model.Token, error) {
	return s.issueTokensForApplication(ctx, user, session, applicationID, scope)
}

func (s *AuthnService) UpsertDevice(ctx context.Context, user model.User, fingerprint, userAgent, ipAddress string, trusted bool) (*model.Device, error) {
	return s.upsertDevice(ctx, user, fingerprint, userAgent, ipAddress, trusted)
}

func (s *AuthnService) FingerprintForDevice(device *model.Device) (string, error) {
	return s.fingerprintForDevice(device)
}
