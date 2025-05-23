//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	search2 "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func BenchmarkSearchAllPods(b *testing.B) {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Deployment),
		))

	pgtestbase := pgtest.ForT(b)
	require.NotNil(b, pgtestbase)
	pool := pgtestbase.DB

	simpleFilter := filter.NewFilter(5, 5, []int{5, 4, 3, 2, 1})
	podsDatastore := NewPostgresDB(pool, nil, nil, simpleFilter)

	podPrototype := fixtures.GetPod().CloneVT()

	const numPods = 1000
	for i := 0; i < numPods; i++ {
		podPrototype.Id = uuid.NewV4().String()
		require.NoError(b, podsDatastore.UpsertPod(ctx, podPrototype))
	}
	b.Run("SearchRetrieval", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			pods, err := podsDatastore.Search(ctx, search2.EmptyQuery())
			assert.NoError(b, err)
			assert.Len(b, pods, numPods)
		}
	})

	b.Run("RawSearchRetrieval", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			pods, err := podsDatastore.SearchRawPods(ctx, search2.EmptyQuery())
			assert.NoError(b, err)
			assert.Len(b, pods, numPods)
		}
	})
}
