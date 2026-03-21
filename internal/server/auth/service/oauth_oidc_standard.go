package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"html"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"pass-pivot/internal/model"
	"pass-pivot/util"
)

type StandardAuthorizeRequest struct {
	SessionID           string
	ClientID            string
	ResponseType        string
	RedirectURI         string
	Scope               string
	State               string
	Nonce               string
	CodeChallenge       string
	CodeChallengeMethod string
	Prompt              string
}

func (s *OIDCService) ValidateAuthorizationRequest(ctx context.Context, in StandardAuthorizeRequest) (*model.Application, string, error) {
	app, err := s.loadAuthorizedApplication(ctx, in.ClientID, in.RedirectURI)
	if err != nil {
		return nil, "", err
	}
	if in.ResponseType != "code" {
		return nil, s.redirectWithOAuthError(in.RedirectURI, "unsupported_response_type", in.State, "only code is supported"), nil
	}
	if strings.TrimSpace(in.CodeChallenge) != "" {
		if !applicationSupportsAuthorizationCodePKCE(app) {
			return nil, s.redirectWithOAuthError(in.RedirectURI, "unauthorized_client", in.State, "pkce is not enabled for this client"), nil
		}
		if !strings.EqualFold(strings.TrimSpace(in.CodeChallengeMethod), "S256") {
			return nil, s.redirectWithOAuthError(in.RedirectURI, "invalid_request", in.State, "only S256 code_challenge_method is supported"), nil
		}
	} else if !applicationSupportsAuthorizationCode(app) {
		return nil, s.redirectWithOAuthError(in.RedirectURI, "unauthorized_client", in.State, "authorization_code grant is not enabled"), nil
	}
	return &app, "", nil
}

func (s *OIDCService) GetSession(ctx context.Context, sessionID string) (*model.Session, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, errors.New("session is required")
	}
	var session model.Session
	if err := s.db.WithContext(ctx).First(&session, "id = ?", sessionID).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *OIDCService) GetSessionUser(ctx context.Context, sessionID string) (*model.User, *model.Session, error) {
	session, err := s.GetSession(ctx, sessionID)
	if err != nil {
		return nil, nil, err
	}
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", session.UserID).Error; err != nil {
		return nil, nil, err
	}
	return &user, session, nil
}

func (s *OIDCService) BuildAuthorizationRedirect(ctx context.Context, in StandardAuthorizeRequest) (string, error) {
	app, redirectError, err := s.ValidateAuthorizationRequest(ctx, in)
	if err != nil {
		return "", err
	}
	if redirectError != "" {
		return redirectError, nil
	}
	if strings.TrimSpace(in.SessionID) == "" {
		if strings.EqualFold(strings.TrimSpace(in.Prompt), "none") {
			return s.redirectWithOAuthError(in.RedirectURI, "login_required", in.State, "interactive login is required"), nil
		}
		return s.redirectWithOAuthError(in.RedirectURI, "login_required", in.State, "session is not active"), nil
	}
	session, err := s.GetSession(ctx, in.SessionID)
	if err != nil {
		return s.redirectWithOAuthError(in.RedirectURI, "login_required", in.State, "session is not active"), nil
	}
	if session.State != "authenticated" {
		return s.redirectWithOAuthError(in.RedirectURI, "login_required", in.State, "session is not authenticated"), nil
	}
	codeValue, err := util.RandomToken(24)
	if err != nil {
		return "", err
	}
	code := &model.AuthorizationCode{
		SessionID:           session.ID,
		UserID:              session.UserID,
		ApplicationID:       app.ID,
		Code:                codeValue,
		RedirectURI:         in.RedirectURI,
		Scope:               strings.TrimSpace(in.Scope),
		Nonce:               strings.TrimSpace(in.Nonce),
		CodeChallenge:       strings.TrimSpace(in.CodeChallenge),
		CodeChallengeMethod: strings.TrimSpace(in.CodeChallengeMethod),
		ExpiresAt:           time.Now().Add(5 * time.Minute),
	}
	storeAuthorizationCode(*code)
	redirect, err := url.Parse(in.RedirectURI)
	if err != nil {
		return "", err
	}
	query := redirect.Query()
	query.Set("code", codeValue)
	if in.State != "" {
		query.Set("state", in.State)
	}
	redirect.RawQuery = query.Encode()
	return redirect.String(), nil
}

func (s *OIDCService) ExchangeRefreshToken(ctx context.Context, audience, clientID, clientSecret, clientAssertionType, clientAssertion, refreshTokenValue, scope string) ([]model.Token, string, error) {
	app, err := s.validateClientAuthentication(ctx, audience, clientID, clientSecret, clientAssertionType, clientAssertion)
	if err != nil {
		return nil, "", err
	}
	if !app.EnableRefreshToken {
		return nil, "", errors.New("refresh_token is not enabled for this application")
	}
	var refresh model.Token
	if err := s.db.WithContext(ctx).
		Where("token = ? AND type = ? AND application_id = ?", refreshTokenValue, "refresh_token", app.ID).
		First(&refresh).Error; err != nil {
		return nil, "", errors.New("invalid refresh token")
	}
	if refresh.RevokedAt != nil || refresh.ExpiresAt.Before(time.Now()) {
		return nil, "", errors.New("invalid refresh token")
	}
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", refresh.UserID).Error; err != nil {
		return nil, "", err
	}
	if user.CurrentUKID != refresh.UKID {
		return nil, "", errors.New("refresh token is no longer valid")
	}
	var session model.Session
	if err := s.db.WithContext(ctx).First(&session, "id = ?", refresh.SessionID).Error; err != nil {
		return nil, "", err
	}
	if session.State != "authenticated" {
		return nil, "", errors.New("session is not active")
	}
	now := time.Now()
	if err := s.db.WithContext(ctx).Model(&refresh).Updates(map[string]any{
		"revoked_at":      &now,
		"revocation_note": "refresh_token_rotated",
	}).Error; err != nil {
		return nil, "", err
	}
	tokenScope := strings.TrimSpace(scope)
	if tokenScope == "" {
		tokenScope = refresh.Scope
	}
	tokens, err := s.auth.IssueTokensForApplication(ctx, user, session, app.ID, tokenScope)
	if err != nil {
		return nil, "", err
	}
	var idToken string
	if applicationReturnsIDToken(app.TokenType) {
		authTime := session.CreatedAt
		idToken, err = s.signIDToken(ctx, app.ID, user, app.ID, tokenScope, "", &authTime, session.ID)
		if err != nil {
			return nil, "", err
		}
	}
	return tokens, idToken, nil
}

func (s *OIDCService) IntrospectToken(ctx context.Context, audience, clientID, clientSecret, clientAssertionType, clientAssertion, tokenValue string) (map[string]any, error) {
	if _, err := s.validateClientAuthentication(ctx, audience, clientID, clientSecret, clientAssertionType, clientAssertion); err != nil {
		return nil, err
	}
	var token model.Token
	if err := s.db.WithContext(ctx).Where("token = ?", tokenValue).First(&token).Error; err != nil {
		return map[string]any{"active": false}, nil
	}
	active := token.RevokedAt == nil && token.ExpiresAt.After(time.Now())
	if !active {
		return map[string]any{"active": false}, nil
	}
	result := map[string]any{
		"active":     true,
		"scope":      token.Scope,
		"client_id":  token.ApplicationID,
		"token_type": token.Type,
		"exp":        token.ExpiresAt.Unix(),
		"sub":        token.UserID,
	}
	if token.SessionID != "" {
		result["sid"] = token.SessionID
	}
	return result, nil
}

func (s *OIDCService) EndSession(ctx context.Context, sessionID, postLogoutRedirectURI, state string) (string, error) {
	if sessionID != "" {
		var session model.Session
		if err := s.db.WithContext(ctx).First(&session, "id = ?", sessionID).Error; err == nil {
			now := time.Now()
			_ = s.db.WithContext(ctx).Model(&model.Token{}).
				Where("session_id = ? AND revoked_at IS NULL", session.ID).
				Updates(map[string]any{"revoked_at": &now, "revocation_note": "oidc_end_session"}).Error
			_ = s.db.WithContext(ctx).Where("id = ?", session.ID).Delete(&model.Session{}).Error
		}
	}
	if strings.TrimSpace(postLogoutRedirectURI) == "" {
		return "", nil
	}
	redirect, err := url.Parse(postLogoutRedirectURI)
	if err != nil {
		return "", err
	}
	if state != "" {
		query := redirect.Query()
		query.Set("state", state)
		redirect.RawQuery = query.Encode()
	}
	return redirect.String(), nil
}

func (s *OIDCService) ValidatePostLogoutRedirectClient(ctx context.Context, clientID, postLogoutRedirectURI string) error {
	app, err := s.loadAuthorizedApplication(ctx, clientID, postLogoutRedirectURI)
	if err != nil {
		return err
	}
	return validatePostLogoutRedirectURI(app, postLogoutRedirectURI)
}

func (s *OIDCService) ValidatePrivateKeyJWTClient(ctx context.Context, clientID, assertionType, assertion, audience string) (model.Application, error) {
	return s.validateClientAuthentication(ctx, audience, clientID, "", assertionType, assertion)
}

func (s *OIDCService) ValidateClientAuthentication(ctx context.Context, clientID, clientSecret, clientAssertionType, clientAssertion, audience string) (model.Application, error) {
	return s.validateClientAuthentication(ctx, audience, clientID, clientSecret, clientAssertionType, clientAssertion)
}

func (s *OIDCService) BuildNamedClientAssertion(ctx context.Context, applicationName, audience string) (string, string, error) {
	var app model.Application
	if err := s.db.WithContext(ctx).Where("name = ?", applicationName).First(&app).Error; err != nil {
		return "", "", errors.New("internal client not found")
	}
	if app.ClientAuthenticationType != "private_key_jwt" {
		return "", "", errors.New("internal client is not configured for private_key_jwt")
	}
	keys, err := s.keys.LoadInternalClientSigningKey(app.ID, app.PublicKey)
	if err != nil {
		return "", "", err
	}
	jti, err := util.RandomToken(18)
	if err != nil {
		return "", "", err
	}
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Issuer:    app.ID,
		Subject:   app.ID,
		Audience:  jwt.ClaimStrings{audience},
		ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
		IssuedAt:  jwt.NewNumericDate(now),
		ID:        jti,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	token.Header["kid"] = keys.KeyID
	assertion, err := token.SignedString(keys.SigningKey)
	if err != nil {
		return "", "", err
	}
	return app.ID, assertion, nil
}

func (s *OIDCService) validateClientAuthentication(ctx context.Context, audience, clientID, clientSecret, clientAssertionType, clientAssertion string) (model.Application, error) {
	var app model.Application
	if err := s.db.WithContext(ctx).Where("id = ?", clientID).First(&app).Error; err != nil {
		return app, errors.New("invalid client")
	}
	switch app.ClientAuthenticationType {
	case "none":
		return app, nil
	case "client_secret_basic", "client_secret_post":
		if app.ClientSecretHash == "" || clientSecret == "" || !util.CheckSecret(app.ClientSecretHash, clientSecret) {
			return app, errors.New("invalid client")
		}
		return app, nil
	case "private_key_jwt":
		return s.validatePrivateKeyJWTAssertion(app, clientAssertionType, clientAssertion, audience)
	default:
		return app, errors.New("unsupported client authentication method")
	}
}

func (s *OIDCService) validatePrivateKeyJWTAssertion(app model.Application, assertionType, assertion, audience string) (model.Application, error) {
	if assertionType != "urn:ietf:params:oauth:client-assertion-type:jwt-bearer" {
		return app, errors.New("invalid client_assertion_type")
	}
	if strings.TrimSpace(assertion) == "" {
		return app, errors.New("client_assertion is required")
	}
	if strings.TrimSpace(app.PublicKey) == "" {
		return app, errors.New("client public key is not configured")
	}
	keys, err := s.keys.LoadClientVerificationKey(app.PublicKey)
	if err != nil {
		return app, err
	}
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(assertion, claims, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodEdDSA.Alg() {
			return nil, errors.New("unsupported client assertion alg")
		}
		return keys.PublicKey, nil
	})
	if err != nil || !token.Valid {
		return app, errors.New("invalid client assertion")
	}
	if claims.Issuer != app.ID || claims.Subject != app.ID {
		return app, errors.New("invalid client assertion subject")
	}
	var audienceMatched bool
	for _, item := range claims.Audience {
		if item == audience {
			audienceMatched = true
			break
		}
	}
	if !audienceMatched {
		return app, errors.New("invalid client assertion audience")
	}
	if claims.ExpiresAt == nil || claims.ExpiresAt.Time.Before(time.Now()) {
		return app, errors.New("client assertion expired")
	}
	return app, nil
}

func (s *OIDCService) loadAuthorizedApplication(ctx context.Context, clientID, redirectURI string) (model.Application, error) {
	var app model.Application
	if err := s.db.WithContext(ctx).Where("id = ?", clientID).First(&app).Error; err != nil {
		return app, errors.New("invalid client_id")
	}
	if !redirectURIAllowed(app.RedirectURIs, redirectURI) {
		return app, errors.New("invalid redirect_uri")
	}
	return app, nil
}

func (s *OIDCService) redirectWithOAuthError(redirectURI, code, state, description string) string {
	redirect, err := url.Parse(redirectURI)
	if err != nil {
		return redirectURI
	}
	query := redirect.Query()
	query.Set("error", code)
	if description != "" {
		query.Set("error_description", description)
	}
	if state != "" {
		query.Set("state", state)
	}
	redirect.RawQuery = query.Encode()
	return redirect.String()
}

func redirectURIAllowed(allowedRaw, candidate string) bool {
	if strings.TrimSpace(candidate) == "" {
		return false
	}
	for _, item := range splitRedirectURIs(allowedRaw) {
		if item == candidate {
			return true
		}
	}
	return false
}

func splitRedirectURIs(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == ' '
	})
	items := make([]string, 0, len(fields))
	for _, item := range fields {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			items = append(items, trimmed)
		}
	}
	return items
}

func ParseBasicClientAuthorization(value string) (string, string, bool) {
	if !strings.HasPrefix(value, "Basic ") {
		return "", "", false
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, "Basic "))
	if err != nil {
		return "", "", false
	}
	parts := strings.SplitN(string(raw), ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func BuildStandardTokenResponse(tokens []model.Token, idToken string) map[string]any {
	result := map[string]any{
		"token_type": "Bearer",
	}
	for i := range tokens {
		token := &tokens[i]
		switch token.Type {
		case "access_token":
			result["access_token"] = token.Token
			result["expires_in"] = tokenExpirySeconds(token)
			if token.Scope != "" {
				result["scope"] = token.Scope
			}
		case "refresh_token":
			result["refresh_token"] = token.Token
		}
	}
	if idToken != "" {
		result["id_token"] = idToken
	}
	return result
}

func tokenExpirySeconds(token *model.Token) int64 {
	if token == nil {
		return 0
	}
	seconds := int64(time.Until(token.ExpiresAt).Seconds())
	if seconds < 0 {
		return 0
	}
	return seconds
}

func validatePostLogoutRedirectURI(app model.Application, redirectURI string) error {
	if strings.TrimSpace(redirectURI) == "" {
		return nil
	}
	if !redirectURIAllowed(app.RedirectURIs, redirectURI) {
		return errors.New("invalid post_logout_redirect_uri")
	}
	return nil
}

func BuildOAuthErrorPage(message string) []byte {
	return []byte(fmt.Sprintf("<html><body><h1>OAuth Error</h1><p>%s</p></body></html>", html.EscapeString(message)))
}
