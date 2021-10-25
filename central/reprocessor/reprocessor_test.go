package reprocessor

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	activeComponentsUpdater "github.com/stackrox/rox/central/activecomponent/updater"
	activeComponentsUpdaterMocks "github.com/stackrox/rox/central/activecomponent/updater/mocks"
	componentCVEEdgeDackbox "github.com/stackrox/rox/central/componentcveedge/dackbox"
	componentCVEEdgeIndex "github.com/stackrox/rox/central/componentcveedge/index"
	cveDackbox "github.com/stackrox/rox/central/cve/dackbox"
	cveIndex "github.com/stackrox/rox/central/cve/index"
	deploymentDackbox "github.com/stackrox/rox/central/deployment/dackbox"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	deploymentIndex "github.com/stackrox/rox/central/deployment/index"
	"github.com/stackrox/rox/central/globalindex"
	indexDackbox "github.com/stackrox/rox/central/image/dackbox"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	imageMocks "github.com/stackrox/rox/central/image/datastore/mocks"
	imageIndex "github.com/stackrox/rox/central/image/index"
	imageComponentDackbox "github.com/stackrox/rox/central/imagecomponent/dackbox"
	imageComponentIndex "github.com/stackrox/rox/central/imagecomponent/index"
	imageComponentEdgeDackbox "github.com/stackrox/rox/central/imagecomponentedge/dackbox"
	imageComponentEdgeIndex "github.com/stackrox/rox/central/imagecomponentedge/index"
	nodeMocks "github.com/stackrox/rox/central/node/datastore/dackbox/datastore/mocks"
	"github.com/stackrox/rox/central/ranking"
	connectionMocks "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	watchedImageMocks "github.com/stackrox/rox/central/watchedimage/datastore/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/indexer"
	"github.com/stackrox/rox/pkg/dackbox/utils/queue"
	"github.com/stackrox/rox/pkg/fixtures"
	imageEnricherMocks "github.com/stackrox/rox/pkg/images/enricher/mocks"
	nodeEnricherMocks "github.com/stackrox/rox/pkg/nodes/enricher/mocks"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	// avoidTickerTriggerDuration is the duration value that we use in the test to avoid any reprocessings
	// being triggered by the normal trigger. The value has to be chosen such that it is strictly larger than
	// the maximal runtime of an individual test case.
	avoidTickerTriggerDuration = 1 * time.Hour

	// maxReprocessDuration is the maximum duration a reprocess loop iteration must last.
	// This is not related to anything under test - we only need some value here such that we can
	// wait safely for a started reprocessing iteration to complete. Therefore, we can chose an almost
	// arbitrarily large value here, since it would only matter if a reprocessing loop iteration would
	// in fact take much longer than expected.
	maxReprocessDuration = 1 * time.Second
)

func TestLoop(t *testing.T) {
	// This test is timing-sensitive and thus prone to flakiness. It should not run in parallel with other
	// tests.
	suite.Run(t, new(loopTestSuite))
}

type loopTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	mockManager       *connectionMocks.MockManager
	mockWatchedImages *watchedImageMocks.MockDataStore
	mockDeployment    *deploymentMocks.MockDataStore
	mockNode          *nodeMocks.MockDataStore
	mockNodeEnricher  *nodeEnricherMocks.MockNodeEnricher
	mockImage         *imageMocks.MockDataStore
	mockImageEnricher *imageEnricherMocks.MockImageEnricher
	mockAcUpdater     activeComponentsUpdater.Updater
}

func (suite *loopTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.mockManager = connectionMocks.NewMockManager(suite.mockCtrl)
	suite.mockImage = imageMocks.NewMockDataStore(suite.mockCtrl)
	suite.mockDeployment = deploymentMocks.NewMockDataStore(suite.mockCtrl)
	suite.mockNode = nodeMocks.NewMockDataStore(suite.mockCtrl)
	suite.mockWatchedImages = watchedImageMocks.NewMockDataStore(suite.mockCtrl)
	suite.mockAcUpdater = activeComponentsUpdaterMocks.NewMockUpdater(suite.mockCtrl)
}

func (suite *loopTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *loopTestSuite) expectCalls(times int, allowMore bool) {
	timesSpec := (*gomock.Call).Times
	if allowMore {
		timesSpec = (*gomock.Call).MinTimes
	}

	timesSpec(suite.mockImage.EXPECT().Search(allAccessCtx, gomock.Any()).Return(nil, nil), times)
	timesSpec(suite.mockNode.EXPECT().Search(allAccessCtx, gomock.Any()).Return(nil, nil), times)
	timesSpec(suite.mockWatchedImages.EXPECT().GetAllWatchedImages(allAccessCtx).Return(nil, nil), times)
	timesSpec(suite.mockManager.EXPECT().BroadcastMessage(&central.MsgToSensor{
		Msg: &central.MsgToSensor_ReassessPolicies{
			ReassessPolicies: &central.ReassessPolicies{},
		},
	}), times)
}

func (suite *loopTestSuite) waitForRun(loop *loopImpl, timeout time.Duration) bool {
	if !concurrency.WaitWithTimeout(&loop.reprocessingStarted, timeout) {
		return false
	}
	suite.Require().Truef(
		concurrency.WaitWithTimeout(&loop.reprocessingComplete, maxReprocessDuration),
		"reprocessing did not finish within %v", maxReprocessDuration)
	return true
}

func (suite *loopTestSuite) TestTimerTicksOnce() {
	duration := 1 * time.Second // Need this to be long enough that the enrichAndDetectTicker won't get called twice during the test.
	loop := newLoopWithDuration(suite.mockManager, suite.mockImageEnricher, suite.mockNodeEnricher, suite.mockDeployment, suite.mockImage, suite.mockNode, nil, suite.mockWatchedImages, duration, duration, time.Minute, suite.mockAcUpdater)
	suite.expectCalls(2, false)
	loop.Start()
	// Wait for initial to complete
	suite.True(suite.waitForRun(loop, 500*time.Millisecond))
	// Wait for next tick, allowing for some margin of error due to a slow machine or similar.
	suite.True(suite.waitForRun(loop, duration+500*time.Millisecond))

	loop.Stop()
}

func (suite *loopTestSuite) TestTimerTicksTwice() {
	duration := 500 * time.Millisecond
	loop := newLoopWithDuration(suite.mockManager, suite.mockImageEnricher, suite.mockNodeEnricher, suite.mockDeployment, suite.mockImage, suite.mockNode, nil, suite.mockWatchedImages, duration, duration, time.Minute, suite.mockAcUpdater)
	suite.expectCalls(3, false)
	loop.Start()

	paddedDuration := duration + 250*time.Millisecond
	suite.True(suite.waitForRun(loop, paddedDuration))
	suite.True(suite.waitForRun(loop, paddedDuration))
	suite.True(suite.waitForRun(loop, paddedDuration))
	loop.Stop()
}

func (suite *loopTestSuite) TestShortCircuitOnce() {
	loop := newLoopWithDuration(suite.mockManager, suite.mockImageEnricher, suite.mockNodeEnricher, suite.mockDeployment, suite.mockImage, suite.mockNode, nil, suite.mockWatchedImages, avoidTickerTriggerDuration, avoidTickerTriggerDuration, time.Minute, suite.mockAcUpdater)
	suite.expectCalls(2, false)
	loop.Start()

	timeout := 500 * time.Millisecond
	suite.True(suite.waitForRun(loop, timeout))
	loop.ShortCircuit()
	suite.True(suite.waitForRun(loop, timeout))
	loop.Stop()
}

func (suite *loopTestSuite) TestShortCircuitTwice() {
	loop := newLoopWithDuration(suite.mockManager, suite.mockImageEnricher, suite.mockNodeEnricher, suite.mockDeployment, suite.mockImage, suite.mockNode, nil, suite.mockWatchedImages, avoidTickerTriggerDuration, avoidTickerTriggerDuration, time.Minute, nil)
	suite.expectCalls(2, true)
	loop.Start()
	timeout := 500 * time.Millisecond
	suite.True(suite.waitForRun(loop, timeout))
	loop.ShortCircuit()
	suite.True(suite.waitForRun(loop, timeout))
	loop.ShortCircuit()
	suite.True(suite.waitForRun(loop, timeout))
	loop.Stop()
}

func (suite *loopTestSuite) TestStopWorks() {
	loop := newLoopWithDuration(suite.mockManager, suite.mockImageEnricher, suite.mockNodeEnricher, suite.mockDeployment, suite.mockImage, suite.mockNode, nil, suite.mockWatchedImages, avoidTickerTriggerDuration, avoidTickerTriggerDuration, time.Minute, nil)
	suite.expectCalls(1, false)
	loop.Start()
	timeout := 500 * time.Millisecond
	suite.True(suite.waitForRun(loop, timeout))
	loop.Stop()
	loop.ShortCircuit()
	suite.False(suite.waitForRun(loop, timeout))
}

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

	imageDS := imageDatastore.New(dacky, concurrency.NewKeyFence(), bleveIndex, false, nil, ranking.NewRanker(), ranking.NewRanker())

	deploymentsDS := deploymentDatastore.New(dacky, concurrency.NewKeyFence(), nil, bleveIndex, bleveIndex, nil, nil, nil, nil,
		nil, filter.NewFilter(5, []int{5}), ranking.NewRanker(), ranking.NewRanker(), ranking.NewRanker())

	loop := NewLoop(nil, nil, nil, deploymentsDS, imageDS, nil, nil, nil, nil).(*loopImpl)

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
