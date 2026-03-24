package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"gorm.io/gorm"

	authservice "pass-pivot/internal/server/auth/service"
	coreservice "pass-pivot/internal/server/core/service"
	authnapi "pass-pivot/internal/server/shared/authnapi"
	sharedhandler "pass-pivot/internal/server/shared/handler"
	sharedweb "pass-pivot/internal/server/shared/web"
)

type authorizeInteractionResponse struct {
	Action             string                                 `json:"action"`
	RedirectTarget     string                                 `json:"redirectTarget,omitempty"`
	Stage              string                                 `json:"stage,omitempty"`
	FlowType           string                                 `json:"flowType,omitempty"`
	ResultStatus       string                                 `json:"resultStatus,omitempty"`
	ResultMessage      string                                 `json:"resultMessage,omitempty"`
	SessionRef         string                                 `json:"sessionRef,omitempty"`
	SecondFactorMethod string                                 `json:"secondFactorMethod,omitempty"`
	MFAOptions         []string                               `json:"mfaOptions,omitempty"`
	Target             *coreservice.LoginTarget               `json:"target,omitempty"`
	CurrentUser        *authorizeCurrentUser                  `json:"currentUser,omitempty"`
	Captcha            *authservice.AuthorizeCaptchaBootstrap `json:"captcha,omitempty"`
}

type authorizeInteractionRequest struct {
	SessionID            string `json:"sessionId"`
	FlowType             string `json:"flowType"`
	UserCode             string `json:"userCode"`
	ClientID             string `json:"clientId"`
	ResponseType         string `json:"responseType"`
	RedirectURI          string `json:"redirectUri"`
	Scope                string `json:"scope"`
	State                string `json:"state"`
	Nonce                string `json:"nonce"`
	CodeChallenge        string `json:"codeChallenge"`
	CodeChallengeMethod  string `json:"codeChallengeMethod"`
	Prompt               string `json:"prompt"`
	SkipAccountSelection bool   `json:"skipAccountSelection"`
}

type authorizeContextResponse struct {
	Action             string                                 `json:"action"`
	RedirectTarget     string                                 `json:"redirectTarget,omitempty"`
	Stage              string                                 `json:"stage,omitempty"`
	FlowType           string                                 `json:"flowType,omitempty"`
	ResultStatus       string                                 `json:"resultStatus,omitempty"`
	ResultMessage      string                                 `json:"resultMessage,omitempty"`
	Error              string                                 `json:"error,omitempty"`
	AuthorizeReturnURL string                                 `json:"authorizeReturnUrl,omitempty"`
	Target             *coreservice.LoginTarget               `json:"target,omitempty"`
	CurrentUser        *authorizeCurrentUser                  `json:"currentUser,omitempty"`
	ApplicationID      string                                 `json:"applicationId,omitempty"`
	SecondFactorMethod string                                 `json:"secondFactorMethod,omitempty"`
	MFAOptions         []authorizeUIMethodOption              `json:"mfaOptions,omitempty"`
	Captcha            *authservice.AuthorizeCaptchaBootstrap `json:"captcha,omitempty"`
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
	var payload authorizeInteractionRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		authnapi.Write(w, http.StatusBadRequest, authnapi.CodeInvalidJSONBody, "invalid JSON body")
		return
	}
	response, err := h.queryAuthorizeInteractionFromPayload(w, r, payload)
	if err != nil {
		authnapi.WriteKnown(w, err)
		return
	}
	sharedweb.JSON(w, http.StatusOK, response)
}

func (h *OIDCHandler) QueryAuthorizeContextAPI(w http.ResponseWriter, r *http.Request) {
	var payload authorizeInteractionRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		authnapi.Write(w, http.StatusBadRequest, authnapi.CodeInvalidJSONBody, "invalid JSON body")
		return
	}
	response, err := h.queryAuthorizeInteractionFromCore(w, r, payload)
	if err != nil {
		authnapi.WriteKnown(w, err)
		return
	}
	if response.Action == "redirect" {
		sharedweb.JSON(w, http.StatusOK, authorizeContextResponse{
			Action:         response.Action,
			RedirectTarget: response.RedirectTarget,
		})
		return
	}
	if response.Target == nil && response.Stage != "done" {
		sharedweb.JSON(w, http.StatusBadRequest, authorizeContextResponse{
			Action: "error",
			Error:  "login target is not available",
		})
		return
	}
	applicationID := ""
	if response.Target != nil {
		applicationID = response.Target.ApplicationID
	}
	sharedweb.JSON(w, http.StatusOK, authorizeContextResponse{
		Action:             "render",
		Stage:              response.Stage,
		FlowType:           response.FlowType,
		ResultStatus:       response.ResultStatus,
		ResultMessage:      response.ResultMessage,
		AuthorizeReturnURL: buildAuthorizeReturnURLFromPayload(payload),
		Target:             response.Target,
		CurrentUser:        response.CurrentUser,
		ApplicationID:      applicationID,
		SecondFactorMethod: response.SecondFactorMethod,
		MFAOptions:         buildAuthorizeMFAOptions(response.MFAOptions),
		Captcha:            response.Captcha,
	})
}

func (h *OIDCHandler) queryAuthorizeInteractionFromCore(w http.ResponseWriter, r *http.Request, payload authorizeInteractionRequest) (authorizeInteractionResponse, error) {
	body, err := h.callAuthnAPI(w, r, "/api/authn/v1/authorize/interaction/query", map[string]any{
		"sessionId":            strings.TrimSpace(payload.SessionID),
		"flowType":             strings.TrimSpace(payload.FlowType),
		"userCode":             strings.TrimSpace(payload.UserCode),
		"clientId":             strings.TrimSpace(payload.ClientID),
		"responseType":         strings.TrimSpace(payload.ResponseType),
		"redirectUri":          strings.TrimSpace(payload.RedirectURI),
		"scope":                strings.TrimSpace(payload.Scope),
		"state":                strings.TrimSpace(payload.State),
		"nonce":                strings.TrimSpace(payload.Nonce),
		"codeChallenge":        strings.TrimSpace(payload.CodeChallenge),
		"codeChallengeMethod":  strings.TrimSpace(payload.CodeChallengeMethod),
		"prompt":               strings.TrimSpace(payload.Prompt),
		"skipAccountSelection": payload.SkipAccountSelection,
	})
	if err != nil {
		return authorizeInteractionResponse{}, err
	}
	var response authorizeInteractionResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return authorizeInteractionResponse{}, err
	}
	if strings.TrimSpace(response.Stage) == "" && response.Action != "redirect" {
		response.Stage = "login"
	}
	return response, nil
}

func buildAuthorizeReturnURLFromPayload(in authorizeInteractionRequest) string {
	query := url.Values{}
	if value := strings.TrimSpace(in.SessionID); value != "" {
		query.Set("ppvt_session_id", value)
	}
	if value := strings.TrimSpace(in.FlowType); value != "" {
		query.Set("type", value)
	}
	if value := strings.TrimSpace(in.UserCode); value != "" {
		query.Set("user_code", value)
	}
	if value := strings.TrimSpace(in.ClientID); value != "" {
		query.Set("client_id", value)
	}
	if value := strings.TrimSpace(in.ResponseType); value != "" {
		query.Set("response_type", value)
	}
	if value := strings.TrimSpace(in.RedirectURI); value != "" {
		query.Set("redirect_uri", value)
	}
	if value := strings.TrimSpace(in.Scope); value != "" {
		query.Set("scope", value)
	}
	if value := strings.TrimSpace(in.State); value != "" {
		query.Set("state", value)
	}
	if value := strings.TrimSpace(in.Nonce); value != "" {
		query.Set("nonce", value)
	}
	if value := strings.TrimSpace(in.CodeChallenge); value != "" {
		query.Set("code_challenge", value)
	}
	if value := strings.TrimSpace(in.CodeChallengeMethod); value != "" {
		query.Set("code_challenge_method", value)
	}
	if value := strings.TrimSpace(in.Prompt); value != "" {
		query.Set("prompt", value)
	}
	u := &url.URL{Path: "/auth/authorize"}
	u.RawQuery = query.Encode()
	return u.String()
}

func (h *OIDCHandler) CreateAuthorizeSessionAPI(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		OrganizationID        string `json:"organizationId"`
		ApplicationID         string `json:"applicationId"`
		Identifier            string `json:"identifier"`
		Secret                string `json:"secret"`
		CaptchaProvider       string `json:"captchaProvider"`
		CaptchaToken          string `json:"captchaToken"`
		CaptchaChallengeToken string `json:"captchaChallengeToken"`
		CaptchaAnswer         string `json:"captchaAnswer"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		authnapi.Write(w, http.StatusBadRequest, authnapi.CodeInvalidJSONBody, "invalid JSON body")
		return
	}
	captchaToken := strings.TrimSpace(payload.CaptchaToken)
	if strings.TrimSpace(payload.CaptchaProvider) == "default" {
		captchaToken = coreservice.BuildDefaultCaptchaResponseToken(strings.TrimSpace(payload.CaptchaChallengeToken), strings.TrimSpace(payload.CaptchaAnswer))
	}
	body, err := h.callAuthnAPI(w, r, "/api/authn/v1/session/create", map[string]any{
		"organizationId":  strings.TrimSpace(payload.OrganizationID),
		"applicationId":   strings.TrimSpace(payload.ApplicationID),
		"identifier":      strings.TrimSpace(payload.Identifier),
		"secret":          payload.Secret,
		"captchaProvider": strings.TrimSpace(payload.CaptchaProvider),
		"captchaToken":    captchaToken,
	})
	if err != nil {
		authnapi.Write(w, http.StatusBadRequest, "", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

func (h *OIDCHandler) CompleteDeviceAuthorizationAPI(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		UserCode string `json:"userCode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		authnapi.Write(w, http.StatusBadRequest, authnapi.CodeInvalidJSONBody, "invalid JSON body")
		return
	}
	sessionID := strings.TrimSpace(sharedhandler.ReadAnyAuthSessionCookie(r))
	if sessionID == "" {
		authnapi.Write(w, http.StatusUnauthorized, "", "session is not authenticated")
		return
	}
	if _, err := h.oidc.ApproveDeviceAuthorization(r.Context(), strings.TrimSpace(payload.UserCode), sessionID); err != nil {
		authnapi.Write(w, http.StatusBadRequest, "", err.Error())
		return
	}
	user, _, err := h.oidc.GetSessionUser(r.Context(), sessionID)
	if err != nil {
		authnapi.Write(w, http.StatusBadRequest, "", err.Error())
		return
	}
	sharedweb.JSON(w, http.StatusOK, map[string]any{
		"ok": true,
		"currentUser": authorizeCurrentUser{
			ID:          user.ID,
			Username:    user.Username,
			Name:        user.Name,
			Email:       user.Email,
			PhoneNumber: user.PhoneNumber,
		},
	})
}

func (h *OIDCHandler) ConfirmAuthorizeSessionAPI(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Accept      bool `json:"accept"`
		TrustDevice bool `json:"trustDevice"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		authnapi.Write(w, http.StatusBadRequest, authnapi.CodeInvalidJSONBody, "invalid JSON body")
		return
	}
	body, err := h.callAuthnAPI(w, r, "/api/authn/v1/session/confirm", map[string]any{
		"sessionId":   resolveLoginSessionRef(r),
		"accept":      payload.Accept,
		"trustDevice": payload.TrustDevice,
	})
	if err != nil {
		authnapi.Write(w, http.StatusBadRequest, "", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

func (h *OIDCHandler) VerifyAuthorizeMFAAPI(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Method      string `json:"method"`
		Code        string `json:"code"`
		TrustDevice bool   `json:"trustDevice"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		authnapi.Write(w, http.StatusBadRequest, authnapi.CodeInvalidJSONBody, "invalid JSON body")
		return
	}
	body, err := h.callAuthnAPI(w, r, "/api/authn/v1/session/verify_mfa", map[string]any{
		"sessionId":   resolveLoginSessionRef(r),
		"method":      strings.TrimSpace(payload.Method),
		"code":        strings.TrimSpace(payload.Code),
		"trustDevice": payload.TrustDevice,
	})
	if err != nil {
		authnapi.Write(w, http.StatusBadRequest, "", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

func (h *OIDCHandler) queryAuthorizeInteractionFromPayload(w http.ResponseWriter, r *http.Request, payload authorizeInteractionRequest) (authorizeInteractionResponse, error) {
	if strings.EqualFold(strings.TrimSpace(payload.FlowType), "device_code") {
		return h.queryDeviceCodeInteraction(w, r, strings.TrimSpace(payload.UserCode))
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
		return authorizeInteractionResponse{}, err
	}
	if redirectError != "" {
		return authorizeInteractionResponse{
			Action:         "redirect",
			RedirectTarget: redirectError,
		}, nil
	}
	target, err := h.platform.GetLoginTarget(r.Context(), in.ClientID)
	if err != nil {
		return authorizeInteractionResponse{}, err
	}
	sessionID := in.SessionID
	if sessionID == "" {
		sessionID = sharedhandler.ReadAuthSessionCookie(r, target.OrganizationID)
	}
	pendingChallenge := ""
	if sessionID == "" {
		pendingChallenge = sharedhandler.ReadPendingLoginChallengeCookie(r)
		if pendingChallenge != "" {
			sessionID = pendingChallenge
		} else {
			sharedhandler.ClearPendingLoginChallengeCookie(w, r)
		}
	}
	if sessionID != "" {
		if _, session, err := h.oidc.ValidateSessionForApplication(r.Context(), sessionID, target.ApplicationID); err == nil {
			if pendingChallenge != "" {
				sharedhandler.ClearPendingLoginChallengeCookie(w, r)
			}
			switch session.State {
			case "authenticated":
				if payload.SkipAccountSelection {
					in.SessionID = sessionID
					redirectTarget, redirectErr := h.oidc.BuildAuthorizationRedirect(r.Context(), in)
					if redirectErr != nil {
						return authorizeInteractionResponse{}, redirectErr
					}
					return authorizeInteractionResponse{
						Action:         "redirect",
						RedirectTarget: redirectTarget,
					}, nil
				}
				user, _, userErr := h.oidc.ValidateSessionForApplication(r.Context(), sessionID, target.ApplicationID)
				if userErr != nil {
					return authorizeInteractionResponse{}, userErr
				}
				return authorizeInteractionResponse{
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
				}, nil
			case "confirmation_required":
				mfaOptions, mfaErr := h.oidc.AvailableMFAMethodsForSession(r.Context(), sessionID)
				if mfaErr != nil {
					return authorizeInteractionResponse{}, mfaErr
				}
				return authorizeInteractionResponse{
					Action:             "render",
					Stage:              "confirmation",
					SessionRef:         session.LoginChallenge,
					SecondFactorMethod: session.SecondFactorMethod,
					MFAOptions:         mfaOptions,
					Target:             target,
				}, nil
			case "mfa_required":
				mfaOptions, mfaErr := h.oidc.AvailableMFAMethodsForSession(r.Context(), sessionID)
				if mfaErr != nil {
					return authorizeInteractionResponse{}, mfaErr
				}
				return authorizeInteractionResponse{
					Action:             "render",
					Stage:              "mfa",
					SessionRef:         session.LoginChallenge,
					SecondFactorMethod: session.SecondFactorMethod,
					MFAOptions:         mfaOptions,
					Target:             target,
				}, nil
			}
		}
	}
	if strings.EqualFold(in.Prompt, "none") {
		return authorizeInteractionResponse{
			Action:         "redirect",
			RedirectTarget: redirectErrorOrDefault(redirectError, in.RedirectURI, in.State),
		}, nil
	}
	return authorizeInteractionResponse{
		Action:  "render",
		Stage:   "login",
		Target:  target,
		Captcha: h.mustBuildAuthorizeCaptcha(r.Context(), target.OrganizationID),
	}, nil
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

func (h *OIDCHandler) RevokeSessionAPI(w http.ResponseWriter, r *http.Request) {
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
		reason = "auth_session_revoked"
	}

	sessionID := strings.TrimSpace(sharedhandler.ReadAnyAuthSessionCookie(r))
	if sessionID != "" {
		if _, err := h.oidc.EndSession(r.Context(), sessionID, "", "", reason); err != nil {
			authnapi.WriteKnown(w, err)
			return
		}
	} else {
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
	}
	sharedhandler.ClearAllAuthSessionCookies(w, r)
	sharedweb.JSON(w, http.StatusOK, map[string]any{"revoked": true})
}
