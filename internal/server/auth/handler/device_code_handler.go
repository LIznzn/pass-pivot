package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	authservice "pass-pivot/internal/server/auth/service"
	sharedhandler "pass-pivot/internal/server/shared/handler"
	sharedweb "pass-pivot/internal/server/shared/web"
)

type deviceVerificationBootstrap struct {
	Title            string                `json:"title"`
	Status           string                `json:"status"`
	Error            string                `json:"error,omitempty"`
	UserCode         string                `json:"userCode"`
	ApplicationName  string                `json:"applicationName,omitempty"`
	OrganizationName string                `json:"organizationName,omitempty"`
	CurrentUser      *authorizeCurrentUser `json:"currentUser,omitempty"`
	LoginAction      string                `json:"loginAction"`
	ConfirmAction    string                `json:"confirmAction"`
	Denied           bool                  `json:"denied,omitempty"`
}

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
	if errMessage != "" {
		h.writeDeviceVerificationPage(w, http.StatusBadRequest, buildDeviceVerificationBootstrap(userCode, nil, nil, false, errMessage))
		return
	}
	_ = sessionID
	h.writeDeviceVerificationPage(w, http.StatusOK, buildDeviceVerificationBootstrap(userCode, view, currentUser, false, ""))
}

func (h *OIDCHandler) DeviceVerificationLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form body", http.StatusBadRequest)
		return
	}
	userCode := strings.TrimSpace(r.Form.Get("user_code"))
	view, err := h.oidc.DeviceAuthorizationByUserCode(r.Context(), userCode)
	if err != nil {
		h.writeDeviceVerificationPage(w, http.StatusBadRequest, buildDeviceVerificationBootstrap(userCode, nil, nil, false, err.Error()))
		return
	}
	body, err := h.callAuthnAPI(w, r, "/api/authn/v1/session/create", map[string]any{
		"organizationId": view.Organization.ID,
		"applicationId":  view.Application.ID,
		"identifier":     strings.TrimSpace(r.Form.Get("identifier")),
		"secret":         r.Form.Get("secret"),
	})
	if err != nil {
		h.writeDeviceVerificationPage(w, http.StatusBadRequest, buildDeviceVerificationBootstrap(userCode, view, nil, false, err.Error()))
		return
	}
	result, parseErr := parseLoginResult(body)
	if parseErr != nil || result.NextStep != "done" {
		h.writeDeviceVerificationPage(w, http.StatusBadRequest, buildDeviceVerificationBootstrap(userCode, view, nil, false, "device verification currently requires a fully authenticated session without pending MFA or confirmation"))
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
		h.writeDeviceVerificationPage(w, http.StatusBadRequest, buildDeviceVerificationBootstrap(userCode, nil, nil, false, err.Error()))
		return
	}
	view, viewErr := h.oidc.DeviceAuthorizationByUserCode(r.Context(), userCode)
	if viewErr != nil {
		h.writeDeviceVerificationPage(w, http.StatusBadRequest, buildDeviceVerificationBootstrap(userCode, nil, nil, false, viewErr.Error()))
		return
	}
	user, _, sessionErr := h.oidc.GetSessionUser(r.Context(), sessionID)
	if sessionErr != nil {
		h.writeDeviceVerificationPage(w, http.StatusBadRequest, buildDeviceVerificationBootstrap(userCode, view, nil, false, sessionErr.Error()))
		return
	}
	currentUser := &authorizeCurrentUser{
		ID:          user.ID,
		Username:    user.Username,
		Name:        user.Name,
		Email:       user.Email,
		PhoneNumber: user.PhoneNumber,
	}
	h.writeDeviceVerificationPage(w, http.StatusOK, buildDeviceVerificationBootstrap(userCode, view, currentUser, strings.EqualFold(strings.TrimSpace(r.Form.Get("deny")), "true"), ""))
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

func buildDeviceVerificationBootstrap(userCode string, view *authservice.DeviceAuthorizationView, currentUser *authorizeCurrentUser, denied bool, errMessage string) deviceVerificationBootstrap {
	title := "Device Verification"
	status := "pending"
	if errMessage != "" {
		title = "Device Verification Error"
		status = "error"
	}
	if errMessage == "" && view != nil && currentUser != nil && (view.Authorization.Status == "approved" || view.Authorization.Status == "denied") {
		status = "done"
	}
	bootstrap := deviceVerificationBootstrap{
		Title:         title,
		Status:        status,
		Error:         errMessage,
		UserCode:      userCode,
		CurrentUser:   currentUser,
		LoginAction:   "/auth/device/login",
		ConfirmAction: "/auth/device/confirm",
		Denied:        denied,
	}
	if view != nil {
		bootstrap.ApplicationName = view.Application.Name
		bootstrap.OrganizationName = view.Organization.Name
	}
	return bootstrap
}

func (h *OIDCHandler) writeDeviceVerificationPage(w http.ResponseWriter, status int, bootstrap deviceVerificationBootstrap) {
	body, err := buildDeviceVerificationAppShell(bootstrap)
	if err != nil {
		http.Error(w, fmt.Sprintf("build device verification page: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func buildDeviceVerificationAppShell(bootstrap deviceVerificationBootstrap) ([]byte, error) {
	payload, err := json.Marshal(bootstrap)
	if err != nil {
		return nil, err
	}
	html := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>PPVT ` + bootstrap.Title + `</title>
  <link rel="stylesheet" href="/auth/device/shared.css" />
  <link rel="stylesheet" href="/auth/device/app.css" />
</head>
<body>
  <div id="app"></div>
  <script>window.__PPVT_DEVICE_BOOTSTRAP__ = ` + string(payload) + `;</script>
  <script type="module" src="/auth/device/app.js"></script>
</body>
</html>`
	return []byte(html), nil
}

func nowTime() time.Time {
	return time.Now()
}
