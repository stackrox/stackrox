package output

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/common/detector/mocks"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

const (
	waitTimeout = 100 * time.Millisecond
)

// fakeDispatcher captures the EventCallback registered by New() so tests can
// invoke it directly, simulating what the real PubSub dispatcher would do.
type fakeDispatcher struct {
	callback pubsub.EventCallback
}

func (f *fakeDispatcher) RegisterConsumerToLane(_ pubsub.ConsumerID, _ pubsub.Topic, _ pubsub.LaneID, cb pubsub.EventCallback) error {
	f.callback = cb
	return nil
}

// outputTestEnv bundles the output queue, its dispatcher (if pubsub), and a
// helper to send events through the correct path.
type outputTestEnv struct {
	queue      component.OutputQueue
	dispatcher *fakeDispatcher
	pubsub     bool
}

func newOutputTestEnv(t *testing.T, det *mocks.MockDetector, pubsubEnabled bool, queueSize int) *outputTestEnv {
	t.Helper()
	t.Setenv(features.SensorInternalPubSub.EnvVar(), fmt.Sprintf("%t", pubsubEnabled))

	env := &outputTestEnv{pubsub: pubsubEnabled}
	var disp pubSubRegister
	if pubsubEnabled {
		env.dispatcher = &fakeDispatcher{}
		disp = env.dispatcher
	}
	q, err := New(det, queueSize, disp)
	assert.NoError(t, err)
	env.queue = q
	return env
}

func (e *outputTestEnv) send(t *testing.T, event *component.ResourceEvent) {
	t.Helper()
	if e.pubsub {
		assert.NoError(t, e.dispatcher.callback(event))
	} else {
		e.queue.Send(event)
	}
}

func shouldForwardMessage(t *testing.T, ch <-chan *message.ExpiringMessage) {
	select {
	case <-ch:
	case <-time.After(waitTimeout):
		t.Error("expecting event to have arrived")
	}
}

func shouldNotForwardMessage(t *testing.T, ch <-chan *message.ExpiringMessage) {
	select {
	case <-ch:
		t.Error("expecting event to not have arrived")
	case <-time.After(waitTimeout):
	}
}

func Test_OutputQueue_ExpiringMessages(t *testing.T) {
	expiredContext, cancel := context.WithCancel(context.Background())
	cancel()

	testCases := map[string]struct {
		message   *component.ResourceEvent
		assertion func(*testing.T, <-chan *message.ExpiringMessage)
	}{
		"Empty context are treated as Background": {
			message: &component.ResourceEvent{
				ForwardMessages: []*central.SensorEvent{{Id: "a"}},
				Context:         nil,
			},
			assertion: shouldForwardMessage,
		},
		"Not expired message is sent": {
			message: &component.ResourceEvent{
				ForwardMessages: []*central.SensorEvent{{Id: "a"}},
				Context:         context.Background(),
			},
			assertion: shouldForwardMessage,
		},
		"Expired message is not sent": {
			message: &component.ResourceEvent{
				ForwardMessages: []*central.SensorEvent{{Id: "a"}},
				Context:         expiredContext,
			},
			assertion: shouldNotForwardMessage,
		},
	}

	for _, pubsubEnabled := range []bool{false, true} {
		for name, tc := range testCases {
			t.Run(fmt.Sprintf("%s/pubsub=%t", name, pubsubEnabled), func(t *testing.T) {
				ctrl := gomock.NewController(t)
				det := mocks.NewMockDetector(ctrl)
				det.EXPECT().ReprocessDeployments(gomock.Eq([]string{}))

				env := newOutputTestEnv(t, det, pubsubEnabled, 10)
				assert.NoError(t, env.queue.Start())
				defer env.queue.Stop()

				env.send(t, tc.message)
				tc.assertion(t, env.queue.ResponsesC())
			})
		}
	}
}

func Test_OutputQueue_ForwardMessages(t *testing.T) {
	expiredCtx, cancel := context.WithCancel(context.Background())
	cancel()

	testCases := map[string]struct {
		message   *component.ResourceEvent
		assertion func(*testing.T, <-chan *message.ExpiringMessage)
	}{
		"nil context treated as Background": {
			message: &component.ResourceEvent{
				Context:         nil,
				ForwardMessages: []*central.SensorEvent{{Id: "a"}},
			},
			assertion: shouldForwardMessage,
		},
		"non-expired message is forwarded": {
			message: &component.ResourceEvent{
				Context:         context.Background(),
				ForwardMessages: []*central.SensorEvent{{Id: "a"}},
			},
			assertion: shouldForwardMessage,
		},
		"expired message is not forwarded": {
			message: &component.ResourceEvent{
				Context:         expiredCtx,
				ForwardMessages: []*central.SensorEvent{{Id: "a"}},
			},
			assertion: shouldNotForwardMessage,
		},
	}

	for _, pubsubEnabled := range []bool{false, true} {
		for name, tc := range testCases {
			t.Run(fmt.Sprintf("%s/pubsub=%t", name, pubsubEnabled), func(t *testing.T) {
				ctrl := gomock.NewController(t)
				det := mocks.NewMockDetector(ctrl)
				det.EXPECT().ReprocessDeployments(gomock.Eq([]string{}))

				env := newOutputTestEnv(t, det, pubsubEnabled, 10)
				assert.NoError(t, env.queue.Start())
				defer env.queue.Stop()

				env.send(t, tc.message)
				tc.assertion(t, env.queue.ResponsesC())
			})
		}
	}
}

func Test_OutputQueue_DetectorCalls(t *testing.T) {
	for _, pubsubEnabled := range []bool{false, true} {
		t.Run(fmt.Sprintf("pubsub=%t", pubsubEnabled), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			det := mocks.NewMockDetector(ctrl)
			det.EXPECT().ReprocessDeployments("dep-a", "dep-b")
			det.EXPECT().ProcessDeployment(
				gomock.Any(),
				gomock.Any(),
				gomock.Eq(central.ResourceAction_CREATE_RESOURCE),
			)

			env := newOutputTestEnv(t, det, pubsubEnabled, 10)
			assert.NoError(t, env.queue.Start())
			defer env.queue.Stop()

			env.send(t, &component.ResourceEvent{
				Context:              context.Background(),
				ReprocessDeployments: []string{"dep-a", "dep-b"},
				DetectorMessages: []component.DeploytimeDetectionRequest{
					{
						Object: &storage.Deployment{Id: "dep-c"},
						Action: central.ResourceAction_CREATE_RESOURCE,
					},
				},
			})
		})
	}
}

// wrongTypeEvent satisfies pubsub.Event but is not *component.ResourceEvent,
// used to exercise the type-assertion guard in ProcessResourceEvent.
type wrongTypeEvent struct{}

func (w *wrongTypeEvent) Topic() pubsub.Topic { return pubsub.ResolvedResourceEventTopic }
func (w *wrongTypeEvent) Lane() pubsub.LaneID { return pubsub.ResolvedResourceEventLane }

func Test_ProcessResourceEvent_WrongType(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")
	ctrl := gomock.NewController(t)
	det := mocks.NewMockDetector(ctrl)

	disp := &fakeDispatcher{}
	q, err := New(det, 10, disp)
	assert.NoError(t, err)
	assert.NoError(t, q.Start())
	defer q.Stop()

	err = disp.callback(&wrongTypeEvent{})
	assert.ErrorContains(t, err, "unable to convert event to *component.ResourceEvent")
}

func Test_ProcessResourceEvent_StopRequested(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")
	ctrl := gomock.NewController(t)
	det := mocks.NewMockDetector(ctrl)

	disp := &fakeDispatcher{}
	q, err := New(det, 10, disp)
	assert.NoError(t, err)
	assert.NoError(t, q.Start())
	q.Stop()

	err = disp.callback(&component.ResourceEvent{
		Context:         context.Background(),
		ForwardMessages: []*central.SensorEvent{{Id: "x"}},
	})
	assert.NoError(t, err)
	shouldNotForwardMessage(t, q.ResponsesC())
}
