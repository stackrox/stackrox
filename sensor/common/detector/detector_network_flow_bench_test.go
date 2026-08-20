package detector

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stretchr/testify/require"
)

// BenchmarkNetworkFlowPipeline_SteadyState measures steady-state end-to-end
// latency per network flow event through the full pipeline (buildFlowDetails →
// enrichFlowOnEntity → queue/publish → detect → output), comparing the legacy
// queue path with the PubSub path. Each flow has two deployment entities, so
// each iteration produces two outputs.
func BenchmarkNetworkFlowPipeline_SteadyState(b *testing.B) {
	for _, pubSubEnabled := range []bool{false, true} {
		b.Run(fmt.Sprintf("pubsub=%t", pubSubEnabled), func(b *testing.B) {
			b.Setenv(features.SensorInternalPubSub.EnvVar(), fmt.Sprintf("%t", pubSubEnabled))

			d := createBenchDetector(b, pubSubEnabled)
			require.NoError(b, d.Start())
			b.Cleanup(d.Stop)

			d.Notify(common.SensorComponentEventCentralReachable)

			flow := &storage.NetworkFlow{
				Props: &storage.NetworkFlowProperties{
					SrcEntity: &storage.NetworkEntityInfo{
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
						Id:   "dep-src",
					},
					DstEntity: &storage.NetworkEntityInfo{
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
						Id:   "dep-dst",
					},
					DstPort:    80,
					L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
				},
			}

			for b.Loop() {
				d.ProcessNetworkFlow(context.Background(), flow)
				// Two outputs: one per deployment entity (src + dst)
				<-d.output
				<-d.output
			}
		})
	}
}
