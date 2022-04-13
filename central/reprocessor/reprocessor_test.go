package reprocessor

import (
	"context"
	"testing"

	componentCVEEdgeDackbox "github.com/stackrox/stackrox/central/componentcveedge/dackbox"
	componentCVEEdgeIndex "github.com/stackrox/stackrox/central/componentcveedge/index"
	cveDackbox "github.com/stackrox/stackrox/central/cve/dackbox"
	cveIndex "github.com/stackrox/stackrox/central/cve/index"
	deploymentDackbox "github.com/stackrox/stackrox/central/deployment/dackbox"
	deploymentDatastore "github.com/stackrox/stackrox/central/deployment/datastore"
	deploymentIndex "github.com/stackrox/stackrox/central/deployment/index"
	"github.com/stackrox/stackrox/central/globalindex"
	indexDackbox "github.com/stackrox/stackrox/central/image/dackbox"
	imageDatastore "github.com/stackrox/stackrox/central/image/datastore"
	imageIndex "github.com/stackrox/stackrox/central/image/index"
	imageComponentDackbox "github.com/stackrox/stackrox/central/imagecomponent/dackbox"
	imageComponentIndex "github.com/stackrox/stackrox/central/imagecomponent/index"
	imageComponentEdgeDackbox "github.com/stackrox/stackrox/central/imagecomponentedge/dackbox"
	imageComponentEdgeIndex "github.com/stackrox/stackrox/central/imagecomponentedge/index"
	"github.com/stackrox/stackrox/central/ranking"
	"github.com/stackrox/stackrox/pkg/concurrency"
	"github.com/stackrox/stackrox/pkg/dackbox"
	"github.com/stackrox/stackrox/pkg/dackbox/indexer"
	"github.com/stackrox/stackrox/pkg/dackbox/utils/queue"
	"github.com/stackrox/stackrox/pkg/fixtures"
	"github.com/stackrox/stackrox/pkg/process/filter"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/require"
)

func TestGetActiveImageIDs(t *testing.T) {
	t.Parallel()

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

	imageDS := imageDatastore.New(dacky, concurrency.NewKeyFence(), bleveIndex, bleveIndex, false, nil, ranking.NewRanker(), ranking.NewRanker())

	deploymentsDS := deploymentDatastore.New(dacky, concurrency.NewKeyFence(), nil, bleveIndex, bleveIndex, nil, nil, nil, nil,
		nil, filter.NewFilter(5, []int{5}), ranking.NewRanker(), ranking.NewRanker(), ranking.NewRanker())

	loop := NewLoop(nil, nil, nil, deploymentsDS, imageDS, nil, nil, nil, nil, queue.NewWaitableQueue()).(*loopImpl)

	ids, err := loop.getActiveImageIDs()
	require.NoError(t, err)
	require.Equal(t, 0, len(ids))

	testCtx := sac.WithAllAccess(context.Background())

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
