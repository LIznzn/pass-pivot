package fido

import (
	"context"
	"net/url"
	"sync"

	"gorm.io/gorm"

	"pass-pivot/internal/config"
)

type Service struct {
	db                      *gorm.DB
	cfg                     config.Config
	recordRegistrationAudit func(context.Context, RegistrationAuditRecord) error
	webauthnSessions        map[string]webauthnChallengeRecord
	webauthnMu              sync.RWMutex
}

type AssertionResult struct {
	OrganizationID string
	UserID         string
	SessionID      string
	CredentialID   string
	Usage          string
}

type RegistrationAuditRecord struct {
	OrganizationID string
	UserID         string
	CredentialID   string
	Purpose        string
}

func NewService(db *gorm.DB, cfg config.Config, recordRegistrationAudit func(context.Context, RegistrationAuditRecord) error) (*Service, error) {
	if _, err := url.Parse(cfg.AuthURL); err != nil {
		return nil, err
	}
	return &Service{
		db:                      db,
		cfg:                     cfg,
		recordRegistrationAudit: recordRegistrationAudit,
		webauthnSessions:        map[string]webauthnChallengeRecord{},
	}, nil
}
