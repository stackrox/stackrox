package detector

import (
	"testing"
	"testing/synctest"

	sensorEvents "github.com/stackrox/rox/sensor/common/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleSensorOnlineEvent_SignalsRuntimeRunning(t *testing.T) {
	d, _, _, _ := createTestDetector(t, true)
	require.False(t, d.runtimeRunning.IsDone())

	require.NoError(t, d.handleSensorOnlineEvent(&sensorEvents.SensorOnlineEvent{}))

	assert.True(t, d.runtimeRunning.IsDone())
}

func TestHandleSensorOfflineEvent_ResetsRuntimeRunning(t *testing.T) {
	d, _, _, _ := createTestDetector(t, true)
	d.runtimeRunning.Signal()
	require.True(t, d.runtimeRunning.IsDone())

	require.NoError(t, d.handleSensorOfflineEvent(&sensorEvents.SensorOfflineEvent{}))

	assert.False(t, d.runtimeRunning.IsDone())
}

func TestHandleSensorOnlineEvent_WrongEventType(t *testing.T) {
	d, _, _, _ := createTestDetector(t, true)

	err := d.handleSensorOnlineEvent(&sensorEvents.SensorOfflineEvent{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

func TestHandleSensorOfflineEvent_WrongEventType(t *testing.T) {
	d, _, _, _ := createTestDetector(t, true)

	err := d.handleSensorOfflineEvent(&sensorEvents.SensorOnlineEvent{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

// TestDetectorStart_RegistersSensorOnlineOfflineConsumers verifies Start()
// wires the online/offline handlers onto the SensorOnline/SensorOffline
// topic and lane, and that a published event flips runtimeRunning end-to-end
// through the dispatcher. This is the regression test for ROX-35620, where
// runtimeRunning was driven by the legacy Notify call even in PubSub mode,
// racing against the dispatcher's own PubSub-driven consumers.
func TestDetectorStart_RegistersSensorOnlineOfflineConsumers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d, _, _, _ := createTestDetector(t, true)
		require.NoError(t, d.Start())
		defer d.Stop()

		require.NoError(t, d.pubSubDispatcher.Publish(&sensorEvents.SensorOnlineEvent{}))
		synctest.Wait()
		assert.True(t, d.runtimeRunning.IsDone())

		require.NoError(t, d.pubSubDispatcher.Publish(&sensorEvents.SensorOfflineEvent{}))
		synctest.Wait()
		assert.False(t, d.runtimeRunning.IsDone())
	})
}
