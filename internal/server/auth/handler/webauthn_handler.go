package handler

import (
	"context"
	"encoding/json"
	"net/http"

	sharedauthn "pass-pivot/internal/server/shared/authn"
	sharedhandler "pass-pivot/internal/server/shared/handler"
	sharedweb "pass-pivot/internal/server/shared/web"
)

type WebAuthnHandler struct {
	service webAuthnLoginService
}

type webAuthnLoginService interface {
	BeginWebAuthnLogin(ctx context.Context, identifier string) (string, any, error)
	FinishWebAuthnLogin(ctx context.Context, challengeID string, payload json.RawMessage, applicationID, deviceKey string) (*sharedauthn.LoginResult, error)
	ParseFingerprint(signedFingerprint string) string
}

func NewWebAuthnHandler(service webAuthnLoginService) *WebAuthnHandler {
	return &WebAuthnHandler{service: service}
}

func (h *WebAuthnHandler) BeginLogin(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Identifier string `json:"identifier"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	challengeID, options, err := h.service.BeginWebAuthnLogin(r.Context(), payload.Identifier)
	if err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"challengeId": challengeID, "options": options})
}

func (h *WebAuthnHandler) FinishLogin(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ChallengeID   string          `json:"challengeId"`
		Response      json.RawMessage `json:"response"`
		ApplicationID string          `json:"applicationId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	deviceKey := h.service.ParseFingerprint(sharedhandler.ReadFingerprintCookie(r))
	result, err := h.service.FinishWebAuthnLogin(r.Context(), payload.ChallengeID, payload.Response, payload.ApplicationID, deviceKey)
	if err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedhandler.WriteFingerprintCookie(w, r, result.Fingerprint)
	sharedhandler.WritePortalSessionCookie(w, r, result.Session.ID)
	sharedweb.JSON(w, http.StatusOK, result)
}
