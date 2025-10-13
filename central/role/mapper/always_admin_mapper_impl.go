package mapper

import (
	"context"

	roleDatastore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/utils"
)

type alwaysAdminMapperImpl struct {
	adminRole permissions.ResolvedRole
}

// FromUserDescriptor always returns admin.
func (rm *alwaysAdminMapperImpl) FromUserDescriptor(_ context.Context, _ *permissions.UserDescriptor) ([]permissions.ResolvedRole, error) {
	return []permissions.ResolvedRole{rm.adminRole}, nil
}

// AlwaysAdminRoleMapper returns an implementation of RoleMapper that always returns the admin role.
func AlwaysAdminRoleMapper() permissions.RoleMapper {
	// It is only valid to store a reference to the Admin role because it is
	// immutable, otherwise we would fetch it on every FromUserDescriptor call.
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())
	adminRole, err := roleDatastore.Singleton().GetAndResolveRole(ctx, accesscontrol.Admin)
	utils.CrashOnError(err)

	return &alwaysAdminMapperImpl{
		adminRole: adminRole,
	}
}
