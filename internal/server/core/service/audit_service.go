package service

import (
	"context"
	"encoding/json"

	"pass-pivot/internal/model"

	"gorm.io/gorm"
)

type AuditService struct {
	db *gorm.DB
}

func NewAuditService(db *gorm.DB) *AuditService {
	return &AuditService{db: db}
}

type AuditEvent struct {
	OrganizationID string
	ProjectID      string
	ApplicationID  string
	ActorType      string
	ActorID        string
	EventType      string
	Result         string
	TargetType     string
	TargetID       string
	IPAddress      string
	UserAgent      string
	CorrelationID  string
	Detail         map[string]any
}

func (s *AuditService) Record(ctx context.Context, event AuditEvent) error {
	detail, _ := json.Marshal(event.Detail)
	return s.db.WithContext(ctx).Create(&model.AuditLog{
		OrganizationID: event.OrganizationID,
		ProjectID:      event.ProjectID,
		ApplicationID:  event.ApplicationID,
		ActorType:      event.ActorType,
		ActorID:        event.ActorID,
		EventType:      event.EventType,
		Result:         event.Result,
		TargetType:     event.TargetType,
		TargetID:       event.TargetID,
		IPAddress:      event.IPAddress,
		UserAgent:      event.UserAgent,
		CorrelationID:  event.CorrelationID,
		Detail:         string(detail),
	}).Error
}
