//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	clusterPostgres "github.com/stackrox/rox/central/cluster/store/cluster/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUncachedClusterLookups(t *testing.T) {
	db := pgtest.ForT(t).DB
	worker, err := GetTestPostgresDataStore(t, pgSearch.WithoutStoreCache(db))
	require.NoError(t, err)
	central := clusterPostgres.New(db)
	ctx := sac.WithAllAccess(context.Background())
	cluster := &storage.Cluster{
		Id: fixtureconsts.Cluster1, Name: "inserted",
		ManagedBy: storage.ManagerType_MANAGER_TYPE_HELM_CHART,
		HelmConfig: &storage.CompleteClusterConfig{DynamicConfig: &storage.DynamicClusterConfig{
			ProcessIndicators: &storage.DynamicClusterConfig_ProcessIndicatorsConfig{ExcludeNamespaceFilter: "^test-"},
		}},
	}
	require.NoError(t, central.Upsert(ctx, cluster))
	_, found, err := worker.GetClusterID(ctx, "Inserted")
	require.NoError(t, err)
	assert.False(t, found, "name resolution is case-sensitive")
	name, found, err := worker.GetClusterName(ctx, fixtureconsts.Cluster1)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "inserted", name)
	id, found, err := worker.GetClusterID(ctx, "inserted")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, fixtureconsts.Cluster1, id)
	exists, err := worker.Exists(ctx, fixtureconsts.Cluster1)
	require.NoError(t, err)
	assert.True(t, exists)
	indicator := &storage.ProcessIndicator{ClusterId: fixtureconsts.Cluster1, Namespace: "test-ns"}
	match, err := worker.MatchProcessIndicator(ctx, indicator)
	require.NoError(t, err)
	assert.True(t, match)
	clusters, err := worker.GetClustersForSAC()
	require.NoError(t, err)
	assert.Len(t, clusters, 1)

	cluster.Name = "renamed"
	cluster.HelmConfig.DynamicConfig.ProcessIndicators.ExcludeNamespaceFilter = "^other-"
	require.NoError(t, central.Upsert(ctx, cluster))
	name, found, err = worker.GetClusterName(ctx, fixtureconsts.Cluster1)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "renamed", name)
	_, found, err = worker.GetClusterID(ctx, "inserted")
	require.NoError(t, err)
	assert.False(t, found)
	id, found, err = worker.GetClusterID(ctx, "renamed")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, fixtureconsts.Cluster1, id)
	match, err = worker.MatchProcessIndicator(ctx, indicator)
	require.NoError(t, err)
	assert.False(t, match)
	indicator.Namespace = "other-ns"
	match, err = worker.MatchProcessIndicator(ctx, indicator)
	require.NoError(t, err)
	assert.True(t, match)
	cluster.HelmConfig.DynamicConfig.ProcessIndicators = nil
	require.NoError(t, central.Upsert(ctx, cluster))
	match, err = worker.MatchProcessIndicator(ctx, indicator)
	require.NoError(t, err)
	assert.False(t, match, "removing a filter must take effect immediately")

	denied := sac.WithNoAccess(ctx)
	_, found, err = worker.GetClusterName(denied, fixtureconsts.Cluster1)
	require.NoError(t, err)
	assert.False(t, found)
	_, found, err = worker.GetClusterID(denied, "renamed")
	require.NoError(t, err)
	assert.False(t, found)
	exists, err = worker.Exists(denied, fixtureconsts.Cluster1)
	require.NoError(t, err)
	assert.False(t, exists)
	match, err = worker.MatchProcessIndicator(denied, indicator)
	require.NoError(t, err)
	assert.False(t, match)

	require.NoError(t, central.Delete(ctx, fixtureconsts.Cluster1))
	_, found, err = worker.GetClusterName(ctx, fixtureconsts.Cluster1)
	require.NoError(t, err)
	assert.False(t, found)
	_, found, err = worker.GetClusterID(ctx, "renamed")
	require.NoError(t, err)
	assert.False(t, found)
	exists, err = worker.Exists(ctx, fixtureconsts.Cluster1)
	require.NoError(t, err)
	assert.False(t, exists)
	match, err = worker.MatchProcessIndicator(ctx, indicator)
	require.NoError(t, err)
	assert.False(t, match)
}

func TestUncachedClusterDuplicateName(t *testing.T) {
	db := pgtest.ForT(t).DB
	worker, err := GetTestPostgresDataStore(t, pgSearch.WithoutStoreCache(db))
	require.NoError(t, err)
	ctx := sac.WithAllAccess(context.Background())
	require.NoError(t, clusterPostgres.New(db).Upsert(ctx, &storage.Cluster{Id: fixtureconsts.Cluster1, Name: "duplicate"}))
	// A preassigned ID makes an incorrect path fail before any creation side effects.
	writeOnly := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS), sac.ResourceScopeKeys(resources.Cluster)))
	_, err = worker.AddCluster(writeOnly, &storage.Cluster{Id: fixtureconsts.Cluster2, Name: "duplicate"})
	require.ErrorContains(t, err, "cannot re-add")
}

func TestUncachedClusterRegistrationResolvesName(t *testing.T) {
	db := pgtest.ForT(t).DB
	worker, err := GetTestPostgresDataStore(t, pgSearch.WithoutStoreCache(db))
	require.NoError(t, err)
	ctx := sac.WithAllAccess(context.Background())
	require.NoError(t, clusterPostgres.New(db).Upsert(ctx, &storage.Cluster{Id: fixtureconsts.Cluster1, Name: "registered"}))
	impl, ok := worker.(*datastoreImpl)
	require.True(t, ok)
	cluster, existing, err := impl.lookupOrCreateCluster(ctx, "", "registered", "unused", clusterConfigData{})
	require.NoError(t, err)
	assert.True(t, existing)
	assert.Equal(t, fixtureconsts.Cluster1, cluster.GetId())

	_, _, err = impl.lookupOrCreateCluster(sac.WithNoAccess(ctx), "", "registered", "unused", clusterConfigData{})
	require.Error(t, err, "name resolution must not bypass the subsequent cluster authorization")
}

func TestUncachedClusterLookupFailures(t *testing.T) {
	db := pgtest.ForT(t).DB
	worker, err := GetTestPostgresDataStore(t, pgSearch.WithoutStoreCache(db))
	require.NoError(t, err)
	db.Close()
	ctx := sac.WithAllAccess(context.Background())
	for name, lookup := range map[string]func() error{
		"name":   func() error { _, _, err := worker.GetClusterName(ctx, fixtureconsts.Cluster1); return err },
		"ID":     func() error { _, _, err := worker.GetClusterID(ctx, "missing"); return err },
		"exists": func() error { _, err := worker.Exists(ctx, fixtureconsts.Cluster1); return err },
		"filter": func() error {
			_, err := worker.MatchProcessIndicator(ctx, &storage.ProcessIndicator{ClusterId: fixtureconsts.Cluster1})
			return err
		},
		"SAC": func() error { _, err := worker.GetClustersForSAC(); return err },
		"duplicate": func() error {
			_, err := worker.AddCluster(ctx, &storage.Cluster{Id: fixtureconsts.Cluster1, Name: "missing"})
			return err
		},
	} {
		t.Run(name, func(t *testing.T) { assert.Error(t, lookup()) })
	}
}
