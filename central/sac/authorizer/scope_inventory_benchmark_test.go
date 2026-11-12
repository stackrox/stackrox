//go:build sql_integration

package authorizer

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5"
	clusterStore "github.com/stackrox/rox/central/cluster/datastore"
	clusterPostgres "github.com/stackrox/rox/central/cluster/store/cluster/postgres"
	namespaceStore "github.com/stackrox/rox/central/namespace/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
)

// Run with -cpu=4 -benchtime=30s to include repeated snapshot refreshes under
// concurrent restricted-user traffic, rather than measuring cache hits alone.
func BenchmarkScopeInventoryAuthorization(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())
	db := pgtest.ForT(b)
	clusters := make([]*storage.Cluster, 16)
	for i := range clusters {
		clusters[i] = &storage.Cluster{Id: uuid.NewV4().String(), Name: fmt.Sprintf("cluster-%d", i)}
	}
	require.NoError(b, clusterPostgres.New(db).UpsertMany(ctx, clusters))

	rows := make([][]any, 16000)
	for i := range rows {
		cluster := clusters[i%len(clusters)]
		team := "other"
		if i%20 == 0 {
			team = "allowed"
		}
		namespace := &storage.NamespaceMetadata{
			Id: uuid.NewV4().String(), Name: fmt.Sprintf("namespace-%d", i),
			ClusterId: cluster.GetId(), ClusterName: cluster.GetName(),
			Labels: map[string]string{"team": team},
		}
		serialized, err := namespace.MarshalVT()
		require.NoError(b, err)
		rows[i] = []any{namespace.GetId(), namespace.GetName(), namespace.GetClusterId(), namespace.GetClusterName(), namespace.GetLabels(), serialized}
	}
	connection, err := db.Acquire(ctx)
	require.NoError(b, err)
	_, err = connection.CopyFrom(ctx, pgx.Identifier{"namespaces"},
		[]string{"id", "name", "clusterid", "clustername", "labels", "serialized"}, pgx.CopyFromRows(rows))
	connection.Release()
	require.NoError(b, err)

	clusterDS, err := clusterStore.GetTestPostgresDataStore(b, db)
	require.NoError(b, err)
	namespaceDS, err := namespaceStore.GetTestPostgresDataStore(b, db)
	require.NoError(b, err)
	var loads atomic.Int64
	cache := &scopeInventoryCache{load: func(ctx context.Context) (inventory, error) {
		loads.Add(1)
		clusters, err := clusterDS.GetClustersForSAC(ctx)
		if err != nil {
			return inventory{}, err
		}
		namespaces, err := namespaceDS.GetNamespacesForSAC(ctx)
		return inventory{clusters: clusters, namespaces: namespaces}, err
	}}
	roles := []permissions.ResolvedRole{role(allResourcesView, &storage.SimpleAccessScope{
		Id: "label-scope",
		Rules: &storage.SimpleAccessScope_Rules{NamespaceLabelSelectors: []*storage.SetBasedLabelSelector{{
			Requirements: []*storage.SetBasedLabelSelector_Requirement{{Key: "team", Op: storage.SetBasedLabelSelector_IN, Values: []string{"allowed"}}},
		}}},
	})}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			inventory, err := cache.get(ctx)
			if err != nil {
				b.Error(err)
				return
			}
			checker := newGlobalScopeCheckerCore(inventory.clusters, inventory.namespaces, roles, nil)
			requestCtx := sac.WithGlobalAccessScopeChecker(ctx, checker)
			if !sac.ForResource(resources.Deployment).ScopeChecker(requestCtx, storage.Access_READ_ACCESS,
				sac.ClusterScopeKey(clusters[0].GetId()), sac.NamespaceScopeKey("namespace-0")).IsAllowed() {
				b.Error("allowed namespace was denied")
				return
			}
		}
	})
	b.ReportMetric(float64(loads.Load()), "inventory-loads")
}
