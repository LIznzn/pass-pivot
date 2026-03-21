package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	authservice "pass-pivot/internal/server/auth/service"
	coreservice "pass-pivot/internal/server/core/service"
	sharedhandler "pass-pivot/internal/server/shared/handler"
)

type authorizeUIBootstrap struct {
	Stage              string                    `json:"stage"`
	Title              string                    `json:"title"`
	Error              string                    `json:"error,omitempty"`
	AuthorizeReturnURL string                    `json:"authorizeReturnUrl"`
	Target             *coreservice.LoginTarget  `json:"target"`
	ApplicationID      string                    `json:"applicationId"`
	LoginAction        string                    `json:"loginAction"`
	ConfirmAction      string                    `json:"confirmAction"`
	MFAAction          string                    `json:"mfaAction"`
	SecondFactorMethod string                    `json:"secondFactorMethod,omitempty"`
	MFAOptions         []authorizeUIMethodOption `json:"mfaOptions"`
	API                authorizeUIAPIConfig      `json:"api"`
}

type authorizeUIMethodOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type authorizeUIAPIConfig struct {
	WebAuthnLoginBegin string `json:"webauthnLoginBegin"`
	WebAuthnLoginEnd   string `json:"webauthnLoginEnd"`
	SessionU2FBegin    string `json:"sessionU2fBegin"`
	SessionU2FFinish   string `json:"sessionU2fFinish"`
	MFAChallenge       string `json:"mfaChallenge"`
}

type authnAPIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (h *OIDCHandler) callAuthnAPI(w http.ResponseWriter, r *http.Request, path string, payload any) ([]byte, error) {
	return h.callAuthnAPIWithHeaders(w, r, path, payload, nil)
}

func (h *OIDCHandler) callAuthnAPIWithHeaders(w http.ResponseWriter, r *http.Request, path string, payload any, extraHeaders map[string]string) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	audience := strings.TrimRight(h.cfg.CoreURL, "/") + "/api/authn"
	clientID, assertion, err := h.BuildNamedClientAssertion(r.Context(), "authn-api", audience)
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
	req.Header.Set("X-PPVT-Original-Remote-Addr", r.RemoteAddr)
	req.Header.Set("X-PPVT-Original-User-Agent", r.UserAgent())
	if cookie := r.Header.Get("Cookie"); cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	for key, value := range extraHeaders {
		if strings.TrimSpace(value) != "" {
			req.Header.Set(key, value)
		}
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	for _, value := range resp.Header.Values("Set-Cookie") {
		w.Header().Add("Set-Cookie", value)
	}
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		var apiErr authnAPIError
		if json.Unmarshal(responseBody, &apiErr) == nil {
			if strings.TrimSpace(apiErr.Message) != "" {
				return nil, errAuthorizeAPI(apiErr.Message)
			}
		}
		return nil, errAuthorizeAPI(strings.TrimSpace(string(responseBody)))
	}
	return responseBody, nil
}

type authorizeAPIError struct {
	message string
}

func (e authorizeAPIError) Error() string {
	return e.message
}

func errAuthorizeAPI(message string) error {
	if strings.TrimSpace(message) == "" {
		message = "authorize api request failed"
	}
	return authorizeAPIError{message: message}
}

func (h *OIDCHandler) AuthorizeSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.writeAuthorizeErrorPage(w, http.StatusBadRequest, "invalid form body")
		return
	}
	switch strings.TrimSpace(r.Form.Get("interaction")) {
	case "login":
		h.AuthorizeLogin(w, r)
	case "confirm":
		h.AuthorizeConfirm(w, r)
	case "mfa":
		h.AuthorizeMFA(w, r)
	default:
		h.writeAuthorizeErrorPage(w, http.StatusBadRequest, "invalid authorize interaction")
	}
}

func (h *OIDCHandler) AuthorizeChallenge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeAuthorizeErrorPage(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := r.ParseForm(); err != nil && !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		h.writeAuthorizeErrorPage(w, http.StatusBadRequest, "invalid request body")
		return
	}
	var payload struct {
		Method string `json:"method"`
	}
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			h.writeAuthorizeErrorPage(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
	} else {
		payload.Method = strings.TrimSpace(r.Form.Get("method"))
	}
	body, err := h.callAuthnAPI(w, r, "/api/authn/v1/session/mfa_challenge/create", map[string]any{
		"method": strings.TrimSpace(payload.Method),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

func (h *OIDCHandler) AuthorizeWebAuthnLoginBegin(w http.ResponseWriter, r *http.Request) {
	body, err := h.callAuthnAPI(w, r, "/api/authn/v1/webauthn/login/begin", map[string]any{
		"identifier": strings.TrimSpace(r.FormValue("identifier")),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

func (h *OIDCHandler) AuthorizeWebAuthnLoginFinish(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ChallengeID   string          `json:"challengeId"`
		Response      json.RawMessage `json:"response"`
		ApplicationID string          `json:"applicationId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	body, err := h.callAuthnAPI(w, r, "/api/authn/v1/webauthn/login/finish", map[string]any{
		"challengeId":   payload.ChallengeID,
		"response":      payload.Response,
		"applicationId": payload.ApplicationID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

func (h *OIDCHandler) AuthorizeSessionU2FBegin(w http.ResponseWriter, r *http.Request) {
	body, err := h.callAuthnAPI(w, r, "/api/authn/v1/session/u2f/begin", map[string]any{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

func (h *OIDCHandler) AuthorizeSessionU2FFinish(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ChallengeID string          `json:"challengeId"`
		Response    json.RawMessage `json:"response"`
		TrustDevice bool            `json:"trustDevice"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	body, err := h.callAuthnAPI(w, r, "/api/authn/v1/session/u2f/finish", map[string]any{
		"challengeId": payload.ChallengeID,
		"response":    payload.Response,
		"trustDevice": payload.TrustDevice,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

func (h *OIDCHandler) AuthorizeLogin(w http.ResponseWriter, r *http.Request) {
	response, err := h.queryAuthorizeInteraction(w, r)
	if err != nil {
		h.writeAuthorizeErrorPage(w, http.StatusBadGateway, err.Error())
		return
	}
	if response.Action == "redirect" && strings.TrimSpace(response.RedirectTarget) != "" {
		http.Redirect(w, r, response.RedirectTarget, http.StatusFound)
		return
	}
	if response.Target == nil {
		h.writeAuthorizeErrorPage(w, http.StatusBadRequest, "login target is not available")
		return
	}
	if err := r.ParseForm(); err != nil {
		h.writeAuthorizeErrorPage(w, http.StatusBadRequest, "invalid form body")
		return
	}

	_, err = h.callAuthnAPI(w, r, "/api/authn/v1/session/create", map[string]any{
		"organizationId": response.Target.OrganizationID,
		"applicationId":  response.Target.ApplicationID,
		"identifier":     strings.TrimSpace(r.Form.Get("identifier")),
		"secret":         r.Form.Get("secret"),
	})
	if err != nil {
		h.renderAuthorizeInteraction(w, r, err.Error())
		return
	}
	http.Redirect(w, r, authorizeURL(r).String(), http.StatusFound)
}

func (h *OIDCHandler) AuthorizeConfirm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.writeAuthorizeErrorPage(w, http.StatusBadRequest, "invalid form body")
		return
	}
	_, err := h.callAuthnAPI(w, r, "/api/authn/v1/session/confirm", map[string]any{
		"accept": strings.EqualFold(r.Form.Get("accept"), "true"),
	})
	if err != nil {
		h.renderAuthorizeInteraction(w, r, err.Error())
		return
	}
	http.Redirect(w, r, authorizeURL(r).String(), http.StatusFound)
}

func (h *OIDCHandler) AuthorizeMFA(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.writeAuthorizeErrorPage(w, http.StatusBadRequest, "invalid form body")
		return
	}
	_, err := h.callAuthnAPI(w, r, "/api/authn/v1/session/verify_mfa", map[string]any{
		"method":      strings.TrimSpace(r.Form.Get("method")),
		"code":        strings.TrimSpace(r.Form.Get("code")),
		"trustDevice": strings.EqualFold(r.Form.Get("trustDevice"), "true"),
	})
	if err != nil {
		h.renderAuthorizeInteraction(w, r, err.Error())
		return
	}
	http.Redirect(w, r, authorizeURL(r).String(), http.StatusFound)
}

func (h *OIDCHandler) renderAuthorizeInteraction(w http.ResponseWriter, r *http.Request, bannerError string) {
	response, err := h.queryAuthorizeInteraction(w, r)
	if err != nil {
		h.writeAuthorizeErrorPage(w, http.StatusBadGateway, err.Error())
		return
	}
	if response.Action == "redirect" {
		http.Redirect(w, r, response.RedirectTarget, http.StatusFound)
		return
	}
	if response.Target == nil {
		h.writeAuthorizeErrorPage(w, http.StatusBadRequest, "login target is not available")
		return
	}
	h.writeAuthorizeApp(w, http.StatusOK, buildAuthorizeBootstrap(r, response.Target, response.Stage, response.SecondFactorMethod, bannerError))
}

func (h *OIDCHandler) queryAuthorizeInteraction(w http.ResponseWriter, r *http.Request) (*authorizeInteractionResponse, error) {
	in := standardAuthorizeRequestFromHTTP(r)
	in.SessionID = strings.TrimSpace(r.URL.Query().Get("ppvt_session_id"))
	if in.SessionID == "" {
		in.SessionID = sharedhandler.ReadPortalSessionCookie(r)
	}
	body, err := h.callAuthnAPI(w, r, "/api/authn/v1/authorize/interaction/query", map[string]any{
		"sessionId":           in.SessionID,
		"clientId":            in.ClientID,
		"responseType":        in.ResponseType,
		"redirectUri":         in.RedirectURI,
		"scope":               in.Scope,
		"state":               in.State,
		"nonce":               in.Nonce,
		"codeChallenge":       in.CodeChallenge,
		"codeChallengeMethod": in.CodeChallengeMethod,
		"prompt":              in.Prompt,
	})
	if err != nil {
		return nil, err
	}
	var response authorizeInteractionResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}
	if strings.TrimSpace(response.Stage) == "" {
		response.Stage = "login"
	}
	return &response, nil
}

func (h *OIDCHandler) writeAuthorizeApp(w http.ResponseWriter, status int, bootstrap authorizeUIBootstrap) {
	body, err := buildAuthorizeAppShell(bootstrap)
	if err != nil {
		h.writeAuthorizeErrorPage(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func (h *OIDCHandler) writeAuthorizeErrorPage(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(authservice.BuildOAuthErrorPage(message))
}

func standardAuthorizeRequestFromHTTP(r *http.Request) authservice.StandardAuthorizeRequest {
	query := r.URL.Query()
	return authservice.StandardAuthorizeRequest{
		ClientID:            strings.TrimSpace(query.Get("client_id")),
		ResponseType:        strings.TrimSpace(query.Get("response_type")),
		RedirectURI:         strings.TrimSpace(query.Get("redirect_uri")),
		Scope:               strings.TrimSpace(query.Get("scope")),
		State:               strings.TrimSpace(query.Get("state")),
		Nonce:               strings.TrimSpace(query.Get("nonce")),
		CodeChallenge:       strings.TrimSpace(query.Get("code_challenge")),
		CodeChallengeMethod: strings.TrimSpace(query.Get("code_challenge_method")),
		Prompt:              strings.TrimSpace(query.Get("prompt")),
	}
}

func authorizeURL(r *http.Request) *url.URL {
	query := r.URL.Query()
	u := &url.URL{Path: "/auth/authorize"}
	u.RawQuery = query.Encode()
	return u
}

func redirectErrorOrDefault(redirectError, redirectURI, state string) string {
	if strings.TrimSpace(redirectError) != "" {
		return redirectError
	}
	redirect, err := url.Parse(redirectURI)
	if err != nil {
		return redirectURI
	}
	query := redirect.Query()
	query.Set("error", "login_required")
	if state != "" {
		query.Set("state", state)
	}
	redirect.RawQuery = query.Encode()
	return redirect.String()
}

func buildAuthorizeBootstrap(r *http.Request, target *coreservice.LoginTarget, stage, secondFactorMethod, bannerError string) authorizeUIBootstrap {
	title := "登录"
	switch stage {
	case "confirmation":
		title = "二次确认"
	case "mfa":
		title = "多因素认证"
	}
	return authorizeUIBootstrap{
		Stage:              stage,
		Title:              title,
		Error:              bannerError,
		AuthorizeReturnURL: authorizeURL(r).String(),
		Target:             target,
		ApplicationID:      target.ApplicationID,
		LoginAction:        "/auth/headless/login?" + r.URL.RawQuery,
		ConfirmAction:      "/auth/headless/confirm?" + r.URL.RawQuery,
		MFAAction:          "/auth/headless/mfa?" + r.URL.RawQuery,
		SecondFactorMethod: strings.TrimSpace(secondFactorMethod),
		MFAOptions: []authorizeUIMethodOption{
			{Value: "totp", Label: "身份验证器（TOTP）"},
			{Value: "email_code", Label: "邮箱验证码"},
			{Value: "sms_code", Label: "手机验证码"},
			{Value: "recovery_code", Label: "备用验证码"},
			{Value: "u2f", Label: "安全密钥"},
		},
		API: authorizeUIAPIConfig{
			WebAuthnLoginBegin: "/auth/headless/login/webauthn/begin?" + r.URL.RawQuery,
			WebAuthnLoginEnd:   "/auth/headless/login/webauthn/finish?" + r.URL.RawQuery,
			SessionU2FBegin:    "/auth/headless/mfa/u2f/begin?" + r.URL.RawQuery,
			SessionU2FFinish:   "/auth/headless/mfa/u2f/finish?" + r.URL.RawQuery,
			MFAChallenge:       "/auth/headless/mfa/challenge/generator?" + r.URL.RawQuery,
		},
	}
}

func buildAuthorizeAppShell(bootstrap authorizeUIBootstrap) ([]byte, error) {
	payload, err := json.Marshal(bootstrap)
	if err != nil {
		return nil, err
	}
	html := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>PPVT ` + bootstrap.Title + `</title>
  <link rel="stylesheet" href="/auth/authorize/app.css" />
</head>
<body>
  <div id="app"></div>
  <script>window.__PPVT_OAUTH_BOOTSTRAP__ = ` + string(payload) + `;</script>
  <script type="module" src="/auth/authorize/app.js"></script>
</body>
</html>`
	return []byte(html), nil
}
