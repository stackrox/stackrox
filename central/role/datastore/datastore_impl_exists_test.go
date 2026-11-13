package datastore

import (
	"context"
	"testing"

	storeMocks "github.com/stackrox/rox/central/role/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newDataStoreForExistsTests(ctrl *gomock.Controller) (*dataStoreImpl, *storeMocks.MockRoleStore,
	*storeMocks.MockPermissionSetStore) {
	roleStore := storeMocks.NewMockRoleStore(ctrl)
	permissionSetStore := storeMocks.NewMockPermissionSetStore(ctrl)
	ds := New(roleStore, permissionSetStore, storeMocks.NewMockSimpleAccessScopeStore(ctrl), nil)
	return ds.(*dataStoreImpl), roleStore, permissionSetStore
}

func ctxWithAccess(access storage.Access) context.Context {
	return sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(access), sac.ResourceScopeKeys(resources.Access)))
}

func TestRoleExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	ds, roleStore, _ := newDataStoreForExistsTests(ctrl)
	readCtx := ctxWithAccess(storage.Access_READ_ACCESS)

	exists, err := ds.RoleExists(sac.WithNoAccess(context.Background()), "role")
	assert.ErrorIs(t, err, sac.ErrResourceAccessDenied)
	assert.False(t, exists)

	roleStore.EXPECT().Exists(gomock.Any(), "role").Return(true, nil)
	exists, err = ds.RoleExists(readCtx, "role")
	require.NoError(t, err)
	assert.True(t, exists)

	roleStore.EXPECT().Exists(gomock.Any(), "role").Return(false, nil)
	exists, err = ds.RoleExists(readCtx, "role")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestPermissionSetExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	ds, _, permissionSetStore := newDataStoreForExistsTests(ctrl)
	readCtx := ctxWithAccess(storage.Access_READ_ACCESS)
	const id = "11111111-1111-1111-1111-111111111111"

	exists, err := ds.PermissionSetExists(sac.WithNoAccess(context.Background()), id)
	assert.ErrorIs(t, err, sac.ErrResourceAccessDenied)
	assert.False(t, exists)

	permissionSetStore.EXPECT().Exists(gomock.Any(), id).Return(true, nil)
	exists, err = ds.PermissionSetExists(readCtx, id)
	require.NoError(t, err)
	assert.True(t, exists)

	permissionSetStore.EXPECT().Exists(gomock.Any(), id).Return(false, nil)
	exists, err = ds.PermissionSetExists(readCtx, id)
	require.NoError(t, err)
	assert.False(t, exists)
}
