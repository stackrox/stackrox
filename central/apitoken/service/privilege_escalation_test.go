package service

import (
	"testing"

	"github.com/stackrox/rox/central/role"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/testutils/roletest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyNoPrivilegeEscalation(t *testing.T) {
	devScope := &storage.SimpleAccessScope{
		Id:   "devScopeId",
		Name: "Dev",
		Rules: &storage.SimpleAccessScope_Rules{
			IncludedClusters: []string{"Dev"},
		},
	}

	writeRole := roletest.NewResolvedRole(
		"Admin",
		map[string]storage.Access{
			"Image":      storage.Access_READ_WRITE_ACCESS,
			"Deployment": storage.Access_READ_WRITE_ACCESS,
		},
		role.AccessScopeIncludeAll,
	)
	readRole := roletest.NewResolvedRole(
		"Analyst",
		map[string]storage.Access{
			"Image":      storage.Access_READ_ACCESS,
			"Deployment": storage.Access_READ_ACCESS,
		},
		role.AccessScopeIncludeAll,
	)
	devWriteRole := roletest.NewResolvedRole(
		"Admin",
		map[string]storage.Access{
			"Image":      storage.Access_READ_WRITE_ACCESS,
			"Deployment": storage.Access_READ_WRITE_ACCESS,
		},
		devScope,
	)

	devReadRole := roletest.NewResolvedRole(
		"Analyst",
		map[string]storage.Access{
			"Image":      storage.Access_READ_ACCESS,
			"Deployment": storage.Access_READ_ACCESS,
		},
		devScope,
	)

	// 0. Both user and requested roles are empty.
	err := verifyNoPrivilegeEscalation(make([]permissions.ResolvedRole, 0), make([]permissions.ResolvedRole, 0))
	assert.NoError(t, err)

	// 1. User roles are empty.
	err = verifyNoPrivilegeEscalation(make([]permissions.ResolvedRole, 0), []permissions.ResolvedRole{writeRole})
	require.Error(t, err)
	assert.Contains(t, err.Error(), newPrivilegeEscalationError("Image", role.AccessScopeIncludeAll.Name, storage.Access_READ_WRITE_ACCESS, storage.Access_NO_ACCESS).Error())
	assert.Contains(t, err.Error(), newPrivilegeEscalationError("Deployment", role.AccessScopeIncludeAll.Name, storage.Access_READ_WRITE_ACCESS, storage.Access_NO_ACCESS).Error())

	// 2. Requested roles are empty.
	err = verifyNoPrivilegeEscalation([]permissions.ResolvedRole{writeRole}, make([]permissions.ResolvedRole, 0))
	assert.NoError(t, err)

	// 3. User role and requested role are the same.
	err = verifyNoPrivilegeEscalation([]permissions.ResolvedRole{readRole}, []permissions.ResolvedRole{readRole})
	assert.NoError(t, err)

	// 4. User roles include requested roles.
	err = verifyNoPrivilegeEscalation([]permissions.ResolvedRole{writeRole, readRole}, []permissions.ResolvedRole{readRole})
	assert.NoError(t, err)

	// 5. User has "Dev" write permissions, requested are "Dev" read permissions.
	err = verifyNoPrivilegeEscalation([]permissions.ResolvedRole{devWriteRole}, []permissions.ResolvedRole{devReadRole})
	assert.NoError(t, err)

	// 6. User has write permissions, requested are "Dev" read permissions.
	err = verifyNoPrivilegeEscalation([]permissions.ResolvedRole{writeRole}, []permissions.ResolvedRole{devReadRole})
	assert.NoError(t, err)

	// 7. Permissions between user roles are united.
	devImageWriteRole := roletest.NewResolvedRole(
		"ImageRead",
		map[string]storage.Access{
			"Image": storage.Access_READ_WRITE_ACCESS,
		},
		devScope,
	)
	deploymentWriteRole := roletest.NewResolvedRole(
		"DeploymentWrite",
		map[string]storage.Access{
			"Deployment": storage.Access_READ_WRITE_ACCESS,
		},
		role.AccessScopeIncludeAll,
	)
	err = verifyNoPrivilegeEscalation([]permissions.ResolvedRole{devImageWriteRole, deploymentWriteRole}, []permissions.ResolvedRole{devWriteRole})
	assert.NoError(t, err)

	// 8. User has read permissions in "Dev" scope, requested are write permissions in "Dev" scope.
	err = verifyNoPrivilegeEscalation([]permissions.ResolvedRole{devReadRole}, []permissions.ResolvedRole{devWriteRole})
	require.Error(t, err)
	assert.Contains(t, err.Error(), newPrivilegeEscalationError("Image", "Dev", storage.Access_READ_WRITE_ACCESS, storage.Access_READ_ACCESS).Error())
	assert.Contains(t, err.Error(), newPrivilegeEscalationError("Deployment", "Dev", storage.Access_READ_WRITE_ACCESS, storage.Access_READ_ACCESS).Error())

	// 9. User has read permissions, requested are write permissions.
	err = verifyNoPrivilegeEscalation([]permissions.ResolvedRole{readRole}, []permissions.ResolvedRole{writeRole})
	require.Error(t, err)
	assert.Contains(t, err.Error(), newPrivilegeEscalationError("Image", role.AccessScopeIncludeAll.Name, storage.Access_READ_WRITE_ACCESS, storage.Access_READ_ACCESS).Error())
	assert.Contains(t, err.Error(), newPrivilegeEscalationError("Deployment", role.AccessScopeIncludeAll.Name, storage.Access_READ_WRITE_ACCESS, storage.Access_READ_ACCESS).Error())

	// 10. User has "dev" write permissions, requested are unrestricted write permissions.
	err = verifyNoPrivilegeEscalation([]permissions.ResolvedRole{devWriteRole}, []permissions.ResolvedRole{writeRole})
	require.Error(t, err)
	assert.Contains(t, err.Error(), newPrivilegeEscalationError("Image", role.AccessScopeIncludeAll.Name, storage.Access_READ_WRITE_ACCESS, storage.Access_NO_ACCESS).Error())
	assert.Contains(t, err.Error(), newPrivilegeEscalationError("Deployment", role.AccessScopeIncludeAll.Name, storage.Access_READ_WRITE_ACCESS, storage.Access_NO_ACCESS).Error())
}
