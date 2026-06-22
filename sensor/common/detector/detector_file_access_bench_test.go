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

// BenchmarkFileAccessPipeline_SteadyState measures steady-state end-to-end
// latency per file access event through the full pipeline (enrich →
// queue/publish → detect → output), comparing the legacy queue path with
// the PubSub path.
func BenchmarkFileAccessPipeline_SteadyState(b *testing.B) {
	for _, pubSubEnabled := range []bool{false, true} {
		b.Run(fmt.Sprintf("pubsub=%t", pubSubEnabled), func(b *testing.B) {
			b.Setenv(features.SensorInternalPubSub.EnvVar(), fmt.Sprintf("%t", pubSubEnabled))

			d := createBenchDetector(b, pubSubEnabled)
			require.NoError(b, d.Start())
			b.Cleanup(d.Stop)

			d.Notify(common.SensorComponentEventCentralReachable)

			access := &storage.FileAccess{
				Process:   &storage.ProcessIndicator{DeploymentId: "dep-1"},
				File:      &storage.FileAccess_File{EffectivePath: "/etc/passwd"},
				Operation: storage.FileAccess_OPEN,
			}

			for b.Loop() {
				d.ProcessFileAccess(context.Background(), access)
				<-d.output
			}
		})
	}
}
