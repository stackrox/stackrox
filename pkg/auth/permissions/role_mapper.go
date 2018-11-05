package permissions

import "github.com/stackrox/rox/generated/api/v1"

// A RoleMapper returns the role corresponding to an identifier
// obtained from a token.
type RoleMapper interface {
	Role(id string) *v1.Role
}
