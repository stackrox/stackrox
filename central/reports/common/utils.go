package common

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authn"
)

// ExtractAccessScopeRules extracts simple access scope rules from the given authenticated user identity
func ExtractAccessScopeRules(identity authn.Identity) []*storage.SimpleAccessScope_Rules {
	roles := identity.Roles()
	accessScopeRules := make([]*storage.SimpleAccessScope_Rules, 0, len(roles))
	for _, role := range roles {
		accessScopeRules = append(accessScopeRules, role.GetAccessScope().GetRules())
	}
	return accessScopeRules
}
