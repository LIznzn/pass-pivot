package system

import (
	"encoding/json"
	"net/http"

	sharedweb "pass-pivot/internal/server/shared/web"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Service() *Service {
	return h.service
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	sharedweb.JSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"service": "ppvt",
	})
}

func (h *Handler) IntrospectToken(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	result, err := h.service.IntrospectToken(r.Context(), payload.Token)
	if err != nil {
		sharedweb.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, result)
}

func (h *Handler) ListPublicExternalIDPs(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ApplicationID string `json:"applicationId"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&payload)
	}
	items, err := h.service.ListPublicExternalIDPsByApplication(r.Context(), payload.ApplicationID)
	if err != nil {
		sharedweb.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	type publicProvider struct {
		ID             string `json:"id"`
		OrganizationID string `json:"organizationId"`
		Protocol       string `json:"protocol"`
		Name           string `json:"name"`
		Issuer         string `json:"issuer"`
	}
	response := make([]publicProvider, 0, len(items))
	for _, item := range items {
		response = append(response, publicProvider{
			ID:             item.ID,
			OrganizationID: item.OrganizationID,
			Protocol:       item.Protocol,
			Name:           item.Name,
			Issuer:         item.Issuer,
		})
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"items": response})
}

func (h *Handler) GetLoginTarget(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ApplicationID string `json:"applicationId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	item, err := h.service.GetLoginTarget(r.Context(), payload.ApplicationID)
	if err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, item)
}
