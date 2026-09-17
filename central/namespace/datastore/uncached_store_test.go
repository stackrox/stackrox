//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	pgStore "github.com/stackrox/rox/central/namespace/datastore/internal/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUncachedNamespacesForSAC(t *testing.T) {
	db := pgtest.ForT(t).DB
	worker, err := GetTestPostgresDataStore(t, pgSearch.WithoutStoreCache(db))
	require.NoError(t, err)
	central := pgStore.New(db)
	workerStore := pgStore.New(pgSearch.WithoutStoreCache(db))
	ctx := sac.WithAllAccess(context.Background())
	for _, name := range []string{"inserted", "updated"} {
		require.NoError(t, central.Upsert(ctx, &storage.NamespaceMetadata{
			Id: fixtureconsts.Namespace1, ClusterId: fixtureconsts.Cluster1, Name: name,
		}))
		namespaces, err := worker.GetNamespacesForSAC()
		require.NoError(t, err)
		require.Len(t, namespaces, 1)
		assert.Equal(t, name, namespaces[0].GetName())
		assert.Equal(t, fixtureconsts.Cluster1, namespaces[0].GetClusterId())
		_, found, err := workerStore.Get(sac.WithNoAccess(ctx), fixtureconsts.Namespace1)
		require.NoError(t, err)
		assert.False(t, found, "ordinary reads still enforce SAC")
		sacNamespaces, err := workerStore.GetAllForSAC(sac.WithNoAccess(ctx))
		require.NoError(t, err)
		require.Len(t, sacNamespaces, 1, "SAC scope construction deliberately bypasses resource permissions")
		assert.Equal(t, name, sacNamespaces[0].GetName())
	}
	require.NoError(t, central.Delete(ctx, fixtureconsts.Namespace1))
	namespaces, err := worker.GetNamespacesForSAC()
	require.NoError(t, err)
	assert.Empty(t, namespaces)

	db.Close()
	namespaces, err = worker.GetNamespacesForSAC()
	require.Error(t, err)
	assert.Nil(t, namespaces)
}
