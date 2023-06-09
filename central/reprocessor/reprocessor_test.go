//go:build sql_integration

package reprocessor

import (
	"context"
	"testing"

	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/globalindex"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	imagePG "github.com/stackrox/rox/central/image/datastore/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	dackboxConcurrency "github.com/stackrox/rox/pkg/dackbox/concurrency"
	"github.com/stackrox/rox/pkg/dackbox/indexer"
	"github.com/stackrox/rox/pkg/dackbox/utils/queue"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/require"
)

func TestGetActiveImageIDs(t *testing.T) {
	t.Parallel()

	testCtx := sac.WithAllAccess(context.Background())

	var (
		pool          postgres.DB
		imageDS       imageDatastore.DataStore
		deploymentsDS deploymentDatastore.DataStore
		indexingQ     queue.WaitableQueue
		err           error
	)

	if env.PostgresDatastoreEnabled.BooleanSetting() {
		testingDB := pgtest.ForT(t)
		pool = testingDB.DB
		defer pool.Close()

		imageDS = imageDatastore.NewWithPostgres(imagePG.New(pool, false, dackboxConcurrency.NewKeyFence()), imagePG.NewIndexer(pool), nil, ranking.ImageRanker(), ranking.ComponentRanker())
		deploymentsDS, err = deploymentDatastore.New(nil, dackboxConcurrency.NewKeyFence(), pool, nil, nil, nil, nil, nil, nil,
			nil, filter.NewFilter(5, 5, []int{5}), ranking.NewRanker(), ranking.NewRanker(), ranking.NewRanker())
		require.NoError(t, err)
	} else {
		rocksDB := rocksdbtest.RocksDBForT(t)

		indexingQ = queue.NewWaitableQueue()
		dacky, err := dackbox.NewRocksDBDackBox(rocksDB, indexingQ, []byte("graph"), []byte("dirty"), []byte("valid"))
		require.NoError(t, err)

		bleveIndex, err := globalindex.MemOnlyIndex()
		require.NoError(t, err)

		reg := indexer.NewWrapperRegistry()
		lazy := indexer.NewLazy(indexingQ, reg, bleveIndex, dacky.AckIndexed)
		lazy.Start()

		imageDS = imageDatastore.New(dacky, dackboxConcurrency.NewKeyFence(), bleveIndex, bleveIndex, false, nil, ranking.NewRanker(), ranking.NewRanker())

		deploymentsDS, err = deploymentDatastore.New(dacky, dackboxConcurrency.NewKeyFence(), nil, bleveIndex, bleveIndex, nil, nil, nil, nil,
			nil, filter.NewFilter(5, 5, []int{5}), ranking.NewRanker(), ranking.NewRanker(), ranking.NewRanker())
		require.NoError(t, err)
	}

	loop := NewLoop(nil, nil, nil, deploymentsDS, imageDS, nil, nil, nil, nil, queue.NewWaitableQueue()).(*loopImpl)

	ids, err := loop.getActiveImageIDs()
	require.NoError(t, err)
	require.Equal(t, 0, len(ids))

	deployment := fixtures.GetDeployment()
	require.NoError(t, deploymentsDS.UpsertDeployment(testCtx, deployment))

	images := fixtures.DeploymentImages()
	imageIDs := make([]string, 0, len(images))
	for _, image := range images {
		require.NoError(t, imageDS.UpsertImage(testCtx, image))
		imageIDs = append(imageIDs, image.GetId())
	}

	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		newSig := concurrency.NewSignal()
		indexingQ.PushSignal(&newSig)
		newSig.Wait()
	}

	ids, err = loop.getActiveImageIDs()
	require.NoError(t, err)
	require.ElementsMatch(t, imageIDs, ids)
}
