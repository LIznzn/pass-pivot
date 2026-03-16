package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"gorm.io/gorm"

	authservice "pass-pivot/internal/server/auth/service"
	coreservice "pass-pivot/internal/server/core/service"
	sharedhandler "pass-pivot/internal/server/shared/handler"
	sharedhttp "pass-pivot/internal/server/shared/web"
)

type authorizeInteractionResponse struct {
	Action             string                   `json:"action"`
	RedirectTarget     string                   `json:"redirectTarget,omitempty"`
	Stage              string                   `json:"stage,omitempty"`
	SecondFactorMethod string                   `json:"secondFactorMethod,omitempty"`
	Target             *coreservice.LoginTarget `json:"target,omitempty"`
}

func (h *OIDCHandler) QueryMetadataAPI(w http.ResponseWriter, r *http.Request) {
	result, err := h.oidc.MetadataByIssuer(r.Context())
	if err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedhttp.JSON(w, http.StatusOK, result)
}

func (h *OIDCHandler) QueryKeysAPI(w http.ResponseWriter, r *http.Request) {
	keys, err := h.oidc.JWKSByIssuer(r.Context())
	if err != nil {
		sharedhttp.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	sharedhttp.JSON(w, http.StatusOK, keys)
}

func (h *OIDCHandler) QueryAuthorizeInteractionAPI(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		SessionID           string `json:"sessionId"`
		ClientID            string `json:"clientId"`
		ResponseType        string `json:"responseType"`
		RedirectURI         string `json:"redirectUri"`
		Scope               string `json:"scope"`
		State               string `json:"state"`
		Nonce               string `json:"nonce"`
		CodeChallenge       string `json:"codeChallenge"`
		CodeChallengeMethod string `json:"codeChallengeMethod"`
		Prompt              string `json:"prompt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	in := authservice.StandardAuthorizeRequest{
		SessionID:           strings.TrimSpace(payload.SessionID),
		ClientID:            strings.TrimSpace(payload.ClientID),
		ResponseType:        strings.TrimSpace(payload.ResponseType),
		RedirectURI:         strings.TrimSpace(payload.RedirectURI),
		Scope:               strings.TrimSpace(payload.Scope),
		State:               strings.TrimSpace(payload.State),
		Nonce:               strings.TrimSpace(payload.Nonce),
		CodeChallenge:       strings.TrimSpace(payload.CodeChallenge),
		CodeChallengeMethod: strings.TrimSpace(payload.CodeChallengeMethod),
		Prompt:              strings.TrimSpace(payload.Prompt),
	}
	_, redirectError, err := h.oidc.ValidateAuthorizationRequest(r.Context(), in)
	if err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if redirectError != "" {
		sharedhttp.JSON(w, http.StatusOK, authorizeInteractionResponse{
			Action:         "redirect",
			RedirectTarget: redirectError,
		})
		return
	}
	target, err := h.platform.GetLoginTarget(r.Context(), in.ClientID)
	if err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sessionID := in.SessionID
	if sessionID == "" {
		sessionID = sharedhandler.ReadPortalSessionCookie(r)
	}
	if sessionID != "" {
		if session, err := h.oidc.GetSession(r.Context(), sessionID); err == nil {
			switch session.State {
			case "authenticated":
				in.SessionID = sessionID
				redirectTarget, redirectErr := h.oidc.BuildAuthorizationRedirect(r.Context(), in)
				if redirectErr != nil {
					sharedhttp.Error(w, http.StatusBadRequest, redirectErr.Error())
					return
				}
				sharedhttp.JSON(w, http.StatusOK, authorizeInteractionResponse{
					Action:         "redirect",
					RedirectTarget: redirectTarget,
				})
				return
			case "confirmation_required":
				sharedhttp.JSON(w, http.StatusOK, authorizeInteractionResponse{
					Action:             "render",
					Stage:              "confirmation",
					SecondFactorMethod: session.SecondFactorMethod,
					Target:             target,
				})
				return
			case "mfa_required":
				sharedhttp.JSON(w, http.StatusOK, authorizeInteractionResponse{
					Action:             "render",
					Stage:              "mfa",
					SecondFactorMethod: session.SecondFactorMethod,
					Target:             target,
				})
				return
			}
		}
	}
	if strings.EqualFold(in.Prompt, "none") {
		sharedhttp.JSON(w, http.StatusOK, authorizeInteractionResponse{
			Action:         "redirect",
			RedirectTarget: redirectErrorOrDefault(redirectError, in.RedirectURI, in.State),
		})
		return
	}
	sharedhttp.JSON(w, http.StatusOK, authorizeInteractionResponse{
		Action: "render",
		Stage:  "login",
		Target: target,
	})
}

func (h *OIDCHandler) ExchangeTokenAPI(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		GrantType           string `json:"grantType"`
		ClientID            string `json:"clientId"`
		ClientSecret        string `json:"clientSecret"`
		ClientAssertionType string `json:"clientAssertionType"`
		ClientAssertion     string `json:"clientAssertion"`
		Code                string `json:"code"`
		RedirectURI         string `json:"redirectUri"`
		CodeVerifier        string `json:"codeVerifier"`
		RefreshToken        string `json:"refreshToken"`
		Username            string `json:"username"`
		Password            string `json:"password"`
		Scope               string `json:"scope"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	audience := requestIssuer(r) + "/auth/token"
	switch strings.TrimSpace(payload.GrantType) {
	case "authorization_code_pkce":
		sharedhttp.Error(w, http.StatusBadRequest, "unsupported grant_type: use authorization_code with code_verifier for PKCE")
		return
	case "code":
		sharedhttp.Error(w, http.StatusBadRequest, "unsupported grant_type: OAuth requires grant_type=authorization_code")
		return
	case "authorization_code":
		pair, idToken, err := h.oidc.ExchangeCode(
			r.Context(),
			audience,
			payload.ClientID,
			payload.ClientSecret,
			payload.ClientAssertionType,
			payload.ClientAssertion,
			payload.Code,
			payload.RedirectURI,
			payload.CodeVerifier,
		)
		if err != nil {
			sharedhttp.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		sharedhttp.JSON(w, http.StatusOK, authservice.BuildStandardTokenResponse(pair, idToken))
	case "client_credentials":
		app, err := h.oidc.ValidateClientAuthentication(
			r.Context(),
			payload.ClientID,
			payload.ClientSecret,
			payload.ClientAssertionType,
			payload.ClientAssertion,
			audience,
		)
		if err != nil {
			sharedhttp.Error(w, http.StatusUnauthorized, err.Error())
			return
		}
		if !coreservice.AppGrantTypesContain(app.GrantType, "client_credentials") {
			sharedhttp.Error(w, http.StatusUnauthorized, "client_credentials grant is not enabled for this application")
			return
		}
		pair, err := h.auth.IssueClientCredentialTokenForApplication(r.Context(), app, payload.Scope)
		if err != nil {
			sharedhttp.Error(w, http.StatusUnauthorized, err.Error())
			return
		}
		sharedhttp.JSON(w, http.StatusOK, authservice.BuildStandardTokenResponse(pair, ""))
	case "refresh_token":
		pair, idToken, err := h.oidc.ExchangeRefreshToken(
			r.Context(),
			audience,
			payload.ClientID,
			payload.ClientSecret,
			payload.ClientAssertionType,
			payload.ClientAssertion,
			payload.RefreshToken,
			payload.Scope,
		)
		if err != nil {
			sharedhttp.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		sharedhttp.JSON(w, http.StatusOK, authservice.BuildStandardTokenResponse(pair, idToken))
	case "password":
		app, err := h.oidc.ValidateClientAuthentication(
			r.Context(),
			payload.ClientID,
			payload.ClientSecret,
			payload.ClientAssertionType,
			payload.ClientAssertion,
			audience,
		)
		if err != nil {
			sharedhttp.Error(w, http.StatusUnauthorized, err.Error())
			return
		}
		if !coreservice.AppGrantTypesContain(app.GrantType, "password") {
			sharedhttp.Error(w, http.StatusUnauthorized, "password grant is not enabled for this application")
			return
		}
		pair, user, session, err := h.auth.IssuePasswordGrantTokenForApplication(
			r.Context(),
			app,
			payload.Username,
			payload.Password,
			payload.Scope,
			sharedhandler.NormalizeRemoteIP(sharedhandler.OriginalRemoteAddr(r)),
			sharedhandler.OriginalUserAgent(r),
		)
		if err != nil {
			sharedhttp.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		idToken := ""
		if coreservice.AppTokenTypesContain(app.TokenType, "id_token") {
			authTime := session.CreatedAt
			idToken, err = h.oidc.SignIDTokenForApplication(r.Context(), app.ID, *user, app.ID, payload.Scope, "", &authTime, session.ID)
			if err != nil {
				sharedhttp.Error(w, http.StatusBadRequest, err.Error())
				return
			}
		}
		sharedhttp.JSON(w, http.StatusOK, authservice.BuildStandardTokenResponse(pair, idToken))
	default:
		sharedhttp.Error(w, http.StatusBadRequest, "unsupported grant_type")
	}
}

func (h *OIDCHandler) QueryUserInfoAPI(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		sharedhttp.Error(w, http.StatusUnauthorized, "missing bearer token")
		return
	}
	profile, err := h.oidc.UserInfo(r.Context(), strings.TrimSpace(strings.TrimPrefix(auth, "Bearer ")))
	if err != nil {
		sharedhttp.Error(w, http.StatusUnauthorized, err.Error())
		return
	}
	sharedhttp.JSON(w, http.StatusOK, profile)
}

func (h *OIDCHandler) ValidateClientAPI(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ClientID            string `json:"clientId"`
		ClientSecret        string `json:"clientSecret"`
		ClientAssertionType string `json:"clientAssertionType"`
		ClientAssertion     string `json:"clientAssertion"`
		Audience            string `json:"audience"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	app, err := h.oidc.ValidateClientAuthentication(
		r.Context(),
		payload.ClientID,
		payload.ClientSecret,
		payload.ClientAssertionType,
		payload.ClientAssertion,
		payload.Audience,
	)
	if err != nil {
		sharedhttp.Error(w, http.StatusUnauthorized, err.Error())
		return
	}
	sharedhttp.JSON(w, http.StatusOK, map[string]any{
		"valid":           true,
		"applicationId":   app.ID,
		"applicationName": app.Name,
	})
}

func (h *OIDCHandler) RevokeTokenAPI(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ClientID            string `json:"clientId"`
		ClientSecret        string `json:"clientSecret"`
		ClientAssertionType string `json:"clientAssertionType"`
		ClientAssertion     string `json:"clientAssertion"`
		Token               string `json:"token"`
		Reason              string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if _, err := h.oidc.ValidateClientAuthentication(
		r.Context(),
		payload.ClientID,
		payload.ClientSecret,
		payload.ClientAssertionType,
		payload.ClientAssertion,
		requestIssuer(r)+"/auth/revoke",
	); err != nil {
		sharedhttp.Error(w, http.StatusUnauthorized, err.Error())
		return
	}
	if err := h.auth.RevokeToken(r.Context(), strings.TrimSpace(payload.Token), strings.TrimSpace(payload.Reason)); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		sharedhttp.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sharedhttp.JSON(w, http.StatusOK, map[string]any{})
}

func (h *OIDCHandler) LogoutAPI(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		Reason       string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sharedhttp.Error(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	reason := strings.TrimSpace(payload.Reason)
	if reason == "" {
		reason = "oidc_end_session"
	}
	if token := strings.TrimSpace(payload.AccessToken); token != "" {
		if err := h.auth.RevokeToken(r.Context(), token, reason); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			sharedhttp.Error(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if token := strings.TrimSpace(payload.RefreshToken); token != "" {
		if err := h.auth.RevokeToken(r.Context(), token, reason); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			sharedhttp.Error(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	sharedhandler.ClearPortalSessionCookie(w, r)
	sharedhttp.JSON(w, http.StatusOK, map[string]any{"logout": true})
}
