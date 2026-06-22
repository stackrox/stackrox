package manager

// Benchmarks for ResourceSyncFinished delivery through both the pubsub and legacy
// internalmessage paths.
//
// To compare old vs new, run with each flag state and feed the results to benchstat:
//
//   ROX_SENSOR_PUBSUB=false go test -bench=Benchmark -benchmem -count=10 -run='^$' \
//     ./sensor/common/networkflow/manager/ > bench_legacy.txt
//   ROX_SENSOR_PUBSUB=true go test -bench=Benchmark -benchmem -count=10 -run='^$' \
//     ./sensor/common/networkflow/manager/ > bench_pubsub.txt
//   benchstat bench_legacy.txt bench_pubsub.txt

import (
	"context"
	"runtime"
	"testing"

	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/common"
	mocksDetector "github.com/stackrox/rox/sensor/common/detector/mocks"
	mocksExternalSrc "github.com/stackrox/rox/sensor/common/externalsrcs/mocks"
	"github.com/stackrox/rox/sensor/common/internalmessage"
	mocksManager "github.com/stackrox/rox/sensor/common/networkflow/manager/mocks"
	"github.com/stackrox/rox/sensor/common/networkflow/updatecomputer"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubDispatcher "github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
	"go.uber.org/mock/gomock"
)

type benchSyncEvent struct {
	validity context.Context
}

func (e *benchSyncEvent) Topic() pubsub.Topic { return pubsub.ResourceSyncFinishedTopic }
func (e *benchSyncEvent) Lane() pubsub.LaneID { return pubsub.ResourceSyncFinishedLane }
func (e *benchSyncEvent) IsExpired() bool     { return false }

func BenchmarkResourceSyncDelivery(b *testing.B) {
	mockCtrl := gomock.NewController(b)
	mockEntityStore := mocksManager.NewMockEntityStore(mockCtrl)
	mockExternalStore := mocksExternalSrc.NewMockStore(mockCtrl)
	mockDetector := mocksDetector.NewMockDetector(mockCtrl)

	pubsubEnabled := features.SensorInternalPubSub.Enabled()
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

	event := &benchSyncEvent{validity: context.Background()}
	legacyMsg := &internalmessage.SensorInternalMessage{
		Kind:     internalmessage.SensorMessageResourceSyncFinished,
		Text:     "bench sync",
		Validity: context.Background(),
	}

	b.ReportAllocs()
	b.ResetTimer()

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
