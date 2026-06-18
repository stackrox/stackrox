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

// BenchmarkIndicatorPipeline_SteadyState measures steady-state end-to-end
// latency per indicator event through the full pipeline (enrich →
// queue/publish → detect → output), comparing the legacy queue path with
// the PubSub path.
func BenchmarkIndicatorPipeline_SteadyState(b *testing.B) {
	for _, pubSubEnabled := range []bool{false, true} {
		b.Run(fmt.Sprintf("pubsub=%t", pubSubEnabled), func(b *testing.B) {
			b.Setenv(features.SensorInternalPubSub.EnvVar(), fmt.Sprintf("%t", pubSubEnabled))

			d := createBenchDetector(b, pubSubEnabled)
			require.NoError(b, d.Start())
			b.Cleanup(d.Stop)

			d.Notify(common.SensorComponentEventCentralReachable)

			pi := &storage.ProcessIndicator{
				Id:           "pi-bench",
				DeploymentId: "dep-1",
				Signal:       &storage.ProcessSignal{ExecFilePath: "/bin/test"},
			}

			for b.Loop() {
				d.ProcessIndicator(context.Background(), pi)
				<-d.output
			}
		})
	}
}
