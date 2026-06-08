package detector

import (
	"fmt"
	"testing"
	"testing/synctest"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	detectorEvents "github.com/stackrox/rox/sensor/common/detector/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestAuditEvents() *sensor.AuditEvents {
	return &sensor.AuditEvents{
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
}

// sendAuditEvent sends an audit event through the appropriate path:
// in PubSub mode, publishes directly to the dispatcher (like the compliance
// service would); in legacy mode, writes to the auditEventsChan.
func sendAuditEvent(d *detectorImpl, pubSubEnabled bool, events *sensor.AuditEvents) {
	if pubSubEnabled {
		_ = d.pubSubDispatcher.Publish(&detectorEvents.AuditLogEvent{AuditEvents: events})
	} else {
		d.auditEventsChan <- events
	}
}

func TestAuditLogPipeline(t *testing.T) {
	tests := map[string]struct {
		setupDetector func(*detectorImpl)
		expectOutput  bool
	}{
		"audit event with alerts reaches output": {
			setupDetector: func(d *detectorImpl) {
				d.unifiedDetector = &fakeUnifiedDetector{
					alerts: []*storage.Alert{{
						Id:     "alert-1",
						Policy: &storage.Policy{Id: "policy-1"},
					}},
				}
			},
			expectOutput: true,
		},
		"audit event with no alerts produces no output": {
			expectOutput: false,
		},
	}

	for _, pubSubEnabled := range []bool{false, true} {
		for name, tc := range tests {
			t.Run(fmt.Sprintf("%s/pubsub=%t", name, pubSubEnabled), func(t *testing.T) {
				t.Setenv(features.SensorInternalPubSub.EnvVar(), fmt.Sprintf("%t", pubSubEnabled))

				synctest.Test(t, func(t *testing.T) {
					d, _, _, _ := createTestDetector(t, pubSubEnabled)
					if tc.setupDetector != nil {
						tc.setupDetector(d)
					}

					require.NoError(t, d.Start())
					defer d.Stop()

					sendAuditEvent(d, pubSubEnabled, newTestAuditEvents())
					synctest.Wait()

					if tc.expectOutput {
						select {
						case msg := <-d.output:
							require.NotNil(t, msg)
							assert.NotEmpty(t, msg.GetEvent().GetAlertResults().GetAlerts())
						default:
							t.Fatal("expected output but none available")
						}
					} else {
						select {
						case msg := <-d.output:
							t.Fatalf("expected no output but got: %v", msg)
						default:
						}
					}
				})
			})
		}
	}
}
