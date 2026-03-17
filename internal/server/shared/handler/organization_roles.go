package handler

import (
	"sort"
	"strings"

	coreservice "pass-pivot/internal/server/core/service"
)

const (
	organizationRoleOwner = "owner"
	organizationRoleAdmin = "admin"
)

func OrganizationOwnerRoleName(organizationID string) string {
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return ""
	}
	return "organization:" + organizationID + ":" + organizationRoleOwner
}

func OrganizationAdminRoleName(organizationID string) string {
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return ""
	}
	return "organization:" + organizationID + ":" + organizationRoleAdmin
}

func parseOrganizationScopedRole(role string) (organizationID string, scope string, ok bool) {
	role = strings.TrimSpace(role)
	parts := strings.Split(role, ":")
	if len(parts) != 3 || parts[0] != "organization" {
		return "", "", false
	}
	if strings.TrimSpace(parts[1]) == "" {
		return "", "", false
	}
	if parts[2] != organizationRoleOwner && parts[2] != organizationRoleAdmin {
		return "", "", false
	}
	return parts[1], parts[2], true
}

func RolesManagedOrganizationIDs(roles []string) []string {
	seen := map[string]struct{}{}
	items := make([]string, 0)
	for _, role := range roles {
		organizationID, _, ok := parseOrganizationScopedRole(role)
		if !ok {
			continue
		}
		if _, exists := seen[organizationID]; exists {
			continue
		}
		seen[organizationID] = struct{}{}
		items = append(items, organizationID)
	}
	sort.Strings(items)
	return items
}

func RolesContainOrganizationManagementRole(roles []string, organizationID string) bool {
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return false
	}
	for _, role := range roles {
		candidateOrganizationID, _, ok := parseOrganizationScopedRole(role)
		if ok && candidateOrganizationID == organizationID {
			return true
		}
	}
	return false
}

func RolesContainAnyOrganizationManagementRole(roles []string) bool {
	for _, role := range roles {
		if _, _, ok := parseOrganizationScopedRole(role); ok {
			return true
		}
	}
	return false
}

func ManagedOrganizationIDs(identity *coreservice.AccessTokenIdentity) []string {
	if identity == nil || identity.User == nil {
		return nil
	}
	return RolesManagedOrganizationIDs(identity.User.Roles)
}

func HasOrganizationManagementRole(identity *coreservice.AccessTokenIdentity, organizationID string) bool {
	if identity == nil || identity.User == nil {
		return false
	}
	return RolesContainOrganizationManagementRole(identity.User.Roles, organizationID)
}

func RolesContainOrganizationOwnerRole(roles []string, organizationID string) bool {
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return false
	}
	ownerRoleName := OrganizationOwnerRoleName(organizationID)
	if ownerRoleName == "" {
		return false
	}
	for _, role := range roles {
		if strings.TrimSpace(role) == ownerRoleName {
			return true
		}
	}
	return false
}

func HasOrganizationOwnerRole(identity *coreservice.AccessTokenIdentity, organizationID string) bool {
	if identity == nil || identity.User == nil {
		return false
	}
	return RolesContainOrganizationOwnerRole(identity.User.Roles, organizationID)
}

func HasAnyOrganizationManagementRole(identity *coreservice.AccessTokenIdentity) bool {
	if identity == nil || identity.User == nil {
		return false
	}
	return RolesContainAnyOrganizationManagementRole(identity.User.Roles)
}
