package tokenbased

import (
	"github.com/stackrox/rox/pkg/auth/permissions"
	"google.golang.org/grpc/metadata"
)

// A RoleMapper returns the role corresponding to an identifier
// obtained from a token.
type RoleMapper interface {
	Role(id string) permissions.Role
}

// An IdentityParser knows how to parse API metadata (gRPC metadata,
// or HTTP headers) into a token-based identity.
//go:generate mockery -name=IdentityParser
type IdentityParser interface {
	// Parse parses API metadata into an identity, with the help of the provider RoleMapper.
	// It returns an error if it couldn't obtain an identity (for whatever reason).
	Parse(md metadata.MD, roleMapper RoleMapper) (Identity, error)
}
