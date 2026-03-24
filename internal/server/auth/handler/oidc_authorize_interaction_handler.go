package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	authservice "pass-pivot/internal/server/auth/service"
	coreservice "pass-pivot/internal/server/core/service"
	sharedauthn "pass-pivot/internal/server/shared/authn"
	sharedhandler "pass-pivot/internal/server/shared/handler"
)

const loginChallengeQueryKey = "ppvt_login_challenge"

type authorizeUIMethodOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
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
		"sessionId": resolveLoginSessionRef(r),
		"method":    strings.TrimSpace(payload.Method),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

func (h *OIDCHandler) AuthorizeWebAuthnLoginBegin(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Identifier    string `json:"identifier"`
		ApplicationID string `json:"applicationId"`
	}
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
	} else {
		payload.Identifier = strings.TrimSpace(r.FormValue("identifier"))
		payload.ApplicationID = strings.TrimSpace(r.FormValue("applicationId"))
	}
	body, err := h.callAuthnAPI(w, r, "/api/authn/v1/webauthn/login/begin", map[string]any{
		"identifier":    strings.TrimSpace(payload.Identifier),
		"applicationId": strings.TrimSpace(payload.ApplicationID),
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

func (h *OIDCHandler) AuthorizeCaptchaRefresh(w http.ResponseWriter, r *http.Request) {
	var organizationID string
	if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("type")), "device_code") {
		response, err := h.queryDeviceCodeInteraction(w, r, strings.TrimSpace(r.URL.Query().Get("user_code")))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		if response.Target != nil {
			organizationID = strings.TrimSpace(response.Target.OrganizationID)
		}
	} else {
		response, err := h.queryAuthorizeInteraction(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		if response.Target != nil {
			organizationID = strings.TrimSpace(response.Target.OrganizationID)
		}
	}
	if organizationID == "" {
		http.Error(w, "login target is not available", http.StatusBadRequest)
		return
	}
	captcha, err := h.oidc.BuildAuthorizeCaptchaChallengeBootstrap(r.Context(), organizationID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if captcha == nil {
		http.Error(w, "captcha is not enabled", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(captcha)
}

func (h *OIDCHandler) AuthorizeSessionU2FBegin(w http.ResponseWriter, r *http.Request) {
	body, err := h.callAuthnAPI(w, r, "/api/authn/v1/session/u2f/begin", map[string]any{
		"sessionId": resolveLoginSessionRef(r),
	})
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
	h.writeAuthorizeApp(w, http.StatusOK, authorizePageTitle(response.Stage), bannerError)
}

func (h *OIDCHandler) queryAuthorizeInteraction(w http.ResponseWriter, r *http.Request) (*authorizeInteractionResponse, error) {
	in := standardAuthorizeRequestFromHTTP(r)
	in.SessionID = strings.TrimSpace(r.URL.Query().Get("ppvt_session_id"))
	if in.SessionID == "" {
		in.SessionID = resolveLoginSessionRef(r)
	}
	body, err := h.callAuthnAPI(w, r, "/api/authn/v1/authorize/interaction/query", map[string]any{
		"sessionId":            in.SessionID,
		"clientId":             in.ClientID,
		"responseType":         in.ResponseType,
		"redirectUri":          in.RedirectURI,
		"scope":                in.Scope,
		"state":                in.State,
		"nonce":                in.Nonce,
		"codeChallenge":        in.CodeChallenge,
		"codeChallengeMethod":  in.CodeChallengeMethod,
		"prompt":               in.Prompt,
		"skipAccountSelection": strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("ppvt_skip_account")), "1"),
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

func (h *OIDCHandler) writeAuthorizeApp(w http.ResponseWriter, status int, title string, _ string) {
	body, err := buildAuthorizeAppShell(title)
	if err != nil {
		h.writeAuthorizeErrorPage(w, http.StatusInternalServerError, err.Error())
		return
	}
	setAuthorizePageNoStore(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func (h *OIDCHandler) writeAuthorizeErrorPage(w http.ResponseWriter, status int, message string) {
	setAuthorizePageNoStore(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(authservice.BuildOAuthErrorPage(message))
}

func setAuthorizePageNoStore(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
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
	query.Del(loginChallengeQueryKey)
	u := &url.URL{Path: "/auth/authorize"}
	u.RawQuery = query.Encode()
	return u
}

func buildAuthorizeReturnURL(r *http.Request) string {
	return authorizeURL(r).String()
}

func resolveLoginSessionRef(r *http.Request) string {
	if value := strings.TrimSpace(r.URL.Query().Get(loginChallengeQueryKey)); value != "" {
		return value
	}
	if value := strings.TrimSpace(sharedhandler.ReadPortalSessionCookie(r)); value != "" {
		return value
	}
	return ""
}

func parseLoginResult(body []byte) (*sharedauthn.LoginResult, error) {
	var result sharedauthn.LoginResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
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

func authorizePageTitle(stage string) string {
	title := "登录"
	switch stage {
	case "account":
		title = "选择账号"
	case "confirmation":
		title = "设备确认"
	case "mfa":
		title = "多因素认证"
	}
	return title
}

func (h *OIDCHandler) mustBuildAuthorizeCaptcha(ctx context.Context, organizationID string) *authservice.AuthorizeCaptchaBootstrap {
	captcha, err := h.oidc.BuildAuthorizeCaptchaBootstrap(ctx, organizationID)
	if err != nil {
		return nil
	}
	return captcha
}

func buildAuthorizeMFAOptions(methods []string) []authorizeUIMethodOption {
	options := make([]authorizeUIMethodOption, 0, len(methods))
	for _, method := range methods {
		value := strings.TrimSpace(method)
		if value == "" {
			continue
		}
		label := value
		switch value {
		case "u2f":
			label = "安全密钥"
		case "totp":
			label = "身份验证器（TOTP）"
		case "email_code":
			label = "邮箱验证码"
		case "sms_code":
			label = "手机验证码"
		case "recovery_code":
			label = "备用验证码"
		}
		options = append(options, authorizeUIMethodOption{
			Value: value,
			Label: label,
		})
	}
	return options
}

func buildAuthorizeAppShell(title string) ([]byte, error) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>PPVT ` + title + `</title>
  <link rel="stylesheet" href="/auth/authorize/app.css" />
</head>
<body>
  <div id="app"></div>
  <script type="module" src="/auth/authorize/app.js"></script>
</body>
</html>`
	return []byte(html), nil
}

func deviceCodeLoginTarget(view *authservice.DeviceAuthorizationView) *coreservice.LoginTarget {
	if view == nil {
		return nil
	}
	return &coreservice.LoginTarget{
		OrganizationID:           view.Organization.ID,
		OrganizationName:         view.Organization.Name,
		DisplayName:              view.Organization.Name,
		OrganizationDisplayNames: map[string]string{},
		WebsiteURL:               "",
		TermsOfServiceURL:        view.Organization.TOSURL,
		PrivacyPolicyURL:         view.Organization.PrivacyPolicyURL,
		ProjectID:                view.Project.ID,
		ProjectName:              view.Project.Name,
		ApplicationID:            view.Application.ID,
		ApplicationName:          view.Application.Name,
		ApplicationDisplayNames: map[string]string{},
		ExternalIDPs:             nil,
	}
}

func (h *OIDCHandler) queryDeviceCodeInteraction(w http.ResponseWriter, r *http.Request, userCode string) (authorizeInteractionResponse, error) {
	view, err := h.oidc.DeviceAuthorizationByUserCode(r.Context(), userCode)
	if err != nil {
		return authorizeInteractionResponse{
			Action:        "render",
			FlowType:      "device_code",
			Stage:         "done",
			ResultStatus:  "error",
			ResultMessage: err.Error(),
		}, nil
	}
	target := deviceCodeLoginTarget(view)
	now := time.Now()
	if view.Authorization.ExpiresAt.Before(now) {
		return authorizeInteractionResponse{
			Action:        "render",
			FlowType:      "device_code",
			Stage:         "done",
			ResultStatus:  "error",
			ResultMessage: "device code has expired",
			Target:        target,
		}, nil
	}
	switch view.Authorization.Status {
	case "approved", "consumed":
		var currentUser *authorizeCurrentUser
		if strings.TrimSpace(view.Authorization.SessionID) != "" {
			if user, _, err := h.oidc.GetSessionUser(r.Context(), view.Authorization.SessionID); err == nil {
				currentUser = &authorizeCurrentUser{
					ID:          user.ID,
					Username:    user.Username,
					Name:        user.Name,
					Email:       user.Email,
					PhoneNumber: user.PhoneNumber,
				}
			}
		}
		return authorizeInteractionResponse{
			Action:        "render",
			FlowType:      "device_code",
			Stage:         "done",
			ResultStatus:  "success",
			ResultMessage: "You have successfully authorized this client.",
			Target:        target,
			CurrentUser:   currentUser,
		}, nil
	case "denied":
		return authorizeInteractionResponse{
			Action:        "render",
			FlowType:      "device_code",
			Stage:         "done",
			ResultStatus:  "error",
			ResultMessage: "The authorization request has been denied.",
			Target:        target,
		}, nil
	}
	sessionID := strings.TrimSpace(sharedhandler.ReadPortalSessionCookie(r))
	if sessionID == "" {
		sessionID = strings.TrimSpace(sharedhandler.ReadPendingLoginChallengeCookie(r))
	}
	if sessionID != "" {
		if session, err := h.oidc.GetSession(r.Context(), sessionID); err == nil {
			switch session.State {
			case "authenticated":
				user, _, userErr := h.oidc.GetSessionUser(r.Context(), sessionID)
				if userErr != nil {
					return authorizeInteractionResponse{}, userErr
				}
				return authorizeInteractionResponse{
					Action:   "render",
					FlowType: "device_code",
					Stage:    "account",
					Target:   target,
					CurrentUser: &authorizeCurrentUser{
						ID:          user.ID,
						Username:    user.Username,
						Name:        user.Name,
						Email:       user.Email,
						PhoneNumber: user.PhoneNumber,
					},
				}, nil
			case "confirmation_required":
				mfaOptions, mfaErr := h.oidc.AvailableMFAMethodsForSession(r.Context(), sessionID)
				if mfaErr != nil {
					return authorizeInteractionResponse{}, mfaErr
				}
				return authorizeInteractionResponse{
					Action:             "render",
					FlowType:           "device_code",
					Stage:              "confirmation",
					Target:             target,
					SecondFactorMethod: session.SecondFactorMethod,
					MFAOptions:         mfaOptions,
				}, nil
			case "mfa_required":
				mfaOptions, mfaErr := h.oidc.AvailableMFAMethodsForSession(r.Context(), sessionID)
				if mfaErr != nil {
					return authorizeInteractionResponse{}, mfaErr
				}
				return authorizeInteractionResponse{
					Action:             "render",
					FlowType:           "device_code",
					Stage:              "mfa",
					Target:             target,
					SecondFactorMethod: session.SecondFactorMethod,
					MFAOptions:         mfaOptions,
				}, nil
			}
		}
	}
	return authorizeInteractionResponse{
		Action:   "render",
		FlowType: "device_code",
		Stage:    "login",
		Target:   target,
		Captcha:  h.mustBuildAuthorizeCaptcha(r.Context(), target.OrganizationID),
	}, nil
}
