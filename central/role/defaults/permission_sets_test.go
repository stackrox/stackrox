package defaults

import (
	"testing"

	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestIsDefaultPermissionSet(t *testing.T) {
	defaultPermissionSetWithTraits := &storage.PermissionSet{Name: Admin,
		Traits: &storage.Traits{Origin: storage.Traits_DEFAULT}}
	defaultPermissionSetWithoutTraits := &storage.PermissionSet{Name: Admin}
	nonDefaultPermissionSet := &storage.PermissionSet{Name: "some-random-permission-set"}

	assert.True(t, IsDefaultPermissionSet(defaultPermissionSetWithTraits))
	assert.True(t, IsDefaultPermissionSet(defaultPermissionSetWithoutTraits))
	assert.False(t, IsDefaultPermissionSet(nonDefaultPermissionSet))
}

func TestAnalystPermSetDoesNotContainAdministration(t *testing.T) {
	analystPermSet := GetDefaultPermissionSet(Analyst)
	// Analyst is one of the default roles.
	assert.NotNil(t, analystPermSet)

	// Contains all resources except one.
	assert.Len(t, analystPermSet.GetResourceToAccess(), len(resources.ListAll())-1)
	// Does not contain Administration resource.
	for resource := range analystPermSet.GetResourceToAccess() {
		assert.NotEqual(t, resource, resources.Administration.GetResource())
	}
}
