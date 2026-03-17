package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"pass-pivot/internal/config"
	authservice "pass-pivot/internal/server/auth/service"
	sharedweb "pass-pivot/internal/server/shared/web"
)

type OAuthHandler struct {
	cfg  config.Config
	oidc *authservice.OIDCService
}

func NewOAuthHandler(cfg config.Config, oidc *authservice.OIDCService) *OAuthHandler {
	return &OAuthHandler{cfg: cfg, oidc: oidc}
}

func (h *OAuthHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid form body")
		return
	}
	clientID, clientSecret, _ := authservice.ParseBasicClientAuthorization(r.Header.Get("Authorization"))
	if clientID == "" {
		clientID = strings.TrimSpace(r.Form.Get("client_id"))
	}
	if clientSecret == "" {
		clientSecret = strings.TrimSpace(r.Form.Get("client_secret"))
	}
	clientAssertionType := strings.TrimSpace(r.Form.Get("client_assertion_type"))
	clientAssertion := strings.TrimSpace(r.Form.Get("client_assertion"))
	body, err := h.callAuthnRevoke(r, map[string]any{
		"clientId":            clientID,
		"clientSecret":        clientSecret,
		"clientAssertionType": clientAssertionType,
		"clientAssertion":     clientAssertion,
		"token":               strings.TrimSpace(r.Form.Get("token")),
		"reason":              "oauth_revoke",
	})
	if err != nil {
		sharedweb.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

func (h *OAuthHandler) Introspect(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid form body")
		return
	}
	clientID, clientSecret, _ := authservice.ParseBasicClientAuthorization(r.Header.Get("Authorization"))
	if clientID == "" {
		clientID = strings.TrimSpace(r.Form.Get("client_id"))
	}
	if clientSecret == "" {
		clientSecret = strings.TrimSpace(r.Form.Get("client_secret"))
	}
	clientAssertionType := strings.TrimSpace(r.Form.Get("client_assertion_type"))
	clientAssertion := strings.TrimSpace(r.Form.Get("client_assertion"))
	if err := h.callAuthnValidateClient(r, clientID, clientSecret, clientAssertionType, clientAssertion, requestIssuer(r)+"/auth/introspect"); err != nil {
		sharedweb.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	result, err := h.callAuthnIntrospect(r, r.Form.Get("token"))
	if err != nil {
		sharedweb.Error(w, http.StatusBadGateway, err.Error())
		return
	}
	if active, _ := result["active"].(bool); active {
		subjectID, _ := result["sub"].(string)
		if subjectID != "" {
			matchedRoles, policies, err := h.callAuthzSubjectPolicyQuery(r, "user", subjectID)
			if err != nil {
				sharedweb.Error(w, http.StatusBadGateway, err.Error())
				return
			}
			result["matched_roles"] = matchedRoles
			result["policies"] = policies
		}
	}
	sharedweb.JSON(w, http.StatusOK, result)
}

func (h *OAuthHandler) callAuthnIntrospect(r *http.Request, token string) (map[string]any, error) {
	payload, err := json.Marshal(map[string]any{"token": strings.TrimSpace(token)})
	if err != nil {
		return nil, err
	}
	audience := strings.TrimRight(h.cfg.CoreURL, "/") + "/api/authn"
	clientID, assertion, err := h.oidc.BuildNamedClientAssertion(r.Context(), "authn-api", audience)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, strings.TrimRight(h.cfg.CoreURL, "/")+"/api/authn/v1/token/introspect", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-PPVT-Client-ID", clientID)
	req.Header.Set("X-PPVT-Client-Assertion-Type", "urn:ietf:params:oauth:client-assertion-type:jwt-bearer")
	req.Header.Set("X-PPVT-Client-Assertion", assertion)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, serviceErrorFromBody(body)
	}
	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (h *OAuthHandler) callAuthnValidateClient(r *http.Request, clientID, clientSecret, clientAssertionType, clientAssertion, audience string) error {
	_, err := h.callAuthnAPI(r, "/api/authn/v1/client/validate", map[string]any{
		"clientId":            clientID,
		"clientSecret":        clientSecret,
		"clientAssertionType": clientAssertionType,
		"clientAssertion":     clientAssertion,
		"audience":            audience,
	})
	return err
}

func (h *OAuthHandler) callAuthnRevoke(r *http.Request, payload any) ([]byte, error) {
	return h.callAuthnAPI(r, "/api/authn/v1/token/revoke", payload)
}

func (h *OAuthHandler) callAuthnAPI(r *http.Request, path string, payload any) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	audience := strings.TrimRight(h.cfg.CoreURL, "/") + "/api/authn"
	clientID, assertion, err := h.oidc.BuildNamedClientAssertion(r.Context(), "authn-api", audience)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, strings.TrimRight(h.cfg.CoreURL, "/")+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-PPVT-Client-ID", clientID)
	req.Header.Set("X-PPVT-Client-Assertion-Type", "urn:ietf:params:oauth:client-assertion-type:jwt-bearer")
	req.Header.Set("X-PPVT-Client-Assertion", assertion)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, serviceErrorFromBody(bodyBytes)
	}
	return bodyBytes, nil
}

func (h *OAuthHandler) callAuthzSubjectPolicyQuery(r *http.Request, subjectType, subjectID string) ([]string, []string, error) {
	payload, err := json.Marshal(map[string]any{
		"subjectType": subjectType,
		"subjectId":   subjectID,
	})
	if err != nil {
		return nil, nil, err
	}
	audience := strings.TrimRight(h.cfg.CoreURL, "/") + "/api/authz"
	clientID, assertion, err := h.oidc.BuildNamedClientAssertion(r.Context(), "authz-api", audience)
	if err != nil {
		return nil, nil, err
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, strings.TrimRight(h.cfg.CoreURL, "/")+"/api/authz/v1/policy/query", bytes.NewReader(payload))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-PPVT-Client-ID", clientID)
	req.Header.Set("X-PPVT-Client-Assertion-Type", "urn:ietf:params:oauth:client-assertion-type:jwt-bearer")
	req.Header.Set("X-PPVT-Client-Assertion", assertion)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, nil, serviceErrorFromBody(body)
	}
	var result struct {
		Roles    []string `json:"roles"`
		Policies []struct {
			Name string `json:"name"`
		} `json:"policies"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, nil, err
	}
	names := make([]string, 0, len(result.Policies))
	for _, item := range result.Policies {
		names = append(names, item.Name)
	}
	return result.Roles, names, nil
}

func serviceErrorFromBody(body []byte) error {
	var payload struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &payload) == nil && strings.TrimSpace(payload.Error) != "" {
		return oauthHandlerError(payload.Error)
	}
	text := strings.TrimSpace(string(body))
	if text == "" {
		text = "internal api request failed"
	}
	return oauthHandlerError(text)
}

type oauthHandlerError string

func (e oauthHandlerError) Error() string {
	return string(e)
}
