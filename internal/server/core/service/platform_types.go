package service

import (
	"time"

	"pass-pivot/internal/model"
)

type ApplicationMutationResult struct {
	Application         model.Application `json:"application"`
	GeneratedPrivateKey string            `json:"generatedPrivateKey,omitempty"`
}

type AccessTokenIdentity struct {
	Token       model.Token
	Session     *model.Session
	User        *model.User
	Application model.Application
}

type UserDetailPasskey struct {
	ID          string    `json:"id"`
	PublicKeyID string    `json:"publicKeyId"`
	Identifier  string    `json:"identifier"`
	SignCount   uint32    `json:"signCount"`
	IsPasskey   bool      `json:"isPasskey"`
	IsU2f       bool      `json:"isU2f"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type UserDetailBinding struct {
	ID            string    `json:"id"`
	ExternalIDPID string    `json:"externalIdpId"`
	ProviderName  string    `json:"providerName"`
	Issuer        string    `json:"issuer"`
	Subject       string    `json:"subject"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type UserDetailRecoverySummary struct {
	Total           int        `json:"total"`
	Available       int        `json:"available"`
	Consumed        int        `json:"consumed"`
	LastGeneratedAt *time.Time `json:"lastGeneratedAt"`
}

type UserDetailDevice struct {
	ID                string     `json:"id"`
	DeviceFingerprint string     `json:"deviceFingerprint"`
	UserAgent         string     `json:"userAgent"`
	Online            bool       `json:"online"`
	Trusted           bool       `json:"trusted"`
	LastLoginIP       string     `json:"lastLoginIp"`
	IPLocation        string     `json:"ipLocation"`
	LastLoginAt       *time.Time `json:"lastLoginAt"`
	FirstLoginAt      *time.Time `json:"firstLoginAt"`
}

type AuditLogView struct {
	model.AuditLog
	IPLocation string `json:"ipLocation"`
}

type UserDetailData struct {
	User               model.User                `json:"user"`
	PasswordCredential bool                      `json:"passwordCredential"`
	Passkeys           []UserDetailPasskey       `json:"passkeys"`
	Bindings           []UserDetailBinding       `json:"bindings"`
	ExternalIDPs       []model.ExternalIDP       `json:"externalIdps"`
	MFAEnrollments     []model.MFAEnrollment     `json:"mfaEnrollments"`
	Devices            []UserDetailDevice        `json:"devices"`
	RecoverySummary    UserDetailRecoverySummary `json:"recoverySummary"`
	RecentAuditLogs    []AuditLogView            `json:"recentAuditLogs"`
}

type PublicExternalIDP struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organizationId"`
	Protocol       string `json:"protocol"`
	Name           string `json:"name"`
	Issuer         string `json:"issuer"`
}

type LoginTarget struct {
	OrganizationID   string              `json:"organizationId"`
	OrganizationName string              `json:"organizationName"`
	ProjectID        string              `json:"projectId"`
	ProjectName      string              `json:"projectName"`
	ApplicationID    string              `json:"applicationId"`
	ApplicationName  string              `json:"applicationName"`
	ExternalIDPs     []PublicExternalIDP `json:"externalIdps"`
}
