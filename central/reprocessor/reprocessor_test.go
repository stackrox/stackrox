package reprocessor

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	componentCVEEdgeDackbox "github.com/stackrox/rox/central/componentcveedge/dackbox"
	componentCVEEdgeIndex "github.com/stackrox/rox/central/componentcveedge/index"
	cveDackbox "github.com/stackrox/rox/central/cve/dackbox"
	cveIndex "github.com/stackrox/rox/central/cve/index"
	deploymentDackbox "github.com/stackrox/rox/central/deployment/dackbox"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	deploymentIndex "github.com/stackrox/rox/central/deployment/index"
	"github.com/stackrox/rox/central/globalindex"
	indexDackbox "github.com/stackrox/rox/central/image/dackbox"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	imagePG "github.com/stackrox/rox/central/image/datastore/store/postgres"
	imageIndex "github.com/stackrox/rox/central/image/index"
	imageComponentDackbox "github.com/stackrox/rox/central/imagecomponent/dackbox"
	imageComponentIndex "github.com/stackrox/rox/central/imagecomponent/index"
	imageComponentEdgeDackbox "github.com/stackrox/rox/central/imagecomponentedge/dackbox"
	imageComponentEdgeIndex "github.com/stackrox/rox/central/imagecomponentedge/index"
	"github.com/stackrox/rox/central/ranking"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	dackboxConcurrency "github.com/stackrox/rox/pkg/dackbox/concurrency"
	"github.com/stackrox/rox/pkg/dackbox/indexer"
	"github.com/stackrox/rox/pkg/dackbox/utils/queue"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/require"
)

func TestGetActiveImageIDs(t *testing.T) {
	t.Parallel()

	testCtx := sac.WithAllAccess(context.Background())

	rocksDB := rocksdbtest.RocksDBForT(t)

	indexingQ := queue.NewWaitableQueue()
	dacky, err := dackbox.NewRocksDBDackBox(rocksDB, indexingQ, []byte("graph"), []byte("dirty"), []byte("valid"))
	require.NoError(t, err)

	bleveIndex, err := globalindex.MemOnlyIndex()
	require.NoError(t, err)

	reg := indexer.NewWrapperRegistry()
	lazy := indexer.NewLazy(indexingQ, reg, bleveIndex, dacky.AckIndexed)
	lazy.Start()

	reg.RegisterWrapper(deploymentDackbox.Bucket, deploymentIndex.Wrapper{})
	reg.RegisterWrapper(indexDackbox.Bucket, imageIndex.Wrapper{})
	reg.RegisterWrapper(cveDackbox.Bucket, cveIndex.Wrapper{})
	reg.RegisterWrapper(componentCVEEdgeDackbox.Bucket, componentCVEEdgeIndex.Wrapper{})
	reg.RegisterWrapper(imageComponentDackbox.Bucket, imageComponentIndex.Wrapper{})
	reg.RegisterWrapper(imageComponentEdgeDackbox.Bucket, imageComponentEdgeIndex.Wrapper{})

	var pool *pgxpool.Pool
	var imageDS imageDatastore.DataStore
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		testingDB := pgtest.ForTIncludeGorm(t)
		pool = testingDB.Pool
		defer pgtest.CloseGormDB(t, testingDB.GormDB)
		defer pool.Close()

		imageDS = imageDatastore.NewWithPostgres(imagePG.CreateTableAndNewStore(testCtx, pool, testingDB.GormDB, false), imagePG.NewIndexer(pool), nil, ranking.ImageRanker(), ranking.ComponentRanker())
	} else {
		imageDS = imageDatastore.New(dacky, dackboxConcurrency.NewKeyFence(), bleveIndex, bleveIndex, false, nil, ranking.NewRanker(), ranking.NewRanker())
	}

	deploymentsDS, err := deploymentDatastore.New(dacky, dackboxConcurrency.NewKeyFence(), pool, bleveIndex, bleveIndex, nil, nil, nil, nil,
		nil, filter.NewFilter(5, []int{5}), ranking.NewRanker(), ranking.NewRanker(), ranking.NewRanker())
	require.NoError(t, err)

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

	newSig := concurrency.NewSignal()
	indexingQ.PushSignal(&newSig)
	newSig.Wait()

	ids, err = loop.getActiveImageIDs()
	require.NoError(t, err)
	require.ElementsMatch(t, imageIDs, ids)
}
