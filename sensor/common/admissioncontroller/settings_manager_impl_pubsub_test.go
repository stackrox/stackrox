package admissioncontroller

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/configmap"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubDispatcher "github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
	storeMocks "github.com/stackrox/rox/sensor/common/store/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newTestSettingsManagerDispatcher(t *testing.T) common.PubSubDispatcher {
	t.Helper()
	dispatcher, err := pubsubDispatcher.NewDispatcher(pubsubDispatcher.WithLaneConfigs(
		[]pubsub.LaneConfig{
			lane.NewBlockingLane(pubsub.AdmCtrlConfigMapLane),
		},
	))
	require.NoError(t, err)
	return dispatcher
}

// TestPubSubPushSettingsPublishesConfigMap verifies pushSettings() publishes a
// configmap.ConfigMapUpdatedEvent instead of writing to the legacy configStream when PubSub is
// enabled, and leaves configStream untouched (that's a separate migration, ROX-36003).
func TestPubSubPushSettingsPublishesConfigMap(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

	dispatcher := newTestSettingsManagerDispatcher(t)
	defer dispatcher.Stop()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	deployments := storeMocks.NewMockDeploymentStore(ctrl)
	pods := storeMocks.NewMockPodStore(ctrl)

	mgr := NewSettingsManager(&mockClusterIDWaiter{id: "cluster-123"}, &mockClusterLabelsGetter{}, deployments, pods, nil, dispatcher)

	eventC := make(chan *configmap.ConfigMapUpdatedEvent, 1)
	require.NoError(t, dispatcher.RegisterConsumer(pubsub.AdmCtrlConfigMapConsumer, pubsub.AdmCtrlConfigMapTopic, func(event pubsub.Event) error {
		e, ok := event.(*configmap.ConfigMapUpdatedEvent)
		if !ok {
			return nil
		}
		eventC <- e
		return nil
	}))

	mgr.UpdatePolicies(nil)
	mgr.UpdateConfig(&storage.DynamicClusterConfig{})

	select {
	case event := <-eventC:
		require.NotNil(t, event.ConfigMap)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for config map update via pubsub")
	}

	require.Nil(t, mgr.(*settingsManager).configStream.Iterator(false).Value(), "configStream should not receive an update in pubsub mode")
}
