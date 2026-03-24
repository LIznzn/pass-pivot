package handler

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

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
		requestIssuer(r)+"/auth/device/code",
		clientID,
		clientSecret,
		strings.TrimSpace(r.Form.Get("client_assertion_type")),
		strings.TrimSpace(r.Form.Get("client_assertion")),
		strings.TrimSpace(r.Form.Get("scope")),
		sharedhandler.NormalizeRemoteIP(sharedhandler.OriginalRemoteAddr(r)),
		strings.TrimSpace(sharedhandler.OriginalUserAgent(r)),
	)
	if err != nil {
		writeOAuthTokenError(w, http.StatusBadRequest, oauthErrorCode(err), err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, result)
}

func (h *OIDCHandler) DeviceVerification(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/auth/authorize?type=device_code&user_code="+url.QueryEscape(strings.TrimSpace(r.URL.Query().Get("user_code"))), http.StatusMovedPermanently)
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
