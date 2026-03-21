package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"gorm.io/gorm"
	"pass-pivot/internal/config"

	"pass-pivot/internal/model"
	"pass-pivot/internal/notify"
	coreservice "pass-pivot/internal/server/core/service"
	sharedauthn "pass-pivot/internal/server/shared/authn"
	sharedfido "pass-pivot/internal/server/shared/fido"
	"pass-pivot/util"
)

type MFAService struct {
	db              *gorm.DB
	cfg             config.Config
	audit           *AuditService
	mu              sync.Mutex
	totp            map[string]pendingTOTPEnrollment
	u2fFIDO         u2fFIDOService
	webAuthnRuntime webAuthnMFARuntime
}

type u2fFIDOService interface {
	BeginAssertionForSession(ctx context.Context, sessionID, usage string) (string, any, error)
	FinishAssertion(ctx context.Context, challengeID string, payload json.RawMessage) (*sharedfido.AssertionResult, error)
}

type webAuthnMFARuntime interface {
	CompleteWebAuthnMFA(ctx context.Context, sessionID, method string, trustDevice bool) (*sharedauthn.LoginResult, error)
}

func NewMFAService(db *gorm.DB, cfg config.Config, audit *AuditService) *MFAService {
	return &MFAService{
		db:    db,
		cfg:   cfg,
		audit: audit,
		totp:  map[string]pendingTOTPEnrollment{},
	}
}

func (s *MFAService) SetFIDOService(fido u2fFIDOService) {
	s.u2fFIDO = fido
}

func (s *MFAService) SetWebAuthnMFARuntime(runtime webAuthnMFARuntime) {
	s.webAuthnRuntime = runtime
}

func (s *MFAService) BeginU2FAssertion(ctx context.Context, sessionID string) (string, any, error) {
	if s.u2fFIDO == nil {
		return "", nil, errors.New("fido service is not configured")
	}
	return s.u2fFIDO.BeginAssertionForSession(ctx, sessionID, "u2f")
}

func (s *MFAService) FinishU2FAssertion(ctx context.Context, challengeID string, payload json.RawMessage, trustDevice bool) (*sharedauthn.LoginResult, error) {
	if s.u2fFIDO == nil {
		return nil, errors.New("fido service is not configured")
	}
	if s.webAuthnRuntime == nil {
		return nil, errors.New("webauthn mfa runtime is not configured")
	}
	assertion, err := s.u2fFIDO.FinishAssertion(ctx, challengeID, payload)
	if err != nil {
		return nil, err
	}
	if assertion.Usage != "u2f" {
		return nil, errors.New("fido assertion usage mismatch")
	}
	return s.webAuthnRuntime.CompleteWebAuthnMFA(ctx, assertion.SessionID, "u2f", trustDevice)
}

type TOTPEnrollmentResult struct {
	EnrollmentID    string `json:"enrollmentId"`
	Secret          string `json:"secret"`
	ProvisioningURI string `json:"provisioningUri"`
	ManualEntryKey  string `json:"manualEntryKey"`
}

type pendingTOTPEnrollment struct {
	UserID         string
	OrganizationID string
	EnrollmentID   string
	Secret         string
	ExpiresAt      time.Time
}

func (s *MFAService) EnrollTOTP(ctx context.Context, userID string) (*TOTPEnrollmentResult, error) {
	return s.EnrollTOTPForApplication(ctx, userID, "")
}

func (s *MFAService) EnrollTOTPForApplication(ctx context.Context, userID, applicationID string) (*TOTPEnrollmentResult, error) {
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}
	settings, err := coreservice.ResolveApplicationSettingsByID(ctx, s.db, s.cfg, applicationID)
	if err != nil {
		return nil, err
	}
	issuer, err := url.Parse(settings.TokenIssuer)
	if err != nil {
		return nil, err
	}
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer.Host,
		AccountName: defaultLoginIdentifier(user),
		Algorithm:   otp.AlgorithmSHA1,
		Digits:      otp.DigitsSix,
		Period:      30,
		SecretSize:  20,
	})
	if err != nil {
		return nil, err
	}
	enrollmentID, err := util.RandomToken(18)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	for key, item := range s.totp {
		if item.UserID == user.ID || time.Now().After(item.ExpiresAt) {
			delete(s.totp, key)
		}
	}
	s.totp[enrollmentID] = pendingTOTPEnrollment{
		UserID:         user.ID,
		OrganizationID: user.OrganizationID,
		EnrollmentID:   enrollmentID,
		Secret:         key.Secret(),
		ExpiresAt:      time.Now().Add(10 * time.Minute),
	}
	s.mu.Unlock()
	return &TOTPEnrollmentResult{
		EnrollmentID:    enrollmentID,
		Secret:          key.Secret(),
		ProvisioningURI: key.URL(),
		ManualEntryKey:  key.Secret(),
	}, nil
}

func (s *MFAService) VerifyTOTPEnrollment(ctx context.Context, userID, enrollmentID, code string) error {
	s.mu.Lock()
	pending, ok := s.totp[enrollmentID]
	if ok && (pending.UserID != userID || time.Now().After(pending.ExpiresAt)) {
		delete(s.totp, enrollmentID)
		ok = false
	}
	s.mu.Unlock()
	if !ok {
		return errors.New("TOTP enrollment expired or not found")
	}
	if !totp.Validate(code, pending.Secret) {
		return errors.New("invalid TOTP code")
	}
	now := time.Now()
	enrollment := model.MFAEnrollment{
		OrganizationID: pending.OrganizationID,
		UserID:         pending.UserID,
		Method:         "totp",
		Label:          "Authenticator App",
		Secret:         pending.Secret,
		Status:         "active",
		LastUsedAt:     &now,
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ? AND method = ?", pending.UserID, "totp").Delete(&model.MFAEnrollment{}).Error; err != nil {
			return err
		}
		return tx.Create(&enrollment).Error
	}); err != nil {
		return err
	}
	s.mu.Lock()
	delete(s.totp, enrollmentID)
	s.mu.Unlock()
	return nil
}

func (s *MFAService) GenerateRecoveryCodes(ctx context.Context, userID string) ([]string, error) {
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}
	_ = s.db.WithContext(ctx).Where("user_id = ?", user.ID).Delete(&model.MFARecoveryCode{}).Error
	codes := sharedauthn.RecoveryCodes()
	for _, code := range codes {
		entry := model.MFARecoveryCode{
			UserID:         user.ID,
			OrganizationID: user.OrganizationID,
			Code:           code,
		}
		if err := s.db.WithContext(ctx).Create(&entry).Error; err != nil {
			return nil, err
		}
	}
	return codes, nil
}

func (s *MFAService) QueryRecoveryCodes(ctx context.Context, userID string) ([]string, error) {
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}
	var records []model.MFARecoveryCode
	if err := s.db.WithContext(ctx).
		Where("user_id = ? AND consumed_at IS NULL AND deleted_at IS NULL", user.ID).
		Order("created_at asc").
		Find(&records).Error; err != nil {
		return nil, err
	}
	codes := make([]string, 0, len(records))
	for _, item := range records {
		if strings.TrimSpace(item.Code) == "" {
			continue
		}
		codes = append(codes, item.Code)
	}
	return codes, nil
}

func (s *MFAService) CreateDeliveryChallenge(ctx context.Context, sessionID, method string) (*model.MFAChallenge, string, error) {
	var session model.Session
	if err := s.db.WithContext(ctx).First(&session, "id = ?", sessionID).Error; err != nil {
		return nil, "", err
	}
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", session.UserID).Error; err != nil {
		return nil, "", err
	}
	target := ""
	switch method {
	case "email_code":
		target = user.Email
	case "sms_code":
		target = user.PhoneNumber
	default:
		return nil, "", errors.New("unsupported delivery method")
	}
	if strings.TrimSpace(target) == "" {
		return nil, "", errors.New("no reachable target for selected method")
	}
	code := fmt.Sprintf("%06d", rand.IntN(1000000))
	hash, err := util.HashSecret(code)
	if err != nil {
		return nil, "", err
	}
	challenge := &model.MFAChallenge{
		BaseModel: model.BaseModel{
			ID: uuid.NewString(),
		},
		SessionID:       session.ID,
		UserID:          user.ID,
		OrganizationID:  user.OrganizationID,
		Method:          method,
		CodeHash:        hash,
		Target:          target,
		ExpiresAt:       time.Now().Add(10 * time.Minute),
		DeliveryMessage: fmt.Sprintf("OTP sent to %s", maskTarget(method, target)),
	}
	challenge.CreatedAt = time.Now()
	challenge.UpdatedAt = challenge.CreatedAt
	storeMFAChallenge(*challenge)
	if method == "email_code" {
		mailer, err := s.mailerForOrganization(ctx, user.OrganizationID)
		if err != nil {
			return nil, "", err
		}
		subject := "PPVT MFA Verification Code"
		body := fmt.Sprintf("Your PPVT verification code is %s. It expires in 10 minutes.", code)
		if err := mailer.Send(ctx, target, subject, body); err != nil {
			return nil, "", err
		}
		challenge.DeliveryMessage = fmt.Sprintf("OTP sent to %s by email", maskTarget(method, target))
		challenge.UpdatedAt = time.Now()
		updateMFAChallenge(*challenge)
	}
	_ = s.audit.Record(ctx, AuditEvent{
		OrganizationID: user.OrganizationID,
		ActorType:      "system",
		ActorID:        user.ID,
		EventType:      "auth.mfa.challenge.created",
		Result:         "success",
		TargetType:     "mfa_challenge",
		TargetID:       challenge.ID,
		Detail: map[string]any{
			"method":   method,
			"target":   maskTarget(method, target),
			"demoCode": code,
		},
	})
	return challenge, code, nil
}

func (s *MFAService) mailerForOrganization(ctx context.Context, organizationID string) (notify.Mailer, error) {
	_, settings, err := coreservice.LoadOrganizationConsoleSettings(ctx, s.db, organizationID)
	if err != nil {
		return nil, err
	}
	channel := settings.MFAPolicy.EmailChannel
	if !channel.Enabled {
		return nil, errors.New("email mfa is not configured for this organization")
	}
	return notify.NewMailer(notify.SMTPConfig{
		From:     strings.TrimSpace(channel.From),
		Host:     strings.TrimSpace(channel.Host),
		Port:     channel.Port,
		Username: strings.TrimSpace(channel.Username),
		Password: channel.Password,
	}), nil
}

func (s *MFAService) Verify(ctx context.Context, sessionID, method, code string) error {
	var session model.Session
	if err := s.db.WithContext(ctx).First(&session, "id = ?", sessionID).Error; err != nil {
		return err
	}
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", session.UserID).Error; err != nil {
		return err
	}
	switch method {
	case "totp":
		var enrollments []model.MFAEnrollment
		if err := s.db.WithContext(ctx).Where("user_id = ? AND method = ? AND status = ?", user.ID, "totp", "active").Find(&enrollments).Error; err != nil {
			return err
		}
		for _, enrollment := range enrollments {
			if totp.Validate(code, enrollment.Secret) {
				now := time.Now()
				_ = s.db.WithContext(ctx).Model(&enrollment).Update("last_used_at", &now).Error
				return nil
			}
		}
		return errors.New("invalid TOTP code")
	case "email_code", "sms_code":
		challenge, ok := latestActiveMFAChallenge(session.ID, method)
		if !ok {
			return errors.New("mfa challenge not found")
		}
		if time.Now().After(challenge.ExpiresAt) {
			return errors.New("MFA challenge expired")
		}
		if !util.CheckSecret(challenge.CodeHash, code) {
			challenge.AttemptCount++
			challenge.UpdatedAt = time.Now()
			updateMFAChallenge(challenge)
			return errors.New("invalid challenge code")
		}
		now := time.Now()
		challenge.ConsumedAt = &now
		challenge.UpdatedAt = now
		updateMFAChallenge(challenge)
		return nil
	case "recovery_code":
		var codes []model.MFARecoveryCode
		if err := s.db.WithContext(ctx).Where("user_id = ? AND consumed_at IS NULL", user.ID).Find(&codes).Error; err != nil {
			return err
		}
		for _, item := range codes {
			if strings.TrimSpace(item.Code) == code || (strings.TrimSpace(item.Code) == "" && util.CheckSecret(item.CodeHash, code)) {
				now := time.Now()
				return s.db.WithContext(ctx).Model(&item).Update("consumed_at", &now).Error
			}
		}
		return errors.New("invalid recovery code")
	default:
		return errors.New("unsupported MFA method")
	}
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

func maskTarget(method, target string) string {
	if target == "" {
		return ""
	}
	if method == "email_code" {
		parts := strings.Split(target, "@")
		if len(parts) != 2 || len(parts[0]) < 2 {
			return target
		}
		return parts[0][:2] + "***@" + parts[1]
	}
	if len(target) <= 4 {
		return target
	}
	return "***" + target[len(target)-4:]
}
