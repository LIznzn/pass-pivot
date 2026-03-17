package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"

	"pass-pivot/internal/config"
	"pass-pivot/internal/model"
	coreservice "pass-pivot/internal/server/core/service"
	"pass-pivot/util"
)

type OIDCService struct {
	db    *gorm.DB
	cfg   config.Config
	audit *AuditService
	auth  oidcAuthService
	keys  *ProviderKeyStore
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

type AuthorizeInput struct {
	SessionID           string
	ClientID            string
	RedirectURI         string
	Scope               string
	CodeChallenge       string
	CodeChallengeMethod string
}

func (s *OIDCService) Metadata(ctx context.Context, clientID, applicationID string) (map[string]any, error) {
	settings, err := s.resolveSettings(ctx, clientID, applicationID)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"issuer":                                settings.TokenIssuer,
		"authorization_endpoint":                settings.TokenIssuer + "/auth/authorize",
		"token_endpoint":                        settings.TokenIssuer + "/auth/token",
		"userinfo_endpoint":                     settings.TokenIssuer + "/auth/userinfo",
		"jwks_uri":                              settings.TokenIssuer + "/auth/keys",
		"revocation_endpoint":                   settings.TokenIssuer + "/auth/revoke",
		"introspection_endpoint":                settings.TokenIssuer + "/auth/introspect",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "client_credentials", "password", "refresh_token"},
		"scopes_supported":                      []string{"openid", "profile", "email", "phone"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post", "private_key_jwt", "none"},
		"token_endpoint_auth_signing_alg_values_supported":         []string{"EdDSA"},
		"revocation_endpoint_auth_methods_supported":               []string{"client_secret_basic", "client_secret_post", "private_key_jwt", "none"},
		"revocation_endpoint_auth_signing_alg_values_supported":    []string{"EdDSA"},
		"introspection_endpoint_auth_methods_supported":            []string{"client_secret_basic", "client_secret_post", "private_key_jwt", "none"},
		"introspection_endpoint_auth_signing_alg_values_supported": []string{"EdDSA"},
		"response_modes_supported":                                 []string{"query"},
		"code_challenge_methods_supported":                         []string{"S256"},
		"id_token_signing_alg_values_supported":                    []string{"RS256"},
		"subject_types_supported":                                  []string{"public"},
		"claims_supported":                                         []string{"sub", "iss", "name", "email", "phone_number", "preferred_username"},
	}, nil
}

func (s *OIDCService) MetadataByIssuer(ctx context.Context) (map[string]any, error) {
	settings := coreservice.ApplicationSettings{TokenIssuer: s.cfg.AuthURL}
	return map[string]any{
		"issuer":                                settings.TokenIssuer,
		"authorization_endpoint":                settings.TokenIssuer + "/auth/authorize",
		"token_endpoint":                        settings.TokenIssuer + "/auth/token",
		"userinfo_endpoint":                     settings.TokenIssuer + "/auth/userinfo",
		"jwks_uri":                              settings.TokenIssuer + "/auth/keys",
		"revocation_endpoint":                   settings.TokenIssuer + "/auth/revoke",
		"introspection_endpoint":                settings.TokenIssuer + "/auth/introspect",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "client_credentials", "password", "refresh_token"},
		"scopes_supported":                      []string{"openid", "profile", "email", "phone"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post", "private_key_jwt", "none"},
		"token_endpoint_auth_signing_alg_values_supported":         []string{"EdDSA"},
		"revocation_endpoint_auth_methods_supported":               []string{"client_secret_basic", "client_secret_post", "private_key_jwt", "none"},
		"revocation_endpoint_auth_signing_alg_values_supported":    []string{"EdDSA"},
		"introspection_endpoint_auth_methods_supported":            []string{"client_secret_basic", "client_secret_post", "private_key_jwt", "none"},
		"introspection_endpoint_auth_signing_alg_values_supported": []string{"EdDSA"},
		"response_modes_supported":                                 []string{"query"},
		"code_challenge_methods_supported":                         []string{"S256"},
		"id_token_signing_alg_values_supported":                    []string{"RS256"},
		"subject_types_supported":                                  []string{"public"},
		"claims_supported":                                         []string{"sub", "iss", "name", "email", "phone_number", "preferred_username"},
	}, nil
}

func (s *OIDCService) JWKS(ctx context.Context, clientID, applicationID string) (map[string]any, error) {
	keys, err := s.keys.Instance()
	if err != nil {
		return nil, err
	}
	return keys.JWKS()
}

func (s *OIDCService) JWKSByIssuer(ctx context.Context) (map[string]any, error) {
	instanceKeys, err := s.keys.Instance()
	if err != nil {
		return nil, err
	}
	return map[string]any{"keys": []jose.JSONWebKey{instanceKeys.PublicJWK()}}, nil
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
	codeValue, err := util.RandomToken(24)
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
	code, ok := loadAuthorizationCode(codeValue)
	if !ok {
		return nil, "", errors.New("code not found")
	}
	if code.ConsumedAt != nil || code.ExpiresAt.Before(time.Now()) {
		deleteAuthorizationCode(codeValue)
		return nil, "", errors.New("code is no longer valid")
	}
	if code.RedirectURI != redirectURI {
		return nil, "", errors.New("redirect_uri mismatch")
	}
	if strings.TrimSpace(code.CodeChallenge) != "" {
		if err := verifyCodeChallenge(code.CodeChallengeMethod, code.CodeChallenge, verifier); err != nil {
			return nil, "", err
		}
	}
	now := time.Now()
	code.ConsumedAt = &now
	storeAuthorizationCode(code)
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
		authTime := session.CreatedAt
		idToken, err = s.signIDToken(ctx, app.ID, user, app.ID, code.Scope, code.Nonce, &authTime, session.ID)
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
	keys, err := s.keys.Instance()
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
