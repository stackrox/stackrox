package output

import (
	"context"
	"fmt"
	"testing"
	"testing/synctest"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/common/detector/mocks"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubDispatcher "github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// testDispatcher extends pubSubRegister with Publish and Stop so tests can
// send events through the real dispatcher machinery.
type testDispatcher interface {
	pubSubRegister
	Publish(pubsub.Event) error
	Stop()
}

// outputTestEnv bundles the output queue, its dispatcher (if pubsub), and a
// helper to send events through the correct path.
type outputTestEnv struct {
	queue    component.OutputQueue
	pubsub   bool
	dispatch testDispatcher
}

func newOutputTestEnv(t *testing.T, det *mocks.MockDetector, pubsubEnabled bool, queueSize int) *outputTestEnv {
	t.Helper()
	t.Setenv(features.SensorInternalPubSub.EnvVar(), fmt.Sprintf("%t", pubsubEnabled))

	env := &outputTestEnv{pubsub: pubsubEnabled}
	var reg pubSubRegister
	if pubsubEnabled {
		d, err := pubsubDispatcher.NewDispatcher(pubsubDispatcher.WithLaneConfigs([]pubsub.LaneConfig{
			lane.NewBlockingLane(pubsub.ResolvedResourceEventLane),
		}))
		require.NoError(t, err)
		t.Cleanup(d.Stop)
		env.dispatch = d
		reg = d
	}
	q, err := New(det, queueSize, reg)
	assert.NoError(t, err)
	env.queue = q
	return env
}

func (e *outputTestEnv) send(t *testing.T, event *component.ResourceEvent) {
	t.Helper()
	if e.pubsub {
		event.SetTopicAndLane(pubsub.ResolvedResourceEventTopic, pubsub.ResolvedResourceEventLane)
		assert.NoError(t, e.dispatch.Publish(event))
	} else {
		e.queue.Send(event)
	}
}

func shouldHaveForwarded(t *testing.T, ch <-chan *message.ExpiringMessage) {
	t.Helper()
	select {
	case <-ch:
	default:
		t.Fatal("expecting event to have arrived")
	}
}

func shouldNotHaveForwarded(t *testing.T, ch <-chan *message.ExpiringMessage) {
	t.Helper()
	select {
	case <-ch:
		t.Error("expecting event to not have arrived")
	default:
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
			assertion: shouldHaveForwarded,
		},
		"Not expired message is sent": {
			message: &component.ResourceEvent{
				ForwardMessages: []*central.SensorEvent{{Id: "a"}},
				Context:         context.Background(),
			},
			assertion: shouldHaveForwarded,
		},
		"Expired message is not sent": {
			message: &component.ResourceEvent{
				ForwardMessages: []*central.SensorEvent{{Id: "a"}},
				Context:         expiredContext,
			},
			assertion: shouldNotHaveForwarded,
		},
	}

	for _, pubsubEnabled := range []bool{false, true} {
		for name, tc := range testCases {
			t.Run(fmt.Sprintf("%s/pubsub=%t", name, pubsubEnabled), func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					ctrl := gomock.NewController(t)
					det := mocks.NewMockDetector(ctrl)
					det.EXPECT().ReprocessDeployments()

					env := newOutputTestEnv(t, det, pubsubEnabled, 10)
					require.NoError(t, env.queue.Start())
					defer env.queue.Stop()

					env.send(t, tc.message)
					synctest.Wait()
					tc.assertion(t, env.queue.ResponsesC())
				})
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
			assertion: shouldHaveForwarded,
		},
		"non-expired message is forwarded": {
			message: &component.ResourceEvent{
				Context:         context.Background(),
				ForwardMessages: []*central.SensorEvent{{Id: "a"}},
			},
			assertion: shouldHaveForwarded,
		},
		"expired message is not forwarded": {
			message: &component.ResourceEvent{
				Context:         expiredCtx,
				ForwardMessages: []*central.SensorEvent{{Id: "a"}},
			},
			assertion: shouldNotHaveForwarded,
		},
	}

	for _, pubsubEnabled := range []bool{false, true} {
		for name, tc := range testCases {
			t.Run(fmt.Sprintf("%s/pubsub=%t", name, pubsubEnabled), func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					ctrl := gomock.NewController(t)
					det := mocks.NewMockDetector(ctrl)
					det.EXPECT().ReprocessDeployments()

					env := newOutputTestEnv(t, det, pubsubEnabled, 10)
					require.NoError(t, env.queue.Start())
					defer env.queue.Stop()

					env.send(t, tc.message)
					synctest.Wait()
					tc.assertion(t, env.queue.ResponsesC())
				})
			})
		}
	}
}

func Test_OutputQueue_DetectorCalls(t *testing.T) {
	for _, pubsubEnabled := range []bool{false, true} {
		t.Run(fmt.Sprintf("pubsub=%t", pubsubEnabled), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctrl := gomock.NewController(t)
				det := mocks.NewMockDetector(ctrl)
				det.EXPECT().ReprocessDeployments("dep-a", "dep-b")
				det.EXPECT().ProcessDeployment(
					gomock.Any(),
					gomock.Any(),
					gomock.Eq(central.ResourceAction_CREATE_RESOURCE),
				)

				env := newOutputTestEnv(t, det, pubsubEnabled, 10)
				require.NoError(t, env.queue.Start())
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
				synctest.Wait()
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

	d, err := pubsubDispatcher.NewDispatcher(pubsubDispatcher.WithLaneConfigs([]pubsub.LaneConfig{
		lane.NewBlockingLane(pubsub.ResolvedResourceEventLane),
	}))
	require.NoError(t, err)
	t.Cleanup(d.Stop)

	q, err := New(det, 10, d)
	assert.NoError(t, err)
	assert.NoError(t, q.Start())
	defer q.Stop()

	impl := q.(*outputQueueImpl)
	err = impl.ProcessResourceEvent(&wrongTypeEvent{})
	assert.ErrorContains(t, err, "unable to convert event to *component.ResourceEvent")
}

func Test_ProcessResourceEvent_StopRequested(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")
		ctrl := gomock.NewController(t)
		det := mocks.NewMockDetector(ctrl)

		d, err := pubsubDispatcher.NewDispatcher(pubsubDispatcher.WithLaneConfigs([]pubsub.LaneConfig{
			lane.NewBlockingLane(pubsub.ResolvedResourceEventLane),
		}))
		require.NoError(t, err)
		t.Cleanup(d.Stop)

		q, err := New(det, 10, d)
		assert.NoError(t, err)
		assert.NoError(t, q.Start())
		q.Stop()

		event := &component.ResourceEvent{
			Context:         context.Background(),
			ForwardMessages: []*central.SensorEvent{{Id: "x"}},
		}
		event.SetTopicAndLane(pubsub.ResolvedResourceEventTopic, pubsub.ResolvedResourceEventLane)
		assert.NoError(t, d.Publish(event))
		synctest.Wait()
		shouldNotHaveForwarded(t, q.ResponsesC())
	})
}
