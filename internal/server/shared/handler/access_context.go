package handler

import (
	"context"
	"net/http"
	"strings"

	coreservice "pass-pivot/internal/server/core/service"
)

type accessContextKey struct{}

func WithAccessTokenIdentity(ctx context.Context, identity *coreservice.AccessTokenIdentity) context.Context {
	return context.WithValue(ctx, accessContextKey{}, identity)
}

func AccessTokenIdentityFromContext(ctx context.Context) (*coreservice.AccessTokenIdentity, bool) {
	identity, ok := ctx.Value(accessContextKey{}).(*coreservice.AccessTokenIdentity)
	return identity, ok && identity != nil
}

func AccessTokenIdentityFromRequest(r *http.Request) (*coreservice.AccessTokenIdentity, bool) {
	return AccessTokenIdentityFromContext(r.Context())
}

func BearerTokenFromRequest(r *http.Request) string {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(auth, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
}

func HasRole(identity *coreservice.AccessTokenIdentity, role string) bool {
	if identity == nil || identity.User == nil {
		return false
	}
	for _, item := range identity.User.Roles {
		if item == role {
			return true
		}
	}
	return false
}

func CurrentUserIDOrTarget(identity *coreservice.AccessTokenIdentity, targetUserID string) (string, bool) {
	if identity == nil || identity.User == nil {
		return "", false
	}
	targetUserID = strings.TrimSpace(targetUserID)
	if targetUserID == "" || targetUserID == identity.User.ID {
		return identity.User.ID, true
	}
	return targetUserID, HasAnyOrganizationManagementRole(identity)
}
