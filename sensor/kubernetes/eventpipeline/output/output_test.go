package output

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
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
	// This test exercises the legacy channel-based output queue path.
	// Disable the feature flag so New does not require a pubsub dispatcher.
	t.Setenv("ROX_SENSOR_PUBSUB", "false")
	ctrl := gomock.NewController(t)
	detector := mocks.NewMockDetector(ctrl)
	q, err := New(detector, 10, nil)
	assert.NoError(t, err)

	assert.NoError(t, q.Start())
	defer q.Stop()

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

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			detector.EXPECT().ReprocessDeployments(gomock.Eq([]string{}))
			q.Send(tc.message)
			tc.assertion(t, q.ResponsesC())
		})
	}
}

// wrongTypeEvent satisfies pubsub.Event but is not *component.ResourceEvent,
// used to exercise the type-assertion guard in ProcessResourceEvent.
type wrongTypeEvent struct{}

func (w *wrongTypeEvent) Topic() pubsub.Topic { return pubsub.ResolvedResourceEventTopic }
func (w *wrongTypeEvent) Lane() pubsub.LaneID { return pubsub.ResolvedResourceEventLane }

func Test_ProcessResourceEvent_WrongType(t *testing.T) {
	ctrl := gomock.NewController(t)
	q := &outputQueueImpl{
		innerQueue:   make(chan *component.ResourceEvent, 1),
		forwardQueue: make(chan *message.ExpiringMessage, 1),
		detector:     mocks.NewMockDetector(ctrl),
		stopper:      concurrency.NewStopper(),
	}
	err := q.ProcessResourceEvent(&wrongTypeEvent{})
	assert.ErrorContains(t, err, "unable to convert event to *component.ResourceEvent")
}

func Test_ProcessResourceEvent_StopRequested(t *testing.T) {
	ctrl := gomock.NewController(t)
	// No detector expectations: the stop guard fires before any processing.
	q := &outputQueueImpl{
		innerQueue:   make(chan *component.ResourceEvent, 1),
		forwardQueue: make(chan *message.ExpiringMessage, 1),
		detector:     mocks.NewMockDetector(ctrl),
		stopper:      concurrency.NewStopper(),
	}
	q.stopper.Client().Stop()
	err := q.ProcessResourceEvent(&component.ResourceEvent{
		Context:         context.Background(),
		ForwardMessages: []*central.SensorEvent{{Id: "x"}},
	})
	assert.NoError(t, err)
	shouldNotForwardMessage(t, q.ResponsesC())
}

func Test_ProcessResourceEvent_ForwardMessages(t *testing.T) {
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

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			det := mocks.NewMockDetector(ctrl)
			det.EXPECT().ReprocessDeployments(gomock.Eq([]string{}))
			q := &outputQueueImpl{
				innerQueue:   make(chan *component.ResourceEvent, 1),
				forwardQueue: make(chan *message.ExpiringMessage, 10),
				detector:     det,
				stopper:      concurrency.NewStopper(),
			}
			assert.NoError(t, q.ProcessResourceEvent(tc.message))
			tc.assertion(t, q.ResponsesC())
		})
	}
}

func Test_ProcessResourceEvent_DetectorCalls(t *testing.T) {
	ctrl := gomock.NewController(t)
	det := mocks.NewMockDetector(ctrl)
	det.EXPECT().ReprocessDeployments("dep-a", "dep-b")
	det.EXPECT().ProcessDeployment(
		gomock.Any(),
		gomock.Any(),
		gomock.Eq(central.ResourceAction_CREATE_RESOURCE),
	)

	q := &outputQueueImpl{
		innerQueue:   make(chan *component.ResourceEvent, 1),
		forwardQueue: make(chan *message.ExpiringMessage, 10),
		detector:     det,
		stopper:      concurrency.NewStopper(),
	}
	assert.NoError(t, q.ProcessResourceEvent(&component.ResourceEvent{
		Context:              context.Background(),
		ReprocessDeployments: []string{"dep-a", "dep-b"},
		DetectorMessages: []component.DeploytimeDetectionRequest{
			{
				Object: &storage.Deployment{Id: "dep-c"},
				Action: central.ResourceAction_CREATE_RESOURCE,
			},
		},
	}))
}
