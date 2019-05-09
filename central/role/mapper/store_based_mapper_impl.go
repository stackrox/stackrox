package mapper

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/prometheus/common/log"
	groupDataStore "github.com/stackrox/rox/central/group/datastore"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	userDataStore "github.com/stackrox/rox/central/user/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/set"
)

type storeBasedMapperImpl struct {
	authProviderID string
	groups         groupDataStore.DataStore
	roles          roleDataStore.DataStore
	users          userDataStore.DataStore
}

// FromTokenClaims interprets the given claim information and converts it to a role.
func (rm *storeBasedMapperImpl) FromTokenClaims(ctx context.Context, claims *tokens.Claims) (*storage.Role, error) {
	// Record the user we are creating a role for.
	rm.recordUser(ctx, claims)
	// Determine the role.
	return rm.getRole(ctx, claims)
}

func (rm *storeBasedMapperImpl) recordUser(ctx context.Context, claims *tokens.Claims) {
	user := rm.createUser(claims)
	if err := rm.users.Upsert(ctx, user); err != nil {
		// Just log since we don't actually need the user information.
		log.Errorf("unable to log user: %s", proto.MarshalTextString(user))
	}
}

func (rm *storeBasedMapperImpl) getRole(ctx context.Context, claims *tokens.Claims) (*storage.Role, error) {
	// Get the groups for the user.
	groups, err := rm.groups.Walk(ctx, rm.authProviderID, claims.ExternalUser.Attributes)
	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return nil, fmt.Errorf("no usable groups for user")
	}

	// Load the roles that apply to the user based on their groups.
	roles, err := rm.rolesForGroups(ctx, groups)
	if err != nil {
		return nil, errors.Wrap(err, "failure to load roles")
	}

	// Generate a role that has the highest permissions of all roles the user has.
	return permissions.NewUnionRole(roles), nil
}

func (rm *storeBasedMapperImpl) rolesForGroups(ctx context.Context, groups []*storage.Group) ([]*storage.Role, error) {
	// Get the role names in all of the groups.
	roleNameSet := set.NewStringSet()
	for _, group := range groups {
		roleNameSet.Add(group.GetRoleName())
	}
	if roleNameSet.Cardinality() == 0 {
		return nil, fmt.Errorf("no roles can be found for user")
	}
	roleNamesSlice := roleNameSet.AsSlice()

	// Load the roles (need to load individually because we want to ignore missing roles)
	var roles = make([]*storage.Role, 0, len(roleNamesSlice))
	for _, roleName := range roleNamesSlice {
		role, err := rm.roles.GetRole(ctx, roleName)
		if err != nil {
			return nil, err
		}
		if role != nil {
			roles = append(roles, role)
		}
	}
	return roles, nil
}

// Helpers
//////////

func (rm *storeBasedMapperImpl) createUser(claims *tokens.Claims) *storage.User {
	// Create a user.
	user := &storage.User{
		Id:             claims.ExternalUser.UserID,
		AuthProviderId: rm.authProviderID,
	}
	addAttributesToUser(user, claims.ExternalUser.Attributes)
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
