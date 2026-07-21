package manager

// Benchmarks for ResourceSyncFinished delivery through both the pubsub and legacy
// internalmessage paths.

import (
	"context"
	"runtime"
	"testing"

	"github.com/stackrox/rox/sensor/common"
	mocksDetector "github.com/stackrox/rox/sensor/common/detector/mocks"
	"github.com/stackrox/rox/sensor/common/events"
	mocksExternalSrc "github.com/stackrox/rox/sensor/common/externalsrcs/mocks"
	"github.com/stackrox/rox/sensor/common/internalmessage"
	mocksManager "github.com/stackrox/rox/sensor/common/networkflow/manager/mocks"
	"github.com/stackrox/rox/sensor/common/networkflow/updatecomputer"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubDispatcher "github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
	"go.uber.org/mock/gomock"
)

func BenchmarkResourceSyncDelivery(b *testing.B) {
	b.Run("legacy", func(b *testing.B) {
		benchmarkResourceSyncDelivery(b, false)
	})
	b.Run("pubsub", func(b *testing.B) {
		benchmarkResourceSyncDelivery(b, true)
	})
}

func benchmarkResourceSyncDelivery(b *testing.B, pubsubEnabled bool) {
	mockCtrl := gomock.NewController(b)
	mockEntityStore := mocksManager.NewMockEntityStore(mockCtrl)
	mockExternalStore := mocksExternalSrc.NewMockStore(mockCtrl)
	mockDetector := mocksDetector.NewMockDetector(mockCtrl)

	msgSub := internalmessage.NewMessageSubscriber()

	var disp common.PubSubDispatcher
	if pubsubEnabled {
		var err error
		disp, err = pubsubDispatcher.NewDispatcher(pubsubDispatcher.WithLaneConfigs(
			[]pubsub.LaneConfig{
				lane.NewBlockingLane(pubsub.ResourceSyncFinishedLane),
			},
		))
		if err != nil {
			b.Fatal(err)
		}
		defer disp.Stop()
	}

	mgr := NewManager(
		mockEntityStore,
		mockExternalStore,
		mockDetector,
		msgSub,
		disp,
		updatecomputer.New(),
	).(*networkFlowManager)

	event := &events.ResourceSyncFinishedEvent{
		LifecycleEvent: events.LifecycleEvent{Validity: context.Background()},
	}
	legacyMsg := &internalmessage.SensorInternalMessage{
		Kind:     internalmessage.SensorMessageResourceSyncFinished,
		Text:     "bench sync",
		Validity: context.Background(),
	}

	b.ReportAllocs()

	for b.Loop() {
		if pubsubEnabled {
			if err := disp.Publish(event); err != nil {
				b.Fatal(err)
			}
		} else {
			if err := msgSub.Publish(legacyMsg); err != nil {
				b.Fatal(err)
			}
		}
		for !mgr.initialSync.Load() {
			runtime.Gosched()
		}
		mgr.initialSync.Store(false)
	}
}
