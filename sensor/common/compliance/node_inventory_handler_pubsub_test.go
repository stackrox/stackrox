package compliance

import (
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/compliance/index"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubDispatcher "github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
	"github.com/stretchr/testify/require"
)

type wrongNodeInventoryEvent struct{}

func (*wrongNodeInventoryEvent) Topic() pubsub.Topic { return pubsub.DefaultTopic }
func (*wrongNodeInventoryEvent) Lane() pubsub.LaneID { return pubsub.DefaultLane }

func newTestNodeInventoryHandlerDispatcher(t *testing.T) common.PubSubDispatcher {
	t.Helper()
	dispatcher, err := pubsubDispatcher.NewDispatcher(pubsubDispatcher.WithLaneConfigs(
		[]pubsub.LaneConfig{
			lane.NewBlockingLane(pubsub.NodeInventoryIntakeLane),
			lane.NewBlockingLane(pubsub.ComplianceAckLane),
		},
	))
	require.NoError(t, err)
	return dispatcher
}

func (s *NodeInventoryHandlerTestSuite) TestPubSubNodeInventoryAndIndexReportDelivered() {
	t := s.T()
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

	dispatcher := newTestNodeInventoryHandlerDispatcher(t)
	defer dispatcher.Stop()

	ch := make(chan *storage.NodeInventory)
	defer close(ch)
	reports := make(chan *index.IndexReportWrap)
	defer close(reports)
	handler := NewNodeInventoryHandler(ch, reports, &mockAlwaysHitNodeIDMatcher{}, &mockRHCOSNodeMatcher{}, dispatcher)
	s.Require().NoError(handler.Start())
	handler.Notify(common.SensorComponentEventCentralReachable)
	defer func() {
		handler.Stop()
		s.NoError(handler.Stopped().Wait())
	}()

	consumer := consumeAndCount(handler.ResponsesC(), 2)

	require.NoError(t, dispatcher.Publish(&NodeInventoryEvent{Inventory: fakeNodeInventory("node1")}))
	require.NoError(t, dispatcher.Publish(&IndexReportWrapEvent{Wrap: &index.IndexReportWrap{
		NodeName:    "node1",
		IndexReport: fakeNodeIndex("x86_64"),
	}}))

	s.NoError(consumer.Stopped().Wait())

	// The legacy raw channels must stay untouched while PubSub is enabled.
	select {
	case <-ch:
		s.Fail("unexpected message on legacy inventories channel while PubSub is enabled")
	case <-time.After(200 * time.Millisecond):
	}
}

// TestPubSubDeliveryAfterStopDoesNotPanic guards against a regression where PubSub events
// delivered after Stop() write to a channel run() already closed. The dispatcher has no
// per-consumer unregistration, so callbacks can still fire post-Stop(); this must be tolerated
// rather than panicking or hanging. Run with -race.
func (s *NodeInventoryHandlerTestSuite) TestPubSubDeliveryAfterStopDoesNotPanic() {
	t := s.T()
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

	dispatcher := newTestNodeInventoryHandlerDispatcher(t)
	defer dispatcher.Stop()

	ch := make(chan *storage.NodeInventory)
	defer close(ch)
	reports := make(chan *index.IndexReportWrap)
	defer close(reports)
	handler := NewNodeInventoryHandler(ch, reports, &mockAlwaysHitNodeIDMatcher{}, &mockRHCOSNodeMatcher{}, dispatcher)
	s.Require().NoError(handler.Start())
	handler.Notify(common.SensorComponentEventCentralReachable)

	handler.Stop()
	s.NoError(handler.Stopped().Wait())

	done := make(chan struct{})
	go func() {
		defer close(done)
		require.NoError(t, dispatcher.Publish(&NodeInventoryEvent{Inventory: fakeNodeInventory("node1")}))
		require.NoError(t, dispatcher.Publish(&IndexReportWrapEvent{Wrap: &index.IndexReportWrap{
			NodeName:    "node1",
			IndexReport: fakeNodeIndex("x86_64"),
		}}))
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		s.FailNow("timed out publishing events after Stop(); a callback is likely blocked sending on ch2Central")
	}
}

func (s *NodeInventoryHandlerTestSuite) TestHandleNodeInventoryEventWrongType() {
	ch := make(chan *storage.NodeInventory)
	defer close(ch)
	reports := make(chan *index.IndexReportWrap)
	defer close(reports)
	handler := NewNodeInventoryHandler(ch, reports, &mockAlwaysHitNodeIDMatcher{}, &mockRHCOSNodeMatcher{}, nil)
	s.Require().NoError(handler.Start())
	defer func() {
		handler.Stop()
		s.NoError(handler.Stopped().Wait())
	}()

	s.Error(handler.handleNodeInventoryEvent(&wrongNodeInventoryEvent{}))
}

func (s *NodeInventoryHandlerTestSuite) TestHandleNodeIndexEventWrongType() {
	ch := make(chan *storage.NodeInventory)
	defer close(ch)
	reports := make(chan *index.IndexReportWrap)
	defer close(reports)
	handler := NewNodeInventoryHandler(ch, reports, &mockAlwaysHitNodeIDMatcher{}, &mockRHCOSNodeMatcher{}, nil)
	s.Require().NoError(handler.Start())
	defer func() {
		handler.Stop()
		s.NoError(handler.Stopped().Wait())
	}()

	s.Error(handler.handleNodeIndexEvent(&wrongNodeInventoryEvent{}))
}

// TestPubSubComplianceAckPublished verifies sendComplianceAck() publishes a ComplianceAckEvent
// instead of writing to the legacy toCompliance channel when PubSub is enabled.
func (s *NodeInventoryHandlerTestSuite) TestPubSubComplianceAckPublished() {
	t := s.T()
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

	dispatcher := newTestNodeInventoryHandlerDispatcher(t)
	defer dispatcher.Stop()

	ch := make(chan *storage.NodeInventory)
	defer close(ch)
	reports := make(chan *index.IndexReportWrap)
	defer close(reports)
	handler := NewNodeInventoryHandler(ch, reports, &mockAlwaysHitNodeIDMatcher{}, &mockRHCOSNodeMatcher{}, dispatcher)
	s.Require().NoError(handler.Start())
	defer func() {
		handler.Stop()
		s.NoError(handler.Stopped().Wait())
	}()

	received := make(chan common.MessageToComplianceWithAddress, 1)
	require.NoError(t, dispatcher.RegisterConsumerToLane(
		pubsub.ComplianceMultiplexerAckConsumer,
		pubsub.ComplianceAckTopic,
		pubsub.ComplianceAckLane,
		func(event pubsub.Event) error {
			ackEvent, ok := event.(*ComplianceAckEvent)
			s.Require().True(ok)
			received <- ackEvent.Msg
			return nil
		},
	))

	handler.sendComplianceAck("node1", 0, 0, "", "")

	select {
	case msg := <-received:
		s.Equal("node1", msg.Hostname)
	case <-time.After(2 * time.Second):
		s.Fail("timed out waiting for compliance ack via pubsub")
	}

	// The legacy toCompliance channel must stay untouched while PubSub is enabled.
	select {
	case <-handler.ComplianceC():
		s.Fail("unexpected message on legacy toCompliance channel while PubSub is enabled")
	case <-time.After(200 * time.Millisecond):
	}
}

// TestArchCacheSafeUnderConcurrentPubSubDelivery publishes interleaved NodeInventoryEvent and
// IndexReportWrapEvent for many different nodes concurrently. Both topics share
// NodeInventoryIntakeLane (a blocking lane), which serializes their delivery through one
// goroutine -- the same guarantee the legacy single-goroutine select loop provided for
// archCache. Run with -race to catch a regression if that guarantee is ever broken.
func (s *NodeInventoryHandlerTestSuite) TestArchCacheSafeUnderConcurrentPubSubDelivery() {
	t := s.T()
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

	dispatcher := newTestNodeInventoryHandlerDispatcher(t)
	defer dispatcher.Stop()

	ch := make(chan *storage.NodeInventory)
	defer close(ch)
	reports := make(chan *index.IndexReportWrap)
	defer close(reports)
	handler := NewNodeInventoryHandler(ch, reports, &mockAlwaysHitNodeIDMatcher{}, &mockRHCOSNodeMatcher{}, dispatcher)
	s.Require().NoError(handler.Start())
	handler.Notify(common.SensorComponentEventCentralReachable)
	defer func() {
		handler.Stop()
		s.NoError(handler.Stopped().Wait())
	}()

	const numNodes = 50
	consumer := consumeAndCount(handler.ResponsesC(), numNodes)

	var wg sync.WaitGroup
	for i := range numNodes {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			nodeName := fmt.Sprintf("node-%d", i)
			require.NoError(t, dispatcher.Publish(&IndexReportWrapEvent{Wrap: &index.IndexReportWrap{
				NodeName:    nodeName,
				IndexReport: fakeNodeIndex("x86_64"),
			}}))
		}(i)
	}
	wg.Wait()

	s.NoError(consumer.Stopped().Wait())
}
