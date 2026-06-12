package detector

import (
	"context"
	"fmt"
	"testing"
	"testing/synctest"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common"
	mockStore "github.com/stackrox/rox/sensor/common/store/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestDeploymentPipeline(t *testing.T) {
	deployment := &storage.Deployment{
		Id:        "dep-1",
		Name:      "test-deployment",
		Namespace: "default",
	}

	tests := map[string]struct {
		setupMocks    func(*mockStore.MockDeploymentStore, *mockStore.MockNetworkPolicyStore)
		setupDetector func(*detectorImpl)
		deployment    *storage.Deployment
		action        central.ResourceAction
		expectOutput  bool
		expectAlerts  bool
	}{
		"create deployment with alerts reaches output": {
			deployment: deployment,
			action:     central.ResourceAction_CREATE_RESOURCE,
			setupMocks: func(_ *mockStore.MockDeploymentStore, nps *mockStore.MockNetworkPolicyStore) {
				nps.EXPECT().Find("default", gomock.Any()).Return(nil)
			},
			setupDetector: func(d *detectorImpl) {
				d.unifiedDetector = &fakeUnifiedDetector{
					alerts: []*storage.Alert{{
						Id:     "alert-1",
						Policy: &storage.Policy{Id: "policy-1"},
					}},
				}
			},
			expectOutput: true,
			expectAlerts: true,
		},
		"create deployment with no alerts still produces output": {
			// The serializer always forwards CREATE results even with empty alerts
			deployment: deployment,
			action:     central.ResourceAction_CREATE_RESOURCE,
			setupMocks: func(_ *mockStore.MockDeploymentStore, nps *mockStore.MockNetworkPolicyStore) {
				nps.EXPECT().Find("default", gomock.Any()).Return(nil)
			},
			expectOutput: true,
			expectAlerts: false,
		},
		"remove deployment produces output with empty alerts": {
			// REMOVE sends an empty AlertResults to mark deploytime alerts as stale
			deployment: deployment,
			action:     central.ResourceAction_REMOVE_RESOURCE,
			setupMocks: func(_ *mockStore.MockDeploymentStore, _ *mockStore.MockNetworkPolicyStore) {},
			setupDetector: func(d *detectorImpl) {
				// Pre-populate the deduper so REMOVE has something to clean up
				d.deduper.addDeployment(deployment)
			},
			expectOutput: true,
			expectAlerts: false,
		},
		"update deployment with changes reaches output": {
			deployment: deployment,
			action:     central.ResourceAction_UPDATE_RESOURCE,
			setupMocks: func(_ *mockStore.MockDeploymentStore, nps *mockStore.MockNetworkPolicyStore) {
				nps.EXPECT().Find("default", gomock.Any()).Return(nil)
			},
			setupDetector: func(d *detectorImpl) {
				d.unifiedDetector = &fakeUnifiedDetector{
					alerts: []*storage.Alert{{
						Id:     "alert-1",
						Policy: &storage.Policy{Id: "policy-1"},
					}},
				}
				// Add the deployment first so the deduper knows about it,
				// then remove it so the update is treated as a change.
				d.deduper.addDeployment(deployment)
				d.deduper.removeDeployment(deployment.GetId())
			},
			expectOutput: true,
			expectAlerts: true,
		},
		"update deployment with no changes is deduplicated": {
			deployment: deployment,
			action:     central.ResourceAction_UPDATE_RESOURCE,
			setupMocks: func(_ *mockStore.MockDeploymentStore, _ *mockStore.MockNetworkPolicyStore) {},
			setupDetector: func(d *detectorImpl) {
				d.unifiedDetector = &fakeUnifiedDetector{
					alerts: []*storage.Alert{{Id: "alert-1", Policy: &storage.Policy{Id: "p1"}}},
				}
				// Add the deployment so the deduper treats the update as a no-op
				d.deduper.addDeployment(deployment)
			},
			expectOutput: false,
		},
	}

	for _, pubSubEnabled := range []bool{false, true} {
		for name, tc := range tests {
			t.Run(fmt.Sprintf("%s/pubsub=%t", name, pubSubEnabled), func(t *testing.T) {
				t.Setenv(features.SensorInternalPubSub.EnvVar(), fmt.Sprintf("%t", pubSubEnabled))

				synctest.Test(t, func(t *testing.T) {
					d, ds, nps, _ := createTestDetector(t, pubSubEnabled)
					ds.EXPECT().GetSnapshot(gomock.Any()).Return(nil).AnyTimes()
					tc.setupMocks(ds, nps)
					if tc.setupDetector != nil {
						tc.setupDetector(d)
					}

					require.NoError(t, d.Start())
					defer d.Stop()

					d.Notify(common.SensorComponentEventCentralReachable)

					d.ProcessDeployment(context.Background(), tc.deployment, tc.action)
					synctest.Wait()

					if tc.expectOutput {
						select {
						case msg := <-d.output:
							require.NotNil(t, msg)
							alertResults := msg.GetEvent().GetAlertResults()
							assert.Equal(t, tc.deployment.GetId(), alertResults.GetDeploymentId())
							if tc.expectAlerts {
								assert.NotEmpty(t, alertResults.GetAlerts())
							} else {
								assert.Empty(t, alertResults.GetAlerts())
							}
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

func TestDeploymentPipelineDropsWhenFull(t *testing.T) {
	bufferSize := 5
	totalEvents := 10000

	// Initialize the rate-limited logger before entering synctest bubbles.
	// It uses a hashicorp LRU that spawns a background timer goroutine;
	// if created inside the bubble, synctest panics on exit because the
	// goroutine remains blocked.
	logging.GetRateLimitedLogger()

	for _, pubSubEnabled := range []bool{false, true} {
		t.Run(fmt.Sprintf("pubsub=%t", pubSubEnabled), func(t *testing.T) {
			t.Setenv(features.SensorInternalPubSub.EnvVar(), fmt.Sprintf("%t", pubSubEnabled))

			synctest.Test(t, func(t *testing.T) {
				d, _, nps, _ := createTestDetectorWithBufferSize(t, pubSubEnabled, bufferSize)
				d.unifiedDetector = &fakeUnifiedDetector{
					alerts: []*storage.Alert{{Id: "alert-1", Policy: &storage.Policy{Id: "p1"}}},
				}
				nps.EXPECT().Find("default", gomock.Any()).Return(nil).AnyTimes()

				require.NoError(t, d.Start())
				defer d.Stop()

				d.Notify(common.SensorComponentEventCentralReachable)

				// Hold the deploymentDetectionLock so the consumer blocks,
				// causing events to queue up and eventually drop.
				d.deploymentDetectionLock.Lock()

				// ProcessDeployment is synchronous up to the drop point:
				// legacy Push checks size and drops inline, PubSub Publish
				// dispatches to the lane which calls TryWrite. After the
				// loop and unlock, synctest.Wait ensures all drops and
				// processing have completed without races.
				for i := range totalEvents {
					d.ProcessDeployment(context.Background(), &storage.Deployment{
						Id:        fmt.Sprintf("dep-%d", i),
						Name:      "test",
						Namespace: "default",
					}, central.ResourceAction_CREATE_RESOURCE)
				}

				// Release the lock so buffered events can be processed
				d.deploymentDetectionLock.Unlock()
				synctest.Wait()

				// The exact number of events processed depends on goroutine
				// scheduling — the consumer may pull 0-2 extra events from
				// the buffer before blocking on the lock. We assert that
				// drops occurred (received < totalEvents) rather than an
				// exact count.
				var received int
				for {
					select {
					case <-d.output:
						received++
					default:
						assert.Greater(t, received, 0, "should have received some events")
						assert.Less(t, received, totalEvents, "should have dropped some events (received %d out of %d)", received, totalEvents)
						return
					}
				}
			})
		})
	}
}
