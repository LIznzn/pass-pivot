package handler

import (
	"context"
	"net/http"
	"strings"

	"pass-pivot/internal/config"
	"pass-pivot/internal/model"
	authservice "pass-pivot/internal/server/auth/service"
	coreservice "pass-pivot/internal/server/core/service"
	sharedauthn "pass-pivot/internal/server/shared/authn"
	sharedhttp "pass-pivot/internal/server/shared/web"
)

type OIDCHandler struct {
	cfg      config.Config
	oidc     *authservice.OIDCService
	auth     oidcAuthClient
	platform oidcPlatformClient
}

type oidcAuthClient interface {
	IssueClientCredentialToken(ctx context.Context, clientID, clientSecret, scope string) (*sharedauthn.TokenPair, error)
	IssueClientCredentialTokenForApplication(ctx context.Context, app model.Application, scope string) (*sharedauthn.TokenPair, error)
	IssuePasswordGrantTokenForApplication(ctx context.Context, app model.Application, identifier, password, scope, ipAddress, userAgent string) (*sharedauthn.TokenPair, *model.User, *model.Session, error)
	RevokeToken(ctx context.Context, tokenValue, reason string) error
}

type oidcPlatformClient interface {
	GetLoginTarget(ctx context.Context, applicationID string) (*coreservice.LoginTarget, error)
}

func NewOIDCHandler(cfg config.Config, oidc *authservice.OIDCService, auth oidcAuthClient, platform oidcPlatformClient) *OIDCHandler {
	return &OIDCHandler{cfg: cfg, oidc: oidc, auth: auth, platform: platform}
}

func (h *OIDCHandler) Config() config.Config {
	return h.cfg
}

func (h *OIDCHandler) Service() *authservice.OIDCService {
	return h.oidc
}

func (h *OIDCHandler) ValidatePrivateKeyJWTClient(ctx context.Context, clientID, assertionType, assertion, audience string) (model.Application, error) {
	return h.oidc.ValidatePrivateKeyJWTClient(ctx, clientID, assertionType, assertion, audience)
}

func (h *OIDCHandler) ValidateClientAuthentication(ctx context.Context, clientID, clientSecret, clientAssertionType, clientAssertion, audience string) (model.Application, error) {
	app, err := h.oidc.ValidateClientAuthentication(ctx, clientID, clientSecret, clientAssertionType, clientAssertion, audience)
	return app, err
}

func (h *OIDCHandler) BuildNamedClientAssertion(ctx context.Context, applicationName, audience string) (string, string, error) {
	return h.oidc.BuildNamedClientAssertion(ctx, applicationName, audience)
}

func (h *OIDCHandler) Metadata(w http.ResponseWriter, r *http.Request) {
	result, err := h.oidc.MetadataByIssuer(r.Context())
	if err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedhttp.JSON(w, http.StatusOK, result)
}

func (h *OIDCHandler) JWKS(w http.ResponseWriter, r *http.Request) {
	keys, err := h.oidc.JWKSByIssuer(r.Context())
	if err != nil {
		sharedhttp.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	sharedhttp.JSON(w, http.StatusOK, keys)
}

func (h *OIDCHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	h.renderAuthorizeInteraction(w, r, "")
}

func (h *OIDCHandler) Token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, "invalid form body")
		return
	}
	clientID, clientSecret, _ := authservice.ParseBasicClientAuthorization(r.Header.Get("Authorization"))
	if clientID == "" {
		clientID = strings.TrimSpace(r.Form.Get("client_id"))
	}
	if clientSecret == "" {
		clientSecret = strings.TrimSpace(r.Form.Get("client_secret"))
	}
	body, err := h.callAuthnAPI(w, r, "/api/authn/v1/token/exchange", map[string]any{
		"grantType":           strings.TrimSpace(r.Form.Get("grant_type")),
		"clientId":            clientID,
		"clientSecret":        clientSecret,
		"clientAssertionType": strings.TrimSpace(r.Form.Get("client_assertion_type")),
		"clientAssertion":     strings.TrimSpace(r.Form.Get("client_assertion")),
		"code":                strings.TrimSpace(r.Form.Get("code")),
		"redirectUri":         strings.TrimSpace(r.Form.Get("redirect_uri")),
		"codeVerifier":        strings.TrimSpace(r.Form.Get("code_verifier")),
		"refreshToken":        strings.TrimSpace(r.Form.Get("refresh_token")),
		"username":            strings.TrimSpace(r.Form.Get("username")),
		"password":            strings.TrimSpace(r.Form.Get("password")),
		"scope":               strings.TrimSpace(r.Form.Get("scope")),
	})
	if err != nil {
		sharedhttp.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

func (h *OIDCHandler) UserInfo(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		sharedhttp.Error(w, http.StatusUnauthorized, "missing bearer token")
		return
	}
	body, err := h.callAuthnAPIWithHeaders(w, r, "/api/authn/v1/userinfo/query", map[string]any{}, map[string]string{
		"Authorization": auth,
	})
	if err != nil {
		sharedhttp.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

func (h *OIDCHandler) EndSession(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	postLogoutRedirectURI := strings.TrimSpace(r.Form.Get("post_logout_redirect_uri"))
	if postLogoutRedirectURI == "" {
		postLogoutRedirectURI = strings.TrimSpace(r.URL.Query().Get("post_logout_redirect_uri"))
	}
	accessToken := bearerTokenFromAuthorization(r.Header.Get("Authorization"))
	if accessToken == "" {
		accessToken = strings.TrimSpace(r.Form.Get("access_token"))
	}
	if _, err := h.callAuthnAPI(w, r, "/api/authn/v1/logout", map[string]any{
		"accessToken":  accessToken,
		"refreshToken": strings.TrimSpace(r.Form.Get("refresh_token")),
		"reason":       "oidc_end_session",
	}); err != nil {
		sharedhttp.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	if postLogoutRedirectURI != "" {
		http.Redirect(w, r, postLogoutRedirectURI, http.StatusFound)
		return
	}
	sharedhttp.JSON(w, http.StatusOK, map[string]any{"logout": true})
}

func bearerTokenFromAuthorization(value string) string {
	if !strings.HasPrefix(value, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(value, "Bearer "))
}

func requestIssuer(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwarded != "" {
		scheme = forwarded
	}
	host := r.Host
	if forwardedHost := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); forwardedHost != "" {
		host = forwardedHost
	}
	return scheme + "://" + host
}
