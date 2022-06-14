package datastore

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stackrox/stackrox/central/globalindex"
	"github.com/stackrox/stackrox/central/pod/datastore/internal/search"
	"github.com/stackrox/stackrox/central/pod/index"
	rocksStore "github.com/stackrox/stackrox/central/pod/store/rocksdb"
	"github.com/stackrox/stackrox/central/role/resources"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/fixtures"
	"github.com/stackrox/stackrox/pkg/process/filter"
	"github.com/stackrox/stackrox/pkg/rocksdb"
	"github.com/stackrox/stackrox/pkg/sac"
	search2 "github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func BenchmarkSearchAllPods(b *testing.B) {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Deployment),
		))

	tempPath := b.TempDir()

	blevePath := filepath.Join(tempPath, "scorch.bleve")

	db, err := rocksdb.NewTemp("benchmark_search_all")
	require.NoError(b, err)
	defer rocksdbtest.TearDownRocksDB(db)

	bleveIndex, err := globalindex.InitializeIndices("main", blevePath, globalindex.EphemeralIndex, "")
	require.NoError(b, err)

	podsStore := rocksStore.New(db)
	podsIndexer := index.New(bleveIndex)
	podsSearcher := search.New(podsStore, podsIndexer)
	simpleFilter := filter.NewFilter(5, []int{5, 4, 3, 2, 1})

	podsDatastore, err := newDatastoreImpl(ctx, podsStore, podsIndexer, podsSearcher, nil, simpleFilter)
	require.NoError(b, err)

	podPrototype := fixtures.GetPod().Clone()

	const numPods = 1000
	for i := 0; i < numPods; i++ {
		if i > 0 && i%100 == 0 {
			fmt.Println("Added", i, "pods")
		}
		podPrototype.Id = fmt.Sprintf("pod%d", i)
		require.NoError(b, podsDatastore.UpsertPod(ctx, podPrototype))
	}
	fmt.Println("Added", numPods, "pods")

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
