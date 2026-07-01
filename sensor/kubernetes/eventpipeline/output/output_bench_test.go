package output

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/common/detector/mocks"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubDispatcher "github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"go.uber.org/mock/gomock"
)

func benchOutputQueue(b *testing.B, pubsubEnabled bool, forwardCount int, makeEvent func() *component.ResourceEvent) {
	b.Helper()
	b.Setenv(features.SensorInternalPubSub.EnvVar(), fmt.Sprintf("%t", pubsubEnabled))

	ctrl := gomock.NewController(b)
	det := mocks.NewMockDetector(ctrl)
	det.EXPECT().ReprocessDeployments(gomock.Any()).AnyTimes()
	det.EXPECT().ProcessDeployment(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	var disp testDispatcher
	var reg pubSubRegister
	if pubsubEnabled {
		d, err := pubsubDispatcher.NewDispatcher(pubsubDispatcher.WithLaneConfigs([]pubsub.LaneConfig{
			lane.NewBlockingLane(pubsub.ResolvedResourceEventLane),
		}))
		if err != nil {
			b.Fatal(err)
		}
		b.Cleanup(d.Stop)
		disp = d
		reg = d
	}

	q, err := New(det, 1024, reg)
	if err != nil {
		b.Fatal(err)
	}
	if err := q.Start(); err != nil {
		b.Fatal(err)
	}
	b.Cleanup(q.Stop)

	b.ResetTimer()
	for b.Loop() {
		event := makeEvent()
		if pubsubEnabled {
			event.SetTopicAndLane(pubsub.ResolvedResourceEventTopic, pubsub.ResolvedResourceEventLane)
			if err := disp.Publish(event); err != nil {
				b.Fatal(err)
			}
		} else {
			q.Send(event)
		}
		for range forwardCount {
			<-q.ResponsesC()
		}
	}
}

func BenchmarkOutputQueue_SingleForward(b *testing.B) {
	makeEvent := func() *component.ResourceEvent {
		return &component.ResourceEvent{
			Context:         context.Background(),
			ForwardMessages: []*central.SensorEvent{{Id: "a"}},
		}
	}
	for _, pubsubEnabled := range []bool{false, true} {
		b.Run(fmt.Sprintf("pubsub=%t", pubsubEnabled), func(b *testing.B) {
			benchOutputQueue(b, pubsubEnabled, 1, makeEvent)
		})
	}
}

func BenchmarkOutputQueue_MultipleForward(b *testing.B) {
	makeEvent := func() *component.ResourceEvent {
		return &component.ResourceEvent{
			Context: context.Background(),
			ForwardMessages: []*central.SensorEvent{
				{Id: "a"},
				{Id: "b"},
				{Id: "c"},
			},
		}
	}
	for _, pubsubEnabled := range []bool{false, true} {
		b.Run(fmt.Sprintf("pubsub=%t", pubsubEnabled), func(b *testing.B) {
			benchOutputQueue(b, pubsubEnabled, 3, makeEvent)
		})
	}
}

func BenchmarkOutputQueue_WithDetector(b *testing.B) {
	makeEvent := func() *component.ResourceEvent {
		return &component.ResourceEvent{
			Context:              context.Background(),
			ForwardMessages:      []*central.SensorEvent{{Id: "a"}},
			ReprocessDeployments: []string{"dep-a", "dep-b"},
			DetectorMessages: []component.DeploytimeDetectionRequest{
				{
					Object: &storage.Deployment{Id: "dep-c"},
					Action: central.ResourceAction_CREATE_RESOURCE,
				},
			},
		}
	}
	for _, pubsubEnabled := range []bool{false, true} {
		b.Run(fmt.Sprintf("pubsub=%t", pubsubEnabled), func(b *testing.B) {
			benchOutputQueue(b, pubsubEnabled, 1, makeEvent)
		})
	}
}
