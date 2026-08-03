package clusterstatus

import (
	"context"
	"testing"
	"testing/synctest"

	configFake "github.com/openshift/client-go/config/clientset/versioned/fake"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/events"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubDispatcher "github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/fake"
)

func newTestOnlineOfflineDispatcher(tb testing.TB) common.PubSubDispatcher {
	tb.Helper()
	dispatcher, err := pubsubDispatcher.NewDispatcher(pubsubDispatcher.WithLaneConfigs([]pubsub.LaneConfig{
		lane.NewBlockingLane(pubsub.SensorOnlineLane),
		lane.NewBlockingLane(pubsub.SensorOfflineLane),
	}))
	require.NoError(tb, err)
	tb.Cleanup(dispatcher.Stop)
	return dispatcher
}

func newTestUpdaterWithDispatcher(tb testing.TB, dispatcher common.PubSubDispatcher) *updaterImpl {
	tb.Helper()
	u := NewUpdater(&fakeClientSet{
		k8s:    fake.NewClientset(),
		config: configFake.NewClientset(),
	}, dispatcher)
	impl := u.(*updaterImpl)
	// Avoid real cloud-provider metadata lookups (e.g. AWS IMDS) from the
	// background run() goroutine started by handleSensorOnlineEvent.
	impl.getProviders = func(context.Context) *storage.ProviderMetadata { return nil }
	return impl
}

func TestHandleSensorOnlineEvent_ClearsOfflineModeAndRuns(t *testing.T) {
	u := newTestUpdaterWithDispatcher(t, newTestOnlineOfflineDispatcher(t))
	require.True(t, u.offlineMode.Load())

	require.NoError(t, u.handleSensorOnlineEvent(&events.SensorOnlineEvent{}))

	assert.False(t, u.offlineMode.Load())
}

func TestHandleSensorOfflineEvent_SetsOfflineMode(t *testing.T) {
	u := newTestUpdaterWithDispatcher(t, newTestOnlineOfflineDispatcher(t))
	u.offlineMode.Store(false)

	require.NoError(t, u.handleSensorOfflineEvent(&events.SensorOfflineEvent{}))

	assert.True(t, u.offlineMode.Load())
}

func TestHandleSensorOnlineEvent_WrongEventType(t *testing.T) {
	u := newTestUpdaterWithDispatcher(t, newTestOnlineOfflineDispatcher(t))

	err := u.handleSensorOnlineEvent(&events.SensorOfflineEvent{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

func TestHandleSensorOfflineEvent_WrongEventType(t *testing.T) {
	u := newTestUpdaterWithDispatcher(t, newTestOnlineOfflineDispatcher(t))

	err := u.handleSensorOfflineEvent(&events.SensorOnlineEvent{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected event type")
}

// TestHandleSensorOnlineEvent_CASGuardPreventsDoubleRun verifies the
// offlineMode CompareAndSwap guard carried over from the legacy Notify path:
// a second consecutive online event must not re-trigger createContext/run.
func TestHandleSensorOnlineEvent_CASGuardPreventsDoubleRun(t *testing.T) {
	u := newTestUpdaterWithDispatcher(t, newTestOnlineOfflineDispatcher(t))

	require.NoError(t, u.handleSensorOnlineEvent(&events.SensorOnlineEvent{}))
	ctxAfterFirst := u.getCurrentContext()

	require.NoError(t, u.handleSensorOnlineEvent(&events.SensorOnlineEvent{}))
	ctxAfterSecond := u.getCurrentContext()

	assert.Same(t, ctxAfterFirst, ctxAfterSecond, "second online event must not recreate the context")
}

// TestUpdaterStart_RegistersSensorOnlineOfflineConsumers verifies Start()
// wires the online/offline handlers onto the SensorOnline/SensorOffline
// topic and lane end-to-end through a real dispatcher.
func TestUpdaterStart_RegistersSensorOnlineOfflineConsumers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dispatcher := newTestOnlineOfflineDispatcher(t)
		u := newTestUpdaterWithDispatcher(t, dispatcher)
		require.NoError(t, u.Start())
		defer u.Stop()

		require.NoError(t, dispatcher.Publish(&events.SensorOnlineEvent{}))
		synctest.Wait()
		assert.False(t, u.offlineMode.Load())

		require.NoError(t, dispatcher.Publish(&events.SensorOfflineEvent{}))
		synctest.Wait()
		assert.True(t, u.offlineMode.Load())
	})
}
