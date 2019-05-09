package permissions

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/tokens"
)

// A RoleMapper returns the role corresponding to an identifier
// obtained from a token.
type RoleMapper interface {
	FromTokenClaims(ctx context.Context, claims *tokens.Claims) (*storage.Role, error)
}

// RoleStore defines an object that provides looking up roles.
type RoleStore interface {
	GetRole(ctx context.Context, roleName string) (*storage.Role, error)
}

// RoleMapperFactory provides an interface for generating a role mapper for an auth provider.
type RoleMapperFactory interface {
	GetRoleMapper(authProviderID string) RoleMapper
}
