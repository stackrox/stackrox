package listener

// Benchmarks for the crdWatcherCallbackWrapper publish path.
//
// To compare old vs new, run with each flag state and feed the results to benchstat:
//
//   ROX_SENSOR_PUBSUB=false go test -bench=Benchmark -benchmem -count=10 -run='^$' \
//     ./sensor/kubernetes/listener/ > bench_legacy.txt
//   ROX_SENSOR_PUBSUB=true go test -bench=Benchmark -benchmem -count=10 -run='^$' \
//     ./sensor/kubernetes/listener/ > bench_pubsub.txt
//   benchstat bench_legacy.txt bench_pubsub.txt

import (
	"context"
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/common/internalmessage"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubDispatcher "github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
	"github.com/stackrox/rox/sensor/kubernetes/listener/watcher"
)

func BenchmarkCrdCallbackPublish(b *testing.B) {
	pubsubEnabled := features.SensorInternalPubSub.Enabled()
	msgSub := internalmessage.NewMessageSubscriber()

	var callbackFired atomic.Bool

	var publisher pubSubPublisher
	if pubsubEnabled {
		disp, err := pubsubDispatcher.NewDispatcher(pubsubDispatcher.WithLaneConfigs(
			[]pubsub.LaneConfig{
				lane.NewBlockingLane(pubsub.SoftRestartLane),
			},
		))
		if err != nil {
			b.Fatal(err)
		}
		defer disp.Stop()
		if err := disp.RegisterConsumerToLane(
			pubsub.CoreSensorConsumer,
			pubsub.SoftRestartTopic,
			pubsub.SoftRestartLane,
			func(_ pubsub.Event) error {
				callbackFired.Store(true)
				return nil
			},
		); err != nil {
			b.Fatal(err)
		}
		publisher = disp
	} else {
		if err := msgSub.Subscribe(internalmessage.SensorMessageSoftRestart, func(_ *internalmessage.SensorInternalMessage) {
			callbackFired.Store(true)
		}); err != nil {
			b.Fatal(err)
		}
		publisher = &noopPublisher{}
	}

	cb := crdWatcherCallbackWrapper(
		context.Background(),
		allResourcesAvailable(),
		msgSub,
		publisher,
		"bench restart",
	)
	status := &watcher.Status{Available: true}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		cb(status)
		for !callbackFired.Load() {
			runtime.Gosched()
		}
		callbackFired.Store(false)
	}
}

type noopPublisher struct{}

func (n *noopPublisher) Publish(_ pubsub.Event) error { return nil }
