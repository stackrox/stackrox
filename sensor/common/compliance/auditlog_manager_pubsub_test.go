package compliance

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/events"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestAuditLogManagerWithDispatcher(tb testing.TB, dispatcher common.PubSubDispatcher) *auditLogCollectionManagerImpl {
	tb.Helper()
	return &auditLogCollectionManagerImpl{
		clusterID:               &fakeClusterIDWaiter{},
		eligibleComplianceNodes: make(map[string]sensor.ComplianceService_CommunicateServer),
		fileStates:              make(map[string]*storage.AuditLogFileState),
		updaterTicker:           time.NewTicker(time.Minute),
		stopper:                 concurrency.NewStopper(),
		forceUpdateSig:          concurrency.NewSignal(),
		centralReady:            concurrency.NewSignal(),
		auditEventMsgs:          make(chan *sensor.MsgFromCompliance),
		fileStateUpdates:        make(chan *message.ExpiringMessage),
		pubSubDispatcher:        dispatcher,
	}
}

func TestAuditLogManagerHandleSensorOnlineEvent_SignalsCentralReady(t *testing.T) {
	a := newTestAuditLogManagerWithDispatcher(t, newTestOnlineOfflineDispatcher(t))
	require.False(t, a.centralReady.IsDone())

	require.NoError(t, a.handleSensorOnlineEvent(&events.SensorOnlineEvent{}))

	assert.True(t, a.centralReady.IsDone())
}

func TestAuditLogManagerHandleSensorOfflineEvent_ResetsCentralReady(t *testing.T) {
	a := newTestAuditLogManagerWithDispatcher(t, newTestOnlineOfflineDispatcher(t))
	a.centralReady.Signal()
	require.True(t, a.centralReady.IsDone())

	require.NoError(t, a.handleSensorOfflineEvent(&events.SensorOfflineEvent{}))

	assert.False(t, a.centralReady.IsDone())
}

func TestAuditLogManagerHandleSensorOnlineEvent_WrongEventType(t *testing.T) {
	a := newTestAuditLogManagerWithDispatcher(t, newTestOnlineOfflineDispatcher(t))

	err := a.handleSensorOnlineEvent(&events.SensorOfflineEvent{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

func TestAuditLogManagerHandleSensorOfflineEvent_WrongEventType(t *testing.T) {
	a := newTestAuditLogManagerWithDispatcher(t, newTestOnlineOfflineDispatcher(t))

	err := a.handleSensorOfflineEvent(&events.SensorOnlineEvent{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

// TestAuditLogManagerStart_RegistersSensorOnlineOfflineConsumers verifies
// Start() wires the online/offline handlers onto the SensorOnline/
// SensorOffline topic and lane end-to-end through a real dispatcher.
func TestAuditLogManagerStart_RegistersSensorOnlineOfflineConsumers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dispatcher := newTestOnlineOfflineDispatcher(t)
		a := newTestAuditLogManagerWithDispatcher(t, dispatcher)
		require.NoError(t, a.Start())
		defer a.Stop()

		require.NoError(t, dispatcher.Publish(&events.SensorOnlineEvent{}))
		synctest.Wait()
		assert.True(t, a.centralReady.IsDone())

		require.NoError(t, dispatcher.Publish(&events.SensorOfflineEvent{}))
		synctest.Wait()
		assert.False(t, a.centralReady.IsDone())
	})
}
