package handler

import (
	"encoding/json"
	"net/http"

	authservice "pass-pivot/internal/server/auth/service"
	sharedhandler "pass-pivot/internal/server/shared/handler"
	sharedhttp "pass-pivot/internal/server/shared/web"
)

type FederationHandler struct {
	federation *authservice.FederationService
}

func NewFederationHandler(federation *authservice.FederationService) *FederationHandler {
	return &FederationHandler{federation: federation}
}

func (h *FederationHandler) StartLogin(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ProviderID    string `json:"providerId"`
		ApplicationID string `json:"applicationId"`
		RedirectURI   string `json:"redirectUri"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	result, err := h.federation.StartLogin(r.Context(), payload.ProviderID, payload.ApplicationID, payload.RedirectURI)
	if err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedhttp.JSON(w, http.StatusOK, result)
}

func (h *FederationHandler) CompleteLogin(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		State         string `json:"state"`
		Code          string `json:"code"`
		ApplicationID string `json:"applicationId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	result, err := h.federation.CompleteLogin(r.Context(), payload.State, payload.Code, payload.ApplicationID)
	if err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedhandler.WritePortalSessionCookie(w, r, result.Session.ID)
	sharedhttp.JSON(w, http.StatusOK, result)
}
