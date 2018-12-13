package mapper

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/prometheus/common/log"
	groupStore "github.com/stackrox/rox/central/group/store"
	roleStore "github.com/stackrox/rox/central/role/store"
	userStore "github.com/stackrox/rox/central/user/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/set"
)

type mapperImpl struct {
	authProviderID string
	groupStore     groupStore.Store
	roleStore      roleStore.Store
	userStore      userStore.Store
}

// FromTokenClaims interprets the given claim information and converts it to a role.
func (rm *mapperImpl) FromTokenClaims(claims *tokens.Claims) (*storage.Role, error) {
	// Record the user we are creating a role for.
	rm.recordUser(claims)
	// Determine the role.
	return rm.getRole(claims)
}

func (rm *mapperImpl) recordUser(claims *tokens.Claims) {
	user := rm.createUser(claims)
	if err := rm.userStore.Upsert(user); err != nil {
		// Just log since we don't actually need the user information.
		log.Errorf("unable to log user: %s", proto.MarshalTextString(user))
	}
}

func (rm *mapperImpl) getRole(claims *tokens.Claims) (*storage.Role, error) {
	// Get the groups for the user.
	groups, err := rm.groupStore.Walk(rm.authProviderID, claims.ExternalUser.Attributes)
	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return nil, fmt.Errorf("no usable groups for user")
	}

	// Load the roles that apply to the user based on their groups.
	roles, err := rm.rolesForGroups(groups)
	if err != nil {
		return nil, fmt.Errorf("failure to load roles: %s", err)
	}

	// Generate a role that has the highest permissions of all roles the user has.
	return permissions.NewUnionRole(roles), nil
}

func (rm *mapperImpl) rolesForGroups(groups []*storage.Group) ([]*storage.Role, error) {
	// Get the roles in all of the groups.
	roleNameSet := set.NewStringSet()
	for _, group := range groups {
		roleNameSet.Add(group.GetRoleName())
	}
	roleNamesSlice := roleNameSet.AsSlice()
	if len(roleNamesSlice) == 0 {
		return nil, fmt.Errorf("no roles can be found for user")
	}
	return rm.roleStore.GetRolesBatch(roleNamesSlice)
}

// Helpers
//////////

func (rm *mapperImpl) createUser(claims *tokens.Claims) *storage.User {
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
