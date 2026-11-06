package datastore

import (
	"context"
	"errors"
	"testing"

	rolePkg "github.com/stackrox/rox/central/role"
	storeMocks "github.com/stackrox/rox/central/role/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSeedDefaultPermissionSetsPreservesExistingConsoleProxy(t *testing.T) {
	console := getDefaultPermissionSet(accesscontrol.ConsoleProxy)
	other := getDefaultPermissionSet(accesscontrol.Admin)
	for name, tc := range map[string]struct {
		existing *storage.PermissionSet
		upsert   bool
	}{
		"missing": {upsert: true},
		"same ID": {existing: console, upsert: true},
		"administrator-created set": {existing: &storage.PermissionSet{
			Id: "e6242a9a-9fb4-4239-9a03-30dcd49d9e68", Name: console.GetName(),
		}},
	} {
		t.Run(name, func(t *testing.T) {
			store := storeMocks.NewMockPermissionSetStore(gomock.NewController(t))
			ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resources.Access)))
			store.EXPECT().UpsertMany(ctx, []*storage.PermissionSet{other}).Return(nil)
			store.EXPECT().Walk(ctx, gomock.Any()).DoAndReturn(
				func(_ context.Context, walk func(*storage.PermissionSet) error) error {
					if tc.existing != nil {
						return walk(tc.existing)
					}
					return nil
				})
			if tc.upsert {
				store.EXPECT().Upsert(ctx, console).Return(nil)
			}
			err := seedDefaultPermissionSets(ctx, &dataStoreImpl{permissionSetStorage: store}, store,
				[]*storage.PermissionSet{console, other})
			require.NoError(t, err)
			if tc.existing != nil && !tc.upsert {
				assert.False(t, rolePkg.IsDefaultPermissionSet(tc.existing))
			}
		})
	}
}

func TestSeedDefaultPermissionSetsDoesNotIgnoreOtherErrors(t *testing.T) {
	store := storeMocks.NewMockPermissionSetStore(gomock.NewController(t))
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.Access)))
	console := getDefaultPermissionSet(accesscontrol.ConsoleProxy)
	want := errors.New("database unavailable")
	store.EXPECT().UpsertMany(ctx, []*storage.PermissionSet{}).Return(nil)
	store.EXPECT().Walk(ctx, gomock.Any()).Return(want)
	err := seedDefaultPermissionSets(ctx, &dataStoreImpl{permissionSetStorage: store}, store,
		[]*storage.PermissionSet{console})
	require.ErrorIs(t, err, want)
}

func TestSeedDefaultPermissionSetsHandlesConcurrentNameCollision(t *testing.T) {
	store := storeMocks.NewMockPermissionSetStore(gomock.NewController(t))
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.Access)))
	console := getDefaultPermissionSet(accesscontrol.ConsoleProxy)
	store.EXPECT().UpsertMany(ctx, []*storage.PermissionSet{}).Return(nil)
	store.EXPECT().Walk(ctx, gomock.Any()).Return(nil)
	store.EXPECT().Upsert(ctx, console).Return(errox.AlreadyExists.New("name collision"))
	store.EXPECT().Walk(ctx, gomock.Any()).DoAndReturn(
		func(_ context.Context, walk func(*storage.PermissionSet) error) error {
			return walk(&storage.PermissionSet{Id: "other-id", Name: console.GetName()})
		})
	err := seedDefaultPermissionSets(ctx, &dataStoreImpl{permissionSetStorage: store}, store,
		[]*storage.PermissionSet{console})
	require.NoError(t, err)
}

func TestSeedDefaultPermissionSetsPropagatesInsertError(t *testing.T) {
	store := storeMocks.NewMockPermissionSetStore(gomock.NewController(t))
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.Access)))
	console := getDefaultPermissionSet(accesscontrol.ConsoleProxy)
	want := errors.New("database unavailable")
	store.EXPECT().UpsertMany(ctx, []*storage.PermissionSet{}).Return(nil)
	store.EXPECT().Walk(ctx, gomock.Any()).Return(nil)
	store.EXPECT().Upsert(ctx, console).Return(want)
	err := seedDefaultPermissionSets(ctx, &dataStoreImpl{permissionSetStorage: store}, store,
		[]*storage.PermissionSet{console})
	require.ErrorIs(t, err, want)
}

func TestSeedDefaultPermissionSetsReusesStoredDefaultID(t *testing.T) {
	store := storeMocks.NewMockPermissionSetStore(gomock.NewController(t))
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.Access)))
	console := getDefaultPermissionSet(accesscontrol.ConsoleProxy)
	existing := console.CloneVT()
	existing.Id = rolePkg.GeneratePermissionSetID()
	store.EXPECT().UpsertMany(ctx, []*storage.PermissionSet{}).Return(nil)
	store.EXPECT().Walk(ctx, gomock.Any()).DoAndReturn(
		func(_ context.Context, walk func(*storage.PermissionSet) error) error { return walk(existing) })
	store.EXPECT().Upsert(ctx, console).Return(nil)
	require.NoError(t, seedDefaultPermissionSets(ctx, &dataStoreImpl{permissionSetStorage: store}, store,
		[]*storage.PermissionSet{console}))
	assert.Equal(t, existing.GetId(), console.GetId())
}
