package reprocessor

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	deploymentDackbox "github.com/stackrox/rox/central/deployment/dackbox"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	deploymentIndex "github.com/stackrox/rox/central/deployment/index"
	"github.com/stackrox/rox/central/globalindex"
	indexDackbox "github.com/stackrox/rox/central/image/dackbox"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	imageMocks "github.com/stackrox/rox/central/image/datastore/mocks"
	imageIndex "github.com/stackrox/rox/central/image/index"
	"github.com/stackrox/rox/central/ranking"
	connectionMocks "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/indexer"
	"github.com/stackrox/rox/pkg/dackbox/utils/queue"
	"github.com/stackrox/rox/pkg/fixtures"
	enricherMocks "github.com/stackrox/rox/pkg/images/enricher/mocks"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestLoop(t *testing.T) {
	suite.Run(t, new(loopTestSuite))
}

type loopTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	mockManager    *connectionMocks.MockManager
	mockDeployment *deploymentMocks.MockDataStore
	mockImage      *imageMocks.MockDataStore
	mockEnricher   *enricherMocks.MockImageEnricher
}

func (suite *loopTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.mockManager = connectionMocks.NewMockManager(suite.mockCtrl)
	suite.mockImage = imageMocks.NewMockDataStore(suite.mockCtrl)
	suite.mockDeployment = deploymentMocks.NewMockDataStore(suite.mockCtrl)
}

func (suite *loopTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *loopTestSuite) expectCalls(times int, allowMore bool) {
	timesSpec := (*gomock.Call).Times
	if allowMore {
		timesSpec = (*gomock.Call).MinTimes
	}

	timesSpec(suite.mockImage.EXPECT().Search(getAndWriteImagesContext, gomock.Any()).Return(nil, nil), times)
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
	if !concurrency.WaitWithTimeout(&loop.reprocessingComplete, 100*time.Millisecond) {
		return false
	}
	return true
}

func (suite *loopTestSuite) TestTimerTicksOnce() {
	duration := 1 * time.Second // Need this to be long enough that the enrichAndDetectTicker won't get called twice during the test.
	loop := newLoopWithDuration(suite.mockManager, suite.mockEnricher, suite.mockDeployment, suite.mockImage, nil, duration, duration, duration).(*loopImpl)
	suite.expectCalls(2, false)
	loop.Start()
	// Wait for initial to complete
	suite.True(suite.waitForRun(loop, 500*time.Millisecond))
	// Wait for next tick
	suite.True(suite.waitForRun(loop, duration+10*time.Millisecond))

	loop.Stop()
}

func (suite *loopTestSuite) TestTimerTicksTwice() {
	duration := 100 * time.Millisecond
	loop := newLoopWithDuration(suite.mockManager, suite.mockEnricher, suite.mockDeployment, suite.mockImage, nil, duration, duration, duration).(*loopImpl)
	suite.expectCalls(3, false)
	loop.Start()

	paddedDuration := duration + 10*time.Millisecond
	suite.True(suite.waitForRun(loop, paddedDuration))
	suite.True(suite.waitForRun(loop, paddedDuration))
	suite.True(suite.waitForRun(loop, paddedDuration))
	loop.Stop()
}

func (suite *loopTestSuite) TestShortCircuitOnce() {
	loop := NewLoop(suite.mockManager, suite.mockEnricher, suite.mockDeployment, suite.mockImage, nil).(*loopImpl)
	suite.expectCalls(2, false)
	loop.Start()

	timeout := 100 * time.Millisecond
	suite.True(suite.waitForRun(loop, timeout))
	loop.ShortCircuit()
	suite.True(suite.waitForRun(loop, timeout))
	loop.Stop()
}

func (suite *loopTestSuite) TestShortCircuitTwice() {
	loop := NewLoop(suite.mockManager, suite.mockEnricher, suite.mockDeployment, suite.mockImage, nil).(*loopImpl)
	suite.expectCalls(2, true)
	loop.Start()
	timeout := 100 * time.Millisecond
	suite.True(suite.waitForRun(loop, timeout))
	loop.ShortCircuit()
	suite.True(suite.waitForRun(loop, timeout))
	loop.ShortCircuit()
	suite.True(suite.waitForRun(loop, timeout))
	loop.Stop()
}

func (suite *loopTestSuite) TestStopWorks() {
	loop := NewLoop(suite.mockManager, suite.mockEnricher, suite.mockDeployment, suite.mockImage, nil).(*loopImpl)
	suite.expectCalls(1, false)
	loop.Start()
	timeout := 100 * time.Millisecond
	suite.True(suite.waitForRun(loop, timeout))
	loop.Stop()
	loop.ShortCircuit()
	suite.False(suite.waitForRun(loop, timeout))
}

func TestGetActiveImageIDs(t *testing.T) {
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

	imageDS, err := imageDatastore.NewBadger(dacky, concurrency.NewKeyFence(), nil, bleveIndex, false, nil, nil, ranking.NewRanker(), ranking.NewRanker())
	require.NoError(t, err)

	deploymentsDS, err := deploymentDatastore.NewBadger(dacky, concurrency.NewKeyFence(), nil, nil, bleveIndex, bleveIndex, nil, nil, nil, nil,
		nil, filter.NewFilter(5, []int{5}), ranking.NewRanker(), ranking.NewRanker(), ranking.NewRanker())
	require.NoError(t, err)

	loop := NewLoop(nil, nil, deploymentsDS, imageDS, nil).(*loopImpl)

	ids, err := loop.getActiveImageIDs()
	require.NoError(t, err)
	require.Len(t, ids, 0)

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
