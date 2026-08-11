package configmap

import (
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubDispatcher "github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func newTestConfigMapDispatcher(t *testing.T) common.PubSubDispatcher {
	t.Helper()
	dispatcher, err := pubsubDispatcher.NewDispatcher(pubsubDispatcher.WithLaneConfigs(
		[]pubsub.LaneConfig{
			lane.NewBlockingLane(pubsub.AdmCtrlConfigMapLane),
		},
	))
	require.NoError(t, err)
	return dispatcher
}

// TestPubSubConfigMapPersisted verifies that publishing a ConfigMapUpdatedEvent results in the
// persister creating (and later updating) the ConfigMap via the Kubernetes API, instead of relying
// on the legacy ValueStream iterator, when PubSub is enabled.
func TestPubSubConfigMapPersisted(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

	dispatcher := newTestConfigMapDispatcher(t)
	defer dispatcher.Stop()

	k8s := fake.NewSimpleClientset()
	persister := NewConfigMapPersister("admissionController", "stackrox", k8s, nil, dispatcher)
	require.NoError(t, persister.Start())
	defer persister.Stop()

	configMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "admission-control"},
		Data:       map[string]string{"key": "v1"},
	}
	require.NoError(t, dispatcher.Publish(&ConfigMapUpdatedEvent{ConfigMap: configMap}))

	require.Eventually(t, func() bool {
		cm, err := k8s.CoreV1().ConfigMaps("stackrox").Get(t.Context(), "admission-control", metav1.GetOptions{})
		return err == nil && cm.Data["key"] == "v1"
	}, 5*time.Second, 10*time.Millisecond, "config map was not created via pubsub")

	updated := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "admission-control"},
		Data:       map[string]string{"key": "v2"},
	}
	require.NoError(t, dispatcher.Publish(&ConfigMapUpdatedEvent{ConfigMap: updated}))

	require.Eventually(t, func() bool {
		cm, err := k8s.CoreV1().ConfigMaps("stackrox").Get(t.Context(), "admission-control", metav1.GetOptions{})
		return err == nil && cm.Data["key"] == "v2"
	}, 5*time.Second, 10*time.Millisecond, "config map was not updated via pubsub")
}

// TestPubSubConfigMapEvent_WrongEventType verifies handleConfigMapEvent rejects events of an
// unexpected type instead of silently ignoring or panicking.
func TestPubSubConfigMapEvent_WrongEventType(t *testing.T) {
	k8s := fake.NewSimpleClientset()
	persister := NewConfigMapPersister("admissionController", "stackrox", k8s, nil, nil).(*configMapPersister)

	require.Error(t, persister.handleConfigMapEvent(&wrongConfigMapEvent{}))
}

type wrongConfigMapEvent struct{}

func (*wrongConfigMapEvent) Topic() pubsub.Topic { return pubsub.DefaultTopic }
func (*wrongConfigMapEvent) Lane() pubsub.LaneID { return pubsub.DefaultLane }
