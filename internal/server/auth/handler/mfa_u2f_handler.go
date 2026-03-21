package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	sharedauthn "pass-pivot/internal/server/shared/authn"
	authnapi "pass-pivot/internal/server/shared/authnapi"
	sharedhandler "pass-pivot/internal/server/shared/handler"
	sharedweb "pass-pivot/internal/server/shared/web"
)

type MFAU2FHandler struct {
	service u2fAssertionService
}

func resolveSessionReference(r *http.Request, explicit string) string {
	if value := strings.TrimSpace(explicit); value != "" {
		return value
	}
	if value := strings.TrimSpace(sharedhandler.ReadPendingLoginChallengeCookie(r)); value != "" {
		return value
	}
	return sharedhandler.ReadPortalSessionCookie(r)
}

type u2fAssertionService interface {
	BeginU2FAssertion(ctx context.Context, sessionID string) (string, any, error)
	FinishU2FAssertion(ctx context.Context, challengeID string, payload json.RawMessage, trustDevice bool) (*sharedauthn.LoginResult, error)
}

func NewMFAU2FHandler(service u2fAssertionService) *MFAU2FHandler {
	return &MFAU2FHandler{service: service}
}

func (h *MFAU2FHandler) BeginAssertion(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		authnapi.Write(w, http.StatusBadRequest, authnapi.CodeInvalidJSONBody, "invalid JSON body")
		return
	}
	sessionID := resolveSessionReference(r, payload.SessionID)
	challengeID, options, err := h.service.BeginU2FAssertion(r.Context(), sessionID)
	if err != nil {
		authnapi.WriteKnown(w, err)
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"challengeId": challengeID, "options": options})
}

func (h *MFAU2FHandler) FinishAssertion(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ChallengeID string          `json:"challengeId"`
		Response    json.RawMessage `json:"response"`
		TrustDevice bool            `json:"trustDevice"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		authnapi.Write(w, http.StatusBadRequest, authnapi.CodeInvalidJSONBody, "invalid JSON body")
		return
	}
	result, err := h.service.FinishU2FAssertion(r.Context(), payload.ChallengeID, payload.Response, payload.TrustDevice)
	if err != nil {
		authnapi.WriteKnown(w, err)
		return
	}
	if result.NextStep == "done" {
		sharedhandler.ClearPendingLoginChallengeCookie(w, r)
		sharedhandler.WritePortalSessionCookie(w, r, result.Session.ID)
	} else {
		sharedhandler.ClearPortalSessionCookie(w, r)
		sharedhandler.WritePendingLoginChallengeCookie(w, r, result.Session.LoginChallenge)
	}
	sharedweb.JSON(w, http.StatusOK, result)
}
