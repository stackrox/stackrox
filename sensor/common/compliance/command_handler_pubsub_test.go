package compliance

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubDispatcher "github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
	"github.com/stretchr/testify/require"
)

type wrongEvent struct{}

func (*wrongEvent) Topic() pubsub.Topic { return pubsub.DefaultTopic }
func (*wrongEvent) Lane() pubsub.LaneID { return pubsub.DefaultLane }

func newTestComplianceReturnDispatcher(t *testing.T) common.PubSubDispatcher {
	t.Helper()
	dispatcher, err := pubsubDispatcher.NewDispatcher(pubsubDispatcher.WithLaneConfigs(
		[]pubsub.LaneConfig{
			lane.NewBlockingLane(pubsub.ComplianceReturnLane),
		},
	))
	require.NoError(t, err)
	return dispatcher
}

func (s *CommandHandlerTestSuite) TestPubSubComplianceReturnDelivered() {
	t := s.T()
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

	dispatcher := newTestComplianceReturnDispatcher(t)
	defer dispatcher.Stop()
	s.cHandler.pubSubDispatcher = dispatcher

	outputChan := make(chan *compliance.ComplianceReturn)
	defer close(outputChan)
	s.startCommandHandler(outputChan)

	s.startScrape("foo", []string{"node1", "node2"})
	s.getScrapeUpdate()

	require.NoError(t, dispatcher.Publish(&ComplianceReturnEvent{Return: &compliance.ComplianceReturn{
		NodeName: "node1",
		ScrapeId: "foo",
	}}))

	select {
	case <-s.cHandler.updates:
	case <-time.After(2 * time.Second):
		s.Require().Fail("timed out waiting for update via pubsub")
	}
}

func (s *CommandHandlerTestSuite) TestHandleComplianceReturnEventWrongType() {
	outputChan := make(chan *compliance.ComplianceReturn)
	defer close(outputChan)
	s.startCommandHandler(outputChan)

	err := s.cHandler.handleComplianceReturnEvent(&wrongEvent{})
	s.Require().Error(err)
}
