package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"

	"pass-pivot/internal/model"
	coreservice "pass-pivot/internal/server/core/service"
	"pass-pivot/utils"

	"gorm.io/gorm"
)

const (
	deviceAuthorizationStatusPending  = "pending"
	deviceAuthorizationStatusApproved = "approved"
	deviceAuthorizationStatusConsumed = "consumed"
	deviceAuthorizationStatusDenied   = "denied"
	deviceAuthorizationTTL            = 5 * time.Minute
	deviceAuthorizationInterval       = 5
)

type DeviceAuthorizationResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int64  `json:"expires_in"`
	Interval                int    `json:"interval,omitempty"`
}

type DeviceAuthorizationView struct {
	Authorization model.DeviceAuthorization
	Application   model.Application
	Project       model.Project
	Organization  model.Organization
}

func (s *OIDCService) CreateDeviceAuthorization(ctx context.Context, audience, clientID, clientSecret, clientAssertionType, clientAssertion, scope string) (*DeviceAuthorizationResponse, error) {
	app, err := s.validateClientAuthentication(ctx, audience, clientID, clientSecret, clientAssertionType, clientAssertion)
	if err != nil {
		return nil, err
	}
	if !coreservice.AppGrantTypesContain(app.GrantType, "device_code") {
		return nil, errors.New("device_code grant is not enabled for this application")
	}
	deviceCode, err := utils.RandomToken(32)
	if err != nil {
		return nil, err
	}
	userCode, err := newUserCode()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	record := model.DeviceAuthorization{
		ApplicationID:   app.ID,
		DeviceCode:      deviceCode,
		UserCode:        userCode,
		Scope:           strings.TrimSpace(scope),
		Status:          deviceAuthorizationStatusPending,
		IntervalSeconds: deviceAuthorizationInterval,
		ExpiresAt:       now.Add(deviceAuthorizationTTL),
	}
	if err := s.db.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, err
	}
	verificationURI := strings.TrimRight(s.cfg.AuthURL, "/") + "/auth/authorize?type=device_code"
	return &DeviceAuthorizationResponse{
		DeviceCode:              record.DeviceCode,
		UserCode:                record.UserCode,
		VerificationURI:         verificationURI,
		VerificationURIComplete: verificationURI + "&user_code=" + record.UserCode,
		ExpiresIn:               int64(deviceAuthorizationTTL / time.Second),
		Interval:                record.IntervalSeconds,
	}, nil
}

func (s *OIDCService) ExchangeDeviceCode(ctx context.Context, audience, clientID, clientSecret, clientAssertionType, clientAssertion, deviceCode string) ([]model.Token, string, error) {
	app, err := s.validateClientAuthentication(ctx, audience, clientID, clientSecret, clientAssertionType, clientAssertion)
	if err != nil {
		return nil, "", err
	}
	if !coreservice.AppGrantTypesContain(app.GrantType, "device_code") {
		return nil, "", errors.New("device_code grant is not enabled for this application")
	}
	var record model.DeviceAuthorization
	if err := s.db.WithContext(ctx).Where("device_code = ? AND application_id = ?", strings.TrimSpace(deviceCode), app.ID).First(&record).Error; err != nil {
		return nil, "", errors.New("invalid device_code")
	}
	now := time.Now()
	if record.ExpiresAt.Before(now) {
		return nil, "", errors.New("expired_token")
	}
	switch record.Status {
	case deviceAuthorizationStatusDenied:
		return nil, "", errors.New("access_denied")
	case deviceAuthorizationStatusConsumed:
		return nil, "", errors.New("invalid device_code")
	case deviceAuthorizationStatusPending:
		if record.LastPolledAt != nil && now.Before(record.LastPolledAt.Add(time.Duration(record.IntervalSeconds)*time.Second)) {
			return nil, "", errors.New("slow_down")
		}
		record.LastPolledAt = &now
		if err := s.db.WithContext(ctx).Model(&record).Update("last_polled_at", now).Error; err != nil {
			return nil, "", err
		}
		return nil, "", errors.New("authorization_pending")
	case deviceAuthorizationStatusApproved:
	default:
		return nil, "", errors.New("invalid device_code")
	}
	if strings.TrimSpace(record.UserID) == "" || strings.TrimSpace(record.SessionID) == "" {
		return nil, "", errors.New("invalid device_code")
	}
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", record.UserID).Error; err != nil {
		return nil, "", err
	}
	var session model.Session
	if err := s.db.WithContext(ctx).First(&session, "id = ?", record.SessionID).Error; err != nil {
		return nil, "", err
	}
	if session.State != "authenticated" {
		return nil, "", errors.New("session is not authenticated")
	}
	tokens, err := s.auth.IssueTokensForApplication(ctx, user, session, app.ID, record.Scope)
	if err != nil {
		return nil, "", err
	}
	idToken := ""
	if applicationReturnsIDToken(app.TokenType) {
		authTime := session.CreatedAt
		idToken, err = s.signIDToken(ctx, app.ID, user, app.ID, record.Scope, "", &authTime, session.ID)
		if err != nil {
			return nil, "", err
		}
	}
	record.Status = deviceAuthorizationStatusConsumed
	if err := s.db.WithContext(ctx).Model(&record).Update("status", record.Status).Error; err != nil {
		return nil, "", err
	}
	return tokens, idToken, nil
}

func (s *OIDCService) DeviceAuthorizationByUserCode(ctx context.Context, userCode string) (*DeviceAuthorizationView, error) {
	if strings.TrimSpace(userCode) == "" {
		return nil, errors.New("user_code is required")
	}
	var record model.DeviceAuthorization
	if err := s.db.WithContext(ctx).Where("user_code = ?", normalizeUserCode(userCode)).First(&record).Error; err != nil {
		return nil, errors.New("device authorization not found")
	}
	var app model.Application
	if err := s.db.WithContext(ctx).First(&app, "id = ?", record.ApplicationID).Error; err != nil {
		return nil, err
	}
	var project model.Project
	if err := s.db.WithContext(ctx).First(&project, "id = ?", app.ProjectID).Error; err != nil {
		return nil, err
	}
	var organization model.Organization
	if err := s.db.WithContext(ctx).First(&organization, "id = ?", project.OrganizationID).Error; err != nil {
		return nil, err
	}
	return &DeviceAuthorizationView{
		Authorization: record,
		Application:   app,
		Project:       project,
		Organization:  organization,
	}, nil
}

func (s *OIDCService) ApproveDeviceAuthorization(ctx context.Context, userCode, sessionID string) (*DeviceAuthorizationView, error) {
	view, err := s.DeviceAuthorizationByUserCode(ctx, userCode)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if view.Authorization.ExpiresAt.Before(now) {
		return nil, errors.New("expired_token")
	}
	if view.Authorization.Status != deviceAuthorizationStatusPending {
		return nil, errors.New("device authorization is no longer pending")
	}
	user, session, err := s.GetSessionUser(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.State != "authenticated" {
		return nil, errors.New("session is not authenticated")
	}
	if user.OrganizationID != view.Organization.ID {
		return nil, errors.New("device authorization organization mismatch")
	}
	allowed, err := userAssignedToApplicationProject(ctx, s.db, view.Application.ID, user.ID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, errors.New("user is not assigned to the target project")
	}
	view.Authorization.Status = deviceAuthorizationStatusApproved
	view.Authorization.UserID = user.ID
	view.Authorization.SessionID = session.ID
	view.Authorization.ApprovedAt = &now
	if err := s.db.WithContext(ctx).Model(&view.Authorization).Updates(map[string]any{
		"status":      view.Authorization.Status,
		"user_id":     view.Authorization.UserID,
		"session_id":  view.Authorization.SessionID,
		"approved_at": now,
	}).Error; err != nil {
		return nil, err
	}
	return view, nil
}

func (s *OIDCService) DenyDeviceAuthorization(ctx context.Context, userCode, sessionID string) (*DeviceAuthorizationView, error) {
	view, err := s.DeviceAuthorizationByUserCode(ctx, userCode)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if view.Authorization.ExpiresAt.Before(now) {
		return nil, errors.New("expired_token")
	}
	user, session, err := s.GetSessionUser(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.State != "authenticated" {
		return nil, errors.New("session is not authenticated")
	}
	if user.OrganizationID != view.Organization.ID {
		return nil, errors.New("device authorization organization mismatch")
	}
	view.Authorization.Status = deviceAuthorizationStatusDenied
	view.Authorization.UserID = user.ID
	view.Authorization.SessionID = session.ID
	view.Authorization.DeniedAt = &now
	if err := s.db.WithContext(ctx).Model(&view.Authorization).Updates(map[string]any{
		"status":      view.Authorization.Status,
		"user_id":     view.Authorization.UserID,
		"session_id":  view.Authorization.SessionID,
		"denied_at":   now,
	}).Error; err != nil {
		return nil, err
	}
	return view, nil
}

func newUserCode() (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	var builder strings.Builder
	builder.Grow(9)
	for i, item := range buf {
		if i == 4 {
			builder.WriteByte('-')
		}
		builder.WriteByte(alphabet[int(item)%len(alphabet)])
	}
	raw := builder.String()
	if len(raw) != 9 {
		return "", fmt.Errorf("invalid generated user code")
	}
	return raw, nil
}

func normalizeUserCode(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "")
	value = strings.ReplaceAll(value, "-", "")
	if len(value) == 8 {
		return value[:4] + "-" + value[4:]
	}
	return value
}

func userAssignedToApplicationProject(ctx context.Context, db *gorm.DB, applicationID, userID string) (bool, error) {
	var app model.Application
	if err := db.WithContext(ctx).First(&app, "id = ?", applicationID).Error; err != nil {
		return false, err
	}
	var project model.Project
	if err := db.WithContext(ctx).Select("id", "status", "user_acl_enabled").First(&project, "id = ?", app.ProjectID).Error; err != nil {
		return false, err
	}
	if strings.TrimSpace(project.Status) == "disabled" {
		return false, nil
	}
	if !project.UserACLEnabled {
		return true, nil
	}
	var count int64
	if err := db.WithContext(ctx).Model(&model.ProjectUserAssignment{}).
		Where("project_id = ? AND user_id = ?", app.ProjectID, userID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
