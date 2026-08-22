package complianceoperator

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubDispatcher "github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
)

func newTestComplianceOperatorDispatcher(t *testing.T) common.PubSubDispatcher {
	t.Helper()
	dispatcher, err := pubsubDispatcher.NewDispatcher(pubsubDispatcher.WithLaneConfigs(
		[]pubsub.LaneConfig{
			lane.NewBlockingLane(pubsub.ComplianceOperatorRequestLane),
		},
	))
	require.NoError(t, err)
	return dispatcher
}

// TestPubSubComplianceRequestRouted verifies ProcessMessage() publishes a
// ComplianceOperatorRequestEvent instead of writing to the legacy request channel when PubSub is
// enabled, and that the response still comes out the other end via ResponsesC().
func TestPubSubComplianceRequestRouted(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

	dispatcher := newTestComplianceOperatorDispatcher(t)
	defer dispatcher.Stop()

	client := fake.NewSimpleDynamicClient(runtime.NewScheme())
	readySignal := concurrency.NewSignal()
	handler := NewRequestHandler(client, nil, &readySignal, dispatcher)
	require.NoError(t, handler.Start())
	defer handler.Stop()

	msg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_ComplianceRequest{
			ComplianceRequest: &central.ComplianceRequest{
				Request: &central.ComplianceRequest_DisableCompliance{
					DisableCompliance: &central.DisableComplianceRequest{
						Id: uuid.NewV4().String(),
					},
				},
			},
		},
	}
	require.NoError(t, handler.ProcessMessage(t.Context(), msg))

	select {
	case response := <-handler.ResponsesC():
		disableResponse := response.Msg.(*central.MsgFromSensor_ComplianceResponse).ComplianceResponse.GetDisableComplianceResponse()
		require.NotNil(t, disableResponse)
		require.Equal(t, msg.GetComplianceRequest().GetDisableCompliance().GetId(), disableResponse.GetId())
		require.Empty(t, disableResponse.GetError())
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for compliance response via pubsub")
	}
}

// TestPubSubComplianceRequest_WrongEventType verifies handleComplianceRequestEvent rejects
// events of an unexpected type instead of silently ignoring or panicking.
func TestPubSubComplianceRequest_WrongEventType(t *testing.T) {
	client := fake.NewSimpleDynamicClient(runtime.NewScheme())
	readySignal := concurrency.NewSignal()
	handler := NewRequestHandler(client, nil, &readySignal, nil).(*handlerImpl)

	require.Error(t, handler.handleComplianceRequestEvent(&wrongComplianceOperatorEvent{}))
}

type wrongComplianceOperatorEvent struct{}

func (*wrongComplianceOperatorEvent) Topic() pubsub.Topic { return pubsub.DefaultTopic }
func (*wrongComplianceOperatorEvent) Lane() pubsub.LaneID { return pubsub.DefaultLane }
