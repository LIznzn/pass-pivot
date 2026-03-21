package handler

import (
	"context"
	"encoding/json"
	"net/http"

	sharedauthn "pass-pivot/internal/server/shared/authn"
	authnapi "pass-pivot/internal/server/shared/authnapi"
	sharedhandler "pass-pivot/internal/server/shared/handler"
	sharedweb "pass-pivot/internal/server/shared/web"
)

type MFAU2FHandler struct {
	service u2fAssertionService
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
	sessionID := payload.SessionID
	if sessionID == "" {
		sessionID = sharedhandler.ReadPortalSessionCookie(r)
	}
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
	sharedhandler.WritePortalSessionCookie(w, r, result.Session.ID)
	sharedweb.JSON(w, http.StatusOK, result)
}
