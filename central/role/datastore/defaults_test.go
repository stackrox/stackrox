package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_getDefaultPermissionSets(t *testing.T) {
	defaults := getDefaultPermissionSets()
	assert.Equal(t, len(accesscontrol.DefaultPermissionSetIDs)+1, len(defaults))
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

func TestConsoleProxyDefaultPermissionSetConstraints(t *testing.T) {
	for _, permissionSet := range append(getDefaultPermissionSets(), getDefaultPermissionSet(accesscontrol.ConsoleProxy)) {
		if permissionSet.GetName() != accesscontrol.ConsoleProxy {
			continue
		}
		require.Equal(t, storage.AccessConstraint_ACCESS_CONSTRAINT_READ, permissionSet.GetAccessConstraint())
		require.Equal(t, storage.ScopeConstraint_SCOPE_CONSTRAINT_CLUSTER_NAMESPACE, permissionSet.GetScopeConstraint())
	}
}
