package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralproxy"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConsoleProxyDefaultPermissionSet(t *testing.T) {
	var consoleProxy *storage.PermissionSet
	for _, permissionSet := range getDefaultPermissionSets() {
		if permissionSet.GetId() == centralproxy.ConsoleProxyPermissionSetID {
			consoleProxy = permissionSet
			break
		}
	}

	require.NotNil(t, consoleProxy)
	assert.Equal(t, accesscontrol.ConsoleProxy, consoleProxy.GetName())
	assert.Equal(t, storage.Traits_DEFAULT, consoleProxy.GetTraits().GetOrigin())
	assert.Equal(t, map[string]storage.Access{
		"Deployment": storage.Access_READ_ACCESS,
		"Image":      storage.Access_READ_ACCESS,
	}, consoleProxy.GetResourceToAccess())
	assert.True(t, accesscontrol.IsDefaultPermissionSet(consoleProxy.GetName()))
	assert.Error(t, verifyNotDefaultPermissionSet(consoleProxy))
	assert.False(t, declarativeconfig.CanModifyResource(context.Background(), consoleProxy))
	for _, role := range getDefaultRoles() {
		assert.NotEqual(t, consoleProxy.GetName(), role.GetName())
	}
}
