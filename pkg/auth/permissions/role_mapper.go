package permissions

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// A RoleMapper returns the role corresponding to an identifier
// obtained from a token.
type RoleMapper interface {
	FromUserDescriptor(ctx context.Context, user *UserDescriptor) ([]*ResolvedRole, error)
}

// UserDescriptor contains the necessary user information to map it to a user
type UserDescriptor struct {
	UserID     string
	Attributes map[string][]string
}

// RoleStore defines an object that provides looking up roles.
type RoleStore interface {
	GetRole(ctx context.Context, roleName string) (*storage.Role, error)
	ResolveRoles(ctx context.Context, roles []*storage.Role) ([]*ResolvedRole, error)
}

// RoleMapperFactory provides an interface for generating a role mapper for an auth provider.
type RoleMapperFactory interface {
	GetRoleMapper(authProviderID string) RoleMapper
}
