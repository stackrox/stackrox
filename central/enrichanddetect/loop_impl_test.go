package enrichanddetect

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	enrichAndDetectorMocks "github.com/stackrox/rox/central/enrichanddetect/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/suite"
)

func TestLoop(t *testing.T) {
	suite.Run(t, new(loopTestSuite))
}

type loopTestSuite struct {
	suite.Suite
	mockDeployments *deploymentMocks.MockDataStore
	mockEnricher    *enrichAndDetectorMocks.MockEnricherAndDetector
	mockCtrl        *gomock.Controller
}

func (suite *loopTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.mockDeployments = deploymentMocks.NewMockDataStore(suite.mockCtrl)
	suite.mockEnricher = enrichAndDetectorMocks.NewMockEnricherAndDetector(suite.mockCtrl)
}

func (suite *loopTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *loopTestSuite) expectCalls(times int, allowMore bool) {
	deployment := fixtures.GetDeployment()
	timesSpec := (*gomock.Call).Times
	if allowMore {
		timesSpec = (*gomock.Call).MinTimes
	}
	timesSpec(suite.mockDeployments.EXPECT().GetDeployments(), times).Return([]*storage.Deployment{deployment}, nil)
	timesSpec(suite.mockEnricher.EXPECT().EnrichAndDetect(deployment), times).Return(nil)
}

func (suite *loopTestSuite) TestTimerDoesNotTick() {
	loop := NewLoop(suite.mockEnricher, suite.mockDeployments)
	loop.Start()
	loop.Stop()
	suite.mockEnricher.EXPECT().EnrichAndDetect(gomock.Any()).MaxTimes(0)
}

func (suite *loopTestSuite) TestTimerTicksOnce() {
	duration := 1 * time.Second // Need this to be long enough that the ticker won't get called twice during the test.
	loop := newLoopWithDuration(suite.mockEnricher, suite.mockDeployments, duration)
	suite.expectCalls(1, false)
	loop.Start()
	time.Sleep(duration + 10*time.Millisecond)
	loop.Stop()
}

func (suite *loopTestSuite) TestTimerTicksTwice() {
	duration := 100 * time.Millisecond
	loop := newLoopWithDuration(suite.mockEnricher, suite.mockDeployments, duration)
	suite.expectCalls(2, true)
	loop.Start()
	time.Sleep((2 * duration) + (10 * time.Millisecond))
	loop.Stop()
}

func (suite *loopTestSuite) TestShortCircuitOnce() {
	loop := NewLoop(suite.mockEnricher, suite.mockDeployments)
	suite.expectCalls(1, false)
	loop.Start()
	go loop.ShortCircuit()
	// Sleep for a little bit of time to allow the mock calls to go through, since they happen asynchronously.
	time.Sleep(500 * time.Millisecond)
	loop.Stop()
}

func (suite *loopTestSuite) TestShortCircuitTwice() {
	loop := NewLoop(suite.mockEnricher, suite.mockDeployments)
	suite.expectCalls(2, false)
	loop.Start()
	go loop.ShortCircuit()
	go loop.ShortCircuit()
	// Sleep for a little bit of time to allow the mock calls to go through, since they happen asynchronously.
	time.Sleep(500 * time.Millisecond)
	loop.Stop()
}

func (suite *loopTestSuite) TestStopWorks() {
	loop := NewLoop(suite.mockEnricher, suite.mockDeployments)
	suite.expectCalls(1, false)
	loop.Start()
	go loop.ShortCircuit()
	time.Sleep(500 * time.Millisecond)
	loop.Stop()
	time.Sleep(100 * time.Millisecond)
	go loop.ShortCircuit()
	// Sleep for a little bit of time to allow the mock calls to go through, since they happen asynchronously.
	time.Sleep(500 * time.Millisecond)
}
