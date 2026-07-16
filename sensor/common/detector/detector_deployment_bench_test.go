package detector

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stretchr/testify/require"
)

// BenchmarkDeploymentPipeline_SteadyState measures steady-state end-to-end
// latency per deployment event through the full pipeline (processDeploymentNoLock →
// enricher.blockingScan → runDetector → serializeDeployTimeOutput → output),
// comparing the legacy queue path with the PubSub path.
func BenchmarkDeploymentPipeline_SteadyState(b *testing.B) {
	for _, pubSubEnabled := range []bool{false, true} {
		b.Run(fmt.Sprintf("pubsub=%t", pubSubEnabled), func(b *testing.B) {
			b.Setenv(features.SensorInternalPubSub.EnvVar(), fmt.Sprintf("%t", pubSubEnabled))

			d := createBenchDetector(b, pubSubEnabled)
			require.NoError(b, d.Start())
			b.Cleanup(d.Stop)

			d.Notify(common.SensorComponentEventCentralReachable)

			deployment := &storage.Deployment{
				Id:        "dep-1",
				Name:      "bench-deployment",
				Namespace: "default",
			}

			for b.Loop() {
				d.ProcessDeployment(context.Background(), deployment, central.ResourceAction_CREATE_RESOURCE)
				<-d.output
			}
		})
	}
}
