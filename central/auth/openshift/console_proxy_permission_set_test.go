package openshift

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralproxy"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestSeedConsoleProxyPermissionSet(t *testing.T) {
	_, roleDS, _ := setupMocks(t)
	roleDS.EXPECT().GetPermissionSet(gomock.Any(), centralproxy.ConsoleProxyPermissionSetID).Return(nil, false, nil)
	roleDS.EXPECT().AddPermissionSet(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, ps *storage.PermissionSet) error {
			assert.Equal(t, centralproxy.ConsoleProxyPermissionSetID, ps.GetId())
			assert.Equal(t, "OpenShift Secured Cluster Console Proxy", ps.GetName())
			assert.Equal(t, "Access for Sensor-proxied OpenShift console requests", ps.GetDescription())
			assert.Equal(t, map[string]storage.Access{
				"Deployment": storage.Access_READ_ACCESS,
				"Image":      storage.Access_READ_ACCESS,
			}, ps.GetResourceToAccess())
			return nil
		})
	SeedConsoleProxyPermissionSet(context.Background(), roleDS)
}

func TestSeedConsoleProxyPermissionSetPreservesChanges(t *testing.T) {
	_, roleDS, _ := setupMocks(t)
	roleDS.EXPECT().GetPermissionSet(gomock.Any(), centralproxy.ConsoleProxyPermissionSetID).
		Return(&storage.PermissionSet{Id: centralproxy.ConsoleProxyPermissionSetID}, true, nil)

	SeedConsoleProxyPermissionSet(context.Background(), roleDS)
}
