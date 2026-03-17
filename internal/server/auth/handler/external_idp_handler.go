package handler

import (
	"encoding/json"
	"net/http"

	authservice "pass-pivot/internal/server/auth/service"
	sharedhandler "pass-pivot/internal/server/shared/handler"
	sharedweb "pass-pivot/internal/server/shared/web"
)

type ExternalIDPHandler struct {
	externalIDP *authservice.ExternalIDPService
}

func NewExternalIDPHandler(externalIDP *authservice.ExternalIDPService) *ExternalIDPHandler {
	return &ExternalIDPHandler{externalIDP: externalIDP}
}

func (h *ExternalIDPHandler) StartLogin(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ProviderID    string `json:"providerId"`
		ApplicationID string `json:"applicationId"`
		RedirectURI   string `json:"redirectUri"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	result, err := h.externalIDP.StartLogin(r.Context(), payload.ProviderID, payload.ApplicationID, payload.RedirectURI)
	if err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, result)
}

func (h *ExternalIDPHandler) CompleteLogin(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		State         string `json:"state"`
		Code          string `json:"code"`
		ApplicationID string `json:"applicationId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	result, err := h.externalIDP.CompleteLogin(r.Context(), payload.State, payload.Code, payload.ApplicationID)
	if err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedhandler.WritePortalSessionCookie(w, r, result.Session.ID)
	sharedweb.JSON(w, http.StatusOK, result)
}
