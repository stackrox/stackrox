package role

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stretchr/testify/assert"
)

func TestIsDefaultRole(t *testing.T) {
	defaultRoleWithTraits := &storage.Role{Name: accesscontrol.Admin, Traits: &storage.Traits{Origin: storage.Traits_DEFAULT}}
	defaultRoleWithoutTraits := &storage.Role{Name: accesscontrol.Admin}
	nonDefaultRole := &storage.Role{Name: "some-random-role"}

	assert.True(t, IsDefaultRole(defaultRoleWithTraits))
	assert.True(t, IsDefaultRole(defaultRoleWithoutTraits))
	assert.False(t, IsDefaultRole(nonDefaultRole))
}

func TestIsDefaultAccessScope(t *testing.T) {
	defaultAccessScopeWithTraits := &storage.SimpleAccessScope{Id: AccessScopeIncludeAll.GetId(),
		Traits: &storage.Traits{Origin: storage.Traits_DEFAULT}}
	defaultAccessScopeWithoutTraits := &storage.SimpleAccessScope{Id: AccessScopeIncludeAll.GetId()}
	nonDefaultAccessScope := &storage.SimpleAccessScope{Id: "some-random-access-scope"}

	assert.True(t, IsDefaultAccessScope(defaultAccessScopeWithTraits))
	assert.True(t, IsDefaultAccessScope(defaultAccessScopeWithoutTraits))
	assert.False(t, IsDefaultAccessScope(nonDefaultAccessScope))
}

func TestIsDefaultPermissionSet(t *testing.T) {
	defaultPermissionSetWithTraits := &storage.PermissionSet{Name: accesscontrol.Admin,
		Traits: &storage.Traits{Origin: storage.Traits_DEFAULT}}
	defaultPermissionSetWithoutTraits := &storage.PermissionSet{Name: accesscontrol.Admin}
	nonDefaultPermissionSet := &storage.PermissionSet{Name: "some-random-permission-set"}

	assert.True(t, IsDefaultPermissionSet(defaultPermissionSetWithTraits))
	assert.True(t, IsDefaultPermissionSet(defaultPermissionSetWithoutTraits))
	assert.False(t, IsDefaultPermissionSet(nonDefaultPermissionSet))
}
