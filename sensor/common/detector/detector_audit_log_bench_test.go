package detector

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	detectorEvents "github.com/stackrox/rox/sensor/common/detector/events"
	"github.com/stretchr/testify/require"
)

// BenchmarkAuditLogPipeline_SteadyState measures steady-state end-to-end
// latency per audit log event through the full pipeline (detect → output),
// comparing the legacy channel path with the PubSub path.
func BenchmarkAuditLogPipeline_SteadyState(b *testing.B) {
	for _, pubSubEnabled := range []bool{false, true} {
		b.Run(fmt.Sprintf("pubsub=%t", pubSubEnabled), func(b *testing.B) {
			b.Setenv(features.SensorInternalPubSub.EnvVar(), fmt.Sprintf("%t", pubSubEnabled))

			d := createBenchDetector(b, pubSubEnabled)
			require.NoError(b, d.Start())
			b.Cleanup(d.Stop)

			events := &sensor.AuditEvents{
				Events: []*storage.KubernetesEvent{
					{
						Id: "event-1",
						Object: &storage.KubernetesEvent_Object{
							Name:      "test-secret",
							Resource:  storage.KubernetesEvent_Object_SECRETS,
							ClusterId: "cluster-1",
							Namespace: "default",
						},
						ApiVerb: storage.KubernetesEvent_CREATE,
					},
				},
			}

			for b.Loop() {
				if pubSubEnabled {
					_ = d.pubSubDispatcher.Publish(&detectorEvents.AuditLogEvent{AuditEvents: events})
				} else {
					d.auditEventsChan <- events
				}
				<-d.output
			}
		})
	}
}
