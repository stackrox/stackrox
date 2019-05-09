package mapper

import (
	"context"

	"github.com/stackrox/rox/central/role"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/tokens"
)

type alwaysAdminMapperImpl struct{}

// FromTokenClaims always returns admin.
func (rm *alwaysAdminMapperImpl) FromTokenClaims(ctx context.Context, claims *tokens.Claims) (*storage.Role, error) {
	return role.DefaultRolesByName[role.Admin], nil
}

// AlwaysAdminRoleMapper returns an implementation of RoleMapper that always returns the admin role.
func AlwaysAdminRoleMapper() permissions.RoleMapper {
	return &alwaysAdminMapperImpl{}
}
