package compliance

import (
	"testing"
	"time"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/compliance/index"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubDispatcher "github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
	"github.com/stretchr/testify/require"
)

func newTestNodeInventoryIntakeDispatcher(t *testing.T) common.PubSubDispatcher {
	t.Helper()
	dispatcher, err := pubsubDispatcher.NewDispatcher(pubsubDispatcher.WithLaneConfigs(
		[]pubsub.LaneConfig{
			lane.NewBlockingLane(pubsub.NodeInventoryIntakeLane),
		},
	))
	require.NoError(t, err)
	return dispatcher
}

func (s *complianceServiceSuite) TestPubSubNodeInventoryDelivered() {
	t := s.T()
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

	dispatcher := newTestNodeInventoryIntakeDispatcher(t)
	defer dispatcher.Stop()
	s.srv.pubSubDispatcher = dispatcher

	received := make(chan *storage.NodeInventory, 1)
	require.NoError(t, dispatcher.RegisterConsumerToLane(
		pubsub.NodeInventoryHandlerNodeInventoryConsumer,
		pubsub.NodeInventoryTopic,
		pubsub.NodeInventoryIntakeLane,
		func(event pubsub.Event) error {
			invEvent, ok := event.(*NodeInventoryEvent)
			s.Require().True(ok)
			received <- invEvent.Inventory
			return nil
		},
	))

	s.Require().NoError(s.stream.Send(&sensor.MsgFromCompliance{
		Msg: &sensor.MsgFromCompliance_NodeInventory{
			NodeInventory: &storage.NodeInventory{NodeName: "node1"},
		},
	}))

	select {
	case inv := <-received:
		s.Require().Equal("node1", inv.GetNodeName())
	case <-time.After(2 * time.Second):
		s.Fail("timed out waiting for node inventory via pubsub")
	}

	select {
	case <-s.srv.nodeInventories:
		s.Fail("unexpected message on legacy nodeInventories channel while PubSub is enabled")
	case <-time.After(200 * time.Millisecond):
	}
}

func (s *complianceServiceSuite) TestPubSubIndexReportDelivered() {
	t := s.T()
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

	dispatcher := newTestNodeInventoryIntakeDispatcher(t)
	defer dispatcher.Stop()
	s.srv.pubSubDispatcher = dispatcher
	s.srv.indexReportWraps = make(chan *index.IndexReportWrap)

	received := make(chan *index.IndexReportWrap, 1)
	require.NoError(t, dispatcher.RegisterConsumerToLane(
		pubsub.NodeInventoryHandlerIndexReportConsumer,
		pubsub.IndexReportWrapTopic,
		pubsub.NodeInventoryIntakeLane,
		func(event pubsub.Event) error {
			wrapEvent, ok := event.(*IndexReportWrapEvent)
			s.Require().True(ok)
			received <- wrapEvent.Wrap
			return nil
		},
	))

	s.Require().NoError(s.stream.Send(&sensor.MsgFromCompliance{
		Msg: &sensor.MsgFromCompliance_IndexReport{
			IndexReport: &v4.IndexReport{},
		},
	}))

	select {
	case wrap := <-received:
		s.Require().NotNil(wrap.IndexReport)
	case <-time.After(2 * time.Second):
		s.Fail("timed out waiting for index report via pubsub")
	}

	select {
	case <-s.srv.indexReportWraps:
		s.Fail("unexpected message on legacy indexReportWraps channel while PubSub is enabled")
	case <-time.After(200 * time.Millisecond):
	}
}

func (s *complianceServiceSuite) TestNodeInventoryFallsBackToRawChannelWhenPubSubDisabled() {
	s.srv.pubSubDispatcher = nil

	go func() {
		s.Require().NoError(s.stream.Send(&sensor.MsgFromCompliance{
			Msg: &sensor.MsgFromCompliance_NodeInventory{
				NodeInventory: &storage.NodeInventory{NodeName: "node1"},
			},
		}))
	}()

	select {
	case inv := <-s.srv.nodeInventories:
		s.Require().Equal("node1", inv.GetNodeName())
	case <-time.After(2 * time.Second):
		s.Fail("timed out waiting for node inventory on legacy channel")
	}
}
