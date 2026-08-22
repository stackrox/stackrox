package centralbound

// Benchmark for Central-bound bridge delivery through a real PubSub dispatcher.
//
//   go test -bench=Benchmark -benchmem -count=10 -run='^$' \
//     ./sensor/common/centralbound/ > bench_bridge.txt

import (
	"runtime"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubDispatcher "github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
)

func BenchmarkBridgeDelivery(b *testing.B) {
	disp, err := pubsubDispatcher.NewDispatcher(pubsubDispatcher.WithLaneConfigs(
		[]pubsub.LaneConfig{
			lane.NewBlockingLane(pubsub.CentralBoundLane),
		},
	))
	if err != nil {
		b.Fatal(err)
	}
	defer disp.Stop()

	bridge, err := NewBridge(disp)
	if err != nil {
		b.Fatal(err)
	}

	evt := &CentralBoundEvent{Msg: message.New(&central.MsgFromSensor{})}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		if err := disp.Publish(evt); err != nil {
			b.Fatal(err)
		}
		for len(bridge.responsesC) == 0 {
			runtime.Gosched()
		}
		<-bridge.responsesC
	}
}
