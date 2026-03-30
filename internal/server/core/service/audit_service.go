package service

import (
	"context"
	"encoding/json"
	"strings"

	"pass-pivot/internal/model"
	sharedauditctx "pass-pivot/internal/server/shared/auditctx"
	sharedaudit "pass-pivot/internal/server/shared/auditlog"

	"gorm.io/gorm"
)

type AuditService struct {
	db *gorm.DB
}

func NewAuditService(db *gorm.DB) *AuditService {
	return &AuditService{db: db}
}

type AuditEvent struct {
	OrganizationID   string
	OrganizationName string
	ProjectID        string
	ProjectName      string
	ApplicationID    string
	ApplicationName  string
	ActorType        string
	ActorID          string
	EventType        string
	Result           string
	TargetType       string
	TargetID         string
	TargetName       string
	IPAddress        string
	UserAgent        string
	RequestMethod    string
	RequestPath      string
	ActorName        string
	Detail           map[string]any
	Changes          []sharedaudit.FieldChange
}

func (s *AuditService) Record(ctx context.Context, event AuditEvent) error {
	requestMeta, _ := sharedauditctx.RequestContextFromContext(ctx)
	if strings.TrimSpace(event.IPAddress) == "" {
		event.IPAddress = requestMeta.IPAddress
	}
	if strings.TrimSpace(event.UserAgent) == "" {
		event.UserAgent = requestMeta.UserAgent
	}
	if strings.TrimSpace(event.RequestMethod) == "" {
		event.RequestMethod = requestMeta.Method
	}
	if strings.TrimSpace(event.RequestPath) == "" {
		event.RequestPath = requestMeta.Path
	}
	auditRecord := model.AuditLog{
		OrganizationID:   event.OrganizationID,
		OrganizationName: s.resolveOrganizationName(ctx, strings.TrimSpace(event.OrganizationID), strings.TrimSpace(event.OrganizationName)),
		ProjectID:        event.ProjectID,
		ProjectName:      s.resolveProjectName(ctx, strings.TrimSpace(event.ProjectID), strings.TrimSpace(event.ProjectName)),
		ApplicationID:    event.ApplicationID,
		ApplicationName:  s.resolveApplicationName(ctx, strings.TrimSpace(event.ApplicationID), strings.TrimSpace(event.ApplicationName)),
		ActorType:        event.ActorType,
		ActorID:          event.ActorID,
		ActorName:        event.ActorName,
		EventType:        event.EventType,
		Result:           event.Result,
		TargetType:       event.TargetType,
		TargetID:         event.TargetID,
		RequestMethod:    event.RequestMethod,
		RequestPath:      event.RequestPath,
		IPAddress:        event.IPAddress,
		UserAgent:        event.UserAgent,
	}
	auditRecord.TargetName = sharedaudit.InferTargetName(strings.TrimSpace(event.TargetName), event.Detail, auditRecord)
	detailPayload := sharedaudit.Detail{
		Request:  sharedaudit.BuildRequestMeta(requestMeta.Method, requestMeta.Path, requestMeta.IPAddress, requestMeta.UserAgent),
		Metadata: event.Detail,
		Changes:  event.Changes,
	}
	if detail, err := json.Marshal(detailPayload); err == nil && string(detail) != "{}" {
		auditRecord.Detail = string(detail)
	}
	return s.db.WithContext(ctx).Create(&auditRecord).Error
}

func (s *AuditService) resolveApplicationName(ctx context.Context, applicationID, applicationName string) string {
	if applicationName != "" {
		return applicationName
	}
	if applicationID == "" {
		return ""
	}
	var app model.Application
	if err := s.db.WithContext(ctx).Select("id", "name").First(&app, "id = ?", applicationID).Error; err == nil {
		if name := strings.TrimSpace(app.Name); name != "" {
			return name
		}
	}
	return applicationID
}

func (s *AuditService) resolveOrganizationName(ctx context.Context, organizationID, organizationName string) string {
	if organizationName != "" {
		return organizationName
	}
	if organizationID == "" {
		return ""
	}
	var organization model.Organization
	if err := s.db.WithContext(ctx).Select("id", "name").First(&organization, "id = ?", organizationID).Error; err == nil {
		if name := strings.TrimSpace(organization.Name); name != "" {
			return name
		}
	}
	return organizationID
}

func (s *AuditService) resolveProjectName(ctx context.Context, projectID, projectName string) string {
	if projectName != "" {
		return projectName
	}
	if projectID == "" {
		return ""
	}
	var project model.Project
	if err := s.db.WithContext(ctx).Select("id", "name").First(&project, "id = ?", projectID).Error; err == nil {
		if name := strings.TrimSpace(project.Name); name != "" {
			return name
		}
	}
	return projectID
}
