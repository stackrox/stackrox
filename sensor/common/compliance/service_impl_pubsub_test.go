package compliance

import (
	"time"

	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubDispatcher "github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
	"github.com/stretchr/testify/require"
)

func (s *complianceServiceSuite) TestComplianceReturnLegacyPath() {
	s.T().Setenv(features.SensorInternalPubSub.EnvVar(), "false")

	s.Require().NoError(s.stream.Send(&sensor.MsgFromCompliance{
		Msg: &sensor.MsgFromCompliance_Return{
			Return: &compliance.ComplianceReturn{NodeName: "node1", ScrapeId: "foo"},
		},
	}))

	select {
	case result := <-s.srv.output:
		s.Require().Equal("foo", result.GetScrapeId())
	case <-time.After(2 * time.Second):
		s.Fail("expected compliance return on legacy output channel")
	}
}

func (s *complianceServiceSuite) TestComplianceReturnPubSubPath() {
	t := s.T()
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

	dispatcher, err := pubsubDispatcher.NewDispatcher(pubsubDispatcher.WithLaneConfigs(
		[]pubsub.LaneConfig{
			lane.NewBlockingLane(pubsub.ComplianceReturnLane),
		},
	))
	require.NoError(t, err)
	defer dispatcher.Stop()

	received := make(chan *ComplianceReturnEvent, 1)
	require.NoError(t, dispatcher.RegisterConsumerToLane(
		pubsub.ComplianceCommandHandlerReturnConsumer,
		pubsub.ComplianceReturnTopic,
		pubsub.ComplianceReturnLane,
		func(event pubsub.Event) error {
			evt, ok := event.(*ComplianceReturnEvent)
			s.Require().True(ok)
			received <- evt
			return nil
		},
	))
	s.srv.pubSubDispatcher = dispatcher

	s.Require().NoError(s.stream.Send(&sensor.MsgFromCompliance{
		Msg: &sensor.MsgFromCompliance_Return{
			Return: &compliance.ComplianceReturn{NodeName: "node1", ScrapeId: "foo"},
		},
	}))

	select {
	case evt := <-received:
		s.Require().Equal("foo", evt.Return.GetScrapeId())
	case <-time.After(2 * time.Second):
		s.Fail("expected compliance return event via pubsub")
	}

	select {
	case <-s.srv.output:
		s.Fail("unexpected message on legacy output channel while pubsub is enabled")
	default:
	}
}
