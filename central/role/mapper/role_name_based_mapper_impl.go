package mapper

import (
	"context"

	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authn"
)

// roleAttributeKey is the UserDescriptor attribute key that directly holds
// the names of the roles to be assigned to the user.
const roleAttributeKey = "role"

// NewRoleBasedMapper returns an implementation of RoleMapper that
// resolves the roles listed in the "role" attribute of the user descriptor.
func NewRoleBasedMapper(roles roleDataStore.DataStore) permissions.RoleMapper {
	return &roleNameBasedMapperImpl{
		roles: roles,
	}
}

type roleNameBasedMapperImpl struct {
	roles roleDataStore.DataStore
}

// FromUserDescriptor resolves the role names given by the "role" attribute.
// Role names that do not correspond to an existing role are ignored.
func (rm *roleNameBasedMapperImpl) FromUserDescriptor(ctx context.Context, user *permissions.UserDescriptor) ([]permissions.ResolvedRole, error) {
	resolvedRoles, _, err := permissions.GetResolvedRolesFromStore(ctx, rm.roles, user.Attributes[roleAttributeKey])
	if err != nil {
		return nil, err
	}
	return authn.FilterOutNoneRole(resolvedRoles), nil
}
