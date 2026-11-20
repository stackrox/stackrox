package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/role"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultPermissionSetConstraints(t *testing.T) {
	tests := map[string]struct {
		access storage.AccessConstraint
		scope  storage.ScopeConstraint
	}{
		accesscontrol.Admin:                 {},
		accesscontrol.Analyst:               {access: storage.AccessConstraint_ACCESS_CONSTRAINT_READ},
		accesscontrol.ContinuousIntegration: {},
		accesscontrol.K8sCentralProxyUser: {
			access: storage.AccessConstraint_ACCESS_CONSTRAINT_READ,
			scope:  storage.ScopeConstraint_SCOPE_CONSTRAINT_CLUSTER_NAMESPACE,
		},
		accesscontrol.NetworkGraphViewer: {
			access: storage.AccessConstraint_ACCESS_CONSTRAINT_READ,
			scope:  storage.ScopeConstraint_SCOPE_CONSTRAINT_CLUSTER_NAMESPACE,
		},
		accesscontrol.None:                            {},
		accesscontrol.SensorCreator:                   {},
		accesscontrol.VulnMgmtApprover:                {},
		accesscontrol.VulnMgmtRequester:               {},
		accesscontrol.VulnReporter:                    {},
		accesscontrol.VulnerabilityManagementConsumer: {},
		accesscontrol.VulnerabilityManagementAdmin:    {},
		accesscontrol.ConfigController:                {},
	}

	permissionSets := getDefaultPermissionSets()
	require.Len(t, permissionSets, len(tests))
	for _, permissionSet := range permissionSets {
		t.Run(permissionSet.GetName(), func(t *testing.T) {
			expected, ok := tests[permissionSet.GetName()]
			require.True(t, ok)
			assert.Equal(t, expected.access, permissionSet.GetAccessConstraint())
			assert.Equal(t, expected.scope, permissionSet.GetScopeConstraint())
			assert.NoError(t, role.ValidatePermissionSet(permissionSet))
			protoassert.Equal(t, permissionSet, getDefaultPermissionSet(permissionSet.GetName()))
		})
	}
}

func TestGetDefaultPermissionSets(t *testing.T) {
	defaults := getDefaultPermissionSets()
	require.Len(t, defaults, len(accesscontrol.DefaultPermissionSetIDs))
	for _, ps := range defaults {
		require.NotNil(t, ps)
		assert.Equal(t, accesscontrol.DefaultPermissionSetIDs[ps.GetName()], ps.GetId())
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
