package middleware

import (
	"context"
	"net/http"
	"strings"

	"pass-pivot/internal/model"
	coreservice "pass-pivot/internal/server/core/service"
	sharedhandler "pass-pivot/internal/server/shared/handler"
	sharedweb "pass-pivot/internal/server/shared/web"
)

type AccessTokenAuthenticator interface {
	AuthenticateAccessToken(ctx context.Context, accessToken string) (*coreservice.AccessTokenIdentity, error)
}

type PrivateKeyJWTAuthenticator interface {
	ValidatePrivateKeyJWTClient(ctx context.Context, clientID, assertionType, assertion, audience string) (model.Application, error)
}

func APIClientAuthentication(platform AccessTokenAuthenticator, oidc PrivateKeyJWTAuthenticator, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requirement := classifyAPIAccess(r.URL.Path, r.Method)
		switch requirement.mode {
		case apiAuthModeAccessToken:
			handleAccessTokenClientAuthentication(platform, requirement, next).ServeHTTP(w, r)
		case apiAuthModePrivateKeyJWT:
			handlePrivateKeyJWTClientAuthentication(oidc, requirement, next).ServeHTTP(w, r)
		default:
			next.ServeHTTP(w, r)
		}
	})
}

func handleAccessTokenClientAuthentication(platform AccessTokenAuthenticator, requirement apiAccessRequirement, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accessToken := sharedhandler.BearerTokenFromRequest(r)
		if accessToken == "" {
			sharedweb.Error(w, http.StatusUnauthorized, "access token is required")
			return
		}
		identity, err := platform.AuthenticateAccessToken(r.Context(), accessToken)
		if err != nil {
			sharedweb.Error(w, http.StatusUnauthorized, err.Error())
			return
		}
		if requirement.requireUser && identity.User == nil {
			sharedweb.Error(w, http.StatusForbidden, "user context is required")
			return
		}
		ctx := sharedhandler.WithAccessTokenIdentity(r.Context(), identity)
		ctx = withAPIApplication(ctx, identity.Application)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func handlePrivateKeyJWTClientAuthentication(oidc PrivateKeyJWTAuthenticator, requirement apiAccessRequirement, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientID := strings.TrimSpace(r.Header.Get("X-PPVT-Client-ID"))
		assertionType := strings.TrimSpace(r.Header.Get("X-PPVT-Client-Assertion-Type"))
		assertion := strings.TrimSpace(r.Header.Get("X-PPVT-Client-Assertion"))
		app, err := oidc.ValidatePrivateKeyJWTClient(
			r.Context(),
			clientID,
			assertionType,
			assertion,
			requestServerIssuer(r)+requirement.privateJWTAud,
		)
		if err != nil {
			sharedweb.Error(w, http.StatusUnauthorized, err.Error())
			return
		}
		next.ServeHTTP(w, r.WithContext(withAPIApplication(r.Context(), app)))
	})
}
