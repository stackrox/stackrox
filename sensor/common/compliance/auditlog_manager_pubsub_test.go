package compliance

import (
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubDispatcher "github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
	"github.com/stretchr/testify/require"
)

func newTestAuditLogManagerDispatcher(t *testing.T) common.PubSubDispatcher {
	t.Helper()
	dispatcher, err := pubsubDispatcher.NewDispatcher(pubsubDispatcher.WithLaneConfigs(
		[]pubsub.LaneConfig{
			lane.NewBlockingLane(pubsub.AuditLogManagerLane),
		},
	))
	require.NoError(t, err)
	return dispatcher
}

// TestPubSubAuditLogManagerEventUpdatesFileState verifies that publishing an
// AuditLogManagerEvent updates file state the same way the legacy AuditMessagesChan() path does.
func (s *AuditLogCollectionManagerTestSuite) TestPubSubAuditLogManagerEventUpdatesFileState() {
	t := s.T()
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

	dispatcher := newTestAuditLogManagerDispatcher(t)
	defer dispatcher.Stop()

	manager := s.getManager(make(map[string]sensor.ComplianceService_CommunicateServer), nil)
	manager.pubSubDispatcher = dispatcher
	manager.enabled.Set(true)

	s.Require().NoError(manager.Start())
	defer manager.Stop()

	expectedFileStates := make(map[string]*storage.AuditLogFileState)
	startTime := time.Now()
	for node := range 2 {
		nodeName := fmt.Sprintf("node-%d", node)
		msg := s.getMsgFromCompliance(nodeName, startTime)
		expectedFileStates[nodeName] = s.getAuditLogFileState(startTime, msg.GetAuditEvents().GetEvents()[0].GetId())

		require.NoError(t, dispatcher.Publish(&AuditLogManagerEvent{Node: msg.GetNode(), AuditEvents: msg.GetAuditEvents()}))
	}

	s.Eventually(func() bool {
		return len(manager.getLatestFileStates()) == len(expectedFileStates)
	}, updateTimeout, 10*time.Millisecond)

	protoassert.MapEqual(t, expectedFileStates, manager.getLatestFileStates())

	// The legacy channel must stay untouched while PubSub is enabled.
	select {
	case <-manager.auditEventMsgs:
		s.Fail("unexpected message on legacy auditEventMsgs channel while PubSub is enabled")
	case <-time.After(200 * time.Millisecond):
	}
}

func (s *AuditLogCollectionManagerTestSuite) TestHandleAuditLogManagerEventWrongType() {
	manager := s.getManager(make(map[string]sensor.ComplianceService_CommunicateServer), nil)
	s.Error(manager.handleAuditLogManagerEvent(&wrongAuditLogManagerEvent{}))
}

// TestPubSubDisabledFallsBackToAuditMessagesChan verifies that with PubSub disabled, the
// manager behaves exactly as before: only the legacy AuditMessagesChan() path is used, even
// with a live dispatcher present.
func (s *AuditLogCollectionManagerTestSuite) TestPubSubDisabledFallsBackToAuditMessagesChan() {
	t := s.T()
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "false")

	dispatcher := newTestAuditLogManagerDispatcher(t)
	defer dispatcher.Stop()

	manager := s.getManager(make(map[string]sensor.ComplianceService_CommunicateServer), nil)
	manager.pubSubDispatcher = dispatcher

	s.Require().NoError(manager.Start())
	defer manager.Stop()

	startTime := time.Now()
	msg := s.getMsgFromCompliance("node-legacy", startTime)
	expected := s.getAuditLogFileState(startTime, msg.GetAuditEvents().GetEvents()[0].GetId())

	manager.AuditMessagesChan() <- msg

	s.Eventually(func() bool {
		return len(manager.getLatestFileStates()) == 1
	}, updateTimeout, 10*time.Millisecond)

	protoassert.MapEqual(t, map[string]*storage.AuditLogFileState{"node-legacy": expected}, manager.getLatestFileStates())
}

type wrongAuditLogManagerEvent struct{}

func (*wrongAuditLogManagerEvent) Topic() pubsub.Topic { return pubsub.DefaultTopic }
func (*wrongAuditLogManagerEvent) Lane() pubsub.LaneID { return pubsub.DefaultLane }
