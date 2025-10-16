package role

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stretchr/testify/assert"
)

func TestIsDefaultRole(t *testing.T) {
	traits := &storage.Traits{}
	traits.SetOrigin(storage.Traits_DEFAULT)
	defaultRoleWithTraits := &storage.Role{}
	defaultRoleWithTraits.SetName(accesscontrol.Admin)
	defaultRoleWithTraits.SetTraits(traits)
	defaultRoleWithoutTraits := &storage.Role{}
	defaultRoleWithoutTraits.SetName(accesscontrol.Admin)
	nonDefaultRole := &storage.Role{}
	nonDefaultRole.SetName("some-random-role")

	assert.True(t, IsDefaultRole(defaultRoleWithTraits))
	assert.True(t, IsDefaultRole(defaultRoleWithoutTraits))
	assert.False(t, IsDefaultRole(nonDefaultRole))
}

func TestIsDefaultAccessScope(t *testing.T) {
	traits := &storage.Traits{}
	traits.SetOrigin(storage.Traits_DEFAULT)
	defaultAccessScopeWithTraits := &storage.SimpleAccessScope{}
	defaultAccessScopeWithTraits.SetId(AccessScopeIncludeAll.GetId())
	defaultAccessScopeWithTraits.SetTraits(traits)
	defaultAccessScopeWithoutTraits := &storage.SimpleAccessScope{}
	defaultAccessScopeWithoutTraits.SetId(AccessScopeIncludeAll.GetId())
	nonDefaultAccessScope := &storage.SimpleAccessScope{}
	nonDefaultAccessScope.SetId("some-random-access-scope")

	assert.True(t, IsDefaultAccessScope(defaultAccessScopeWithTraits))
	assert.True(t, IsDefaultAccessScope(defaultAccessScopeWithoutTraits))
	assert.False(t, IsDefaultAccessScope(nonDefaultAccessScope))
}

func TestIsDefaultPermissionSet(t *testing.T) {
	traits := &storage.Traits{}
	traits.SetOrigin(storage.Traits_DEFAULT)
	defaultPermissionSetWithTraits := &storage.PermissionSet{}
	defaultPermissionSetWithTraits.SetName(accesscontrol.Admin)
	defaultPermissionSetWithTraits.SetTraits(traits)
	defaultPermissionSetWithoutTraits := &storage.PermissionSet{}
	defaultPermissionSetWithoutTraits.SetName(accesscontrol.Admin)
	nonDefaultPermissionSet := &storage.PermissionSet{}
	nonDefaultPermissionSet.SetName("some-random-permission-set")

	assert.True(t, IsDefaultPermissionSet(defaultPermissionSetWithTraits))
	assert.True(t, IsDefaultPermissionSet(defaultPermissionSetWithoutTraits))
	assert.False(t, IsDefaultPermissionSet(nonDefaultPermissionSet))
}
