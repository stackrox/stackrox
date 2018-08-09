package tokenbased

import "github.com/stackrox/rox/pkg/auth/permissions"

// A RoleMapper returns the role corresponding to an identifier
// obtained from a token.
type RoleMapper interface {
	Role(id string) (role permissions.Role, exists bool)
}

// An IdentityParser knows how to parse API metadata (gRPC metadata,
// or HTTP headers) into a token-based identity.
//go:generate mockery -name=IdentityParser
type IdentityParser interface {
	// Parse parses API metadata into an identity, with the help of the provider RoleMapper.
	// It returns an error if it couldn't obtain an identity (for whatever reason).
	Parse(headers map[string][]string, roleMapper RoleMapper) (Identity, error)
}
