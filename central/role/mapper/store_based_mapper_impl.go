package mapper

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	groupDataStore "github.com/stackrox/rox/central/group/datastore"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	userDataStore "github.com/stackrox/rox/central/user/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
)

var (
	log = logging.LoggerForModule()
)

type storeBasedMapperImpl struct {
	authProviderID string
	groups         groupDataStore.DataStore
	roles          roleDataStore.DataStore
	users          userDataStore.DataStore
}

func (rm *storeBasedMapperImpl) FromUserDescriptor(ctx context.Context, user *permissions.UserDescriptor) ([]permissions.ResolvedRole, error) {
	rm.recordUser(ctx, user)
	return rm.getRoles(ctx, user)
}

func (rm *storeBasedMapperImpl) recordUser(ctx context.Context, descriptor *permissions.UserDescriptor) {
	user := rm.createUser(descriptor)

	// Telemetry logic: add the first time logging in users to the group of
	// other players like central and fleet manager under the tenant group, so
	// that the users share the common tenant properties like organization ID
	// available for analytics purposes:
	if existing, _ := rm.users.GetUser(ctx, user.GetId()); existing == nil {
		addUserToTenantGroup(user)
	}

	if err := rm.users.Upsert(ctx, user); err != nil {
		// Just log since we don't actually need the user information.
		log.Errorf("unable to log user: %s: %v", proto.MarshalTextString(user), err)
	}
}

func (rm *storeBasedMapperImpl) getRoles(ctx context.Context, user *permissions.UserDescriptor) ([]permissions.ResolvedRole, error) {
	// Get the groups for the user.
	groups, err := rm.groups.Walk(ctx, rm.authProviderID, user.Attributes)
	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return nil, nil
	}

	// Load the roles that apply to the user based on their groups.
	roles, err := rm.rolesForGroups(ctx, groups)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load roles for user with id: %q", user.UserID)
	}

	return roles, nil
}

func (rm *storeBasedMapperImpl) rolesForGroups(ctx context.Context, groups []*storage.Group) ([]permissions.ResolvedRole, error) {
	// Get the role names in all of the groups.
	roleNameSet := set.NewStringSet()
	for _, group := range groups {
		roleNameSet.Add(group.GetRoleName())
	}
	if roleNameSet.IsEmpty() {
		return nil, errors.New("no roles can be found for user")
	}

	// Load the roles individually because we want to ignore missing roles.
	var resolvedRoles = make([]permissions.ResolvedRole, 0, roleNameSet.Cardinality())
	for roleName := range roleNameSet {
		role, err := rm.roles.GetAndResolveRole(ctx, roleName)
		if err != nil {
			return nil, errors.Wrapf(err, "resolving role %q", roleName)
		}
		if role != nil && role.GetRoleName() != authn.NoneRole {
			resolvedRoles = append(resolvedRoles, role)
		}
	}
	return resolvedRoles, nil
}

// Helpers
//////////

func (rm *storeBasedMapperImpl) createUser(descriptor *permissions.UserDescriptor) *storage.User {
	// Create a user.
	user := &storage.User{
		Id:             descriptor.UserID,
		AuthProviderId: rm.authProviderID,
	}
	addAttributesToUser(user, descriptor.Attributes)
	return user
}

func addAttributesToUser(user *storage.User, attributes map[string][]string) {
	if len(attributes) == 0 {
		return
	}
	for k, vs := range attributes {
		for _, v := range vs {
			user.Attributes = append(user.Attributes, &storage.UserAttribute{Key: k, Value: v})
		}
	}
}
