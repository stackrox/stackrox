package manager

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	mocksDetector "github.com/stackrox/rox/sensor/common/detector/mocks"
	mocksManager "github.com/stackrox/rox/sensor/common/networkflow/manager/mocks"
	"github.com/stackrox/rox/sensor/common/networkflow/updatecomputer"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	// sendToCentralTimeout is the maximum duration to wait for channel operations (send/receive) with Central.
	// Includes time for enrichmentDoneSignal + goroutine scheduling + actual channel send operation.
	// Set to 100ms to accommodate race detector overhead and scheduling delays.
	sendToCentralTimeout = 100 * time.Millisecond

	// enrichmentTimeout is the maximum duration to wait for an enrichment cycle to complete.
	// This includes goroutine scheduling, entity lookups, and enrichment processing.
	enrichmentTimeout = 500 * time.Millisecond

	// tickerSendTimeout is the maximum duration to wait for ticker send to enrichment goroutine.
	// Short timeout provides fast failure detection if enrichment goroutine becomes unresponsive.
	tickerSendTimeout = 50 * time.Millisecond
)

func TestSendNetworkFlows(t *testing.T) {
	t.Setenv(env.ProcessesListeningOnPort.EnvVar(), "true")
	suite.Run(t, new(sendNetflowsSuite))
}

// sendNetflowsSuite focuses on the manager sending enrichment results to Central
type sendNetflowsSuite struct {
	suite.Suite
	mockCtrl     *gomock.Controller
	mockEntity   *mocksManager.MockEntityStore
	uc           updatecomputer.UpdateComputer
	m            *networkFlowManager
	mockDetector *mocksDetector.MockDetector
	fakeTicker   chan time.Time
}

const (
	srcID = "src-id"
	dstID = "dst-id"
)

func (b *sendNetflowsSuite) SetupTest() {
	b.mockCtrl = gomock.NewController(b.T())
	enrichTickerC := make(chan time.Time)
	defer close(enrichTickerC)
	// Need to expose the concrete type of update computer for deduper assertions
	b.uc = updatecomputer.NewTransitionBased()
	b.m, b.mockEntity, _, b.mockDetector = createManager(b.mockCtrl, enrichTickerC)
	b.m.updateComputer = b.uc

	// Set up RecordTick mock expectation (called after each enrichment cycle)
	b.mockEntity.EXPECT().RecordTick().AnyTimes()

	b.fakeTicker = make(chan time.Time)
	go b.m.enrichConnections(b.fakeTicker)
}

func (b *sendNetflowsSuite) TeardownTest() {
	b.m.stopper.Client().Stop()
}

func (b *sendNetflowsSuite) updateConn(pair *connectionPair) {
	addHostConnection(b.m, createHostnameConnections("hostname").withConnectionPair(pair))
}

func (b *sendNetflowsSuite) updateEp(pair *endpointPair) {
	addHostConnection(b.m, createHostnameConnections("hostname").withEndpointPair(pair))
}

func (b *sendNetflowsSuite) expectContainerLookups(n int) {
	expectEntityLookupContainerHelper(b.mockEntity, n, clusterentities.ContainerMetadata{
		DeploymentID: srcID,
	}, true, false)()
}

func (b *sendNetflowsSuite) expectLookups(n int) {
	expectEntityLookupContainerHelper(b.mockEntity, n, clusterentities.ContainerMetadata{
		DeploymentID: srcID,
	}, true, false)()
	expectEntityLookupEndpointHelper(b.mockEntity, n, []clusterentities.LookupResult{
		{
			Entity:         networkgraph.Entity{ID: dstID},
			ContainerPorts: []uint16{80},
		},
	})()
}

func (b *sendNetflowsSuite) expectFailedLookup(n int) {
	expectEntityLookupContainerHelper(b.mockEntity, n, clusterentities.ContainerMetadata{}, false, false)()
}

func (b *sendNetflowsSuite) expectDetections(n int) {
	expectDetectorHelper(b.mockDetector, n)()
}

func (b *sendNetflowsSuite) TestUpdateConnectionGeneratesNetflow() {
	b.expectLookups(1)
	b.expectDetections(1)

	b.updateConn(createConnectionPair())
	b.thenTickerTicks()
	b.assertOneUpdatedOpenConnection()
}

func (b *sendNetflowsSuite) TestCloseConnection() {
	b.expectLookups(1)
	b.expectDetections(1)

	b.updateConn(createConnectionPair().lastSeen(timestamp.Now()))
	b.thenTickerTicks()
	b.assertOneUpdatedCloseConnection()
}

func (b *sendNetflowsSuite) TestCloseConnectionFailedLookup() {
	b.expectFailedLookup(1)

	b.updateConn(createConnectionPair().lastSeen(timestamp.Now()))
	b.thenTickerTicks()
	b.assertNoMoreUpdatesToCentral()
}

func (b *sendNetflowsSuite) TestCloseOldConnectionFailedLookup() {
	b.expectFailedLookup(1)
	b.expectDetections(1)

	pair := createConnectionPair().
		firstSeen(timestamp.Now().Add(-env.ContainerIDResolutionGracePeriod.DurationSetting() * 2)).
		lastSeen(timestamp.Now())
	b.m.activeConnections[*pair.conn] = &networkConnIndicatorWithAge{}
	b.updateConn(pair)
	b.thenTickerTicks()
	b.assertOneUpdatedCloseConnection()
}

func (b *sendNetflowsSuite) TestCloseEndpoint() {
	b.expectContainerLookups(1)

	b.updateEp(createEndpointPair(timestamp.Now().Add(-time.Hour), timestamp.Now()).lastSeen(timestamp.Now()))
	b.thenTickerTicks()
	b.assertOneUpdatedEndpoint(false)
}

func (b *sendNetflowsSuite) TestCloseEndpointFailedLookup() {
	b.expectFailedLookup(1)

	b.updateEp(createEndpointPair(timestamp.Now().Add(-time.Hour), timestamp.Now()).lastSeen(timestamp.Now()))
	b.thenTickerTicks()
	b.assertNoMoreUpdatesToCentral()
}

func (b *sendNetflowsSuite) TestCloseOldEndpointFailedLookup() {
	b.expectFailedLookup(1)

	pair := createEndpointPair(
		timestamp.Now().Add(-env.ContainerIDResolutionGracePeriod.DurationSetting()*2), timestamp.Now()).
		lastSeen(timestamp.Now())
	b.m.activeEndpoints[*pair.endpoint] = &containerEndpointIndicatorWithAge{}
	b.updateEp(pair)
	b.thenTickerTicks()
	b.assertOneUpdatedEndpoint(false)
}

func (b *sendNetflowsSuite) TestUnchangedConnection() {
	b.expectLookups(1)
	b.expectDetections(1)

	b.updateConn(createConnectionPair().lastSeen(timestamp.InfiniteFuture))
	b.thenTickerTicks()
	b.assertOneUpdatedOpenConnection()

	// There should be no second update, the connection did not change
	b.thenTickerTicks()
	b.assertNoMoreUpdatesToCentral()
}

func (b *sendNetflowsSuite) TestSendTwoUpdatesOnConnectionChanged() {
	b.expectLookups(2)
	b.expectDetections(2)

	pair := createConnectionPair()
	b.updateConn(pair.lastSeen(timestamp.FromProtobuf(protoconv.NowMinus(time.Hour))))
	b.thenTickerTicks()
	b.assertOneUpdatedCloseConnection()

	pair.lastSeen(timestamp.Now())
	b.updateConn(pair)
	b.thenTickerTicks()
	b.assertOneUpdatedCloseConnection()
}

func (b *sendNetflowsSuite) TestUpdatesGetBufferedWhenUnread() {
	b.expectLookups(4)
	b.expectDetections(4)

	startingSendCycles := b.m.sendCyclesCompleted.Load()

	// four times without reading
	for i := 4; i > 0; i-- {
		ts := protoconv.NowMinus(time.Duration(i) * time.Hour)
		b.updateConn(createConnectionPair().lastSeen(timestamp.FromProtobuf(ts)))
		b.thenTickerTicks()
	}

	// Wait for all 4 send cycles to complete before reading
	b.waitForSendCycles(startingSendCycles + 4)

	// should be able to read four buffered updates in sequence
	for i := 0; i < 4; i++ {
		b.assertOneUpdatedCloseConnection()
	}
}

func (b *sendNetflowsSuite) TestCallsDetectionEvenOnFullBuffer() {
	b.expectLookups(6)
	b.expectDetections(6)

	startingSendCycles := b.m.sendCyclesCompleted.Load()

	for i := 6; i > 0; i-- {
		ts := protoconv.NowMinus(time.Duration(i) * time.Hour)
		b.updateConn(createConnectionPair().lastSeen(timestamp.FromProtobuf(ts)))
		b.thenTickerTicks()
	}

	// Wait for all 6 send cycles to complete (ensures all channel operations attempted)
	b.waitForSendCycles(startingSendCycles + 6)

	// Will only store 5 network flow updates, as it's the maximum buffer size in the test
	for i := 0; i < 5; i++ {
		b.assertOneUpdatedCloseConnection()
	}

	b.assertNoMoreUpdatesToCentral()
}

func (b *sendNetflowsSuite) thenTickerTicks() {
	b.m.enrichmentDoneSignal.Reset()
	select {
	case b.fakeTicker <- time.Now():
	case <-time.After(tickerSendTimeout):
		b.T().Fatal("ticker send blocked - enrichment goroutine not responsive")
	}
	b.waitForEnrichmentDone()
}

func (b *sendNetflowsSuite) waitForEnrichmentDone() {
	start := time.Now()
	// Wait for enrichment cycle to complete (signal from manager) with timeout
	select {
	case <-b.m.enrichmentDoneSignal.Done():
		elapsed := time.Since(start)
		b.T().Logf("enrichment completed in %v", elapsed)
	case <-time.After(enrichmentTimeout):
		b.T().Fatal("enrichment did not complete within timeout")
	}
}

func (b *sendNetflowsSuite) waitForSendCycles(expectedCycles uint64) {
	start := time.Now()
	deadline := time.Now().Add(sendToCentralTimeout)

	for {
		currentCycles := b.m.sendCyclesCompleted.Load()
		if currentCycles >= expectedCycles {
			elapsed := time.Since(start)
			b.T().Logf("send cycles completed: %d/%d in %v", currentCycles, expectedCycles, elapsed)
			return
		}

		if time.Now().After(deadline) {
			b.T().Fatalf("send cycles did not complete within timeout: got %d, expected %d", currentCycles, expectedCycles)
		}

		time.Sleep(1 * time.Millisecond) // Poll every 1ms
	}
}

func (b *sendNetflowsSuite) assertOneUpdatedOpenConnection() {
	msg := mustSendToCentralWithoutBlock(b.T(), b.m.sensorUpdates)
	netflowUpdate, ok := msg.Msg.(*central.MsgFromSensor_NetworkFlowUpdate)
	b.Require().True(ok, "message is NetworkFlowUpdate")
	b.Require().Len(netflowUpdate.NetworkFlowUpdate.GetUpdated(), 1, "one updated connection")
	b.Assert().Equal(int32(0), netflowUpdate.NetworkFlowUpdate.GetUpdated()[0].GetLastSeenTimestamp().GetNanos(), "the connection should be open")
}

func (b *sendNetflowsSuite) assertOneUpdatedCloseConnection() {
	msg := mustSendToCentralWithoutBlock(b.T(), b.m.sensorUpdates)
	netflowUpdate, ok := msg.Msg.(*central.MsgFromSensor_NetworkFlowUpdate)
	b.Require().True(ok, "message is NetworkFlowUpdate")
	b.Require().Len(netflowUpdate.NetworkFlowUpdate.GetUpdated(), 1, "one updated connection")
	b.Assert().NotEqual(int32(0), netflowUpdate.NetworkFlowUpdate.GetUpdated()[0].GetLastSeenTimestamp().GetNanos(), "the connection should not be open")
}

func (b *sendNetflowsSuite) assertOneUpdatedEndpoint(isOpen bool) {
	msg := mustSendToCentralWithoutBlock(b.T(), b.m.sensorUpdates)
	netflowUpdate, ok := msg.Msg.(*central.MsgFromSensor_NetworkFlowUpdate)
	b.Require().True(ok, "message is NetworkFlowUpdate")
	b.Require().Len(netflowUpdate.NetworkFlowUpdate.GetUpdatedEndpoints(), 1, "one updated endpint")
	closeTS := netflowUpdate.NetworkFlowUpdate.GetUpdatedEndpoints()[0].GetLastActiveTimestamp().GetNanos()
	if isOpen {
		b.Assert().Equal(int32(0), closeTS, "the endpoint should be open but is closed")
	} else {
		b.Assert().NotEqual(int32(0), closeTS, "the endpoint should be closed but is open")
	}
}

func mustNotRead[T any](t *testing.T, ch chan T) {
	select {
	case <-ch:
		t.Fatal("should not receive in channel")
	case <-time.After(sendToCentralTimeout):
	}
}

func mustReadTimeout[T any](t *testing.T, ch chan T, timeout time.Duration) T {
	var result T
	select {
	case v, more := <-ch:
		require.True(t, more, "channel should never close")
		result = v
	case <-time.After(timeout):
		t.Fatal("blocked on reading from channel")
	}
	return result
}

func mustSendToCentralWithoutBlock[T any](t *testing.T, ch chan T) T {
	return mustReadTimeout(t, ch, sendToCentralTimeout)
}
