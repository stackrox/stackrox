package datastore

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/pod/datastore/internal/search"
	"github.com/stackrox/rox/central/pod/index"
	badgerStore "github.com/stackrox/rox/central/pod/store/badger"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/sac"
	search2 "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func BenchmarkSearchAllPods(b *testing.B) {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Deployment),
		))

	tempPath, err := ioutil.TempDir("", "")
	require.NoError(b, err)

	blevePath := filepath.Join(tempPath, "scorch.bleve")

	db, dir, err := badgerhelper.NewTemp("benchmark_search_all")
	require.NoError(b, err)
	defer utils.IgnoreError(db.Close)
	defer func() { _ = os.RemoveAll(dir) }()

	bleveIndex, err := globalindex.InitializeIndices(blevePath, globalindex.EphemeralIndex)
	require.NoError(b, err)

	podsStore := badgerStore.New(db)
	podsIndexer := index.New(bleveIndex)
	podsSearcher := search.New(podsIndexer)
	simpleFilter := filter.NewFilter(5, []int{5, 4, 3, 2, 1})

	podsDatastore, err := newDatastoreImpl(podsStore, podsIndexer, podsSearcher, nil, simpleFilter)
	require.NoError(b, err)

	podPrototype := proto.Clone(fixtures.GetPod()).(*storage.Pod)

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
}
