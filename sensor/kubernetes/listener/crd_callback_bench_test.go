package listener

// Benchmarks for the crdWatcherCallbackWrapper publish path.

import (
	"context"
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/stackrox/rox/sensor/common/internalmessage"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubDispatcher "github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
	"github.com/stackrox/rox/sensor/kubernetes/listener/watcher"
)

func BenchmarkCrdCallbackPublish(b *testing.B) {
	b.Run("legacy", func(b *testing.B) {
		benchmarkCrdCallbackPublish(b, false)
	})
	b.Run("pubsub", func(b *testing.B) {
		benchmarkCrdCallbackPublish(b, true)
	})
}

func benchmarkCrdCallbackPublish(b *testing.B, pubsubEnabled bool) {
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
			pubsub.SensorSoftRestartConsumer,
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
