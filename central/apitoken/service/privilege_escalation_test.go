package service

import (
	"testing"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/testutils/roletest"
	"github.com/stretchr/testify/assert"
)

func TestVerifyNoPrivilegeEscalation(t *testing.T) {
	devScope := &storage.SimpleAccessScope{
		Name: "Dev",
		Rules: &storage.SimpleAccessScope_Rules{
			IncludedClusters: []string{"Dev"},
		},
	}

	writeRole := roletest.NewResolvedRoleWithGlobalScope(
		"Admin",
		map[string]storage.Access{
			"Image":      storage.Access_READ_WRITE_ACCESS,
			"Deployment": storage.Access_READ_WRITE_ACCESS,
		},
	)
	readRole := roletest.NewResolvedRoleWithGlobalScope(
		"Analyst",
		map[string]storage.Access{
			"Image":      storage.Access_READ_ACCESS,
			"Deployment": storage.Access_READ_ACCESS,
		},
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
	assert.Nil(t, err)
	// 1. User roles are empty.
	err = verifyNoPrivilegeEscalation(make([]permissions.ResolvedRole, 0), []permissions.ResolvedRole{writeRole})
	multiErr := multierror.Append(nil, []error{
		buildError("Image", "", storage.Access_READ_WRITE_ACCESS, storage.Access_NO_ACCESS),
		buildError("Deployment", "", storage.Access_READ_WRITE_ACCESS, storage.Access_NO_ACCESS),
	}...)
	assert.EqualError(t, err, multiErr.Error())
	// 2. Requested roles are empty.
	err = verifyNoPrivilegeEscalation([]permissions.ResolvedRole{writeRole}, make([]permissions.ResolvedRole, 0))
	assert.Nil(t, err)

	// 3. User role and requested role are the same.
	err = verifyNoPrivilegeEscalation([]permissions.ResolvedRole{readRole}, []permissions.ResolvedRole{readRole})
	assert.Nil(t, err)

	// 4. User roles include requested roles.
	err = verifyNoPrivilegeEscalation([]permissions.ResolvedRole{writeRole, readRole}, []permissions.ResolvedRole{readRole})
	assert.Nil(t, err)

	// 5. User has "Dev" write permissions, requested are "Dev" read permissions.
	err = verifyNoPrivilegeEscalation([]permissions.ResolvedRole{devWriteRole}, []permissions.ResolvedRole{devReadRole})
	assert.Nil(t, err)

	// 6. User has write permissions, requested are "Dev" read permissions.
	err = verifyNoPrivilegeEscalation([]permissions.ResolvedRole{writeRole}, []permissions.ResolvedRole{devReadRole})
	assert.Nil(t, err)

	// 7. Permissions between user roles are united.
	devImageWriteRole := roletest.NewResolvedRole(
		"ImageRead",
		map[string]storage.Access{
			"Image": storage.Access_READ_WRITE_ACCESS,
		},
		devScope,
	)
	deploymentWriteRole := roletest.NewResolvedRoleWithGlobalScope(
		"DeploymentWrite",
		map[string]storage.Access{
			"Deployment": storage.Access_READ_WRITE_ACCESS,
		},
	)
	err = verifyNoPrivilegeEscalation([]permissions.ResolvedRole{devImageWriteRole, deploymentWriteRole}, []permissions.ResolvedRole{devWriteRole})
	assert.Nil(t, err)

	// 8. User has read permissions in "Dev" scope, requested are write permissions in "Dev" scope.
	err = verifyNoPrivilegeEscalation([]permissions.ResolvedRole{devReadRole}, []permissions.ResolvedRole{devWriteRole})
	multiErr = multierror.Append(nil, []error{
		buildError("Image", "Dev", storage.Access_READ_WRITE_ACCESS, storage.Access_READ_ACCESS),
		buildError("Deployment", "Dev", storage.Access_READ_WRITE_ACCESS, storage.Access_READ_ACCESS),
	}...)
	assert.EqualError(t, err, multiErr.Error())

	// 9. User has read permissions, requested are write permissions.
	err = verifyNoPrivilegeEscalation([]permissions.ResolvedRole{readRole}, []permissions.ResolvedRole{writeRole})
	multiErr = multierror.Append(nil, []error{
		buildError("Image", "", storage.Access_READ_WRITE_ACCESS, storage.Access_READ_ACCESS),
		buildError("Deployment", "", storage.Access_READ_WRITE_ACCESS, storage.Access_READ_ACCESS),
	}...)
	assert.EqualError(t, err, multiErr.Error())

	// 10. User has "dev" write permissions, requested are unrestricted write permissions.
	err = verifyNoPrivilegeEscalation([]permissions.ResolvedRole{devWriteRole}, []permissions.ResolvedRole{writeRole})
	multiErr = multierror.Append(nil, []error{
		buildError("Image", "", storage.Access_READ_WRITE_ACCESS, storage.Access_NO_ACCESS),
		buildError("Deployment", "", storage.Access_READ_WRITE_ACCESS, storage.Access_NO_ACCESS),
	}...)
	assert.EqualError(t, err, multiErr.Error())
}

func buildError(requestedResource, scopeName string, requestedAccess, userAccess storage.Access) error {
	return errors.Errorf("resource=%s, access scope=%q: requested access is %s, when user access is %s",
		requestedResource, scopeName, requestedAccess, userAccess)
}
