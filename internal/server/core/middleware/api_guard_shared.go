package middleware

import (
	"context"
	"net/http"
	"strings"

	"pass-pivot/internal/model"
)

type apiAuthMode string

const (
	apiAuthModeNone          apiAuthMode = "none"
	apiAuthModeAccessToken   apiAuthMode = "access_token"
	apiAuthModePrivateKeyJWT apiAuthMode = "private_key_jwt"
)

type apiAccessRequirement struct {
	mode          apiAuthMode
	requireUser   bool
	privateJWTAud string
}

type apiApplicationContextKey string

const apiApplicationKey apiApplicationContextKey = "ppvt_api_application"

func classifyAPIAccess(path string, method string) apiAccessRequirement {
	if method != http.MethodPost {
		return apiAccessRequirement{mode: apiAuthModeNone}
	}
	switch {
	case strings.HasPrefix(path, "/api/user/v1/"):
		return apiAccessRequirement{mode: apiAuthModeAccessToken, requireUser: true}
	case strings.HasPrefix(path, "/api/manage/v1/"):
		return apiAccessRequirement{mode: apiAuthModeAccessToken}
	case strings.HasPrefix(path, "/api/authn/v1/"):
		return apiAccessRequirement{mode: apiAuthModePrivateKeyJWT, privateJWTAud: "/api/authn"}
	case strings.HasPrefix(path, "/api/authz/v1/"):
		return apiAccessRequirement{mode: apiAuthModePrivateKeyJWT, privateJWTAud: "/api/authz"}
	default:
		return apiAccessRequirement{mode: apiAuthModeNone}
	}
}

func requestServerIssuer(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwarded != "" {
		scheme = forwarded
	}
	host := r.Host
	if forwardedHost := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); forwardedHost != "" {
		host = forwardedHost
	}
	return scheme + "://" + host
}

func withAPIApplication(ctx context.Context, application model.Application) context.Context {
	return context.WithValue(ctx, apiApplicationKey, application)
}

func apiApplicationFromContext(ctx context.Context) (model.Application, bool) {
	application, ok := ctx.Value(apiApplicationKey).(model.Application)
	return application, ok
}
