package authn

import (
	"github.com/stackrox/rox/pkg/auth/permissions"
)

// NoneRole identifies default role with no permissions.
const NoneRole = "None"

// FilterOutNoneRole returns a copy of the input role list from which
// occurrences of the None role were removed.
func FilterOutNoneRole(source []permissions.ResolvedRole) []permissions.ResolvedRole {
	target := make([]permissions.ResolvedRole, 0, len(source))
	for _, role := range source {
		if role != nil && role.GetRoleName() != NoneRole {
			target = append(target, role)
		}
	}
	return target
}
