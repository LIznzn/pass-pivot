package authz

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"

	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	"gorm.io/gorm"

	internalcasbin "pass-pivot/internal/casbin"
	ppvtmodel "pass-pivot/internal/model"
	coreservice "pass-pivot/internal/server/core/service"
	sharedhandler "pass-pivot/internal/server/shared/handler"
)

type AuthzService struct {
	db    *gorm.DB
	audit *coreservice.AuditService
}

type PolicyCheckResult struct {
	Allowed         bool     `json:"allowed"`
	SubjectType     string   `json:"subjectType"`
	SubjectID       string   `json:"subjectId"`
	MatchedRole     string   `json:"matchedRole"`
	MatchedPolicyID string   `json:"matchedPolicyId"`
	MatchedPolicy   string   `json:"matchedPolicy"`
	MatchedEffect   string   `json:"matchedEffect"`
	MatchedPriority int      `json:"matchedPriority"`
	MatchedPath     string   `json:"matchedPath"`
	MatchedMethod   string   `json:"matchedMethod"`
	AvailableRoles  []string `json:"availableRoles"`
	DecisionSource  string   `json:"decisionSource"`
	Reason          string   `json:"reason"`
}

type SubjectPolicySummary struct {
	SubjectType string             `json:"subjectType"`
	SubjectID   string             `json:"subjectId"`
	Roles       []string           `json:"roles"`
	Policies    []ppvtmodel.Policy `json:"policies"`
}

func NewAuthzService(db *gorm.DB, audit *coreservice.AuditService) *AuthzService {
	return &AuthzService{db: db, audit: audit}
}

func ensureCanManageOrganization(ctx context.Context, organizationID string) error {
	identity, ok := sharedhandler.AccessTokenIdentityFromContext(ctx)
	if !ok || identity.User == nil {
		return nil
	}
	if sharedhandler.HasOrganizationManagementRole(identity, organizationID) {
		return nil
	}
	return errors.New("organization management role is required")
}

func currentActorFromContext(ctx context.Context) (string, string) {
	identity, ok := sharedhandler.AccessTokenIdentityFromContext(ctx)
	if !ok || identity.User == nil {
		return "", ""
	}
	if text := strings.TrimSpace(identity.User.Email); text != "" {
		return identity.User.ID, text
	}
	return identity.User.ID, identity.User.ID
}

func (s *AuthzService) CreateRole(ctx context.Context, role ppvtmodel.Role) (*ppvtmodel.Role, error) {
	if strings.TrimSpace(role.OrganizationID) == "" || strings.TrimSpace(role.Name) == "" {
		return nil, errors.New("organizationId and name are required")
	}
	if err := ensureCanManageOrganization(ctx, role.OrganizationID); err != nil {
		return nil, err
	}
	if !roleTypeOptions[strings.TrimSpace(role.Type)] {
		return nil, errors.New("invalid role type")
	}
	if err := s.db.WithContext(ctx).
		Where("organization_id = ? AND name = ?", role.OrganizationID, role.Name).
		First(&ppvtmodel.Role{}).Error; err == nil {
		return nil, errors.New("role name already exists")
	}
	if err := s.db.WithContext(ctx).Create(&role).Error; err != nil {
		return nil, err
	}
	actorID, actorName := currentActorFromContext(ctx)
	_ = s.audit.Record(ctx, coreservice.AuditEvent{
		OrganizationID: role.OrganizationID,
		ActorType:      "admin",
		ActorID:        actorID,
		ActorName:   actorName,
		EventType:      "role.created",
		Result:         "success",
		TargetType:     "role",
		TargetID:       role.ID,
		TargetName:  role.Name,
		Detail: map[string]any{
			"type": role.Type,
		},
	})
	return &role, nil
}

func (s *AuthzService) UpdateRole(ctx context.Context, role ppvtmodel.Role) (*ppvtmodel.Role, error) {
	if strings.TrimSpace(role.ID) == "" {
		return nil, errors.New("id is required")
	}
	if role.Type != "" && !roleTypeOptions[strings.TrimSpace(role.Type)] {
		return nil, errors.New("invalid role type")
	}
	var existing ppvtmodel.Role
	if err := s.db.WithContext(ctx).First(&existing, "id = ?", role.ID).Error; err != nil {
		return nil, err
	}
	if err := ensureCanManageOrganization(ctx, existing.OrganizationID); err != nil {
		return nil, err
	}
	nextName := coalesceString(role.Name, existing.Name)
	if err := s.db.WithContext(ctx).
		Where("organization_id = ? AND name = ? AND id <> ?", existing.OrganizationID, nextName, existing.ID).
		First(&ppvtmodel.Role{}).Error; err == nil {
		return nil, errors.New("role name already exists")
	}
	updates := map[string]any{
		"name":        nextName,
		"type":        coalesceString(role.Type, existing.Type),
		"description": coalesceString(role.Description, existing.Description),
	}
	if err := s.db.WithContext(ctx).Model(&existing).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).First(&existing, "id = ?", role.ID).Error; err != nil {
		return nil, err
	}
	actorID, actorName := currentActorFromContext(ctx)
	_ = s.audit.Record(ctx, coreservice.AuditEvent{
		OrganizationID: existing.OrganizationID,
		ActorType:      "admin",
		ActorID:        actorID,
		ActorName:   actorName,
		EventType:      "role.updated",
		Result:         "success",
		TargetType:     "role",
		TargetID:       existing.ID,
		TargetName:  existing.Name,
		Detail: map[string]any{
			"type": existing.Type,
		},
	})
	return &existing, nil
}

func (s *AuthzService) DeleteRoles(ctx context.Context, roleIDs []string) error {
	if len(roleIDs) == 0 {
		return errors.New("roleIds is required")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var roles []ppvtmodel.Role
		if err := tx.Where("id IN ?", roleIDs).Find(&roles).Error; err != nil {
			return err
		}
		if len(roles) == 0 {
			return errors.New("role not found")
		}
		for _, role := range roles {
			if err := ensureCanManageOrganization(ctx, role.OrganizationID); err != nil {
				return err
			}
		}
		if err := tx.Where("role_id IN ?", roleIDs).Delete(&ppvtmodel.Policy{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id IN ?", roleIDs).Delete(&ppvtmodel.Role{}).Error; err != nil {
			return err
		}
		for _, role := range roles {
			_ = s.audit.Record(ctx, coreservice.AuditEvent{
				OrganizationID: role.OrganizationID,
				ActorType:      "admin",
				EventType:      "role.deleted",
				Result:         "success",
				TargetType:     "role",
				TargetID:       role.ID,
			})
		}
		return nil
	})
}

func (s *AuthzService) ListRoles(ctx context.Context, organizationID string) ([]ppvtmodel.Role, error) {
	var items []ppvtmodel.Role
	query := s.db.WithContext(ctx)
	if organizationID != "" {
		if err := ensureCanManageOrganization(ctx, organizationID); err != nil {
			return nil, err
		}
		query = query.Where("organization_id = ?", organizationID)
	} else if identity, ok := sharedhandler.AccessTokenIdentityFromContext(ctx); ok && identity.User != nil {
		managedOrganizationIDs := sharedhandler.ManagedOrganizationIDs(identity)
		if len(managedOrganizationIDs) > 0 {
			query = query.Where("organization_id IN ?", managedOrganizationIDs)
		}
	}
	err := query.Order("type asc, name asc").Find(&items).Error
	return items, err
}

func (s *AuthzService) CreatePolicy(ctx context.Context, policy ppvtmodel.Policy) (*ppvtmodel.Policy, error) {
	if strings.TrimSpace(policy.OrganizationID) == "" || strings.TrimSpace(policy.RoleID) == "" || strings.TrimSpace(policy.Name) == "" {
		return nil, errors.New("organizationId, roleId and name are required")
	}
	if err := ensureCanManageOrganization(ctx, policy.OrganizationID); err != nil {
		return nil, err
	}
	var role ppvtmodel.Role
	if err := s.db.WithContext(ctx).First(&role, "id = ?", policy.RoleID).Error; err != nil {
		return nil, errors.New("role not found")
	}
	if role.OrganizationID != policy.OrganizationID {
		return nil, errors.New("role does not belong to organization")
	}
	if err := validatePolicy(policy); err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Create(&policy).Error; err != nil {
		return nil, err
	}
	actorID, actorName := currentActorFromContext(ctx)
	_ = s.audit.Record(ctx, coreservice.AuditEvent{
		OrganizationID: policy.OrganizationID,
		ActorType:      "admin",
		ActorID:        actorID,
		ActorName:   actorName,
		EventType:      "policy.created",
		Result:         "success",
		TargetType:     "policy",
		TargetID:       policy.ID,
		TargetName:  policy.Name,
		Detail: map[string]any{
			"effect":   policy.Effect,
			"priority": policy.Priority,
		},
	})
	return &policy, nil
}

func (s *AuthzService) UpdatePolicy(ctx context.Context, policy ppvtmodel.Policy) (*ppvtmodel.Policy, error) {
	if strings.TrimSpace(policy.ID) == "" {
		return nil, errors.New("id is required")
	}
	var existing ppvtmodel.Policy
	if err := s.db.WithContext(ctx).First(&existing, "id = ?", policy.ID).Error; err != nil {
		return nil, err
	}
	if err := ensureCanManageOrganization(ctx, existing.OrganizationID); err != nil {
		return nil, err
	}
	var role ppvtmodel.Role
	if err := s.db.WithContext(ctx).First(&role, "id = ?", existing.RoleID).Error; err != nil {
		return nil, errors.New("role not found")
	}
	if role.OrganizationID != existing.OrganizationID {
		return nil, errors.New("role does not belong to organization")
	}
	updated := existing
	if strings.TrimSpace(policy.Name) != "" {
		updated.Name = strings.TrimSpace(policy.Name)
	}
	if strings.TrimSpace(policy.Effect) != "" {
		updated.Effect = strings.TrimSpace(policy.Effect)
	}
	if policy.Priority != 0 {
		updated.Priority = policy.Priority
	}
	if policy.APIRules != nil {
		updated.APIRules = policy.APIRules
	}
	if err := validatePolicy(updated); err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(&existing).Updates(map[string]any{
		"name":      updated.Name,
		"effect":    updated.Effect,
		"priority":  updated.Priority,
		"api_rules": updated.APIRules,
	}).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).First(&existing, "id = ?", policy.ID).Error; err != nil {
		return nil, err
	}
	actorID, actorName := currentActorFromContext(ctx)
	_ = s.audit.Record(ctx, coreservice.AuditEvent{
		OrganizationID: existing.OrganizationID,
		ActorType:      "admin",
		ActorID:        actorID,
		ActorName:   actorName,
		EventType:      "policy.updated",
		Result:         "success",
		TargetType:     "policy",
		TargetID:       existing.ID,
		TargetName:  existing.Name,
		Detail: map[string]any{
			"effect":   existing.Effect,
			"priority": existing.Priority,
		},
	})
	return &existing, nil
}

func (s *AuthzService) DeletePolicies(ctx context.Context, policyIDs []string) error {
	if len(policyIDs) == 0 {
		return errors.New("policyIds is required")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var policies []ppvtmodel.Policy
		if err := tx.Where("id IN ?", policyIDs).Find(&policies).Error; err != nil {
			return err
		}
		for _, policy := range policies {
			if err := ensureCanManageOrganization(ctx, policy.OrganizationID); err != nil {
				return err
			}
		}
		return tx.Where("id IN ?", policyIDs).Delete(&ppvtmodel.Policy{}).Error
	})
}

func (s *AuthzService) ListPolicies(ctx context.Context, organizationID, roleID string) ([]ppvtmodel.Policy, error) {
	var items []ppvtmodel.Policy
	query := s.db.WithContext(ctx)
	if organizationID != "" {
		if err := ensureCanManageOrganization(ctx, organizationID); err != nil {
			return nil, err
		}
		query = query.Where("organization_id = ?", organizationID)
	} else if identity, ok := sharedhandler.AccessTokenIdentityFromContext(ctx); ok && identity.User != nil {
		managedOrganizationIDs := sharedhandler.ManagedOrganizationIDs(identity)
		if len(managedOrganizationIDs) > 0 {
			query = query.Where("organization_id IN ?", managedOrganizationIDs)
		}
	}
	if roleID != "" {
		query = query.Where("role_id = ?", roleID)
	}
	err := query.Order("priority asc, name asc").Find(&items).Error
	return items, err
}

func (s *AuthzService) CheckPolicy(ctx context.Context, subjectType, subjectID, method, path string) (*PolicyCheckResult, error) {
	subjectType = strings.TrimSpace(subjectType)
	subjectID = strings.TrimSpace(subjectID)
	method = strings.TrimSpace(method)
	path = strings.TrimSpace(path)
	if subjectType != "user" && subjectType != "application" {
		return nil, errors.New("subjectType must be user or application")
	}
	if subjectID == "" || method == "" || path == "" {
		return nil, errors.New("subjectId, method and path are required")
	}
	enforcer, matchIndex, rolesBySubject, err := s.loadEnforcer(ctx)
	if err != nil {
		return nil, err
	}
	casbinSubject := casbinSubjectKey(subjectType, subjectID)
	allowed, explain, err := enforcer.EnforceEx(casbinSubject, path, method)
	if err != nil {
		return nil, err
	}
	result := &PolicyCheckResult{
		Allowed:        allowed,
		SubjectType:    subjectType,
		SubjectID:      subjectID,
		AvailableRoles: rolesBySubject[casbinSubject],
		DecisionSource: "casbin",
		Reason:         "default_deny",
	}
	if len(explain) >= 6 {
		matchKey := policyMatchKey(explain[0], explain[1], explain[2], explain[3], explain[4], explain[5])
		if match, ok := matchIndex[matchKey]; ok {
			result.MatchedRole = match.RoleName
			result.MatchedPolicyID = match.PolicyID
			result.MatchedPolicy = match.PolicyName
			result.MatchedEffect = match.Effect
			result.MatchedPriority = match.Priority
			result.MatchedPath = match.Path
			result.MatchedMethod = match.Method
			result.Reason = "matched_policy"
		}
	}
	return result, nil
}

func (s *AuthzService) ListSubjectPolicies(ctx context.Context, subjectType, subjectID string) (*SubjectPolicySummary, error) {
	roles, policies, err := s.loadSubjectRolesAndPolicies(ctx, subjectType, subjectID)
	if err != nil {
		return nil, err
	}
	return &SubjectPolicySummary{
		SubjectType: subjectType,
		SubjectID:   subjectID,
		Roles:       roles,
		Policies:    policies,
	}, nil
}

type policyMatch struct {
	RoleName   string
	PolicyID   string
	PolicyName string
	Effect     string
	Priority   int
	Path       string
	Method     string
}

func (s *AuthzService) loadEnforcer(ctx context.Context) (*casbin.Enforcer, map[string]policyMatch, map[string][]string, error) {
	casbinModelDef, err := casbinmodel.NewModelFromString(internalcasbin.Model)
	if err != nil {
		return nil, nil, nil, err
	}
	enforcer, err := casbin.NewEnforcer(casbinModelDef)
	if err != nil {
		return nil, nil, nil, err
	}

	var roles []ppvtmodel.Role
	if err := s.db.WithContext(ctx).Find(&roles).Error; err != nil {
		return nil, nil, nil, err
	}
	roleByName := make(map[string]ppvtmodel.Role, len(roles))
	for _, role := range roles {
		roleByName[role.Name] = role
	}

	var users []ppvtmodel.User
	if err := s.db.WithContext(ctx).Find(&users).Error; err != nil {
		return nil, nil, nil, err
	}
	var applications []ppvtmodel.Application
	if err := s.db.WithContext(ctx).Find(&applications).Error; err != nil {
		return nil, nil, nil, err
	}

	rolesBySubject := map[string][]string{}
	for _, user := range users {
		subject := casbinSubjectKey("user", user.ID)
		for _, roleName := range user.Roles {
			role, ok := roleByName[roleName]
			if !ok || role.Type != "user" {
				continue
			}
			_, _ = enforcer.AddGroupingPolicy(subject, casbinRoleKey(role.Name))
			rolesBySubject[subject] = append(rolesBySubject[subject], role.Name)
		}
		sort.Strings(rolesBySubject[subject])
	}
	for _, application := range applications {
		subject := casbinSubjectKey("application", application.ID)
		for _, roleName := range application.Roles {
			role, ok := roleByName[roleName]
			if !ok || role.Type != "application" {
				continue
			}
			_, _ = enforcer.AddGroupingPolicy(subject, casbinRoleKey(role.Name))
			rolesBySubject[subject] = append(rolesBySubject[subject], role.Name)
		}
		sort.Strings(rolesBySubject[subject])
	}

	var policies []ppvtmodel.Policy
	if err := s.db.WithContext(ctx).Order("priority asc, name asc").Find(&policies).Error; err != nil {
		return nil, nil, nil, err
	}
	matchIndex := make(map[string]policyMatch)
	for _, policy := range policies {
		role, ok := findRoleByID(roles, policy.RoleID)
		if !ok {
			continue
		}
		for _, rule := range policy.APIRules {
			path := strings.TrimSpace(rule.Path)
			method := strings.TrimSpace(rule.Method)
			if path == "" || method == "" {
				continue
			}
			values := []string{
				intToString(policy.Priority),
				casbinRoleKey(role.Name),
				path,
				method,
				strings.TrimSpace(policy.Effect),
				strings.TrimSpace(policy.Name),
			}
			policyArgs := make([]any, 0, len(values))
			for _, value := range values {
				policyArgs = append(policyArgs, value)
			}
			_, _ = enforcer.AddPolicy(policyArgs...)
			matchIndex[policyMatchKey(values[0], values[1], values[2], values[3], values[4], values[5])] = policyMatch{
				RoleName:   role.Name,
				PolicyID:   policy.ID,
				PolicyName: policy.Name,
				Effect:     policy.Effect,
				Priority:   policy.Priority,
				Path:       path,
				Method:     method,
			}
		}
	}
	return enforcer, matchIndex, rolesBySubject, nil
}

func (s *AuthzService) loadSubjectRolesAndPolicies(ctx context.Context, subjectType, subjectID string) ([]string, []ppvtmodel.Policy, error) {
	subjectType = strings.TrimSpace(subjectType)
	subjectID = strings.TrimSpace(subjectID)
	if subjectType != "user" && subjectType != "application" {
		return nil, nil, errors.New("subjectType must be user or application")
	}
	if subjectID == "" {
		return nil, nil, errors.New("subjectId is required")
	}
	roleNames := []string{}
	organizationID := ""
	if subjectType == "user" {
		var user ppvtmodel.User
		if err := s.db.WithContext(ctx).First(&user, "id = ?", subjectID).Error; err != nil {
			return nil, nil, err
		}
		roleNames = normalizeRoleNames(user.Roles)
		organizationID = user.OrganizationID
	} else {
		var application ppvtmodel.Application
		if err := s.db.WithContext(ctx).First(&application, "id = ?", subjectID).Error; err != nil {
			return nil, nil, err
		}
		roleNames = normalizeRoleNames(application.Roles)
		var project ppvtmodel.Project
		if err := s.db.WithContext(ctx).First(&project, "id = ?", application.ProjectID).Error; err != nil {
			return nil, nil, err
		}
		organizationID = project.OrganizationID
	}
	if len(roleNames) == 0 {
		return []string{}, []ppvtmodel.Policy{}, nil
	}
	var roles []ppvtmodel.Role
	if err := s.db.WithContext(ctx).
		Where("organization_id = ? AND name IN ?", organizationID, roleNames).
		Find(&roles).Error; err != nil {
		return nil, nil, err
	}
	roleIDs := make([]string, 0, len(roles))
	filteredNames := make([]string, 0, len(roles))
	for _, role := range roles {
		if role.Type != subjectType {
			continue
		}
		roleIDs = append(roleIDs, role.ID)
		filteredNames = append(filteredNames, role.Name)
	}
	if len(roleIDs) == 0 {
		return []string{}, []ppvtmodel.Policy{}, nil
	}
	var policies []ppvtmodel.Policy
	if err := s.db.WithContext(ctx).Where("role_id IN ?", roleIDs).Order("priority asc, name asc").Find(&policies).Error; err != nil {
		return nil, nil, err
	}
	sort.Strings(filteredNames)
	return filteredNames, policies, nil
}

func validatePolicy(policy ppvtmodel.Policy) error {
	if strings.TrimSpace(policy.Name) == "" {
		return errors.New("policy name is required")
	}
	effect := strings.TrimSpace(policy.Effect)
	if effect != "allow" && effect != "deny" {
		return errors.New("policy effect must be allow or deny")
	}
	if len(policy.APIRules) == 0 {
		return errors.New("apiRules is required")
	}
	for _, rule := range policy.APIRules {
		if strings.TrimSpace(rule.Method) == "" || strings.TrimSpace(rule.Path) == "" {
			return errors.New("apiRules.method and apiRules.path are required")
		}
	}
	return nil
}

func casbinSubjectKey(subjectType, subjectID string) string {
	return strings.TrimSpace(subjectType) + ":" + strings.TrimSpace(subjectID)
}

func casbinRoleKey(roleName string) string {
	return "role:" + strings.TrimSpace(roleName)
}

func findRoleByID(roles []ppvtmodel.Role, roleID string) (ppvtmodel.Role, bool) {
	for _, role := range roles {
		if role.ID == roleID {
			return role, true
		}
	}
	return ppvtmodel.Role{}, false
}

func intToString(value int) string {
	return strconv.Itoa(value)
}

func policyMatchKey(priority, role, path, method, effect, name string) string {
	return strings.Join([]string{priority, role, path, method, effect, name}, "|")
}

var roleTypeOptions = map[string]bool{
	"user":        true,
	"application": true,
}

func normalizeRoleNames(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, item := range values {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	sort.Strings(result)
	return result
}

func coalesceString(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return fallback
}
