package datastore

import (
	"context"
	"testing"

	storeMocks "github.com/stackrox/rox/central/role/store/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
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

	roleStore.EXPECT().Count(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, q *v1.Query) (int, error) {
			protoassert.Equal(t, search.NewQueryBuilder().AddDocIDs("role").ProtoQuery(), q)
			return 1, nil
		})
	exists, err = ds.RoleExists(readCtx, "role")
	require.NoError(t, err)
	assert.True(t, exists)

	roleStore.EXPECT().Count(gomock.Any(), gomock.Any()).Return(0, nil)
	exists, err = ds.RoleExists(readCtx, "role")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestPermissionSetExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	ds, _, permissionSetStore := newDataStoreForExistsTests(ctrl)
	readCtx := ctxWithAccess(storage.Access_READ_ACCESS)

	exists, err := ds.PermissionSetExists(sac.WithNoAccess(context.Background()), "11111111-1111-1111-1111-111111111111")
	assert.ErrorIs(t, err, sac.ErrResourceAccessDenied)
	assert.False(t, exists)

	permissionSetStore.EXPECT().Count(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, q *v1.Query) (int, error) {
			protoassert.Equal(t, search.NewQueryBuilder().AddDocIDs("11111111-1111-1111-1111-111111111111").ProtoQuery(), q)
			return 1, nil
		})
	exists, err = ds.PermissionSetExists(readCtx, "11111111-1111-1111-1111-111111111111")
	require.NoError(t, err)
	assert.True(t, exists)

	permissionSetStore.EXPECT().Count(gomock.Any(), gomock.Any()).Return(0, nil)
	exists, err = ds.PermissionSetExists(readCtx, "11111111-1111-1111-1111-111111111111")
	require.NoError(t, err)
	assert.False(t, exists)
}
