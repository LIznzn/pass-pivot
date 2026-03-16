package user

import (
	"encoding/json"
	"net/http"

	"pass-pivot/internal/model"
	sharedhandler "pass-pivot/internal/server/shared/handler"
	sharedhttp "pass-pivot/internal/server/shared/web"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetCurrentUserProfile(w http.ResponseWriter, r *http.Request) {
	identity, ok := sharedhandler.AccessTokenIdentityFromRequest(r)
	if !ok || identity.User == nil {
		sharedhttp.Error(w, http.StatusUnauthorized, "access token is required")
		return
	}
	sharedhttp.JSON(w, http.StatusOK, identity.User)
}

func (h *Handler) UpdateCurrentUserProfile(w http.ResponseWriter, r *http.Request) {
	identity, ok := sharedhandler.AccessTokenIdentityFromRequest(r)
	if !ok || identity.User == nil {
		sharedhttp.Error(w, http.StatusUnauthorized, "access token is required")
		return
	}
	var payload struct {
		Username    string `json:"username"`
		Name        string `json:"name"`
		Email       string `json:"email"`
		PhoneNumber string `json:"phoneNumber"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	item, err := h.service.platform.UpdateCurrentUserProfile(r.Context(), identity.Token.SessionID, model.User{
		Username:    payload.Username,
		Name:        payload.Name,
		Email:       payload.Email,
		PhoneNumber: payload.PhoneNumber,
	})
	if err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedhttp.JSON(w, http.StatusOK, item)
}

func (h *Handler) GetCurrentUserDetail(w http.ResponseWriter, r *http.Request) {
	identity, ok := sharedhandler.AccessTokenIdentityFromRequest(r)
	if !ok || identity.User == nil {
		sharedhttp.Error(w, http.StatusUnauthorized, "access token is required")
		return
	}
	item, err := h.service.platform.GetUserDetail(r.Context(), identity.User.ID)
	if err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedhttp.JSON(w, http.StatusOK, item)
}

func (h *Handler) GetCurrentUserSetting(w http.ResponseWriter, r *http.Request) {
	identity, ok := sharedhandler.AccessTokenIdentityFromRequest(r)
	if !ok || identity.User == nil {
		sharedhttp.Error(w, http.StatusUnauthorized, "access token is required")
		return
	}
	item, err := h.service.platform.GetCurrentUserSetting(r.Context(), identity.Token.SessionID)
	if err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedhttp.JSON(w, http.StatusOK, item)
}

func (h *Handler) UpdateCurrentUserSetting(w http.ResponseWriter, r *http.Request) {
	identity, ok := sharedhandler.AccessTokenIdentityFromRequest(r)
	if !ok || identity.User == nil {
		sharedhttp.Error(w, http.StatusUnauthorized, "access token is required")
		return
	}
	var payload struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := h.service.platform.UpdateCurrentUserPassword(r.Context(), identity.Token.SessionID, payload.CurrentPassword, payload.NewPassword); err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedhttp.JSON(w, http.StatusOK, map[string]any{"updated": true})
}

func (h *Handler) UpdateCurrentUserMFAMethod(w http.ResponseWriter, r *http.Request) {
	identity, ok := sharedhandler.AccessTokenIdentityFromRequest(r)
	if !ok || identity.User == nil {
		sharedhttp.Error(w, http.StatusUnauthorized, "access token is required")
		return
	}
	var payload struct {
		Method  string `json:"method"`
		Enabled bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := h.service.platform.SetCurrentUserMFAMethod(r.Context(), identity.Token.SessionID, payload.Method, payload.Enabled); err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedhttp.JSON(w, http.StatusOK, map[string]any{"updated": true})
}

func (h *Handler) DeleteCurrentUserMFAEnrollment(w http.ResponseWriter, r *http.Request) {
	identity, ok := sharedhandler.AccessTokenIdentityFromRequest(r)
	if !ok || identity.User == nil {
		sharedhttp.Error(w, http.StatusUnauthorized, "access token is required")
		return
	}
	var payload struct {
		Method string `json:"method"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := h.service.platform.DeleteCurrentUserMFAEnrollments(r.Context(), identity.Token.SessionID, payload.Method); err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedhttp.JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (h *Handler) DeleteCurrentUserPasskey(w http.ResponseWriter, r *http.Request) {
	identity, ok := sharedhandler.AccessTokenIdentityFromRequest(r)
	if !ok || identity.User == nil {
		sharedhttp.Error(w, http.StatusUnauthorized, "access token is required")
		return
	}
	var payload struct {
		CredentialID string `json:"credentialId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := h.service.platform.DeleteCurrentUserPasskey(r.Context(), identity.Token.SessionID, payload.CredentialID); err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedhttp.JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (h *Handler) UntrustCurrentDevice(w http.ResponseWriter, r *http.Request) {
	identity, ok := sharedhandler.AccessTokenIdentityFromRequest(r)
	if !ok || identity.User == nil {
		sharedhttp.Error(w, http.StatusUnauthorized, "access token is required")
		return
	}
	var payload struct {
		DeviceID string `json:"deviceId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := h.service.platform.UntrustCurrentDevice(r.Context(), identity.Token.SessionID, payload.DeviceID); err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedhttp.JSON(w, http.StatusOK, map[string]any{"updated": true})
}

func (h *Handler) CreateCurrentExternalIdentityBinding(w http.ResponseWriter, r *http.Request) {
	identity, ok := sharedhandler.AccessTokenIdentityFromRequest(r)
	if !ok || identity.User == nil {
		sharedhttp.Error(w, http.StatusUnauthorized, "access token is required")
		return
	}
	var payload struct {
		ExternalIDPID string `json:"externalIdpId"`
		Issuer        string `json:"issuer"`
		Subject       string `json:"subject"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	item, err := h.service.platform.CreateCurrentExternalIdentityBinding(r.Context(), identity.Token.SessionID, model.ExternalIdentityBinding{
		ExternalIDPID: payload.ExternalIDPID,
		Issuer:        payload.Issuer,
		Subject:       payload.Subject,
	})
	if err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedhttp.JSON(w, http.StatusCreated, item)
}

func (h *Handler) DeleteCurrentExternalIdentityBinding(w http.ResponseWriter, r *http.Request) {
	identity, ok := sharedhandler.AccessTokenIdentityFromRequest(r)
	if !ok || identity.User == nil {
		sharedhttp.Error(w, http.StatusUnauthorized, "access token is required")
		return
	}
	var payload struct {
		BindingID string `json:"bindingId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := h.service.platform.DeleteCurrentExternalIdentityBinding(r.Context(), identity.Token.SessionID, payload.BindingID); err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedhttp.JSON(w, http.StatusOK, map[string]any{"deleted": true})
}
