//go:build sql_integration

package authorizer

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	clusterPostgres "github.com/stackrox/rox/central/cluster/store/cluster/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
)

// Run with -cpu=4 -benchtime=30s to measure SQL scope evaluation under
// sustained concurrent restricted-user traffic.
func BenchmarkScopeInventoryAuthorization(b *testing.B) {
	benchmarkSQLScope(b, func(_ []*storage.Cluster) *storage.SimpleAccessScope_Rules {
		return &storage.SimpleAccessScope_Rules{NamespaceLabelSelectors: []*storage.SetBasedLabelSelector{{
			Requirements: []*storage.SetBasedLabelSelector_Requirement{{Key: "team", Op: storage.SetBasedLabelSelector_IN, Values: []string{"allowed"}}},
		}}}
	}, 0, "namespace-0")
}

func BenchmarkScopeNamespaceName(b *testing.B) {
	benchmarkSQLScope(b, func(clusters []*storage.Cluster) *storage.SimpleAccessScope_Rules {
		return &storage.SimpleAccessScope_Rules{IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{{
			ClusterId: clusters[0].GetId(), NamespaceName: "namespace-0",
		}}}
	}, 0, "namespace-0")
}

func BenchmarkScopeBroadNamespaceLabel(b *testing.B) {
	benchmarkSQLScope(b, func(_ []*storage.Cluster) *storage.SimpleAccessScope_Rules {
		return &storage.SimpleAccessScope_Rules{NamespaceLabelSelectors: []*storage.SetBasedLabelSelector{{
			Requirements: []*storage.SetBasedLabelSelector_Requirement{{Key: "team", Op: storage.SetBasedLabelSelector_NOT_IN, Values: []string{"allowed"}}},
		}}}
	}, 1, "namespace-1")
}

func benchmarkSQLScope(b *testing.B, rules func([]*storage.Cluster) *storage.SimpleAccessScope_Rules, clusterIndex int, namespaceName string) {
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

	roles := []permissions.ResolvedRole{role(allResourcesView, &storage.SimpleAccessScope{Id: "scope", Rules: rules(clusters)})}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			checker := newScopeChecker(roles, nil, func(rules *storage.SimpleAccessScope_Rules) (*effectiveaccessscope.ScopeTree, error) {
				return computeSQLScope(ctx, db, rules)
			})
			requestCtx := sac.WithGlobalAccessScopeChecker(ctx, checker)
			if !sac.ForResource(resources.Deployment).ScopeChecker(requestCtx, storage.Access_READ_ACCESS,
				sac.ClusterScopeKey(clusters[clusterIndex].GetId()), sac.NamespaceScopeKey(namespaceName)).IsAllowed() {
				b.Error("allowed namespace was denied")
				return
			}
		}
	})
}
