package common

import (
	rolePkg "github.com/stackrox/rox/central/role"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authn"
)

// ExtractAccessScopeRules extracts simple access scope rules from the given authenticated user identity
// For the purpose of vulnerability reporting, nil/empty list of rules would mean allow access to all clusters/namespaces.
func ExtractAccessScopeRules(identity authn.Identity) []*storage.SimpleAccessScope_Rules {
	roles := identity.Roles()
	accessScopeRules := make([]*storage.SimpleAccessScope_Rules, 0, len(roles))
	for _, role := range roles {
		// Note: This mirrors the scope resolution logic in `func (c *authorizerDataCache) computeEffectiveAccessScope(...)`
		//  defined in central/sac/authorizer/builtin_scoped_authorizer.go .
		//  The reason for doing this that the system access scope "AccessScopeIncludeAll" which includes all clusters/namespaces
		//  has rules = nil. However, for any other access scope, nil/empty scope rules translate to an "exclude all" access scope.
		accessScope := role.GetAccessScope()
		if accessScope == nil || accessScope.Id == rolePkg.AccessScopeExcludeAll.Id {
			accessScopeRules = append(accessScopeRules, rolePkg.AccessScopeExcludeAll.GetRules())
		} else if accessScope.Id == rolePkg.AccessScopeIncludeAll.Id {
			return nil
		} else if accessScope.GetRules() == nil {
			// nil/empty rules in a non-nil access scope rolePkg.AccessScopeIncludeAll.Id
			// would mean exclude all scope
			accessScopeRules = append(accessScopeRules, rolePkg.AccessScopeExcludeAll.GetRules())
		} else {
			accessScopeRules = append(accessScopeRules, accessScope.GetRules())
		}
	}
	if len(accessScopeRules) == 0 {
		accessScopeRules = append(accessScopeRules, rolePkg.AccessScopeExcludeAll.GetRules())
	}
	return accessScopeRules
}
