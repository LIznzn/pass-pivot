package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"gorm.io/gorm"

	authservice "pass-pivot/internal/server/auth/service"
	coreservice "pass-pivot/internal/server/core/service"
	authnapi "pass-pivot/internal/server/shared/authnapi"
	sharedhandler "pass-pivot/internal/server/shared/handler"
	sharedweb "pass-pivot/internal/server/shared/web"
)

type authorizeInteractionResponse struct {
	Action             string                   `json:"action"`
	RedirectTarget     string                   `json:"redirectTarget,omitempty"`
	Stage              string                   `json:"stage,omitempty"`
	SessionRef         string                   `json:"sessionRef,omitempty"`
	SecondFactorMethod string                   `json:"secondFactorMethod,omitempty"`
	MFAOptions         []string                 `json:"mfaOptions,omitempty"`
	Target             *coreservice.LoginTarget `json:"target,omitempty"`
	CurrentUser        *authorizeCurrentUser    `json:"currentUser,omitempty"`
}

type authorizeCurrentUser struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phoneNumber"`
}

func (h *OIDCHandler) QueryMetadataAPI(w http.ResponseWriter, r *http.Request) {
	result, err := h.oidc.MetadataByIssuer(r.Context())
	if err != nil {
		authnapi.WriteKnown(w, err)
		return
	}
	sharedweb.JSON(w, http.StatusOK, result)
}

func (h *OIDCHandler) QueryKeysAPI(w http.ResponseWriter, r *http.Request) {
	keys, err := h.oidc.JWKSByIssuer(r.Context())
	if err != nil {
		authnapi.WriteKnown(w, err)
		return
	}
	sharedweb.JSON(w, http.StatusOK, keys)
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
		SkipAccountSelection bool  `json:"skipAccountSelection"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		authnapi.Write(w, http.StatusBadRequest, authnapi.CodeInvalidJSONBody, "invalid JSON body")
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
		authnapi.WriteKnown(w, err)
		return
	}
	if redirectError != "" {
		sharedweb.JSON(w, http.StatusOK, authorizeInteractionResponse{
			Action:         "redirect",
			RedirectTarget: redirectError,
		})
		return
	}
	target, err := h.platform.GetLoginTarget(r.Context(), in.ClientID)
	if err != nil {
		authnapi.WriteKnown(w, err)
		return
	}
	sessionID := in.SessionID
	if sessionID == "" {
		sessionID = sharedhandler.ReadPortalSessionCookie(r)
	}
	pendingChallenge := ""
	if sessionID == "" {
		pendingChallenge = sharedhandler.ReadPendingLoginChallengeCookie(r)
		sessionID = pendingChallenge
	}
	if sessionID != "" {
		if session, err := h.oidc.GetSession(r.Context(), sessionID); err == nil {
			if pendingChallenge != "" {
				sharedhandler.ClearPendingLoginChallengeCookie(w, r)
			}
			switch session.State {
			case "authenticated":
				if payload.SkipAccountSelection {
					in.SessionID = sessionID
					redirectTarget, redirectErr := h.oidc.BuildAuthorizationRedirect(r.Context(), in)
					if redirectErr != nil {
						authnapi.WriteKnown(w, redirectErr)
						return
					}
					sharedweb.JSON(w, http.StatusOK, authorizeInteractionResponse{
						Action:         "redirect",
						RedirectTarget: redirectTarget,
					})
					return
				}
				user, _, userErr := h.oidc.GetSessionUser(r.Context(), sessionID)
				if userErr != nil {
					authnapi.WriteKnown(w, userErr)
					return
				}
				sharedweb.JSON(w, http.StatusOK, authorizeInteractionResponse{
					Action: "render",
					Stage:  "account",
					Target: target,
					CurrentUser: &authorizeCurrentUser{
						ID:          user.ID,
						Username:    user.Username,
						Name:        user.Name,
						Email:       user.Email,
						PhoneNumber: user.PhoneNumber,
					},
				})
				return
			case "confirmation_required":
				mfaOptions, mfaErr := h.oidc.AvailableMFAMethodsForSession(r.Context(), sessionID)
				if mfaErr != nil {
					authnapi.WriteKnown(w, mfaErr)
					return
				}
				sharedweb.JSON(w, http.StatusOK, authorizeInteractionResponse{
					Action:             "render",
					Stage:              "confirmation",
					SessionRef:         session.LoginChallenge,
					SecondFactorMethod: session.SecondFactorMethod,
					MFAOptions:         mfaOptions,
					Target:             target,
				})
				return
			case "mfa_required":
				mfaOptions, mfaErr := h.oidc.AvailableMFAMethodsForSession(r.Context(), sessionID)
				if mfaErr != nil {
					authnapi.WriteKnown(w, mfaErr)
					return
				}
				sharedweb.JSON(w, http.StatusOK, authorizeInteractionResponse{
					Action:             "render",
					Stage:              "mfa",
					SessionRef:         session.LoginChallenge,
					SecondFactorMethod: session.SecondFactorMethod,
					MFAOptions:         mfaOptions,
					Target:             target,
				})
				return
			}
		}
	}
	if strings.EqualFold(in.Prompt, "none") {
		sharedweb.JSON(w, http.StatusOK, authorizeInteractionResponse{
			Action:         "redirect",
			RedirectTarget: redirectErrorOrDefault(redirectError, in.RedirectURI, in.State),
		})
		return
	}
	sharedweb.JSON(w, http.StatusOK, authorizeInteractionResponse{
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
		authnapi.Write(w, http.StatusBadRequest, authnapi.CodeInvalidJSONBody, "invalid JSON body")
		return
	}
	audience := requestIssuer(r) + "/auth/token"
	switch strings.TrimSpace(payload.GrantType) {
	case "authorization_code_pkce":
		authnapi.WriteKnown(w, errors.New("unsupported grant_type: use authorization_code with code_verifier for PKCE"))
		return
	case "code":
		authnapi.WriteKnown(w, errors.New("unsupported grant_type: OAuth requires grant_type=authorization_code"))
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
			authnapi.WriteKnown(w, err)
			return
		}
		sharedweb.JSON(w, http.StatusOK, authservice.BuildStandardTokenResponse(pair, idToken))
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
			authnapi.WriteKnown(w, err)
			return
		}
		if !coreservice.AppGrantTypesContain(app.GrantType, "client_credentials") {
			authnapi.WriteKnown(w, errors.New("client_credentials grant is not enabled for this application"))
			return
		}
		pair, err := h.auth.IssueClientCredentialTokenForApplication(r.Context(), app, payload.Scope)
		if err != nil {
			authnapi.WriteKnown(w, err)
			return
		}
		sharedweb.JSON(w, http.StatusOK, authservice.BuildStandardTokenResponse(pair, ""))
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
			authnapi.WriteKnown(w, err)
			return
		}
		sharedweb.JSON(w, http.StatusOK, authservice.BuildStandardTokenResponse(pair, idToken))
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
			authnapi.WriteKnown(w, err)
			return
		}
		if !coreservice.AppGrantTypesContain(app.GrantType, "password") {
			authnapi.WriteKnown(w, errors.New("password grant is not enabled for this application"))
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
			authnapi.WriteKnown(w, err)
			return
		}
		idToken := ""
		if coreservice.AppTokenTypesContain(app.TokenType, "id_token") {
			authTime := session.CreatedAt
			idToken, err = h.oidc.SignIDTokenForApplication(r.Context(), app.ID, *user, app.ID, payload.Scope, "", &authTime, session.ID)
			if err != nil {
				authnapi.WriteKnown(w, err)
				return
			}
		}
		sharedweb.JSON(w, http.StatusOK, authservice.BuildStandardTokenResponse(pair, idToken))
	default:
		authnapi.WriteKnown(w, errors.New("unsupported grant_type"))
	}
}

func (h *OIDCHandler) QueryUserInfoAPI(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		authnapi.WriteKnown(w, errors.New("missing bearer token"))
		return
	}
	profile, err := h.oidc.UserInfo(r.Context(), strings.TrimSpace(strings.TrimPrefix(auth, "Bearer ")))
	if err != nil {
		authnapi.WriteKnown(w, err)
		return
	}
	sharedweb.JSON(w, http.StatusOK, profile)
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
		authnapi.Write(w, http.StatusBadRequest, authnapi.CodeInvalidJSONBody, "invalid JSON body")
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
		authnapi.WriteKnown(w, err)
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{
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
		authnapi.Write(w, http.StatusBadRequest, authnapi.CodeInvalidJSONBody, "invalid JSON body")
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
		authnapi.WriteKnown(w, err)
		return
	}
	if err := h.auth.RevokeToken(r.Context(), strings.TrimSpace(payload.Token), strings.TrimSpace(payload.Reason)); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		authnapi.WriteKnown(w, err)
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{})
}

func (h *OIDCHandler) LogoutAPI(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		Reason       string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		authnapi.Write(w, http.StatusBadRequest, authnapi.CodeInvalidJSONBody, "invalid JSON body")
		return
	}
	reason := strings.TrimSpace(payload.Reason)
	if reason == "" {
		reason = "oidc_end_session"
	}
	if token := strings.TrimSpace(payload.AccessToken); token != "" {
		if err := h.auth.RevokeToken(r.Context(), token, reason); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			authnapi.WriteKnown(w, err)
			return
		}
	}
	if token := strings.TrimSpace(payload.RefreshToken); token != "" {
		if err := h.auth.RevokeToken(r.Context(), token, reason); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			authnapi.WriteKnown(w, err)
			return
		}
	}
	sharedhandler.ClearPortalSessionCookie(w, r)
	sharedweb.JSON(w, http.StatusOK, map[string]any{"logout": true})
}
