package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"gorm.io/gorm"

	"pass-pivot/internal/config"
	"pass-pivot/internal/model"
	sharedauthn "pass-pivot/internal/server/shared/authn"
	"pass-pivot/util"
)

type FederationService struct {
	db    *gorm.DB
	cfg   config.Config
	audit *AuditService
	auth  federationAuthService
}

type federationAuthService interface {
	IssueTokenPair(ctx context.Context, user model.User, session model.Session, scope string) (*sharedauthn.TokenPair, error)
}

func NewFederationService(db *gorm.DB, cfg config.Config, audit *AuditService, auth federationAuthService) *FederationService {
	return &FederationService{db: db, cfg: cfg, audit: audit, auth: auth}
}

type StartFederationResult struct {
	AuthURL string `json:"authUrl"`
	State   string `json:"state"`
}

type FederationCallbackResult struct {
	Session model.Session `json:"session"`
	Tokens  []model.Token `json:"tokens"`
}

func (s *FederationService) StartLogin(ctx context.Context, providerID, applicationID, redirectURI string) (*StartFederationResult, error) {
	var provider model.ExternalIDP
	if err := s.db.WithContext(ctx).First(&provider, "id = ?", providerID).Error; err != nil {
		return nil, err
	}
	state, err := util.RandomToken(24)
	if err != nil {
		return nil, err
	}
	nonce, err := util.RandomToken(24)
	if err != nil {
		return nil, err
	}
	verifier, err := util.RandomToken(32)
	if err != nil {
		return nil, err
	}
	authState := model.ExternalAuthState{
		BaseModel: model.BaseModel{
			ID: uuid.NewString(),
		},
		OrganizationID: provider.OrganizationID,
		ProviderID:     provider.ID,
		State:          state,
		Nonce:          nonce,
		RedirectURI:    redirectURI,
		CodeVerifier:   verifier,
		ExpiresAt:      time.Now().Add(10 * time.Minute),
	}
	authState.CreatedAt = time.Now()
	authState.UpdatedAt = authState.CreatedAt
	storeExternalAuthState(authState)
	oidcProvider, err := oidc.NewProvider(ctx, provider.Issuer)
	if err != nil {
		return nil, err
	}
	oauthConfig := oauth2.Config{
		ClientID:     provider.ClientID,
		ClientSecret: provider.ClientSecret,
		RedirectURL:  redirectURI,
		Endpoint:     oidcProvider.Endpoint(),
		Scopes:       federationScopes(provider.Scopes),
	}
	authURL := oauthConfig.AuthCodeURL(state, oidc.Nonce(nonce))
	return &StartFederationResult{AuthURL: authURL, State: state}, nil
}

func (s *FederationService) CompleteLogin(ctx context.Context, stateValue, code, applicationID string) (*FederationCallbackResult, error) {
	authState, ok := loadExternalAuthState(stateValue)
	if !ok {
		return nil, errors.New("federation state not found")
	}
	if authState.ExpiresAt.Before(time.Now()) {
		deleteExternalAuthState(stateValue)
		return nil, errors.New("federation state expired")
	}
	var provider model.ExternalIDP
	if err := s.db.WithContext(ctx).First(&provider, "id = ?", authState.ProviderID).Error; err != nil {
		return nil, err
	}
	oidcProvider, err := oidc.NewProvider(ctx, provider.Issuer)
	if err != nil {
		return nil, err
	}
	oauthConfig := oauth2.Config{
		ClientID:     provider.ClientID,
		ClientSecret: provider.ClientSecret,
		RedirectURL:  authState.RedirectURI,
		Endpoint:     oidcProvider.Endpoint(),
		Scopes:       federationScopes(provider.Scopes),
	}
	token, err := oauthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, errors.New("missing id_token from provider")
	}
	verifier := oidcProvider.Verifier(&oidc.Config{ClientID: provider.ClientID})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, err
	}
	var claims struct {
		Subject string `json:"sub"`
		Email   string `json:"email"`
		Phone   string `json:"phone_number"`
		Name    string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, err
	}
	if claims.Subject == "" {
		return nil, errors.New("missing subject from provider")
	}
	var binding model.ExternalIdentityBinding
	if err := s.db.WithContext(ctx).
		Where("organization_id = ? AND issuer = ? AND subject = ?", provider.OrganizationID, provider.Issuer, claims.Subject).
		First(&binding).Error; err != nil {
		return nil, errors.New("external identity is not bound to an existing user")
	}
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", binding.UserID).Error; err != nil {
		return nil, err
	}
	session := model.Session{
		OrganizationID: provider.OrganizationID,
		UserID:         user.ID,
		ApplicationID:  applicationID,
		State:          "authenticated",
		RiskLevel:      "medium",
	}
	if err := s.db.WithContext(ctx).Create(&session).Error; err != nil {
		return nil, err
	}
	pair, err := s.auth.IssueTokenPair(ctx, user, session, "openid profile email phone")
	if err != nil {
		return nil, err
	}
	_ = s.audit.Record(ctx, AuditEvent{
		OrganizationID: provider.OrganizationID,
		ApplicationID:  applicationID,
		ActorType:      "user",
		ActorID:        user.ID,
		EventType:      "auth.federation.succeeded",
		Result:         "success",
		TargetType:     "session",
		TargetID:       session.ID,
		Detail: map[string]any{
			"providerId": provider.ID,
			"issuer":     provider.Issuer,
		},
	})
	deleteExternalAuthState(stateValue)
	return &FederationCallbackResult{
		Session: session,
		Tokens:  sharedauthn.CompactTokens(pair),
	}, nil
}

func federationScopes(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{oidc.ScopeOpenID, "profile", "email", "phone"}
	}
	parts := strings.Fields(raw)
	scopes := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			scopes = append(scopes, part)
		}
	}
	return scopes
}
