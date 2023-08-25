package common

import (
	rolePkg "github.com/stackrox/rox/central/role"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/search"
)

// ExtractAccessScopeRules extracts simple access scope rules from the given authenticated user identity
// For the purpose of vulnerability reporting, nil/empty list of rules would mean allow access to all clusters/namespaces.
func ExtractAccessScopeRules(identity authn.Identity) []*storage.SimpleAccessScope_Rules {
	roles := identity.Roles()
	accessScopeRulesList := make([]*storage.SimpleAccessScope_Rules, 0, len(roles))
	for _, role := range roles {
		// Note: This mirrors the scope resolution logic in `func (c *authorizerDataCache) computeEffectiveAccessScope(...)`
		//  defined in central/sac/authorizer/builtin_scoped_authorizer.go .
		//  The reason for doing this is that the system access scope "AccessScopeIncludeAll" which includes all clusters/namespaces
		//  has rules = nil. However, for any other access scope, nil/empty scope rules translate to an "exclude all" access scope.
		accessScope := role.GetAccessScope()
		if accessScope == nil {
			accessScopeRulesList = append(accessScopeRulesList, rolePkg.AccessScopeExcludeAll.GetRules())
		} else if accessScope.Id == rolePkg.AccessScopeIncludeAll.Id {
			return nil
		} else if accessScope.Id == rolePkg.AccessScopeExcludeAll.Id || accessScope.GetRules() == nil {
			// nil/empty rules in a non-nil access scope means exclude all clusters/namespaces
			// if the access scope is not same as rolePkg.AccessScopeIncludeAll
			accessScopeRulesList = append(accessScopeRulesList, rolePkg.AccessScopeExcludeAll.GetRules())
		} else {
			accessScopeRulesList = append(accessScopeRulesList, accessScope.GetRules())
		}
	}
	if len(accessScopeRulesList) == 0 {
		accessScopeRulesList = append(accessScopeRulesList, rolePkg.AccessScopeExcludeAll.GetRules())
	}
	return accessScopeRulesList
}

// IsV1ReportConfig returns true if the given config belongs to reporting version 1.0
func IsV1ReportConfig(config *storage.ReportConfiguration) bool {
	return config.GetResourceScope() == nil
}

// IsV2ReportConfig returns true if the given config belongs to reporting version 2.0
func IsV2ReportConfig(config *storage.ReportConfiguration) bool {
	return config.GetResourceScope() != nil
}

// WithoutV2ReportConfigs adds a conjunction query to exclude v2 report configs
func WithoutV2ReportConfigs(query *v1.Query) *v1.Query {
	return search.ConjunctionQuery(
		query,
		search.NewQueryBuilder().AddExactMatches(search.CollectionID, "").ProtoQuery())
}

// WithoutV1ReportConfigs adds a conjunction query to exclude v1 report configs
func WithoutV1ReportConfigs(query *v1.Query) *v1.Query {
	return search.ConjunctionQuery(
		query,
		search.NewQueryBuilder().AddExactMatches(search.EmbeddedCollectionID, "").ProtoQuery())
}
