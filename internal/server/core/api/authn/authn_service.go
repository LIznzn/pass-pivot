package authn

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"regexp"
	"strings"
	"sync"
	"time"

	"pass-pivot/internal/config"
	"pass-pivot/internal/model"
	"pass-pivot/internal/notify"
	authservice "pass-pivot/internal/server/auth/service"
	coreservice "pass-pivot/internal/server/core/service"
	sharedauthn "pass-pivot/internal/server/shared/authn"
	sharedfido "pass-pivot/internal/server/shared/fido"
	sharedhandler "pass-pivot/internal/server/shared/handler"
	captchaprovider "pass-pivot/provider/captcha"
	"pass-pivot/utils"

	"gorm.io/gorm"
)

const (
	passwordResetChallengeTTL = 10 * time.Minute
	maxPasswordResetAttempts  = 5
)

var (
	passwordNumberPattern = regexp.MustCompile(`[0-9]`)
	passwordUpperPattern  = regexp.MustCompile(`[A-Z]`)
	passwordLowerPattern  = regexp.MustCompile(`[a-z]`)
	passwordSymbolPattern = regexp.MustCompile(`[^A-Za-z0-9]`)
)

type passwordResetChallengeRecord struct {
	OrganizationID string
	UserID         string
	Identifier     string
	Method         string
	Target         string
	CodeHash       string
	ExpiresAt      time.Time
	ConsumedAt     *time.Time
	AttemptCount   int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type passwordResetScopeInput struct {
	OrganizationID string
	ClientID       string
	Domain         string
}

type PasswordResetMethodOption struct {
	Method       string `json:"method"`
	MaskedTarget string `json:"maskedTarget"`
}

type PasswordResetOptions struct {
	Methods []PasswordResetMethodOption `json:"methods"`
}

var passwordResetStore = struct {
	mu         sync.RWMutex
	challenges map[string]passwordResetChallengeRecord
}{
	challenges: map[string]passwordResetChallengeRecord{},
}

func passwordResetKey(organizationID, identifier string) string {
	return strings.TrimSpace(organizationID) + ":" + strings.TrimSpace(identifier)
}

func storePasswordResetChallenge(record passwordResetChallengeRecord) {
	passwordResetStore.mu.Lock()
	defer passwordResetStore.mu.Unlock()
	passwordResetStore.challenges[passwordResetKey(record.OrganizationID, record.Identifier)] = record
}

func loadPasswordResetChallenge(organizationID, identifier string) (passwordResetChallengeRecord, bool) {
	key := passwordResetKey(organizationID, identifier)
	passwordResetStore.mu.RLock()
	record, ok := passwordResetStore.challenges[key]
	passwordResetStore.mu.RUnlock()
	if !ok {
		return passwordResetChallengeRecord{}, false
	}
	now := time.Now()
	if record.ExpiresAt.Before(now) || record.ConsumedAt != nil {
		deletePasswordResetChallenge(organizationID, identifier)
		return passwordResetChallengeRecord{}, false
	}
	return record, true
}

func updatePasswordResetChallenge(record passwordResetChallengeRecord) {
	passwordResetStore.mu.Lock()
	defer passwordResetStore.mu.Unlock()
	passwordResetStore.challenges[passwordResetKey(record.OrganizationID, record.Identifier)] = record
}

func deletePasswordResetChallenge(organizationID, identifier string) {
	passwordResetStore.mu.Lock()
	defer passwordResetStore.mu.Unlock()
	delete(passwordResetStore.challenges, passwordResetKey(organizationID, identifier))
}

func deletePasswordResetChallengesByUser(userID string) {
	passwordResetStore.mu.Lock()
	defer passwordResetStore.mu.Unlock()
	for key, record := range passwordResetStore.challenges {
		if record.UserID == userID {
			delete(passwordResetStore.challenges, key)
		}
	}
}

func cleanupPasswordResetChallenges() {
	now := time.Now()
	passwordResetStore.mu.Lock()
	defer passwordResetStore.mu.Unlock()
	for key, record := range passwordResetStore.challenges {
		if record.ExpiresAt.Before(now) || record.ConsumedAt != nil {
			delete(passwordResetStore.challenges, key)
		}
	}
}

func init() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			cleanupPasswordResetChallenges()
		}
	}()
}

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
	var organization model.Organization
	if err := s.db.WithContext(ctx).First(&organization, "id = ?", in.OrganizationID).Error; err != nil {
		return nil, err
	}
	if !organizationIsActive(organization) {
		return nil, errors.New("organization is disabled")
	}
	settings := coreservice.NormalizeOrganizationConsoleSettings(&model.OrganizationSetting{
		SupportEmail:   organization.SupportEmail,
		LogoURL:        organization.LogoURL,
		Domains:        organization.Domains,
		LoginPolicy:    organization.LoginPolicy,
		PasswordPolicy: organization.PasswordPolicy,
		MFAPolicy:      organization.MFAPolicy,
		Captcha:        organization.Captcha,
	})
	if err := s.verifyLoginCaptcha(settings.Captcha, organization.ID, in.CaptchaProvider, in.CaptchaToken); err != nil {
		return nil, err
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
	if strings.TrimSpace(user.PasswordHash) == "" || !utils.CheckSecret(user.PasswordHash, in.Secret) {
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
	session.LoginChallenge, _ = utils.RandomToken(18)
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

func validatePasswordAgainstPolicy(password string, policy model.OrganizationPasswordPolicy) error {
	if strings.TrimSpace(password) == "" {
		return errors.New("password is required")
	}
	minLength := policy.MinLength
	if minLength <= 0 {
		minLength = 8
	}
	if len(password) < minLength {
		return errors.New("password does not meet minimum length requirement")
	}
	if policy.RequireUppercase && !passwordUpperPattern.MatchString(password) {
		return errors.New("password must include an uppercase letter")
	}
	if policy.RequireLowercase && !passwordLowerPattern.MatchString(password) {
		return errors.New("password must include a lowercase letter")
	}
	if policy.RequireNumber && !passwordNumberPattern.MatchString(password) {
		return errors.New("password must include a number")
	}
	if policy.RequireSymbol && !passwordSymbolPattern.MatchString(password) {
		return errors.New("password must include a symbol")
	}
	return nil
}

func (s *AuthnService) passwordResetMailer(ctx context.Context, organizationID string) (notify.Mailer, error) {
	_, settings, err := coreservice.LoadOrganizationConsoleSettings(ctx, s.db, organizationID)
	if err != nil {
		return nil, err
	}
	if !coreservice.OrganizationMailSettingsReady(settings.Mail) {
		return nil, errors.New("email password reset is not configured for this organization")
	}
	return notify.NewMailer(notify.MailConfig{
		Provider:       settings.Mail.Provider,
		From:           strings.TrimSpace(settings.Mail.From),
		SMTPHost:       strings.TrimSpace(settings.Mail.SMTPHost),
		SMTPPort:       settings.Mail.SMTPPort,
		SMTPUser:       strings.TrimSpace(settings.Mail.SMTPUser),
		SMTPPass:       settings.Mail.SMTPPass,
		MailgunDomain:  strings.TrimSpace(settings.Mail.MailgunDomain),
		MailgunAPIKey:  strings.TrimSpace(settings.Mail.MailgunAPIKey),
		MailgunAPIBase: strings.TrimSpace(settings.Mail.MailgunAPIBase),
		SendGridAPIKey: strings.TrimSpace(settings.Mail.SendGridAPIKey),
	}), nil
}

func (s *AuthnService) passwordResetMethodOptions(settings model.OrganizationSetting, user model.User) []PasswordResetMethodOption {
	options := make([]PasswordResetMethodOption, 0, 2)
	if coreservice.OrganizationMailSettingsReady(settings.Mail) && strings.TrimSpace(user.Email) != "" {
		options = append(options, PasswordResetMethodOption{
			Method:       "email_code",
			MaskedTarget: maskEmailForDisplay(user.Email),
		})
	}
	if settings.MFAPolicy.AllowSmsCode && strings.TrimSpace(user.PhoneNumber) != "" {
		options = append(options, PasswordResetMethodOption{
			Method:       "sms_code",
			MaskedTarget: maskPhoneForDisplay(user.PhoneNumber),
		})
	}
	return options
}

func (s *AuthnService) QueryPasswordResetOptions(ctx context.Context, organizationID, clientID, identifier string) (*PasswordResetOptions, error) {
	trimmedIdentifier := strings.TrimSpace(identifier)
	if trimmedIdentifier == "" {
		return nil, errors.New("identifier is required")
	}
	organization, err := s.resolvePasswordResetOrganization(ctx, passwordResetScopeInput{
		OrganizationID: organizationID,
		ClientID:       clientID,
	})
	if err != nil {
		return nil, err
	}
	settings := coreservice.NormalizeOrganizationConsoleSettings(&model.OrganizationSetting{
		SupportEmail:   organization.SupportEmail,
		LogoURL:        organization.LogoURL,
		Domains:        organization.Domains,
		LoginPolicy:    organization.LoginPolicy,
		PasswordPolicy: organization.PasswordPolicy,
		MFAPolicy:      organization.MFAPolicy,
		Captcha:        organization.Captcha,
	})
	user, err := s.findUserByIdentifier(ctx, organization.ID, trimmedIdentifier)
	if err != nil || user.Status != "active" {
		return &PasswordResetOptions{Methods: []PasswordResetMethodOption{}}, nil
	}
	return &PasswordResetOptions{
		Methods: s.passwordResetMethodOptions(settings, user),
	}, nil
}

func (s *AuthnService) resolvePasswordResetOrganization(ctx context.Context, in passwordResetScopeInput) (model.Organization, error) {
	if organizationID := strings.TrimSpace(in.OrganizationID); organizationID != "" {
		var organization model.Organization
		if err := s.db.WithContext(ctx).First(&organization, "id = ?", organizationID).Error; err != nil {
			return model.Organization{}, err
		}
		if !organizationIsActive(organization) {
			return model.Organization{}, errors.New("organization is disabled")
		}
		return organization, nil
	}
	if clientID := strings.TrimSpace(in.ClientID); clientID != "" {
		var application model.Application
		if err := s.db.WithContext(ctx).First(&application, "id = ?", clientID).Error; err != nil {
			return model.Organization{}, err
		}
		if !applicationIsActive(application) {
			return model.Organization{}, errors.New("application is disabled")
		}
		var project model.Project
		if err := s.db.WithContext(ctx).First(&project, "id = ?", application.ProjectID).Error; err != nil {
			return model.Organization{}, err
		}
		if strings.TrimSpace(project.Status) == "disabled" {
			return model.Organization{}, errors.New("project is disabled")
		}
		var organization model.Organization
		if err := s.db.WithContext(ctx).First(&organization, "id = ?", project.OrganizationID).Error; err != nil {
			return model.Organization{}, err
		}
		if !organizationIsActive(organization) {
			return model.Organization{}, errors.New("organization is disabled")
		}
		return organization, nil
	}
	if strings.TrimSpace(in.Domain) != "" {
		// Reserve host/domain-based resolution here so the API contract can stay stable.
	}
	return model.Organization{}, errors.New("password reset scope is not available")
}

func (s *AuthnService) StartPasswordReset(ctx context.Context, organizationID, clientID, identifier, method, contact, captchaProvider, captchaToken string) error {
	trimmedIdentifier := strings.TrimSpace(identifier)
	if trimmedIdentifier == "" {
		return errors.New("identifier is required")
	}
	trimmedMethod := strings.TrimSpace(method)
	if trimmedMethod == "" {
		return errors.New("password reset method is required")
	}
	trimmedContact := strings.TrimSpace(contact)
	if trimmedContact == "" {
		return errors.New("password reset contact is required")
	}
	organization, err := s.resolvePasswordResetOrganization(ctx, passwordResetScopeInput{
		OrganizationID: organizationID,
		ClientID:       clientID,
	})
	if err != nil {
		return err
	}
	settings := coreservice.NormalizeOrganizationConsoleSettings(&model.OrganizationSetting{
		SupportEmail:   organization.SupportEmail,
		LogoURL:        organization.LogoURL,
		Domains:        organization.Domains,
		LoginPolicy:    organization.LoginPolicy,
		PasswordPolicy: organization.PasswordPolicy,
		MFAPolicy:      organization.MFAPolicy,
		Captcha:        organization.Captcha,
	})
	if err := s.verifyLoginCaptcha(settings.Captcha, organization.ID, captchaProvider, captchaToken); err != nil {
		return err
	}
	user, err := s.findUserByIdentifier(ctx, organization.ID, trimmedIdentifier)
	if err != nil || user.Status != "active" {
		return nil
	}
	options := s.passwordResetMethodOptions(settings, user)
	selectedTarget := ""
	selectedMaskedTarget := ""
	for _, option := range options {
		if option.Method != trimmedMethod {
			continue
		}
		selectedMaskedTarget = option.MaskedTarget
		switch trimmedMethod {
		case "email_code":
			selectedTarget = strings.TrimSpace(user.Email)
		case "sms_code":
			selectedTarget = strings.TrimSpace(user.PhoneNumber)
		}
		break
	}
	if selectedTarget == "" {
		return nil
	}
	if !passwordResetContactMatches(trimmedMethod, selectedTarget, trimmedContact) {
		return nil
	}
	code := fmt.Sprintf("%06d", rand.IntN(1000000))
	hash, err := utils.HashSecret(code)
	if err != nil {
		return err
	}
	now := time.Now()
	storePasswordResetChallenge(passwordResetChallengeRecord{
		OrganizationID: organization.ID,
		UserID:         user.ID,
		Identifier:     trimmedIdentifier,
		Method:         trimmedMethod,
		Target:         selectedTarget,
		CodeHash:       hash,
		ExpiresAt:      now.Add(passwordResetChallengeTTL),
		CreatedAt:      now,
		UpdatedAt:      now,
	})
	switch trimmedMethod {
	case "email_code":
		mailer, err := s.passwordResetMailer(ctx, organization.ID)
		if err != nil {
			return err
		}
		if err := mailer.Send(ctx, selectedTarget, "PPVT Password Reset Code", fmt.Sprintf("Your PPVT password reset code is %s. It expires in 10 minutes.", code)); err != nil {
			return err
		}
	case "sms_code":
		// SMS delivery is modeled as an optional channel. Keep the reset challenge active
		// so the transport can be integrated later without changing the API contract.
	}
	_ = s.audit.Record(ctx, coreservice.AuditEvent{
		OrganizationID: organization.ID,
		ActorType:      "anonymous",
		EventType:      "auth.password_reset.started",
		Result:         "success",
		TargetType:     "user",
		TargetID:       user.ID,
		Detail: map[string]any{
			"identifier": trimmedIdentifier,
			"method":     trimmedMethod,
			"target":     selectedMaskedTarget,
		},
	})
	return nil
}

func (s *AuthnService) FinishPasswordReset(ctx context.Context, organizationID, clientID, identifier, code, newPassword string) error {
	trimmedIdentifier := strings.TrimSpace(identifier)
	if trimmedIdentifier == "" {
		return errors.New("identifier is required")
	}
	if strings.TrimSpace(code) == "" {
		return errors.New("password reset code is required")
	}
	organization, err := s.resolvePasswordResetOrganization(ctx, passwordResetScopeInput{
		OrganizationID: organizationID,
		ClientID:       clientID,
	})
	if err != nil {
		return err
	}
	challenge, ok := loadPasswordResetChallenge(organization.ID, trimmedIdentifier)
	if !ok {
		return errors.New("password reset challenge not found")
	}
	if time.Now().After(challenge.ExpiresAt) {
		deletePasswordResetChallenge(organization.ID, trimmedIdentifier)
		return errors.New("password reset challenge expired")
	}
	if challenge.AttemptCount >= maxPasswordResetAttempts {
		now := time.Now()
		challenge.ConsumedAt = &now
		challenge.UpdatedAt = now
		updatePasswordResetChallenge(challenge)
		return errors.New("password reset challenge max attempts exceeded")
	}
	if !utils.CheckSecret(challenge.CodeHash, strings.TrimSpace(code)) {
		challenge.AttemptCount++
		challenge.UpdatedAt = time.Now()
		updatePasswordResetChallenge(challenge)
		return errors.New("invalid password reset code")
	}
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", challenge.UserID).Error; err != nil {
		return err
	}
	var passwordPolicyOrganization model.Organization
	if err := s.db.WithContext(ctx).Select("id", "password_policy").First(&passwordPolicyOrganization, "id = ?", user.OrganizationID).Error; err != nil {
		return err
	}
	if err := validatePasswordAgainstPolicy(newPassword, passwordPolicyOrganization.PasswordPolicy); err != nil {
		return err
	}
	hash, err := utils.HashSecret(newPassword)
	if err != nil {
		return err
	}
	now := time.Now()
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&user).Update("password_hash", hash).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.Token{}).
			Where("user_id = ? AND revoked_at IS NULL", user.ID).
			Updates(map[string]any{"revoked_at": now, "revocation_note": "password_recovered"}).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	challenge.ConsumedAt = &now
	challenge.UpdatedAt = now
	updatePasswordResetChallenge(challenge)
	deletePasswordResetChallengesByUser(user.ID)
	_ = s.audit.Record(ctx, coreservice.AuditEvent{
		OrganizationID: user.OrganizationID,
		ActorType:      "anonymous",
		EventType:      "auth.password_reset.completed",
		Result:         "success",
		TargetType:     "user",
		TargetID:       user.ID,
		Detail: map[string]any{
			"identifier": trimmedIdentifier,
		},
	})
	return nil
}

func maskEmail(value string) string {
	parts := strings.Split(strings.TrimSpace(value), "@")
	if len(parts) != 2 {
		return value
	}
	if len(parts[0]) <= 2 {
		return parts[0] + "***@" + parts[1]
	}
	return parts[0][:2] + "***@" + parts[1]
}

func maskEmailForDisplay(value string) string {
	parts := strings.Split(strings.TrimSpace(value), "@")
	if len(parts) != 2 {
		return value
	}
	local := parts[0]
	domainParts := strings.Split(parts[1], ".")
	domainSuffix := ""
	if len(domainParts) > 1 {
		domainSuffix = "." + domainParts[len(domainParts)-1]
	}
	if local == "" {
		local = "*"
	}
	return local[:1] + "*****@*****" + domainSuffix
}

func normalizePhoneDigits(value string) string {
	var builder strings.Builder
	for _, r := range strings.TrimSpace(value) {
		if r >= '0' && r <= '9' {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func passwordResetContactMatches(method, actual, provided string) bool {
	actual = strings.TrimSpace(actual)
	provided = strings.TrimSpace(provided)
	switch strings.TrimSpace(method) {
	case "sms_code":
		actualDigits := normalizePhoneDigits(actual)
		providedDigits := normalizePhoneDigits(provided)
		if len(actualDigits) < 4 || len(providedDigits) != 4 {
			return false
		}
		return strings.HasSuffix(actualDigits, providedDigits)
	default:
		return strings.EqualFold(actual, provided)
	}
}

func maskPhoneForDisplay(value string) string {
	digits := normalizePhoneDigits(value)
	if len(digits) < 5 {
		return "****"
	}
	countryCode := ""
	localDigits := digits
	switch {
	case strings.HasPrefix(digits, "86") && len(digits) >= 13:
		countryCode = "+86 "
		localDigits = digits[2:]
	case strings.HasPrefix(digits, "81") && len(digits) >= 12:
		countryCode = "+81 "
		localDigits = digits[2:]
	default:
		if strings.HasPrefix(strings.TrimSpace(value), "+") && len(digits) > 10 {
			countryCode = "+" + digits[:len(digits)-10] + " "
			localDigits = digits[len(digits)-10:]
		}
	}
	if len(localDigits) < 4 {
		return countryCode + "****"
	}
	prefix := localDigits
	if len(prefix) > 3 {
		prefix = prefix[:3]
	}
	last := localDigits[len(localDigits)-1:]
	return countryCode + prefix + " **** ***" + last
}

func (s *AuthnService) verifyLoginCaptcha(settings model.OrganizationCaptchaSettings, organizationID, providerName, token string) error {
	settings = coreservice.NormalizeOrganizationConsoleSettings(&model.OrganizationSetting{Captcha: settings}).Captcha
	switch settings.Provider {
	case "disabled":
		return nil
	case "default":
		if strings.TrimSpace(providerName) != "default" {
			return errors.New("captcha is required")
		}
		secret := authservice.DefaultCaptchaSecret(s.cfg.Secret, organizationID)
		ok, err := captchaprovider.VerifyCaptchaByCaptchaType("Default", token, "", secret)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("invalid captcha")
		}
		return nil
	case "google":
		if strings.TrimSpace(providerName) != "google" {
			return errors.New("captcha is required")
		}
		ok, err := captchaprovider.VerifyCaptchaByCaptchaType("Google reCAPTCHA", token, settings.ClientKey, settings.ClientSecret)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("invalid captcha")
		}
		return nil
	case "cloudflare":
		if strings.TrimSpace(providerName) != "cloudflare" {
			return errors.New("captcha is required")
		}
		ok, err := captchaprovider.VerifyCaptchaByCaptchaType("Cloudflare Turnstile", token, settings.ClientKey, settings.ClientSecret)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("invalid captcha")
		}
		return nil
	default:
		return errors.New("invalid captcha provider")
	}
}

func (s *AuthnService) evaluateMFALoginRequirement(ctx context.Context, user model.User) (bool, string, error) {
	_, settings, err := coreservice.LoadOrganizationConsoleSettings(ctx, s.db, user.OrganizationID)
	if err != nil {
		return false, "", err
	}
	mfaRequired := settings.MFAPolicy.RequireForAllUsers
	if !mfaRequired {
		var enabledCount int64
		if err := s.db.WithContext(ctx).Model(&model.MFAEnrollment{}).
			Where("user_id = ? AND method = ? AND status = ? AND deleted_at IS NULL", user.ID, "mfa", "active").
			Count(&enabledCount).Error; err != nil {
			return false, "", err
		}
		mfaRequired = enabledCount > 0
	}
	if !mfaRequired {
		return false, "", nil
	}
	primaryMethods := make([]string, 0, 5)
	hasRecoveryCode := false

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
			primaryMethods = append(primaryMethods, "u2f")
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
			primaryMethods = append(primaryMethods, "webauthn")
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
			primaryMethods = append(primaryMethods, "totp")
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
			hasRecoveryCode = true
		}
	}

	if settings.MFAPolicy.AllowEmailCode &&
		strings.TrimSpace(user.Email) != "" &&
		coreservice.OrganizationMailSettingsReady(settings.Mail) {
		primaryMethods = append(primaryMethods, "email_code")
	}

	if settings.MFAPolicy.AllowSmsCode && strings.TrimSpace(user.PhoneNumber) != "" {
		primaryMethods = append(primaryMethods, "sms_code")
	}

	if len(primaryMethods) == 0 {
		return false, "", nil
	}
	if hasRecoveryCode {
		primaryMethods = append(primaryMethods, "recovery_code")
	}
	return true, primaryMethods[0], nil
}

func (s *AuthnService) ConfirmSession(ctx context.Context, sessionID string, accept bool, trustDevice bool) (*sharedauthn.LoginResult, error) {
	session, err := s.findSessionByReference(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if !accept {
		session.State = "rejected"
		if err := s.db.WithContext(ctx).Save(session).Error; err != nil {
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
	if err := s.db.WithContext(ctx).Save(session).Error; err != nil {
		return nil, err
	}
	if session.RequiresMFA {
		return &sharedauthn.LoginResult{Session: *session, NextStep: "mfa"}, nil
	}
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", session.UserID).Error; err != nil {
		return nil, err
	}
	s.recordLoginSucceeded(ctx, *session, user.ID, session.IPAddress, session.UserAgent, map[string]any{
		"trustedDevice": trustDevice,
	})
	tokens, err := s.issueTokens(ctx, user, *session, "openid profile email phone")
	if err != nil {
		return nil, err
	}
	return &sharedauthn.LoginResult{Session: *session, NextStep: "done", Tokens: tokens}, nil
}

func (s *AuthnService) VerifyMFA(ctx context.Context, sessionID, method, code string, trustDevice bool) (*sharedauthn.LoginResult, error) {
	session, err := s.findSessionByReference(ctx, sessionID)
	if err != nil {
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
		if err := s.mfa.Verify(ctx, session.ID, method, code); err != nil {
			return nil, err
		}
	}
	return s.completeMFASession(ctx, user, session, method, trustDevice)
}

func (s *AuthnService) CompleteWebAuthnMFA(ctx context.Context, sessionID, method string, trustDevice bool) (*sharedauthn.LoginResult, error) {
	session, err := s.findSessionByReference(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.State != "mfa_required" && session.State != "confirmation_required" {
		return nil, errors.New("session is not awaiting mfa")
	}
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", session.UserID).Error; err != nil {
		return nil, err
	}
	return s.completeMFASession(ctx, user, session, method, trustDevice)
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
		fingerprint, err = utils.GenerateFingerprint()
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
	deviceKey, ok := utils.VerifyFingerprint(signedFingerprint, s.cfg.Secret)
	if !ok {
		return ""
	}
	return deviceKey
}

func (s *AuthnService) fingerprintForDevice(device *model.Device) (string, error) {
	if device == nil {
		return "", nil
	}
	return utils.SignFingerprint(device.Fingerprint, s.cfg.Secret)
}

func (s *AuthnService) RequestMFAChallenge(ctx context.Context, sessionID, method string) (*model.MFAChallenge, string, error) {
	session, err := s.findSessionByReference(ctx, sessionID)
	if err != nil {
		return nil, "", err
	}
	return s.mfa.CreateDeliveryChallenge(ctx, session.ID, method)
}

func (s *AuthnService) userIDFromSession(ctx context.Context, sessionID string) (string, error) {
	if sessionID == "" {
		return "", errors.New("sessionId is required")
	}
	session, err := s.findSessionByReference(ctx, sessionID)
	if err != nil {
		return "", err
	}
	return session.UserID, nil
}

func (s *AuthnService) findSessionByReference(ctx context.Context, sessionRef string) (*model.Session, error) {
	ref := strings.TrimSpace(sessionRef)
	if ref == "" {
		return nil, errors.New("session is required")
	}
	var session model.Session
	if err := s.db.WithContext(ctx).First(&session, "id = ?", ref).Error; err == nil {
		return &session, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Where("login_challenge = ?", ref).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
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

func (s *AuthnService) QueryRecoveryCodes(ctx context.Context, userID string) ([]string, error) {
	return s.mfa.QueryRecoveryCodes(ctx, userID)
}

func (s *AuthnService) QueryCurrentUserRecoveryCodes(ctx context.Context, sessionID string) ([]string, error) {
	userID, err := s.userIDFromSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return s.QueryRecoveryCodes(ctx, userID)
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
	if app.ClientSecretHash == "" || !utils.CheckSecret(app.ClientSecretHash, clientSecret) {
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
	tokenValue, err := utils.RandomToken(32)
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
	if strings.TrimSpace(user.PasswordHash) == "" || !utils.CheckSecret(user.PasswordHash, password) {
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
	token.RevokedAt = new(time.Now())
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
	newUKID, err := utils.RandomToken(18)
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
		accessValue, err := utils.RandomToken(32)
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
		refreshValue, err := utils.RandomToken(32)
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
