package permissions

import (
	"context"
)

// A RoleMapper returns the role corresponding to an identifier
// obtained from a token.
type RoleMapper interface {
	FromUserDescriptor(ctx context.Context, user *UserDescriptor) ([]ResolvedRole, error)
}

// UserDescriptor contains the necessary user information to map it to a user
type UserDescriptor struct {
	UserID     string
	Attributes map[string][]string
}

// RoleStore defines an object that provides looking up roles.
type RoleStore interface {
	GetAndResolveRole(ctx context.Context, name string) (ResolvedRole, error)
	GetAllResolvedRoles(ctx context.Context) ([]ResolvedRole, error)
}

// RoleMapperFactory provides an interface for generating a role mapper for an auth provider.
type RoleMapperFactory interface {
	GetRoleMapper(authProviderID string) RoleMapper
}
