package compliance

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubDispatcher "github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
	"github.com/stretchr/testify/require"
)

func newTestServiceDispatcher(t *testing.T) common.PubSubDispatcher {
	t.Helper()
	dispatcher, err := pubsubDispatcher.NewDispatcher(pubsubDispatcher.WithLaneConfigs(
		[]pubsub.LaneConfig{
			lane.NewBlockingLane(pubsub.DetectorAuditLogLane),
			lane.NewBlockingLane(pubsub.AuditLogManagerLane),
		},
	))
	require.NoError(t, err)
	return dispatcher
}

// TestPubSubAuditEventsRoutedToAuditLogManager verifies that Communicate()'s AuditEvents case
// publishes an AuditLogManagerEvent instead of writing to the legacy AuditMessagesChan() when
// PubSub is enabled.
func (s *complianceServiceSuite) TestPubSubAuditEventsRoutedToAuditLogManager() {
	t := s.T()
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

	dispatcher := newTestServiceDispatcher(t)
	defer dispatcher.Stop()
	s.srv.pubSubDispatcher = dispatcher

	received := make(chan *AuditLogManagerEvent, 1)
	require.NoError(t, dispatcher.RegisterConsumerToLane(
		pubsub.AuditLogManagerAuditEventsConsumer,
		pubsub.AuditLogManagerTopic,
		pubsub.AuditLogManagerLane,
		func(event pubsub.Event) error {
			auditLogManagerEvent, ok := event.(*AuditLogManagerEvent)
			require.True(t, ok)
			received <- auditLogManagerEvent
			return nil
		},
	))

	s.online()
	s.Require().NoError(s.stream.Send(&sensor.MsgFromCompliance{
		Msg: &sensor.MsgFromCompliance_AuditEvents{
			AuditEvents: &sensor.AuditEvents{
				Events: []*storage.KubernetesEvent{{Id: "1"}},
			},
		},
	}))

	select {
	case event := <-received:
		s.NotNil(event.AuditEvents)
	case <-time.After(2 * time.Second):
		s.Fail("timed out waiting for audit log manager event via pubsub")
	}

	// The legacy audit log manager channel must stay untouched while PubSub is enabled.
	select {
	case <-s.mockAuditLogC:
		s.Fail("unexpected message on legacy audit log manager channel while PubSub is enabled")
	case <-time.After(200 * time.Millisecond):
	}
}
