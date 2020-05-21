package datastore

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stackrox/rox/central/deployment/datastore/internal/search"
	"github.com/stackrox/rox/central/deployment/index"
	badgerStore "github.com/stackrox/rox/central/deployment/store/badger"
	"github.com/stackrox/rox/central/globalindex"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/ranking"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/sac"
	search2 "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func BenchmarkSearchAllDeployments(b *testing.B) {
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

	dacky, err := dackbox.NewDackBox(db, nil, []byte("graph"), []byte("dirty"), []byte("valid"))
	require.NoError(b, err)

	bleveIndex, err := globalindex.InitializeIndices("main", blevePath, globalindex.EphemeralIndex)
	require.NoError(b, err)

	deploymentsStore, err := badgerStore.New(db)
	require.NoError(b, err)

	deploymentsIndexer := index.New(bleveIndex, bleveIndex)
	deploymentsSearcher := search.New(deploymentsStore, dacky, nil, nil, nil, nil, nil, deploymentsIndexer)

	imageDS, err := imageDatastore.NewBadger(dacky, concurrency.NewKeyFence(), db, bleveIndex, false, nil, nil, ranking.NewRanker(), ranking.NewRanker())
	require.NoError(b, err)

	deploymentsDatastore, err := newDatastoreImpl(deploymentsStore, nil, deploymentsIndexer, deploymentsSearcher, imageDS, nil, nil, nil, nil,
		nil, ranking.NewRanker(), ranking.NewRanker(), ranking.NewRanker())
	require.NoError(b, err)

	deploymentPrototype := fixtures.GetDeployment().Clone()

	const numDeployments = 1000
	for i := 0; i < numDeployments; i++ {
		if i > 0 && i%100 == 0 {
			fmt.Println("Added", i, "deployments")
		}
		deploymentPrototype.Id = fmt.Sprintf("deployment%d", i)
		require.NoError(b, deploymentsDatastore.UpsertDeployment(ctx, deploymentPrototype))
	}

	b.Run("SearchRetrievalList", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			deployments, err := deploymentsDatastore.SearchListDeployments(ctx, search2.EmptyQuery())
			assert.NoError(b, err)
			assert.Len(b, deployments, numDeployments)
		}
	})

	b.Run("SearchRetrievalFull", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			deployments, err := deploymentsDatastore.SearchRawDeployments(ctx, search2.EmptyQuery())
			assert.NoError(b, err)
			assert.Len(b, deployments, numDeployments)
		}
	})
}
