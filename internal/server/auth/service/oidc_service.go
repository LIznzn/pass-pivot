package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"

	"pass-pivot/internal/config"
	"pass-pivot/internal/model"
	coreservice "pass-pivot/internal/server/core/service"
	"pass-pivot/utils"
)

type OIDCService struct {
	db    *gorm.DB
	cfg   config.Config
	audit *AuditService
	auth  oidcAuthService
	keys  *ProviderKeyStore
}

type AuthorizeCaptchaBootstrap struct {
	Provider  string `json:"provider"`
	ClientKey string `json:"client_key,omitempty"`
}

type AuthorizeCaptchaChallengeBootstrap struct {
	ImageDataURL   string `json:"imageDataUrl,omitempty"`
	ChallengeToken string `json:"challengeToken,omitempty"`
}

type OIDCMetadata struct {
	Issuer                                     string   `json:"issuer"`
	JWKSURI                                    string   `json:"jwks_uri"`
	AuthorizationEndpoint                      string   `json:"authorization_endpoint"`
	DeviceAuthorizationEndpoint                string   `json:"device_authorization_endpoint,omitempty"`
	TokenEndpoint                              string   `json:"token_endpoint"`
	UserInfoEndpoint                           string   `json:"userinfo_endpoint"`
	RevocationEndpoint                         string   `json:"revocation_endpoint,omitempty"`
	IntrospectionEndpoint                      string   `json:"introspection_endpoint,omitempty"`
	EndSessionEndpoint                         string   `json:"end_session_endpoint,omitempty"`
	ScopesSupported                            []string `json:"scopes_supported"`
	ResponseTypesSupported                     []string `json:"response_types_supported"`
	ResponseModesSupported                     []string `json:"response_modes_supported,omitempty"`
	GrantTypesSupported                        []string `json:"grant_types_supported,omitempty"`
	SubjectTypesSupported                      []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported           []string `json:"id_token_signing_alg_values_supported"`
	TokenEndpointAuthMethodsSupported          []string `json:"token_endpoint_auth_methods_supported,omitempty"`
	TokenEndpointAuthSigningAlgValuesSupported []string `json:"token_endpoint_auth_signing_alg_values_supported,omitempty"`
	ClaimsSupported                            []string `json:"claims_supported,omitempty"`
	RequestURIParameterSupported               bool     `json:"request_uri_parameter_supported"`
	CodeChallengeMethodsSupported              []string `json:"code_challenge_methods_supported,omitempty"`
}

type oidcAuthService interface {
	IssueClientCredentialTokenForApplication(ctx context.Context, app model.Application, scope string) ([]model.Token, error)
	IssuePasswordGrantTokenForApplication(ctx context.Context, app model.Application, identifier, password, scope, ipAddress, userAgent string) ([]model.Token, *model.User, *model.Session, error)
	IssueTokensForApplication(ctx context.Context, user model.User, session model.Session, applicationID, scope string) ([]model.Token, error)
	RevokeToken(ctx context.Context, tokenValue, reason string) error
	ResetUserUKID(ctx context.Context, userID string) (string, error)
}

func NewOIDCService(db *gorm.DB, cfg config.Config, audit *AuditService, auth oidcAuthService, keys *ProviderKeyStore) *OIDCService {
	return &OIDCService{db: db, cfg: cfg, audit: audit, auth: auth, keys: keys}
}

func DefaultCaptchaSecret(rootSecret, organizationID string) string {
	return strings.TrimSpace(rootSecret) + ":" + strings.TrimSpace(organizationID) + ":default-captcha"
}

func (s *OIDCService) BuildAuthorizeCaptchaBootstrap(ctx context.Context, organizationID string) (*AuthorizeCaptchaBootstrap, error) {
	config, err := s.authorizeCaptchaPublicConfig(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, nil
	}
	return config, nil
}

func (s *OIDCService) BuildAuthorizeCaptchaChallengeBootstrap(ctx context.Context, organizationID string) (*AuthorizeCaptchaChallengeBootstrap, error) {
	config, err := s.authorizeCaptchaPublicConfig(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, nil
	}
	switch config.Provider {
	case "default":
		challenge, err := coreservice.CreateDefaultCaptcha(DefaultCaptchaSecret(s.cfg.Secret, organizationID), time.Now())
		if err != nil {
			return nil, err
		}
		return &AuthorizeCaptchaChallengeBootstrap{
			ImageDataURL:   challenge.ImageDataURL,
			ChallengeToken: challenge.ChallengeToken,
		}, nil
	case "google", "cloudflare":
		return &AuthorizeCaptchaChallengeBootstrap{}, nil
	default:
		return nil, errors.New("invalid captcha provider")
	}
}

func (s *OIDCService) authorizeCaptchaPublicConfig(ctx context.Context, organizationID string) (*AuthorizeCaptchaBootstrap, error) {
	_, settings, err := coreservice.LoadOrganizationConsoleSettings(ctx, s.db, organizationID)
	if err != nil {
		return nil, err
	}
	switch settings.Captcha.Provider {
	case "disabled":
		return nil, nil
	case "default":
		return &AuthorizeCaptchaBootstrap{Provider: "default"}, nil
	case "google", "cloudflare":
		return &AuthorizeCaptchaBootstrap{
			Provider:  settings.Captcha.Provider,
			ClientKey: strings.TrimSpace(settings.Captcha.ClientKey),
		}, nil
	default:
		return nil, errors.New("invalid captcha provider")
	}
}

func (s *OIDCService) AvailableMFAMethodsForSession(ctx context.Context, sessionID string) ([]string, error) {
	session, err := s.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", session.UserID).Error; err != nil {
		return nil, err
	}
	_, settings, err := coreservice.LoadOrganizationConsoleSettings(ctx, s.db, user.OrganizationID)
	if err != nil {
		return nil, err
	}
	mfaRequired := settings.MFAPolicy.RequireForAllUsers
	if !mfaRequired {
		var enabledCount int64
		if err := s.db.WithContext(ctx).Model(&model.MFAEnrollment{}).
			Where("user_id = ? AND method = ? AND status = ? AND deleted_at IS NULL", user.ID, "mfa", "active").
			Count(&enabledCount).Error; err != nil {
			return nil, err
		}
		mfaRequired = enabledCount > 0
	}
	if !mfaRequired {
		return []string{}, nil
	}
	primaryMethods := make([]string, 0, 5)
	hasRecoveryCode := false

	if settings.MFAPolicy.AllowU2F {
		var u2fCount int64
		if err := s.db.WithContext(ctx).Model(&model.SecureKey{}).
			Where("user_id = ? AND u2f_enable = ? AND deleted_at IS NULL", user.ID, true).
			Count(&u2fCount).Error; err != nil {
			return nil, err
		}
		var u2fEnrollmentCount int64
		if err := s.db.WithContext(ctx).Model(&model.MFAEnrollment{}).
			Where("user_id = ? AND method = ? AND status = ? AND deleted_at IS NULL", user.ID, "u2f", "active").
			Count(&u2fEnrollmentCount).Error; err != nil {
			return nil, err
		}
		if u2fCount > 0 && u2fEnrollmentCount > 0 {
			primaryMethods = append(primaryMethods, "u2f")
		}
	}

	if settings.MFAPolicy.AllowTotp {
		var totpCount int64
		if err := s.db.WithContext(ctx).Model(&model.MFAEnrollment{}).
			Where("user_id = ? AND method = ? AND status = ? AND deleted_at IS NULL", user.ID, "totp", "active").
			Count(&totpCount).Error; err != nil {
			return nil, err
		}
		if totpCount > 0 {
			primaryMethods = append(primaryMethods, "totp")
		}
	}

	if settings.MFAPolicy.AllowEmailCode &&
		strings.TrimSpace(user.Email) != "" &&
		settings.MFAPolicy.EmailChannel.Enabled &&
		strings.TrimSpace(settings.MFAPolicy.EmailChannel.From) != "" &&
		strings.TrimSpace(settings.MFAPolicy.EmailChannel.Host) != "" &&
		settings.MFAPolicy.EmailChannel.Port > 0 {
		primaryMethods = append(primaryMethods, "email_code")
	}

	if settings.MFAPolicy.AllowSmsCode && strings.TrimSpace(user.PhoneNumber) != "" {
		primaryMethods = append(primaryMethods, "sms_code")
	}

	if settings.MFAPolicy.AllowRecoveryCode {
		var recoveryCount int64
		if err := s.db.WithContext(ctx).Model(&model.MFARecoveryCode{}).
			Where("user_id = ? AND consumed_at IS NULL AND deleted_at IS NULL", user.ID).
			Count(&recoveryCount).Error; err != nil {
			return nil, err
		}
		if recoveryCount > 0 {
			hasRecoveryCode = true
		}
	}

	if len(primaryMethods) == 0 {
		return []string{}, nil
	}
	if hasRecoveryCode {
		primaryMethods = append(primaryMethods, "recovery_code")
	}
	return primaryMethods, nil
}

type AuthorizeInput struct {
	SessionID           string
	ClientID            string
	RedirectURI         string
	Scope               string
	CodeChallenge       string
	CodeChallengeMethod string
}

func (s *OIDCService) Metadata(ctx context.Context, clientID, applicationID string) (OIDCMetadata, error) {
	settings, err := s.resolveSettings(ctx, clientID, applicationID)
	if err != nil {
		return OIDCMetadata{}, err
	}
	return buildOIDCMetadata(settings.TokenIssuer), nil
}

func (s *OIDCService) MetadataByIssuer(ctx context.Context) (OIDCMetadata, error) {
	settings := coreservice.ApplicationSettings{TokenIssuer: s.cfg.AuthURL}
	return buildOIDCMetadata(settings.TokenIssuer), nil
}

func (s *OIDCService) JWKS(ctx context.Context, clientID, applicationID string) (map[string]any, error) {
	keys, err := s.keys.ProviderJWKs(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{"keys": keys}, nil
}

func (s *OIDCService) JWKSByIssuer(ctx context.Context) (map[string]any, error) {
	keys, err := s.keys.ProviderJWKs(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{"keys": keys}, nil
}

func (s *OIDCService) Authorize(ctx context.Context, in AuthorizeInput) (*model.AuthorizationCode, error) {
	var app model.Application
	if err := s.db.WithContext(ctx).Where("id = ?", in.ClientID).First(&app).Error; err != nil {
		return nil, errors.New("client not found")
	}
	var session model.Session
	if err := s.db.WithContext(ctx).First(&session, "id = ?", in.SessionID).Error; err != nil {
		return nil, err
	}
	if session.State != "authenticated" {
		return nil, errors.New("session is not authenticated")
	}
	codeValue, err := utils.RandomToken(24)
	if err != nil {
		return nil, err
	}
	code := &model.AuthorizationCode{
		SessionID:           session.ID,
		UserID:              session.UserID,
		ApplicationID:       app.ID,
		Code:                codeValue,
		RedirectURI:         in.RedirectURI,
		Scope:               in.Scope,
		Nonce:               "",
		CodeChallenge:       in.CodeChallenge,
		CodeChallengeMethod: in.CodeChallengeMethod,
		ExpiresAt:           time.Now().Add(5 * time.Minute),
	}
	storeAuthorizationCode(*code)
	return code, nil
}

func (s *OIDCService) ExchangeCode(ctx context.Context, audience, clientID, clientSecret, clientAssertionType, clientAssertion, codeValue, redirectURI, verifier string) ([]model.Token, string, error) {
	app, err := s.validateClientAuthentication(ctx, audience, clientID, clientSecret, clientAssertionType, clientAssertion)
	if err != nil {
		return nil, "", err
	}
	if !applicationSupportsAuthorizationCode(app) {
		return nil, "", errors.New("authorization_code grant is not enabled for this application")
	}
	now := time.Now()
	code, ok := consumeAuthorizationCode(codeValue, now)
	if !ok {
		return nil, "", errors.New("code is no longer valid")
	}
	if code.RedirectURI != redirectURI {
		deleteAuthorizationCode(codeValue)
		return nil, "", errors.New("redirect_uri mismatch")
	}
	if strings.TrimSpace(code.CodeChallenge) != "" {
		if err := verifyCodeChallenge(code.CodeChallengeMethod, code.CodeChallenge, verifier); err != nil {
			deleteAuthorizationCode(codeValue)
			return nil, "", err
		}
	}
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", code.UserID).Error; err != nil {
		return nil, "", err
	}
	var session model.Session
	if err := s.db.WithContext(ctx).First(&session, "id = ?", code.SessionID).Error; err != nil {
		return nil, "", err
	}
	tokens, err := s.auth.IssueTokensForApplication(ctx, user, session, code.ApplicationID, code.Scope)
	if err != nil {
		return nil, "", err
	}
	var idToken string
	if applicationReturnsIDToken(app.TokenType) {
		idToken, err = s.signIDToken(ctx, app.ID, user, app.ID, code.Scope, code.Nonce, new(session.CreatedAt), session.ID)
		if err != nil {
			return nil, "", err
		}
	}
	return tokens, idToken, nil
}

func (s *OIDCService) UserInfo(ctx context.Context, accessToken string) (map[string]any, error) {
	var token model.Token
	if err := s.db.WithContext(ctx).Where("token = ? AND type = ?", accessToken, "access_token").First(&token).Error; err != nil {
		return nil, errors.New("token not found")
	}
	if token.ExpiresAt.Before(time.Now()) || token.RevokedAt != nil {
		return nil, errors.New("token expired or revoked")
	}
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", token.UserID).Error; err != nil {
		return nil, err
	}
	settings, err := coreservice.ResolveApplicationSettingsByID(ctx, s.db, s.cfg, token.ApplicationID)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"iss":                settings.TokenIssuer,
		"sub":                user.ID,
		"name":               user.Name,
		"email":              user.Email,
		"phone_number":       user.PhoneNumber,
		"preferred_username": user.Username,
	}, nil
}

func verifyCodeChallenge(method string, challenge string, verifier string) error {
	if challenge == "" || verifier == "" {
		return errors.New("pkce is required")
	}
	switch strings.ToUpper(method) {
	case "S256":
		hash := sha256.Sum256([]byte(verifier))
		encoded := base64.RawURLEncoding.EncodeToString(hash[:])
		if encoded != challenge {
			return errors.New("pkce verifier mismatch")
		}
	default:
		return errors.New("unsupported code challenge method")
	}
	return nil
}

func applicationReturnsIDToken(tokenType []string) bool {
	return coreservice.TokenTypesContain(tokenType, "id_token")
}

func applicationSupportsAuthorizationCode(app model.Application) bool {
	return coreservice.AppGrantTypesContain(app.GrantType, "authorization_code") ||
		coreservice.AppGrantTypesContain(app.GrantType, "authorization_code_pkce")
}

func applicationSupportsAuthorizationCodePKCE(app model.Application) bool {
	return coreservice.AppGrantTypesContain(app.GrantType, "authorization_code_pkce")
}

func (s *OIDCService) signIDToken(ctx context.Context, applicationID string, user model.User, audience, scope, nonce string, authTime *time.Time, sessionID string) (string, error) {
	settings, err := coreservice.ResolveApplicationSettingsByID(ctx, s.db, s.cfg, applicationID)
	if err != nil {
		return "", err
	}
	keys, err := s.keys.ProviderKeysForApplication(ctx, applicationID)
	if err != nil {
		return "", err
	}
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   settings.TokenIssuer,
		"sub":   user.ID,
		"aud":   audience,
		"iat":   now.Unix(),
		"exp":   now.Add(10 * time.Minute).Unix(),
		"scope": scope,
	}
	if nonce != "" {
		claims["nonce"] = nonce
	}
	if authTime != nil {
		claims["auth_time"] = authTime.Unix()
	}
	if sessionID != "" {
		claims["sid"] = sessionID
	}
	if user.Name != "" {
		claims["name"] = user.Name
	}
	if user.Email != "" {
		claims["email"] = user.Email
	}
	if user.PhoneNumber != "" {
		claims["phone_number"] = user.PhoneNumber
	}
	if user.Username != "" {
		claims["preferred_username"] = user.Username
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = keys.KeyID
	return token.SignedString(keys.SigningKey)
}

func (s *OIDCService) SignIDTokenForApplication(ctx context.Context, applicationID string, user model.User, audience, scope, nonce string, authTime *time.Time, sessionID string) (string, error) {
	return s.signIDToken(ctx, applicationID, user, audience, scope, nonce, authTime, sessionID)
}

func (s *OIDCService) resolveSettings(ctx context.Context, clientID, applicationID string) (coreservice.ApplicationSettings, error) {
	if clientID != "" {
		_, settings, err := coreservice.ResolveApplicationSettingsByClientID(ctx, s.db, s.cfg, clientID)
		return settings, err
	}
	return coreservice.ResolveApplicationSettingsByID(ctx, s.db, s.cfg, applicationID)
}

func buildOIDCMetadata(issuer string) OIDCMetadata {
	return OIDCMetadata{
		Issuer:                                     issuer,
		AuthorizationEndpoint:                      issuer + "/auth/authorize",
		TokenEndpoint:                              issuer + "/auth/token",
		DeviceAuthorizationEndpoint:                issuer + "/auth/device/code",
		UserInfoEndpoint:                           issuer + "/auth/userinfo",
		JWKSURI:                                    issuer + "/auth/keys",
		EndSessionEndpoint:                         issuer + "/auth/end_session",
		ScopesSupported:                            []string{"openid", "profile", "email", "phone"},
		ResponseTypesSupported:                     []string{"code", "token", "id_token", "id_token token"},
		ResponseModesSupported:                     []string{"query", "fragment"},
		GrantTypesSupported:                        []string{"authorization_code", "implicit", "client_credentials", "password", "refresh_token", "urn:ietf:params:oauth:grant-type:device_code"},
		SubjectTypesSupported:                      []string{"public"},
		IDTokenSigningAlgValuesSupported:           []string{"RS256"},
		TokenEndpointAuthMethodsSupported:          []string{"client_secret_basic", "client_secret_post", "private_key_jwt", "none"},
		TokenEndpointAuthSigningAlgValuesSupported: []string{"RS256"},
		ClaimsSupported:                            []string{"sub", "iss", "aud", "iat", "exp", "auth_time", "nonce", "sid", "name", "email", "phone_number", "preferred_username"},
		RequestURIParameterSupported:               false,
		CodeChallengeMethodsSupported:              []string{"S256"},
		RevocationEndpoint:                         issuer + "/auth/revoke",
		IntrospectionEndpoint:                      issuer + "/auth/introspect",
	}
}
