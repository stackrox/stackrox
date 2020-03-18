package reprocessor

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	"github.com/stackrox/rox/central/globalindex"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	imageMocks "github.com/stackrox/rox/central/image/datastore/mocks"
	"github.com/stackrox/rox/central/ranking"
	connectionMocks "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	enricherMocks "github.com/stackrox/rox/pkg/images/enricher/mocks"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
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

	timesSpec(suite.mockDeployment.EXPECT().Search(getDeploymentsContext, gomock.Any()).Return(nil, nil), times)
	timesSpec(suite.mockManager.EXPECT().BroadcastMessage(&central.MsgToSensor{
		Msg: &central.MsgToSensor_ReassessPolicies{
			ReassessPolicies: &central.ReassessPolicies{},
		},
	}), times)
}

func (suite *loopTestSuite) TestTimerDoesNotTick() {
	loop := NewLoop(suite.mockManager, suite.mockEnricher, suite.mockDeployment, suite.mockImage)
	loop.Start()
	loop.Stop()
	suite.mockManager.EXPECT().GetActiveConnections().MaxTimes(0)
}

func (suite *loopTestSuite) TestTimerTicksOnce() {
	duration := 1 * time.Second // Need this to be long enough that the enrichAndDetectTicker won't get called twice during the test.
	loop := newLoopWithDuration(suite.mockManager, suite.mockEnricher, suite.mockDeployment, suite.mockImage, duration, duration, duration)
	suite.expectCalls(2, false)
	loop.Start()
	time.Sleep(duration + 10*time.Millisecond)
	loop.Stop()
}

func (suite *loopTestSuite) TestTimerTicksTwice() {
	duration := 100 * time.Millisecond
	loop := newLoopWithDuration(suite.mockManager, suite.mockEnricher, suite.mockDeployment, suite.mockImage, duration, duration, duration)
	suite.expectCalls(2, true)
	loop.Start()
	time.Sleep((2 * duration) + (10 * time.Millisecond))
	loop.Stop()
}

func (suite *loopTestSuite) TestShortCircuitOnce() {
	loop := NewLoop(suite.mockManager, suite.mockEnricher, suite.mockDeployment, suite.mockImage)
	suite.expectCalls(2, false)
	loop.Start()
	go loop.ShortCircuit()
	// Sleep for a little bit of time to allow the mock calls to go through, since they happen asynchronously.
	time.Sleep(500 * time.Millisecond)
	loop.Stop()
}

func (suite *loopTestSuite) TestShortCircuitTwice() {
	loop := NewLoop(suite.mockManager, suite.mockEnricher, suite.mockDeployment, suite.mockImage)
	suite.expectCalls(3, false)
	loop.Start()
	go loop.ShortCircuit()
	go loop.ShortCircuit()
	// Sleep for a little bit of time to allow the mock calls to go through, since they happen asynchronously.
	time.Sleep(500 * time.Millisecond)
	loop.Stop()
}

func (suite *loopTestSuite) TestStopWorks() {
	loop := NewLoop(suite.mockManager, suite.mockEnricher, suite.mockDeployment, suite.mockImage)
	suite.expectCalls(2, false)
	loop.Start()
	go loop.ShortCircuit()
	time.Sleep(500 * time.Millisecond)
	loop.Stop()
	time.Sleep(100 * time.Millisecond)
	go loop.ShortCircuit()
	// Sleep for a little bit of time to allow the mock calls to go through, since they happen asynchronously.
	time.Sleep(500 * time.Millisecond)
}

func TestGetActiveImageIDs(t *testing.T) {
	envIso := testutils.NewEnvIsolator(t)
	envIso.Setenv(features.Dackbox.EnvVar(), "false")
	defer envIso.RestoreAll()

	badgerDB := testutils.BadgerDBForT(t)

	dacky, err := dackbox.NewDackBox(badgerDB, nil, []byte("graph"), []byte("dirty"), []byte("valid"))
	require.NoError(t, err)

	bleveIndex, err := globalindex.MemOnlyIndex()
	require.NoError(t, err)

	imageDS, err := imageDatastore.NewBadger(dacky, concurrency.NewKeyFence(), badgerDB, bleveIndex, false, nil, nil, ranking.NewRanker(), ranking.NewRanker())
	require.NoError(t, err)

	deploymentsDS, err := deploymentDatastore.NewBadger(dacky, concurrency.NewKeyFence(), badgerDB, bleveIndex, nil, nil, nil, nil, nil,
		nil, filter.NewFilter(5, []int{5}), ranking.NewRanker(), ranking.NewRanker(), ranking.NewRanker())
	require.NoError(t, err)

	loop := NewLoop(nil, nil, deploymentsDS, imageDS).(*loopImpl)

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

	ids, err = loop.getActiveImageIDs()
	require.NoError(t, err)
	require.ElementsMatch(t, imageIDs, ids)
}
