package middleware

import (
	"errors"
	"net/http"
	"strings"

	apiauthz "pass-pivot/internal/server/core/api/authz"
	sharedhandler "pass-pivot/internal/server/shared/handler"
	sharedhttp "pass-pivot/internal/server/shared/web"
)

func APIPolicyAuthorization(authz *apiauthz.AuthzService, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requirement := classifyAPIAccess(r.URL.Path, r.Method)
		if requirement.mode == apiAuthModeNone {
			next.ServeHTTP(w, r)
			return
		}
		application, ok := apiApplicationFromContext(r.Context())
		if !ok || strings.TrimSpace(application.ID) == "" {
			sharedhttp.Error(w, http.StatusUnauthorized, "application context is required")
			return
		}
		if err := ensureApplicationAllowed(r, authz, application.ID); err != nil {
			sharedhttp.Error(w, http.StatusForbidden, err.Error())
			return
		}
		if requirement.requireUser {
			identity, ok := sharedhandler.AccessTokenIdentityFromContext(r.Context())
			if !ok || identity.User == nil {
				sharedhttp.Error(w, http.StatusForbidden, "user context is required")
				return
			}
			userDecision, err := authz.CheckPolicy(r.Context(), "user", identity.User.ID, r.Method, r.URL.Path)
			if err != nil {
				sharedhttp.Error(w, http.StatusForbidden, err.Error())
				return
			}
			if !userDecision.Allowed {
				sharedhttp.Error(w, http.StatusForbidden, "user is not allowed to call this api")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func ensureApplicationAllowed(r *http.Request, authz *apiauthz.AuthzService, applicationID string) error {
	decision, err := authz.CheckPolicy(r.Context(), "application", applicationID, r.Method, r.URL.Path)
	if err != nil {
		return err
	}
	if !decision.Allowed {
		return errors.New("application is not allowed to call this api")
	}
	return nil
}
