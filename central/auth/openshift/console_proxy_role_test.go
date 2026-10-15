package openshift

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralproxy"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestSeedConsoleProxyRole(t *testing.T) {
	_, roleDS, _ := setupMocks(t)
	roleDS.EXPECT().GetPermissionSet(gomock.Any(), consoleProxyPermissionSetID).Return(nil, false, nil)
	roleDS.EXPECT().AddPermissionSet(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, ps *storage.PermissionSet) error {
			assert.Equal(t, consoleProxyPermissionSetID, ps.GetId())
			assert.Equal(t, map[string]storage.Access{
				"Deployment": storage.Access_READ_ACCESS,
				"Image":      storage.Access_READ_ACCESS,
			}, ps.GetResourceToAccess())
			return nil
		})
	roleDS.EXPECT().GetRole(gomock.Any(), centralproxy.ConsoleProxyRole).Return(nil, false, nil)
	roleDS.EXPECT().AddRole(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, role *storage.Role) error {
			assert.Equal(t, centralproxy.ConsoleProxyRole, role.GetName())
			assert.Equal(t, consoleProxyPermissionSetID, role.GetPermissionSetId())
			assert.Equal(t, accesscontrol.DefaultAccessScopeIDs[accesscontrol.DenyAllAccessScope], role.GetAccessScopeId())
			return nil
		})

	SeedConsoleProxyRole(context.Background(), roleDS)
}

func TestSeedConsoleProxyRolePreservesChanges(t *testing.T) {
	_, roleDS, _ := setupMocks(t)
	roleDS.EXPECT().GetPermissionSet(gomock.Any(), consoleProxyPermissionSetID).
		Return(&storage.PermissionSet{Id: consoleProxyPermissionSetID}, true, nil)
	roleDS.EXPECT().GetRole(gomock.Any(), centralproxy.ConsoleProxyRole).
		Return(&storage.Role{PermissionSetId: "custom"}, true, nil)

	SeedConsoleProxyRole(context.Background(), roleDS)
}
