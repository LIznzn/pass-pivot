package manage

import (
	"encoding/json"
	"net/http"

	"pass-pivot/internal/model"
	sharedweb "pass-pivot/internal/server/shared/web"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListOrganizations(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListOrganizations(r.Context())
	if err != nil {
		sharedweb.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) CreateOrganization(w http.ResponseWriter, r *http.Request) {
	var payload model.Organization
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	item, err := h.service.CreateOrganization(r.Context(), payload)
	if err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusCreated, item)
}

func (h *Handler) UpdateOrganization(w http.ResponseWriter, r *http.Request) {
	var payload model.Organization
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	item, err := h.service.UpdateOrganization(r.Context(), payload)
	if err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, item)
}

func (h *Handler) DisableOrganization(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		OrganizationID string `json:"organizationId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := h.service.DisableOrganization(r.Context(), payload.OrganizationID); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"disabled": true})
}

func (h *Handler) DeleteOrganization(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		OrganizationID string `json:"organizationId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := h.service.DeleteOrganization(r.Context(), payload.OrganizationID); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (h *Handler) ListProjects(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		OrganizationID string `json:"organizationId"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&payload)
	}
	items, err := h.service.ListProjects(r.Context(), payload.OrganizationID)
	if err != nil {
		sharedweb.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) CreateProject(w http.ResponseWriter, r *http.Request) {
	var payload model.Project
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	item, err := h.service.CreateProject(r.Context(), payload)
	if err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusCreated, item)
}

func (h *Handler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	var payload model.Project
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	item, err := h.service.UpdateProject(r.Context(), payload)
	if err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, item)
}

func (h *Handler) DisableProject(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ProjectID string `json:"projectId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := h.service.DisableProject(r.Context(), payload.ProjectID); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"disabled": true})
}

func (h *Handler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ProjectID string `json:"projectId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := h.service.DeleteProject(r.Context(), payload.ProjectID); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (h *Handler) UpdateProjectUserAssignments(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ProjectID string   `json:"projectId"`
		UserIDs   []string `json:"userIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	userIDs, err := h.service.UpdateProjectUserAssignments(r.Context(), payload.ProjectID, payload.UserIDs)
	if err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{
		"projectId": payload.ProjectID,
		"userIds":   userIDs,
	})
}

func (h *Handler) ListApplications(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ProjectID string `json:"projectId"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&payload)
	}
	items, err := h.service.ListApplications(r.Context(), payload.ProjectID)
	if err != nil {
		sharedweb.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) CreateApplication(w http.ResponseWriter, r *http.Request) {
	var payload model.Application
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	item, err := h.service.CreateApplication(r.Context(), payload)
	if err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response := map[string]any{
		"id":                       item.Application.ID,
		"projectId":                item.Application.ProjectID,
		"name":                     item.Application.Name,
		"metadata":                 item.Application.Metadata,
		"description":              item.Application.Description,
		"redirectUris":             item.Application.RedirectURIs,
		"status":                   item.Application.Status,
		"applicationType":          item.Application.ApplicationType,
		"grantType":                item.Application.GrantType,
		"enableRefreshToken":       item.Application.EnableRefreshToken,
		"clientAuthenticationType": item.Application.ClientAuthenticationType,
		"tokenType":                item.Application.TokenType,
		"roles":                    item.Application.Roles,
		"publicKey":                item.Application.PublicKey,
		"accessTokenTTLMinutes":    item.Application.AccessTokenTTLMinutes,
		"refreshTokenTTLHours":     item.Application.RefreshTokenTTLHours,
		"createdAt":                item.Application.CreatedAt,
		"updatedAt":                item.Application.UpdatedAt,
		"generatedPrivateKey":      item.GeneratedPrivateKey,
	}
	sharedweb.JSON(w, http.StatusCreated, response)
}

func (h *Handler) ResetApplicationKey(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ApplicationID string `json:"applicationId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	item, err := h.service.ResetApplicationKey(r.Context(), payload.ApplicationID)
	if err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response := map[string]any{
		"id":                  item.Application.ID,
		"publicKey":           item.Application.PublicKey,
		"generatedPrivateKey": item.GeneratedPrivateKey,
	}
	sharedweb.JSON(w, http.StatusOK, response)
}

func (h *Handler) UpdateApplication(w http.ResponseWriter, r *http.Request) {
	var payload model.Application
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	item, err := h.service.UpdateApplication(r.Context(), payload)
	if err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response := map[string]any{
		"id":                       item.Application.ID,
		"projectId":                item.Application.ProjectID,
		"name":                     item.Application.Name,
		"metadata":                 item.Application.Metadata,
		"description":              item.Application.Description,
		"redirectUris":             item.Application.RedirectURIs,
		"status":                   item.Application.Status,
		"applicationType":          item.Application.ApplicationType,
		"grantType":                item.Application.GrantType,
		"enableRefreshToken":       item.Application.EnableRefreshToken,
		"clientAuthenticationType": item.Application.ClientAuthenticationType,
		"tokenType":                item.Application.TokenType,
		"roles":                    item.Application.Roles,
		"publicKey":                item.Application.PublicKey,
		"accessTokenTTLMinutes":    item.Application.AccessTokenTTLMinutes,
		"refreshTokenTTLHours":     item.Application.RefreshTokenTTLHours,
		"createdAt":                item.Application.CreatedAt,
		"updatedAt":                item.Application.UpdatedAt,
		"generatedPrivateKey":      item.GeneratedPrivateKey,
	}
	sharedweb.JSON(w, http.StatusOK, response)
}

func (h *Handler) DisableApplication(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ApplicationID string `json:"applicationId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := h.service.DisableApplication(r.Context(), payload.ApplicationID); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"disabled": true})
}

func (h *Handler) DeleteApplication(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ApplicationID string `json:"applicationId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := h.service.DeleteApplication(r.Context(), payload.ApplicationID); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		OrganizationID string `json:"organizationId"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&payload)
	}
	items, err := h.service.ListUsers(r.Context(), payload.OrganizationID)
	if err != nil {
		sharedweb.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		OrganizationID string   `json:"organizationId"`
		ApplicationID  string   `json:"applicationId"`
		Username       string   `json:"username"`
		Name           string   `json:"name"`
		Email          string   `json:"email"`
		PhoneNumber    string   `json:"phoneNumber"`
		Roles          []string `json:"roles"`
		Status         string   `json:"status"`
		Identifier     string   `json:"identifier"`
		Password       string   `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	item, err := h.service.CreateUser(r.Context(), model.User{
		OrganizationID: payload.OrganizationID,
		Username:       payload.Username,
		Name:           payload.Name,
		Email:          payload.Email,
		PhoneNumber:    payload.PhoneNumber,
		Roles:          payload.Roles,
		Status:         payload.Status,
	}, payload.Identifier, payload.Password, payload.ApplicationID)
	if err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusCreated, item)
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	var payload model.User
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	item, err := h.service.UpdateUser(r.Context(), payload)
	if err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, item)
}

func (h *Handler) UpdateUserMFAMethod(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		UserID  string `json:"userId"`
		Method  string `json:"method"`
		Enabled bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := h.service.SetUserMFAMethod(r.Context(), payload.UserID, payload.Method, payload.Enabled); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"updated": true})
}

func (h *Handler) DeleteUserMFAEnrollment(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		UserID string `json:"userId"`
		Method string `json:"method"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := h.service.DeleteUserMFAEnrollments(r.Context(), payload.UserID, payload.Method); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (h *Handler) DeleteUserSecureKey(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		UserID       string `json:"userId"`
		CredentialID string `json:"credentialId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := h.service.DeleteUserSecureKey(r.Context(), payload.UserID, payload.CredentialID); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (h *Handler) BeginUserSecureKeyRegistration(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		UserID  string `json:"userId"`
		Purpose string `json:"purpose"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	challengeID, options, err := h.service.BeginUserSecureKeyRegistration(r.Context(), payload.UserID, payload.Purpose)
	if err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"challengeId": challengeID, "options": options})
}

func (h *Handler) FinishUserSecureKeyRegistration(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ChallengeID string          `json:"challengeId"`
		Response    json.RawMessage `json:"response"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := h.service.FinishSecureKeyRegistration(r.Context(), payload.ChallengeID, payload.Response); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"registered": true})
}

func (h *Handler) DeleteUserRecoveryCodes(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		UserID string `json:"userId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := h.service.DeleteUserRecoveryCodes(r.Context(), payload.UserID); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (h *Handler) GetUserDetail(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		UserID string `json:"userId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	item, err := h.service.GetUserDetail(r.Context(), payload.UserID)
	if err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, item)
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		UserID  string   `json:"userId"`
		UserIDs []string `json:"userIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if payload.UserID != "" {
		payload.UserIDs = append(payload.UserIDs, payload.UserID)
	}
	if err := h.service.DeleteUsers(r.Context(), payload.UserIDs); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (h *Handler) DisableUser(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		UserID string `json:"userId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := h.service.DisableUser(r.Context(), payload.UserID); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"disabled": true})
}

func (h *Handler) EnableUser(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		UserID string `json:"userId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := h.service.EnableUser(r.Context(), payload.UserID); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"enabled": true})
}

func (h *Handler) ResetUserPassword(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		UserID   string `json:"userId"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := h.service.ResetUserPassword(r.Context(), payload.UserID, payload.Password); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"reset": true})
}

func (h *Handler) ResetUserUKID(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		UserID string `json:"userId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	ukid, err := h.service.ResetUserUKID(r.Context(), payload.UserID)
	if err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"ukid": ukid})
}

func (h *Handler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		OrganizationID string `json:"organizationId"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&payload)
	}
	items, err := h.service.ListAuditLogs(r.Context(), payload.OrganizationID)
	if err != nil {
		sharedweb.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) ListExternalIDPs(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		OrganizationID string `json:"organizationId"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&payload)
	}
	items, err := h.service.ListExternalIDPs(r.Context(), payload.OrganizationID)
	if err != nil {
		sharedweb.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) CreateExternalIDP(w http.ResponseWriter, r *http.Request) {
	var payload model.ExternalIDP
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	item, err := h.service.CreateExternalIDP(r.Context(), payload)
	if err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusCreated, item)
}

func (h *Handler) UpdateExternalIDP(w http.ResponseWriter, r *http.Request) {
	var payload model.ExternalIDP
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	item, err := h.service.UpdateExternalIDP(r.Context(), payload)
	if err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, item)
}

func (h *Handler) DeleteExternalIdentityBinding(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		UserID    string `json:"userId"`
		BindingID string `json:"bindingId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := h.service.DeleteExternalIdentityBinding(r.Context(), payload.UserID, payload.BindingID); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (h *Handler) UntrustUserDevice(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		UserID   string `json:"userId"`
		DeviceID string `json:"deviceId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := h.service.UntrustUserDevice(r.Context(), payload.UserID, payload.DeviceID); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"updated": true})
}

func (h *Handler) RevokeUserSessions(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		UserID string `json:"userId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := h.service.RevokeUserSessions(r.Context(), payload.UserID); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"revoked": true})
}

func (h *Handler) RotateUserToken(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		UserID string `json:"userId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := h.service.RotateUserToken(r.Context(), payload.UserID); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"rotated": true})
}

func (h *Handler) CreateExternalIdentityBinding(w http.ResponseWriter, r *http.Request) {
	var payload model.ExternalIdentityBinding
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedweb.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	item, err := h.service.CreateExternalIdentityBinding(r.Context(), payload)
	if err != nil {
		sharedweb.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusCreated, item)
}
