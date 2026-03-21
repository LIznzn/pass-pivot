package authn

import (
	"encoding/json"
	"net/http"

	sharedauthn "pass-pivot/internal/server/shared/authn"
	authnapi "pass-pivot/internal/server/shared/authnapi"
	sharedhandler "pass-pivot/internal/server/shared/handler"
	sharedweb "pass-pivot/internal/server/shared/web"
)

type Handler struct {
	service *AuthnService
}

func NewHandler(service *AuthnService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		OrganizationID      string `json:"organizationId"`
		ApplicationID       string `json:"applicationId"`
		Identifier          string `json:"identifier"`
		Secret              string `json:"secret"`
		TrustCurrentDevice  bool   `json:"trustCurrentDevice"`
		RequireAnnouncement bool   `json:"requireAnnouncement"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		authnapi.Write(w, http.StatusBadRequest, authnapi.CodeInvalidJSONBody, "invalid JSON body")
		return
	}
	deviceKey := h.service.ParseFingerprint(sharedhandler.ReadFingerprintCookie(r))
	ipAddress := sharedhandler.NormalizeRemoteIP(sharedhandler.OriginalRemoteAddr(r))
	result, err := h.service.LoginWithUserCredential(r.Context(), sharedauthn.LoginInput{
		OrganizationID:      payload.OrganizationID,
		ApplicationID:       payload.ApplicationID,
		Identifier:          payload.Identifier,
		Secret:              payload.Secret,
		IPAddress:           ipAddress,
		UserAgent:           sharedhandler.OriginalUserAgent(r),
		DeviceKey:           deviceKey,
		TrustCurrentDevice:  payload.TrustCurrentDevice,
		RequireAnnouncement: payload.RequireAnnouncement,
	})
	if err != nil {
		authnapi.WriteKnown(w, err)
		return
	}
	sharedhandler.WriteFingerprintCookie(w, r, result.Fingerprint)
	sharedhandler.WritePortalSessionCookie(w, r, result.Session.ID)
	sharedweb.JSON(w, http.StatusOK, result)
}

func (h *Handler) Confirm(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		SessionID string `json:"sessionId"`
		Accept    bool   `json:"accept"`
		TrustDevice bool `json:"trustDevice"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		authnapi.Write(w, http.StatusBadRequest, authnapi.CodeInvalidJSONBody, "invalid JSON body")
		return
	}
	sessionID := payload.SessionID
	if sessionID == "" {
		sessionID = sharedhandler.ReadPortalSessionCookie(r)
	}
	result, err := h.service.ConfirmSession(r.Context(), sessionID, payload.Accept, payload.TrustDevice)
	if err != nil {
		authnapi.WriteKnown(w, err)
		return
	}
	if !payload.Accept {
		sharedhandler.ClearPortalSessionCookie(w, r)
	} else {
		sharedhandler.WritePortalSessionCookie(w, r, result.Session.ID)
	}
	sharedweb.JSON(w, http.StatusOK, result)
}

func (h *Handler) VerifyMFA(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		SessionID   string `json:"sessionId"`
		Method      string `json:"method"`
		Code        string `json:"code"`
		TrustDevice bool   `json:"trustDevice"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		authnapi.Write(w, http.StatusBadRequest, authnapi.CodeInvalidJSONBody, "invalid JSON body")
		return
	}
	sessionID := payload.SessionID
	if sessionID == "" {
		sessionID = sharedhandler.ReadPortalSessionCookie(r)
	}
	result, err := h.service.VerifyMFA(r.Context(), sessionID, payload.Method, payload.Code, payload.TrustDevice)
	if err != nil {
		authnapi.WriteKnown(w, err)
		return
	}
	sharedhandler.WritePortalSessionCookie(w, r, result.Session.ID)
	sharedweb.JSON(w, http.StatusOK, result)
}

func (h *Handler) CreateMFAChallenge(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		SessionID string `json:"sessionId"`
		Method    string `json:"method"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		authnapi.Write(w, http.StatusBadRequest, authnapi.CodeInvalidJSONBody, "invalid JSON body")
		return
	}
	sessionID := payload.SessionID
	if sessionID == "" {
		sessionID = sharedhandler.ReadPortalSessionCookie(r)
	}
	challenge, demoCode, err := h.service.RequestMFAChallenge(r.Context(), sessionID, payload.Method)
	if err != nil {
		authnapi.WriteKnown(w, err)
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{
		"challenge": challenge,
		"demoCode":  demoCode,
	})
}

func (h *Handler) EnrollTOTP(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		UserID        string `json:"userId"`
		ApplicationID string `json:"applicationId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		authnapi.Write(w, http.StatusBadRequest, authnapi.CodeInvalidJSONBody, "invalid JSON body")
		return
	}
	var (
		result any
		err    error
	)
	if identity, ok := sharedhandler.AccessTokenIdentityFromRequest(r); ok && identity.User != nil {
		targetUserID, allowed := sharedhandler.CurrentUserIDOrTarget(identity, payload.UserID)
		if !allowed {
			authnapi.Write(w, http.StatusForbidden, authnapi.CodeForbidden, "organization management role is required")
			return
		}
		if targetUserID != identity.User.ID {
			allowed, err = h.service.CanManageUser(r.Context(), identity.User.Roles, targetUserID)
			if err != nil {
				authnapi.WriteKnown(w, err)
				return
			}
			if !allowed {
				authnapi.Write(w, http.StatusForbidden, authnapi.CodeForbidden, "organization management role is required")
				return
			}
		}
		result, err = h.service.EnrollTOTP(r.Context(), targetUserID, payload.ApplicationID)
	} else {
		result, err = h.service.EnrollTOTP(r.Context(), payload.UserID, payload.ApplicationID)
	}
	if err != nil {
		authnapi.WriteKnown(w, err)
		return
	}
	sharedweb.JSON(w, http.StatusOK, result)
}

func (h *Handler) EnrollPortalTOTP(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ApplicationID string `json:"applicationId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		authnapi.Write(w, http.StatusBadRequest, authnapi.CodeInvalidJSONBody, "invalid JSON body")
		return
	}
	result, err := h.service.EnrollCurrentUserTOTP(r.Context(), sharedhandler.ReadPortalSessionCookie(r), payload.ApplicationID)
	if err != nil {
		authnapi.WriteKnown(w, err)
		return
	}
	sharedweb.JSON(w, http.StatusOK, result)
}

func (h *Handler) VerifyTOTPEnrollment(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		UserID       string `json:"userId"`
		EnrollmentID string `json:"enrollmentId"`
		Code         string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		authnapi.Write(w, http.StatusBadRequest, authnapi.CodeInvalidJSONBody, "invalid JSON body")
		return
	}
	var err error
	if identity, ok := sharedhandler.AccessTokenIdentityFromRequest(r); ok && identity.User != nil {
		targetUserID, allowed := sharedhandler.CurrentUserIDOrTarget(identity, payload.UserID)
		if !allowed {
			authnapi.Write(w, http.StatusForbidden, authnapi.CodeForbidden, "organization management role is required")
			return
		}
		if targetUserID != identity.User.ID {
			allowed, err = h.service.CanManageUser(r.Context(), identity.User.Roles, targetUserID)
			if err != nil {
				authnapi.WriteKnown(w, err)
				return
			}
			if !allowed {
				authnapi.Write(w, http.StatusForbidden, authnapi.CodeForbidden, "organization management role is required")
				return
			}
		}
		err = h.service.VerifyTOTPEnrollment(r.Context(), targetUserID, payload.EnrollmentID, payload.Code)
	} else {
		err = h.service.VerifyTOTPEnrollment(r.Context(), payload.UserID, payload.EnrollmentID, payload.Code)
	}
	if err != nil {
		authnapi.WriteKnown(w, err)
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"verified": true})
}

func (h *Handler) VerifyPortalTOTPEnrollment(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		EnrollmentID string `json:"enrollmentId"`
		Code         string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		authnapi.Write(w, http.StatusBadRequest, authnapi.CodeInvalidJSONBody, "invalid JSON body")
		return
	}
	if err := h.service.VerifyCurrentUserTOTPEnrollment(r.Context(), sharedhandler.ReadPortalSessionCookie(r), payload.EnrollmentID, payload.Code); err != nil {
		authnapi.WriteKnown(w, err)
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"verified": true})
}

func (h *Handler) GenerateRecoveryCodes(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		UserID string `json:"userId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		authnapi.Write(w, http.StatusBadRequest, authnapi.CodeInvalidJSONBody, "invalid JSON body")
		return
	}
	var (
		codes []string
		err   error
	)
	if identity, ok := sharedhandler.AccessTokenIdentityFromRequest(r); ok && identity.User != nil {
		targetUserID, allowed := sharedhandler.CurrentUserIDOrTarget(identity, payload.UserID)
		if !allowed {
			authnapi.Write(w, http.StatusForbidden, authnapi.CodeForbidden, "organization management role is required")
			return
		}
		if targetUserID != identity.User.ID {
			allowed, err = h.service.CanManageUser(r.Context(), identity.User.Roles, targetUserID)
			if err != nil {
				authnapi.WriteKnown(w, err)
				return
			}
			if !allowed {
				authnapi.Write(w, http.StatusForbidden, authnapi.CodeForbidden, "organization management role is required")
				return
			}
		}
		codes, err = h.service.GenerateRecoveryCodes(r.Context(), targetUserID)
	} else {
		codes, err = h.service.GenerateRecoveryCodes(r.Context(), payload.UserID)
	}
	if err != nil {
		authnapi.WriteKnown(w, err)
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"codes": codes})
}

func (h *Handler) GeneratePortalRecoveryCodes(w http.ResponseWriter, r *http.Request) {
	codes, err := h.service.GenerateCurrentUserRecoveryCodes(r.Context(), sharedhandler.ReadPortalSessionCookie(r))
	if err != nil {
		authnapi.WriteKnown(w, err)
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{"codes": codes})
}

func (h *Handler) ResetUKID(w http.ResponseWriter, r *http.Request) {
	if identity, ok := sharedhandler.AccessTokenIdentityFromRequest(r); ok && identity.User != nil {
		ukid, err := h.service.ResetUserUKID(r.Context(), identity.User.ID)
		if err != nil {
			authnapi.WriteKnown(w, err)
			return
		}
		sharedweb.JSON(w, http.StatusOK, map[string]any{"reset": true, "ukid": ukid})
		return
	}
	authnapi.Write(w, http.StatusForbidden, authnapi.CodeForbidden, "user context is required")
}
