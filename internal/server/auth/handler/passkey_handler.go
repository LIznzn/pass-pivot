package handler

import (
	"encoding/json"
	"net/http"

	authservice "pass-pivot/internal/server/auth/service"
	sharedhandler "pass-pivot/internal/server/shared/handler"
	sharedhttp "pass-pivot/internal/server/shared/web"
)

type PasskeyHandler struct {
	passkey *authservice.PasskeyService
}

func NewPasskeyHandler(passkey *authservice.PasskeyService) *PasskeyHandler {
	return &PasskeyHandler{passkey: passkey}
}

func (h *PasskeyHandler) BeginRegistration(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		UserID  string `json:"userId"`
		Purpose string `json:"purpose"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	var (
		challengeID string
		options     any
		err         error
	)
	if identity, ok := sharedhandler.AccessTokenIdentityFromRequest(r); ok && identity.User != nil {
		targetUserID, allowed := sharedhandler.CurrentUserIDOrTarget(identity, payload.UserID)
		if !allowed {
			sharedhttp.Error(w, http.StatusForbidden, "console:admin role is required")
			return
		}
		challengeID, options, err = h.passkey.BeginRegistration(r.Context(), targetUserID, payload.Purpose)
	} else {
		challengeID, options, err = h.passkey.BeginRegistration(r.Context(), payload.UserID, payload.Purpose)
	}
	if err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedhttp.JSON(w, http.StatusOK, map[string]any{"challengeId": challengeID, "options": options})
}

func (h *PasskeyHandler) BeginPortalRegistration(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Purpose string `json:"purpose"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && err.Error() != "EOF" {
		sharedhttp.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	challengeID, options, err := h.passkey.BeginRegistrationForSession(r.Context(), sharedhandler.ReadPortalSessionCookie(r), payload.Purpose)
	if err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedhttp.JSON(w, http.StatusOK, map[string]any{"challengeId": challengeID, "options": options})
}

func (h *PasskeyHandler) FinishRegistration(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ChallengeID string          `json:"challengeId"`
		Response    json.RawMessage `json:"response"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := h.passkey.FinishRegistration(r.Context(), payload.ChallengeID, payload.Response); err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedhttp.JSON(w, http.StatusOK, map[string]any{"registered": true})
}

func (h *PasskeyHandler) BeginLogin(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Identifier string `json:"identifier"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	challengeID, options, err := h.passkey.BeginLogin(r.Context(), payload.Identifier)
	if err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedhttp.JSON(w, http.StatusOK, map[string]any{"challengeId": challengeID, "options": options})
}

func (h *PasskeyHandler) FinishLogin(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ChallengeID   string          `json:"challengeId"`
		Response      json.RawMessage `json:"response"`
		ApplicationID string          `json:"applicationId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	deviceKey := h.passkey.ParseFingerprint(sharedhandler.ReadFingerprintCookie(r))
	result, err := h.passkey.FinishLogin(r.Context(), payload.ChallengeID, payload.Response, payload.ApplicationID, deviceKey)
	if err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedhandler.WriteFingerprintCookie(w, r, result.Fingerprint)
	sharedhandler.WritePortalSessionCookie(w, r, result.Session.ID)
	sharedhttp.JSON(w, http.StatusOK, result)
}

func (h *PasskeyHandler) BeginSessionMFA(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	sessionID := payload.SessionID
	if sessionID == "" {
		sessionID = sharedhandler.ReadPortalSessionCookie(r)
	}
	challengeID, options, err := h.passkey.BeginMFA(r.Context(), sessionID)
	if err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedhttp.JSON(w, http.StatusOK, map[string]any{"challengeId": challengeID, "options": options})
}

func (h *PasskeyHandler) FinishSessionMFA(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ChallengeID string          `json:"challengeId"`
		Response    json.RawMessage `json:"response"`
		TrustDevice bool            `json:"trustDevice"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	result, err := h.passkey.FinishMFA(r.Context(), payload.ChallengeID, payload.Response, payload.TrustDevice)
	if err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedhandler.WritePortalSessionCookie(w, r, result.Session.ID)
	sharedhttp.JSON(w, http.StatusOK, result)
}
