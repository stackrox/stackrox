package datastore

import (
	"testing"

	"github.com/stackrox/rox/central/role"
	"github.com/stackrox/rox/generated/storage"
	permissionsUtils "github.com/stackrox/rox/pkg/auth/permissions/utils"
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
		accesscontrol.ConsoleProxy: {
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

func TestDefaultPermissionSetIdentities(t *testing.T) {
	defaults := getDefaultPermissionSets()
	require.Len(t, defaults, len(accesscontrol.DefaultPermissionSetIDs))
	for _, ps := range defaults {
		assert.Equal(t, accesscontrol.DefaultPermissionSetIDs[ps.GetName()], ps.GetId())
		assert.Equal(t, storage.Traits_DEFAULT, ps.GetTraits().GetOrigin())
		assert.Error(t, verifyNotDefaultPermissionSet(ps))
		assert.False(t, declarativeconfig.CanModifyResource(t.Context(), ps))
	}
}

func TestDefaultRoles(t *testing.T) {
	defaults := getDefaultRoles()
	assert.Len(t, defaults, len(accesscontrol.DefaultRoleNames))
	for _, defaultRole := range defaults {
		assert.True(t, accesscontrol.IsDefaultRole(defaultRole.GetName()))
		assert.Equal(t, storage.Traits_DEFAULT, defaultRole.GetTraits().GetOrigin())
		assert.Error(t, verifyNotDefaultRole(defaultRole))
		assert.False(t, declarativeconfig.CanModifyResource(t.Context(), defaultRole))
		assert.NotEqual(t, accesscontrol.DefaultPermissionSetIDs[accesscontrol.ConsoleProxy], defaultRole.GetPermissionSetId())
		assert.NotEqual(t, accesscontrol.OpenShiftPrometheusMetricsReaderID, defaultRole.GetPermissionSetId())
	}
}

func TestConsoleProxyPermissionSet(t *testing.T) {
	ps := getDefaultPermissionSet(accesscontrol.ConsoleProxy)
	require.NotNil(t, ps)
	assert.Equal(t, "ffffffff-ffff-fff4-f5ff-fffffffffff2", ps.GetId())
	assert.Equal(t, "OpenShift Secured Cluster Console Proxy", ps.GetName())
	assert.Equal(t, map[string]storage.Access{
		"Deployment": storage.Access_READ_ACCESS,
		"Image":      storage.Access_READ_ACCESS,
	}, ps.GetResourceToAccess())
	require.NoError(t, permissionsUtils.ValidateScopedReadOnlyPermissionSet(ps))
}
