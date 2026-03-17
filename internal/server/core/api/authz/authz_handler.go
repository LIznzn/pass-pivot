package authz

import (
	"encoding/json"
	"net/http"

	"pass-pivot/internal/model"
	sharedweb "pass-pivot/internal/server/shared/web"
)

type Handler struct {
	service *AuthzService
}

func NewHandler(service *AuthzService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Service() *AuthzService {
	return h.service
}

func (h *Handler) ListRoles(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		OrganizationID string `json:"organizationId"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&payload)
	}
	items, err := h.service.ListRoles(r.Context(), payload.OrganizationID)
	if err != nil {
		sharedweb.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) CreateRole(w http.ResponseWriter, r *http.Request) {
	var payload model.Role
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	item, err := h.service.CreateRole(r.Context(), payload)
	if err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusCreated, item)
}

func (h *Handler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	var payload model.Role
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	item, err := h.service.UpdateRole(r.Context(), payload)
	if err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, item)
}

func (h *Handler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		RoleID  string   `json:"roleId"`
		RoleIDs []string `json:"roleIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if payload.RoleID != "" {
		payload.RoleIDs = append(payload.RoleIDs, payload.RoleID)
	}
	if err := h.service.DeleteRoles(r.Context(), payload.RoleIDs); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (h *Handler) ListPolicies(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		OrganizationID string `json:"organizationId"`
		RoleID         string `json:"roleId"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&payload)
	}
	items, err := h.service.ListPolicies(r.Context(), payload.OrganizationID, payload.RoleID)
	if err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) CreatePolicy(w http.ResponseWriter, r *http.Request) {
	var payload model.Policy
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	item, err := h.service.CreatePolicy(r.Context(), payload)
	if err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusCreated, item)
}

func (h *Handler) UpdatePolicy(w http.ResponseWriter, r *http.Request) {
	var payload model.Policy
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	item, err := h.service.UpdatePolicy(r.Context(), payload)
	if err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, item)
}

func (h *Handler) DeletePolicy(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PolicyID  string   `json:"policyId"`
		PolicyIDs []string `json:"policyIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if payload.PolicyID != "" {
		payload.PolicyIDs = append(payload.PolicyIDs, payload.PolicyID)
	}
	if err := h.service.DeletePolicies(r.Context(), payload.PolicyIDs); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (h *Handler) CheckPolicy(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		SubjectType string `json:"subjectType"`
		SubjectID   string `json:"subjectId"`
		Method      string `json:"method"`
		Path        string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	result, err := h.service.CheckPolicy(r.Context(), payload.SubjectType, payload.SubjectID, payload.Method, payload.Path)
	if err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	status := http.StatusOK
	if !result.Allowed {
		status = http.StatusForbidden
	}
	sharedweb.JSON(w, status, result)
}

func (h *Handler) ListSubjectPolicies(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		SubjectType string `json:"subjectType"`
		SubjectID   string `json:"subjectId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	result, err := h.service.ListSubjectPolicies(r.Context(), payload.SubjectType, payload.SubjectID)
	if err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, result)
}
