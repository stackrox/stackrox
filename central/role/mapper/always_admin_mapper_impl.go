package mapper

import (
	"context"

	"github.com/stackrox/rox/central/role"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
)

type alwaysAdminMapperImpl struct{}

// FromUserDescriptor always returns admin.
func (rm *alwaysAdminMapperImpl) FromUserDescriptor(ctx context.Context, user *permissions.UserDescriptor) (*storage.Role, error) {
	return role.DefaultRolesByName[role.Admin], nil
}

// AlwaysAdminRoleMapper returns an implementation of RoleMapper that always returns the admin role.
func AlwaysAdminRoleMapper() permissions.RoleMapper {
	return &alwaysAdminMapperImpl{}
}
