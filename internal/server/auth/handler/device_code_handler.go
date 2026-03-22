package handler

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"time"

	authservice "pass-pivot/internal/server/auth/service"
	sharedhandler "pass-pivot/internal/server/shared/handler"
	sharedweb "pass-pivot/internal/server/shared/web"
)

func (h *OIDCHandler) DeviceAuthorization(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeOAuthTokenError(w, http.StatusBadRequest, "invalid_request", "invalid form body")
		return
	}
	clientID, clientSecret, _ := authservice.ParseBasicClientAuthorization(r.Header.Get("Authorization"))
	if clientID == "" {
		clientID = strings.TrimSpace(r.Form.Get("client_id"))
	}
	if clientSecret == "" {
		clientSecret = strings.TrimSpace(r.Form.Get("client_secret"))
	}
	result, err := h.oidc.CreateDeviceAuthorization(
		r.Context(),
		requestIssuer(r)+"/auth/device_authorization",
		clientID,
		clientSecret,
		strings.TrimSpace(r.Form.Get("client_assertion_type")),
		strings.TrimSpace(r.Form.Get("client_assertion")),
		strings.TrimSpace(r.Form.Get("scope")),
	)
	if err != nil {
		writeOAuthTokenError(w, http.StatusBadRequest, oauthErrorCode(err), err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, result)
}

func (h *OIDCHandler) DeviceVerification(w http.ResponseWriter, r *http.Request) {
	userCode := strings.TrimSpace(r.URL.Query().Get("user_code"))
	view, currentUser, sessionID, errMessage := h.loadDeviceVerificationPage(r, userCode)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if errMessage != "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(buildDeviceVerificationPage(userCode, nil, nil, "", errMessage)))
		return
	}
	_, _ = w.Write([]byte(buildDeviceVerificationPage(userCode, view, currentUser, sessionID, "")))
}

func (h *OIDCHandler) DeviceVerificationLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form body", http.StatusBadRequest)
		return
	}
	userCode := strings.TrimSpace(r.Form.Get("user_code"))
	view, err := h.oidc.DeviceAuthorizationByUserCode(r.Context(), userCode)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(buildDeviceVerificationPage(userCode, nil, nil, "", err.Error())))
		return
	}
	body, err := h.callAuthnAPI(w, r, "/api/authn/v1/session/create", map[string]any{
		"organizationId": view.Organization.ID,
		"applicationId":  view.Application.ID,
		"identifier":     strings.TrimSpace(r.Form.Get("identifier")),
		"secret":         r.Form.Get("secret"),
	})
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(buildDeviceVerificationPage(userCode, view, nil, "", err.Error())))
		return
	}
	result, parseErr := parseLoginResult(body)
	if parseErr != nil || result.NextStep != "done" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(buildDeviceVerificationPage(userCode, view, nil, "", "device verification currently requires a fully authenticated session without pending MFA or confirmation")))
		return
	}
	http.Redirect(w, r, "/auth/device?user_code="+url.QueryEscape(userCode), http.StatusFound)
}

func (h *OIDCHandler) DeviceVerificationConfirm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form body", http.StatusBadRequest)
		return
	}
	userCode := strings.TrimSpace(r.Form.Get("user_code"))
	sessionID := strings.TrimSpace(sharedhandler.ReadPortalSessionCookie(r))
	if sessionID == "" {
		http.Redirect(w, r, "/auth/device?user_code="+url.QueryEscape(userCode), http.StatusFound)
		return
	}
	var err error
	if strings.EqualFold(strings.TrimSpace(r.Form.Get("deny")), "true") {
		_, err = h.oidc.DenyDeviceAuthorization(r.Context(), userCode, sessionID)
	} else {
		_, err = h.oidc.ApproveDeviceAuthorization(r.Context(), userCode, sessionID)
	}
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(buildDeviceVerificationPage(userCode, nil, nil, "", err.Error())))
		return
	}
	_, currentUser, _, _ := h.loadDeviceVerificationPage(r, userCode)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(buildDeviceVerificationDonePage(currentUser, strings.EqualFold(strings.TrimSpace(r.Form.Get("deny")), "true"))))
}

func writeOAuthTokenError(w http.ResponseWriter, status int, code, description string) {
	if strings.TrimSpace(code) == "" {
		code = "server_error"
	}
	response := map[string]any{"error": code}
	if strings.TrimSpace(description) != "" {
		response["error_description"] = description
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func oauthErrorCode(err error) string {
	message := strings.TrimSpace(err.Error())
	switch message {
	case "invalid client", "invalid client credentials", "unsupported client authentication method":
		return "invalid_client"
	case "invalid device_code":
		return "invalid_grant"
	case "expired_token", "authorization_pending", "slow_down", "access_denied":
		return message
	case "device_code grant is not enabled for this application":
		return "unauthorized_client"
	default:
		return "invalid_request"
	}
}

func (h *OIDCHandler) loadDeviceVerificationPage(r *http.Request, userCode string) (*authservice.DeviceAuthorizationView, *authorizeCurrentUser, string, string) {
	view, err := h.oidc.DeviceAuthorizationByUserCode(r.Context(), userCode)
	if err != nil {
		return nil, nil, "", err.Error()
	}
	if view.Authorization.ExpiresAt.Before(nowTime()) {
		return nil, nil, "", "device code has expired"
	}
	switch view.Authorization.Status {
	case "approved":
		return view, nil, "", "device code has already been approved"
	case "consumed":
		return nil, nil, "", "device code has already been consumed"
	case "denied":
		return nil, nil, "", "device code has been denied"
	}
	sessionID := strings.TrimSpace(sharedhandler.ReadPortalSessionCookie(r))
	if sessionID == "" {
		return view, nil, "", ""
	}
	user, session, err := h.oidc.GetSessionUser(r.Context(), sessionID)
	if err != nil || session.State != "authenticated" {
		return view, nil, "", ""
	}
	return view, &authorizeCurrentUser{
		ID:          user.ID,
		Username:    user.Username,
		Name:        user.Name,
		Email:       user.Email,
		PhoneNumber: user.PhoneNumber,
	}, sessionID, ""
}

func buildDeviceVerificationPage(userCode string, view *authservice.DeviceAuthorizationView, currentUser *authorizeCurrentUser, sessionID, errMessage string) string {
	title := "Device Verification"
	if errMessage != "" {
		title = "Device Verification Error"
	}
	var content strings.Builder
	content.WriteString("<!DOCTYPE html><html><head><meta charset=\"utf-8\" /><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\" /><title>")
	content.WriteString(title)
	content.WriteString("</title><style>body{font-family:sans-serif;max-width:560px;margin:48px auto;padding:0 16px;line-height:1.5}form{display:flex;flex-direction:column;gap:12px}input{padding:10px;font-size:16px}button{padding:10px 14px;font-size:16px}code{font-size:18px} .error{color:#b00020}</style></head><body>")
	content.WriteString("<h1>Device Verification</h1>")
	if errMessage != "" {
		content.WriteString("<p class=\"error\">")
		content.WriteString(html.EscapeString(errMessage))
		content.WriteString("</p>")
	}
	if view != nil {
		content.WriteString("<p>Application: <strong>")
		content.WriteString(html.EscapeString(view.Application.Name))
		content.WriteString("</strong></p><p>Organization: <strong>")
		content.WriteString(html.EscapeString(view.Organization.Name))
		content.WriteString("</strong></p>")
	}
	content.WriteString("<p>User code: <code>")
	content.WriteString(html.EscapeString(userCode))
	content.WriteString("</code></p>")
	if currentUser == nil {
		content.WriteString("<form method=\"post\" action=\"/auth/device/login\">")
		content.WriteString("<input type=\"hidden\" name=\"user_code\" value=\"")
		content.WriteString(html.EscapeString(userCode))
		content.WriteString("\" />")
		content.WriteString("<label>Email or username<input name=\"identifier\" autocomplete=\"username\" required /></label>")
		content.WriteString("<label>Password<input type=\"password\" name=\"secret\" autocomplete=\"current-password\" required /></label>")
		content.WriteString("<button type=\"submit\">Sign in</button></form>")
	} else {
		content.WriteString("<p>Signed in as <strong>")
		content.WriteString(html.EscapeString(currentUser.Email))
		content.WriteString("</strong></p>")
		content.WriteString("<form method=\"post\" action=\"/auth/device/confirm\">")
		content.WriteString("<input type=\"hidden\" name=\"user_code\" value=\"")
		content.WriteString(html.EscapeString(userCode))
		content.WriteString("\" />")
		content.WriteString("<button type=\"submit\">Approve</button></form>")
		content.WriteString("<form method=\"post\" action=\"/auth/device/confirm\" style=\"margin-top:12px\">")
		content.WriteString("<input type=\"hidden\" name=\"user_code\" value=\"")
		content.WriteString(html.EscapeString(userCode))
		content.WriteString("\" /><input type=\"hidden\" name=\"deny\" value=\"true\" />")
		content.WriteString("<button type=\"submit\">Deny</button></form>")
	}
	content.WriteString("</body></html>")
	return content.String()
}

func buildDeviceVerificationDonePage(currentUser *authorizeCurrentUser, denied bool) string {
	action := "approved"
	if denied {
		action = "denied"
	}
	email := ""
	if currentUser != nil {
		email = currentUser.Email
	}
	return fmt.Sprintf("<!DOCTYPE html><html><head><meta charset=\"utf-8\" /><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\" /><title>Device Verification</title></head><body style=\"font-family:sans-serif;max-width:560px;margin:48px auto;padding:0 16px\"><h1>Device Verification</h1><p>The request has been %s.</p><p>%s</p></body></html>", html.EscapeString(action), html.EscapeString(email))
}

func nowTime() time.Time {
	return time.Now()
}
