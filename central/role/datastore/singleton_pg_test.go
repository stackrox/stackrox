//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	rolePkg "github.com/stackrox/rox/central/role"
	permissionSetPGStore "github.com/stackrox/rox/central/role/store/permissionset/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeedConsoleProxyPermissionSetNameCollision(t *testing.T) {
	pgTest := pgtest.ForT(t)
	require.NotNil(t, pgTest)
	defer pgTest.Close()

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.Access)))
	store := permissionSetPGStore.New(pgTest.DB)
	custom := &storage.PermissionSet{
		Id:   rolePkg.GeneratePermissionSetID(),
		Name: accesscontrol.ConsoleProxy,
	}
	require.NoError(t, store.Upsert(ctx, custom))

	roleDS := &dataStoreImpl{permissionSetStorage: store}
	for range 2 {
		require.NoError(t, seedDefaultPermissionSets(ctx, roleDS, store,
			[]*storage.PermissionSet{getDefaultPermissionSet(accesscontrol.ConsoleProxy)}))
	}
	stored, found, err := store.Get(ctx, custom.GetId())
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, custom.GetName(), stored.GetName())
	assert.Equal(t, custom.GetTraits().GetOrigin(), stored.GetTraits().GetOrigin())
	count, err := store.Count(ctx, search.EmptyQuery())
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}
