package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stretchr/testify/assert"
)

func Test_getDefaultPermissionSets(t *testing.T) {
	defaults := getDefaultPermissionSets()
	assert.Equal(t, len(defaults), len(accesscontrol.DefaultPermissionSetIDs))
	for _, ps := range defaults {
		assert.True(t, accesscontrol.IsDefaultPermissionSet(ps.GetName()))
		assert.Equal(t, storage.Traits_DEFAULT, ps.GetTraits().GetOrigin())
		assert.Error(t, verifyNotDefaultPermissionSet(ps))
		assert.False(t, declarativeconfig.CanModifyResource(context.Background(), ps))
	}
}

func Test_getDefaultRoles(t *testing.T) {
	defaults := getDefaultRoles()
	assert.Equal(t, len(defaults), len(accesscontrol.DefaultRoleNames))
	for _, role := range defaults {
		assert.True(t, accesscontrol.IsDefaultRole(role.GetName()))
		assert.Equal(t, storage.Traits_DEFAULT, role.GetTraits().GetOrigin())
		assert.Error(t, verifyNotDefaultRole(role))
		assert.False(t, declarativeconfig.CanModifyResource(context.Background(), role))
	}
}
