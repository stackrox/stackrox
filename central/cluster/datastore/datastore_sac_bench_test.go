//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"testing"

	pgStore "github.com/stackrox/rox/central/cluster/store/cluster/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
)

// BenchmarkGetClustersForSAC measures loading the objects needed to construct
// an access scope. Database setup and fixture generation are outside the timer.
func BenchmarkGetClustersForSAC(b *testing.B) {
	for _, count := range []int{20, 100} {
		b.Run(fmt.Sprintf("%d_clusters", count), func(b *testing.B) {
			db := pgtest.ForT(b)
			store := pgStore.New(db.DB)
			ds := &datastoreImpl{clusterStorage: store}
			clusters := make([]*storage.Cluster, 0, count)
			for i := range count {
				cluster := fixtures.GetCluster(fmt.Sprintf("cluster-%03d", i))
				cluster.Id = uuid.NewV4().String()
				cluster.Labels = map[string]string{
					"team":        fmt.Sprintf("team-%02d", i%20),
					"region":      fmt.Sprintf("region-%d", i%3),
					"environment": "production",
				}
				clusters = append(clusters, cluster)
			}
			require.NoError(b, store.UpsertMany(sac.WithAllAccess(context.Background()), clusters))
			ctx := context.Background()
			_, err := ds.GetClustersForSAC(ctx)
			require.NoError(b, err)
			b.ReportAllocs()
			for b.Loop() {
				result, err := ds.GetClustersForSAC(ctx)
				require.NoError(b, err)
				require.Len(b, result, count)
			}
		})
	}
}
