package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MFAChannel string

type BaseModel struct {
	ID        string         `gorm:"primaryKey;size:36" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (b *BaseModel) BeforeCreate(tx *gorm.DB) error {
	if b.ID == "" {
		b.ID = uuid.NewString()
	}
	return nil
}

type Organization struct {
	BaseModel
	Name              string               `gorm:"size:128" json:"name"`
	Metadata          map[string]string    `gorm:"serializer:json;type:json" json:"metadata"`
	AllowJWTAccess    bool                 `json:"allowJwtAccess"`
	AllowBasicAccess  bool                 `json:"allowBasicAccess"`
	AllowNoAuthAccess bool                 `json:"allowNoAuthAccess"`
	AllowRefreshToken bool                 `json:"allowRefreshToken"`
	AllowAuthCode     bool                 `json:"allowAuthorizationCode"`
	AllowPKCE         bool                 `json:"allowPKCE"`
	TOSURL            string               `gorm:"size:255" json:"-"`
	PrivacyPolicyURL  string               `gorm:"size:255" json:"-"`
	SupportEmail      string               `gorm:"size:255" json:"-"`
	LogoURL           string               `gorm:"size:255" json:"-"`
	Domains           []OrganizationDomain `gorm:"serializer:json;type:json" json:"-"`
	LoginPolicy       OrganizationLoginPolicy `gorm:"serializer:json;type:json" json:"-"`
	PasswordPolicy    OrganizationPasswordPolicy `gorm:"serializer:json;type:json" json:"-"`
	MFAPolicy         OrganizationMFAPolicy `gorm:"serializer:json;type:json" json:"-"`
	ConsoleSettings   *OrganizationSetting `gorm:"-" json:"consoleSettings,omitempty"`
	Projects          []Project            `json:"projects,omitempty"`
	Users             []User               `json:"users,omitempty"`
	Roles             []Role               `json:"roles,omitempty"`
	ExternalIDPs      []ExternalIDP        `json:"externalIdps,omitempty"`
}

func (Organization) TableName() string {
	return "organization"
}

type OrganizationDomain struct {
	Host     string `json:"host"`
	Verified bool   `json:"verified"`
}

type OrganizationLoginPolicy struct {
	PasswordLoginEnabled bool   `json:"passwordLoginEnabled"`
	PasskeyLoginEnabled  bool   `json:"passkeyLoginEnabled"`
	AllowUsername        bool   `json:"allowUsername"`
	AllowEmail           bool   `json:"allowEmail"`
	AllowPhone           bool   `json:"allowPhone"`
	UsernameMode         string `json:"usernameMode"`
	EmailMode            string `json:"emailMode"`
	PhoneMode            string `json:"phoneMode"`
}

type OrganizationPasswordPolicy struct {
	MinLength        int  `json:"minLength"`
	RequireUppercase bool `json:"requireUppercase"`
	RequireLowercase bool `json:"requireLowercase"`
	RequireNumber    bool `json:"requireNumber"`
	RequireSymbol    bool `json:"requireSymbol"`
	PasswordExpires  bool `json:"passwordExpires"`
	ExpiryDays       int  `json:"expiryDays"`
}

type OrganizationEmailChannel struct {
	Enabled  bool   `json:"enabled"`
	From     string `json:"from"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type OrganizationMFAPolicy struct {
	RequireForAllUsers bool                     `json:"requireForAllUsers"`
	AllowPasskey       bool                     `json:"allowPasskey"`
	AllowTotp          bool                     `json:"allowTotp"`
	AllowEmailCode     bool                     `json:"allowEmailCode"`
	AllowSmsCode       bool                     `json:"allowSmsCode"`
	AllowU2F           bool                     `json:"allowU2f"`
	AllowRecoveryCode  bool                     `json:"allowRecoveryCode"`
	EmailChannel       OrganizationEmailChannel `json:"emailChannel"`
}

type OrganizationSetting struct {
	TOSURL           string                     `gorm:"size:255" json:"tosUrl"`
	PrivacyPolicyURL string                     `gorm:"size:255" json:"privacyPolicyUrl"`
	SupportEmail     string                     `gorm:"size:255" json:"supportEmail"`
	LogoURL          string                     `gorm:"size:255" json:"logoUrl"`
	Domains          []OrganizationDomain       `gorm:"serializer:json;type:json" json:"domains"`
	LoginPolicy      OrganizationLoginPolicy    `gorm:"serializer:json;type:json" json:"loginPolicy"`
	PasswordPolicy   OrganizationPasswordPolicy `gorm:"serializer:json;type:json" json:"passwordPolicy"`
	MFAPolicy        OrganizationMFAPolicy      `gorm:"serializer:json;type:json" json:"mfaPolicy"`
}

type Project struct {
	BaseModel
	OrganizationID string        `gorm:"index;size:36" json:"organizationId"`
	Name           string        `gorm:"size:128" json:"name"`
	Description    string        `gorm:"size:255" json:"description"`
	Applications   []Application `json:"applications,omitempty"`
}

func (Project) TableName() string {
	return "project"
}

type Application struct {
	BaseModel
	ProjectID                string   `gorm:"index;size:36" json:"projectId"`
	Name                     string   `gorm:"size:128" json:"name"`
	Description              string   `gorm:"size:255" json:"description"`
	RedirectURIs             string   `gorm:"type:text" json:"redirectUris"`
	ApplicationType          string   `gorm:"size:32;default:web" json:"applicationType"`
	GrantType                []string `gorm:"serializer:json;type:json" json:"grantType"`
	EnableRefreshToken       bool     `json:"enableRefreshToken"`
	ClientAuthenticationType string   `gorm:"size:64;default:none" json:"clientAuthenticationType"`
	TokenType                []string `gorm:"serializer:json;type:json" json:"tokenType"`
	Roles                    []string `gorm:"serializer:json;type:json" json:"roles"`
	ClientSecretHash         string   `gorm:"size:255" json:"-"`
	PublicKey                string   `gorm:"size:64" json:"publicKey"`
	AccessTokenTTLMinutes    int      `json:"accessTokenTTLMinutes"`
	RefreshTokenTTLHours     int      `json:"refreshTokenTTLHours"`
}

func (Application) TableName() string {
	return "application"
}

type User struct {
	BaseModel
	OrganizationID string       `gorm:"index;size:36" json:"organizationId"`
	Username       string       `gorm:"index;size:128" json:"username"`
	Name           string       `gorm:"size:128" json:"name"`
	Email          string       `gorm:"index;size:128" json:"email"`
	PhoneNumber    string       `gorm:"index;size:64" json:"phoneNumber"`
	Roles          []string     `gorm:"serializer:json;type:json" json:"roles"`
	Status         string       `gorm:"size:32;default:active" json:"status"`
	PasswordHash   string       `gorm:"size:255" json:"-"`
	CurrentUKID    string       `gorm:"column:current_ukid;size:64;index" json:"currentUkid"`
	IsTrusted      bool         `json:"isTrusted"`
	MFAPasskeys    []MFAPasskey `json:"mfaPasskeys,omitempty"`
	Sessions       []Session    `json:"sessions,omitempty"`
}

func (User) TableName() string {
	return "user"
}

type MFAPasskey struct {
	BaseModel
	OrganizationID string `gorm:"index;size:36" json:"organizationId"`
	UserID         string `gorm:"index;size:36" json:"userId"`
	Type           string `gorm:"size:32" json:"type"`
	Identifier     string `gorm:"size:128;index" json:"identifier"`
	PublicKey      string `gorm:"type:text" json:"publicKey"`
	PublicKeyID    string `gorm:"size:128;index" json:"publicKeyId"`
	SignCount      uint32 `json:"signCount"`
	IsPasskey      bool   `json:"isPasskey"`
	IsU2f          bool   `json:"isU2f"`
	BackupEligible bool   `json:"backupEligible"`
	BackupState    bool   `json:"backupState"`
	Transports     string `gorm:"size:255" json:"transports"`
}

func (MFAPasskey) TableName() string {
	return "mfa_passkey"
}

type MFAEnrollment struct {
	BaseModel
	OrganizationID string     `gorm:"index;size:36" json:"organizationId"`
	UserID         string     `gorm:"index;size:36" json:"userId"`
	Method         string     `gorm:"size:32;index" json:"method"`
	Label          string     `gorm:"size:128" json:"label"`
	Secret         string     `gorm:"size:255" json:"-"`
	Target         string     `gorm:"size:255" json:"target"`
	Status         string     `gorm:"size:32;default:active" json:"status"`
	LastUsedAt     *time.Time `json:"lastUsedAt"`
}

func (MFAEnrollment) TableName() string {
	return "mfa_enrollment"
}

type MFAChallenge struct {
	BaseModel
	SessionID       string     `gorm:"index;size:36" json:"sessionId"`
	UserID          string     `gorm:"index;size:36" json:"userId"`
	OrganizationID  string     `gorm:"index;size:36" json:"organizationId"`
	Method          string     `gorm:"size:32;index" json:"method"`
	CodeHash        string     `gorm:"size:255" json:"-"`
	Target          string     `gorm:"size:255" json:"target"`
	ExpiresAt       time.Time  `json:"expiresAt"`
	ConsumedAt      *time.Time `json:"consumedAt"`
	AttemptCount    int        `json:"attemptCount"`
	DeliveryMessage string     `gorm:"size:255" json:"deliveryMessage"`
}

func (MFAChallenge) TableName() string {
	return "mfa_challenge"
}

type MFARecoveryCode struct {
	BaseModel
	UserID         string     `gorm:"index;size:36" json:"userId"`
	OrganizationID string     `gorm:"index;size:36" json:"organizationId"`
	CodeHash       string     `gorm:"size:255" json:"-"`
	ConsumedAt     *time.Time `json:"consumedAt"`
}

func (MFARecoveryCode) TableName() string {
	return "mfa_recovery_code"
}

type Session struct {
	BaseModel
	OrganizationID        string `gorm:"index;size:36" json:"organizationId"`
	UserID                string `gorm:"index;size:36" json:"userId"`
	ApplicationID         string `gorm:"index;size:36" json:"applicationId"`
	DeviceID              string `gorm:"index;size:36" json:"deviceId"`
	State                 string `gorm:"size:32;default:pending" json:"state"`
	RequiresConfirmation  bool   `json:"requiresConfirmation"`
	RequiresMFA           bool   `json:"requiresMFA"`
	TrustedDeviceEligible bool   `json:"trustedDeviceEligible"`
	SecondFactorMethod    string `gorm:"size:32" json:"secondFactorMethod"`
	IPAddress             string `gorm:"size:64" json:"ipAddress"`
	UserAgent             string `gorm:"size:255" json:"userAgent"`
	RiskLevel             string `gorm:"size:16;default:medium" json:"riskLevel"`
	LoginChallenge        string `gorm:"size:128;index" json:"loginChallenge"`
}

func (Session) TableName() string {
	return "session"
}

type AuthorizationCode struct {
	BaseModel
	SessionID           string     `gorm:"index;size:36" json:"sessionId"`
	UserID              string     `gorm:"index;size:36" json:"userId"`
	ApplicationID       string     `gorm:"index;size:36" json:"applicationId"`
	Code                string     `gorm:"uniqueIndex;size:128" json:"code"`
	RedirectURI         string     `gorm:"size:255" json:"redirectUri"`
	Scope               string     `gorm:"size:255" json:"scope"`
	Nonce               string     `gorm:"size:255" json:"nonce"`
	CodeChallenge       string     `gorm:"size:255" json:"codeChallenge"`
	CodeChallengeMethod string     `gorm:"size:16" json:"codeChallengeMethod"`
	ExpiresAt           time.Time  `json:"expiresAt"`
	ConsumedAt          *time.Time `json:"consumedAt"`
}

func (AuthorizationCode) TableName() string {
	return "authorization_code"
}

type Token struct {
	BaseModel
	SessionID      string     `gorm:"index;size:36" json:"sessionId"`
	UserID         string     `gorm:"index;size:36" json:"userId"`
	ApplicationID  string     `gorm:"index;size:36" json:"applicationId"`
	Type           string     `gorm:"size:32" json:"type"`
	Token          string     `gorm:"uniqueIndex;size:255" json:"token"`
	Scope          string     `gorm:"size:255" json:"scope"`
	UKID           string     `gorm:"size:64;index" json:"ukid"`
	ExpiresAt      time.Time  `json:"expiresAt"`
	RevokedAt      *time.Time `json:"revokedAt"`
	RevocationNote string     `gorm:"size:255" json:"revocationNote"`
}

func (Token) TableName() string {
	return "token"
}

type Device struct {
	BaseModel
	UserID         string     `gorm:"index;size:36" json:"userId"`
	OrganizationID string     `gorm:"index;size:36" json:"organizationId"`
	Fingerprint    string     `gorm:"size:255;index" json:"fingerprint"`
	Description    string     `gorm:"size:255" json:"description"`
	UserAgent      string     `gorm:"size:255" json:"userAgent"`
	LastLoginIP    string     `gorm:"size:64" json:"lastLoginIp"`
	FirstSeenAt    *time.Time `json:"firstSeenAt"`
	LastSeenAt     time.Time  `json:"lastSeenAt"`
	Trusted        bool       `json:"trusted"`
}

func (Device) TableName() string {
	return "device"
}

type Role struct {
	BaseModel
	OrganizationID string `gorm:"index;size:36" json:"organizationId"`
	Name           string `gorm:"size:128" json:"name"`
	Type           string `gorm:"size:32;index" json:"type"`
	Description    string `gorm:"size:255" json:"description"`
}

func (Role) TableName() string {
	return "role"
}

type PolicyAPIRule struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

type Policy struct {
	BaseModel
	OrganizationID string          `gorm:"index;size:36" json:"organizationId"`
	RoleID         string          `gorm:"index;size:36" json:"roleId"`
	Name           string          `gorm:"size:128;index" json:"name"`
	Effect         string          `gorm:"size:16;default:allow" json:"effect"`
	Priority       int             `gorm:"index" json:"priority"`
	APIRules       []PolicyAPIRule `gorm:"serializer:json;type:json" json:"apiRules"`
}

func (Policy) TableName() string {
	return "policy"
}

type AuditLog struct {
	BaseModel
	OrganizationID string `gorm:"index;size:36" json:"organizationId"`
	ProjectID      string `gorm:"index;size:36" json:"projectId"`
	ApplicationID  string `gorm:"index;size:36" json:"applicationId"`
	ActorType      string `gorm:"size:32" json:"actorType"`
	ActorID        string `gorm:"size:36" json:"actorId"`
	EventType      string `gorm:"size:64;index" json:"eventType"`
	Result         string `gorm:"size:16" json:"result"`
	TargetType     string `gorm:"size:32" json:"targetType"`
	TargetID       string `gorm:"size:36" json:"targetId"`
	IPAddress      string `gorm:"size:64" json:"ipAddress"`
	UserAgent      string `gorm:"size:255" json:"userAgent"`
	CorrelationID  string `gorm:"size:64;index" json:"correlationId"`
	Detail         string `gorm:"type:text" json:"detail"`
}

func (AuditLog) TableName() string {
	return "audit_log"
}

type ExternalIDP struct {
	BaseModel
	OrganizationID   string            `gorm:"index;size:36" json:"organizationId"`
	Protocol         string            `gorm:"size:32;default:oidc" json:"protocol"`
	Name             string            `gorm:"size:128" json:"name"`
	Issuer           string            `gorm:"size:255" json:"issuer"`
	ClientID         string            `gorm:"size:128" json:"clientId"`
	ClientSecret     string            `gorm:"size:255" json:"-"`
	AuthorizationURL string            `gorm:"size:255" json:"authorizationUrl"`
	TokenURL         string            `gorm:"size:255" json:"tokenUrl"`
	UserInfoURL      string            `gorm:"size:255" json:"userInfoUrl"`
	JWKSURL          string            `gorm:"size:255" json:"jwksUrl"`
	Scopes           string            `gorm:"size:255" json:"scopes"`
	Metadata         map[string]string `gorm:"serializer:json;type:json" json:"metadata"`
	AutoCreateUser   bool              `gorm:"default:false" json:"autoCreateUser"`
}

func (ExternalIDP) TableName() string {
	return "external_idp"
}

type ExternalIdentityBinding struct {
	BaseModel
	OrganizationID string `gorm:"index;size:36" json:"organizationId"`
	UserID         string `gorm:"index;size:36" json:"userId"`
	ExternalIDPID  string `gorm:"column:external_idp_id;index;size:36" json:"externalIdpId"`
	Issuer         string `gorm:"size:255;index" json:"issuer"`
	Subject        string `gorm:"size:255;index" json:"subject"`
}

func (ExternalIdentityBinding) TableName() string {
	return "external_identity_binding"
}

type ExternalAuthState struct {
	BaseModel
	OrganizationID string    `gorm:"index;size:36" json:"organizationId"`
	ProviderID     string    `gorm:"index;size:36" json:"providerId"`
	UserID         string    `gorm:"index;size:36" json:"userId"`
	State          string    `gorm:"uniqueIndex;size:128" json:"state"`
	Nonce          string    `gorm:"size:128" json:"nonce"`
	RedirectURI    string    `gorm:"size:255" json:"redirectUri"`
	CodeVerifier   string    `gorm:"size:255" json:"codeVerifier"`
	ExpiresAt      time.Time `json:"expiresAt"`
}

func (ExternalAuthState) TableName() string {
	return "external_auth_state"
}
