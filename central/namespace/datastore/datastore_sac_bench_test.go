//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"testing"

	pgStore "github.com/stackrox/rox/central/namespace/datastore/internal/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
)

// BenchmarkGetNamespacesForSAC measures loading the objects needed to construct
// an access scope. Database setup and fixture generation are outside the timer.
func BenchmarkGetNamespacesForSAC(b *testing.B) {
	for _, count := range []int{1000, 5000} {
		b.Run(fmt.Sprintf("%d_namespaces", count), func(b *testing.B) {
			db := pgtest.ForT(b)
			store := pgStore.New(db.DB)
			ds := &datastoreImpl{store: store}
			namespaces := make([]*storage.NamespaceMetadata, 0, count)
			for i := range count {
				clusterID := fmt.Sprintf("00000000-0000-4000-8000-%012d", i%100)
				namespace := fixtures.GetNamespace(clusterID, fmt.Sprintf("cluster-%03d", i%100), fmt.Sprintf("namespace-%04d", i))
				namespace.Id = uuid.NewV4().String()
				namespace.Labels = map[string]string{
					"team":        fmt.Sprintf("team-%02d", i%20),
					"application": fmt.Sprintf("application-%04d", i),
					"environment": "production",
				}
				namespace.Annotations = map[string]string{"description": "Namespace used to benchmark access-scope construction"}
				namespaces = append(namespaces, namespace)
			}
			require.NoError(b, store.UpsertMany(sac.WithAllAccess(context.Background()), namespaces))
			ctx := context.Background()
			_, err := ds.GetNamespacesForSAC(ctx)
			require.NoError(b, err)
			b.ReportAllocs()
			for b.Loop() {
				result, err := ds.GetNamespacesForSAC(ctx)
				require.NoError(b, err)
				require.Len(b, result, count)
			}
		})
	}
}
