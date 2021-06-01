package mapper

import (
	"context"

	"github.com/stackrox/rox/central/role"
	roleDatastore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/utils"
)

type alwaysAdminMapperImpl struct {
	adminRole *storage.Role
}

// FromUserDescriptor always returns admin.
func (rm *alwaysAdminMapperImpl) FromUserDescriptor(ctx context.Context, user *permissions.UserDescriptor) ([]*storage.Role, error) {
	return []*storage.Role{rm.adminRole}, nil
}

// AlwaysAdminRoleMapper returns an implementation of RoleMapper that always returns the admin role.
func AlwaysAdminRoleMapper() permissions.RoleMapper {
	// It is only valid to store a reference to the Admin role because it is
	// immutable, otherwise we would fetch it on every FromUserDescriptor call.
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())
	adminRole, err := roleDatastore.Singleton().GetRole(ctx, role.Admin)
	utils.Must(err)

	return &alwaysAdminMapperImpl{
		adminRole: adminRole,
	}
}
