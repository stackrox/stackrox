package output

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/common/detector/mocks"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"go.uber.org/mock/gomock"
)

func benchOutputQueue(b *testing.B, pubsubEnabled bool, makeEvent func() *component.ResourceEvent) {
	b.Helper()
	b.Setenv(features.SensorInternalPubSub.EnvVar(), fmt.Sprintf("%t", pubsubEnabled))

	ctrl := gomock.NewController(b)
	det := mocks.NewMockDetector(ctrl)
	det.EXPECT().ReprocessDeployments(gomock.Any()).AnyTimes()
	det.EXPECT().ProcessDeployment(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	var disp *fakeDispatcher
	var reg pubSubRegister
	if pubsubEnabled {
		disp = &fakeDispatcher{}
		reg = disp
	}
	q, err := New(det, 1024, reg)
	if err != nil {
		b.Fatal(err)
	}
	if err := q.Start(); err != nil {
		b.Fatal(err)
	}

	stopDrain := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		ch := q.ResponsesC()
		for {
			select {
			case <-ch:
			case <-stopDrain:
				return
			}
		}
	}()
	b.Cleanup(func() {
		q.Stop()
		close(stopDrain)
		<-done
	})

	b.ResetTimer()
	for b.Loop() {
		event := makeEvent()
		if pubsubEnabled {
			if err := disp.callback(event); err != nil {
				b.Fatal(err)
			}
		} else {
			q.Send(event)
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
			benchOutputQueue(b, pubsubEnabled, makeEvent)
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
			benchOutputQueue(b, pubsubEnabled, makeEvent)
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
			benchOutputQueue(b, pubsubEnabled, makeEvent)
		})
	}
}
