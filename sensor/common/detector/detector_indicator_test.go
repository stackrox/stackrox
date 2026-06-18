package detector

import (
	"context"
	"fmt"
	"testing"
	"testing/synctest"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common"
	mockStore "github.com/stackrox/rox/sensor/common/store/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestIndicatorPipeline(t *testing.T) {
	deployment := &storage.Deployment{
		Id:        "dep-1",
		Name:      "test-deployment",
		Namespace: "default",
	}

	tests := map[string]struct {
		setupMocks    func(*mockStore.MockDeploymentStore, *mockStore.MockNetworkPolicyStore)
		setupDetector func(*detectorImpl)
		indicator     *storage.ProcessIndicator
		expectOutput  bool
	}{
		"indicator with alerts reaches output": {
			indicator: &storage.ProcessIndicator{
				Id:           "pi-1",
				DeploymentId: "dep-1",
				Signal:       &storage.ProcessSignal{ExecFilePath: "/bin/bash"},
			},
			setupMocks: func(ds *mockStore.MockDeploymentStore, nps *mockStore.MockNetworkPolicyStore) {
				ds.EXPECT().GetSnapshot("dep-1").Return(deployment)
				nps.EXPECT().Find("default", gomock.Any()).Return(nil)
			},
			setupDetector: func(d *detectorImpl) {
				d.unifiedDetector = &fakeUnifiedDetector{
					alerts: []*storage.Alert{{
						Id:     "alert-1",
						Policy: &storage.Policy{Id: "policy-1", Name: "test-policy"},
					}},
				}
			},
			expectOutput: true,
		},
		"indicator with no alerts produces no output": {
			indicator: &storage.ProcessIndicator{
				Id:           "pi-2",
				DeploymentId: "dep-1",
				Signal:       &storage.ProcessSignal{ExecFilePath: "/bin/ls"},
			},
			setupMocks: func(ds *mockStore.MockDeploymentStore, nps *mockStore.MockNetworkPolicyStore) {
				ds.EXPECT().GetSnapshot("dep-1").Return(deployment)
				nps.EXPECT().Find("default", gomock.Any()).Return(nil)
			},
			expectOutput: false,
		},
		"indicator for missing deployment produces no output": {
			indicator: &storage.ProcessIndicator{
				Id:           "pi-3",
				DeploymentId: "dep-missing",
			},
			setupMocks: func(ds *mockStore.MockDeploymentStore, _ *mockStore.MockNetworkPolicyStore) {
				ds.EXPECT().GetSnapshot("dep-missing").Return(nil)
			},
			expectOutput: false,
		},
	}

	for _, pubSubEnabled := range []bool{false, true} {
		for name, tc := range tests {
			t.Run(fmt.Sprintf("%s/pubsub=%t", name, pubSubEnabled), func(t *testing.T) {
				t.Setenv(features.SensorInternalPubSub.EnvVar(), fmt.Sprintf("%t", pubSubEnabled))

				synctest.Test(t, func(t *testing.T) {
					d, ds, nps := createTestDetector(t, pubSubEnabled)
					tc.setupMocks(ds, nps)
					if tc.setupDetector != nil {
						tc.setupDetector(d)
					}

					require.NoError(t, d.Start())
					defer d.Stop()

					d.Notify(common.SensorComponentEventCentralReachable)

					d.ProcessIndicator(context.Background(), tc.indicator)
					synctest.Wait()

					if tc.expectOutput {
						select {
						case msg := <-d.output:
							require.NotNil(t, msg)
							alertResults := msg.GetEvent().GetAlertResults()
							assert.Equal(t, tc.indicator.GetDeploymentId(), alertResults.GetDeploymentId())
							assert.NotEmpty(t, alertResults.GetAlerts())
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

func TestIndicatorPipelineOfflineBlocks(t *testing.T) {
	deployment := &storage.Deployment{
		Id:        "dep-1",
		Name:      "test-deployment",
		Namespace: "default",
	}

	for _, pubSubEnabled := range []bool{false, true} {
		t.Run(fmt.Sprintf("pubsub=%t", pubSubEnabled), func(t *testing.T) {
			t.Setenv(features.SensorInternalPubSub.EnvVar(), fmt.Sprintf("%t", pubSubEnabled))

			synctest.Test(t, func(t *testing.T) {
				d, ds, nps := createTestDetector(t, pubSubEnabled)
				d.unifiedDetector = &fakeUnifiedDetector{
					alerts: []*storage.Alert{{Id: "alert-1", Policy: &storage.Policy{Id: "p1"}}},
				}
				ds.EXPECT().GetSnapshot("dep-1").Return(deployment).AnyTimes()
				nps.EXPECT().Find("default", gomock.Any()).Return(nil).AnyTimes()

				require.NoError(t, d.Start())
				defer d.Stop()

				// Pipeline starts offline — send an indicator
				d.ProcessIndicator(context.Background(), &storage.ProcessIndicator{
					Id: "pi-offline", DeploymentId: "dep-1",
					Signal: &storage.ProcessSignal{ExecFilePath: "/bin/test"},
				})
				synctest.Wait()

				// No output while offline
				select {
				case <-d.output:
					t.Fatal("expected no output while offline")
				default:
				}

				// Go online — buffered event should be processed
				d.Notify(common.SensorComponentEventCentralReachable)
				synctest.Wait()

				select {
				case msg := <-d.output:
					require.NotNil(t, msg)
					assert.NotEmpty(t, msg.GetEvent().GetAlertResults().GetAlerts())
				default:
					t.Fatal("expected output after going online")
				}
			})
		})
	}
}

func TestIndicatorPipelineDropsWhenFull(t *testing.T) {
	deployment := &storage.Deployment{
		Id:        "dep-1",
		Name:      "test-deployment",
		Namespace: "default",
	}
	bufferSize := 5
	totalEvents := bufferSize + 20

	// Initialize the rate-limited logger before entering synctest bubbles.
	// The hashicorp LRU it uses spawns a timer goroutine that would cause
	// synctest to panic if created inside the bubble.
	logging.GetRateLimitedLogger()

	for _, pubSubEnabled := range []bool{false, true} {
		t.Run(fmt.Sprintf("pubsub=%t", pubSubEnabled), func(t *testing.T) {
			t.Setenv(features.SensorInternalPubSub.EnvVar(), fmt.Sprintf("%t", pubSubEnabled))

			synctest.Test(t, func(t *testing.T) {
				d, ds, nps := createTestDetectorWithBufferSize(t, pubSubEnabled, bufferSize)
				d.unifiedDetector = &fakeUnifiedDetector{
					alerts: []*storage.Alert{{Id: "alert-1", Policy: &storage.Policy{Id: "p1"}}},
				}
				ds.EXPECT().GetSnapshot("dep-1").Return(deployment).AnyTimes()
				nps.EXPECT().Find("default", gomock.Any()).Return(nil).AnyTimes()

				require.NoError(t, d.Start())
				defer d.Stop()

				// Pipeline starts offline — send more events than the buffer can hold
				for i := range totalEvents {
					d.ProcessIndicator(context.Background(), &storage.ProcessIndicator{
						Id: fmt.Sprintf("pi-%d", i), DeploymentId: "dep-1",
						Signal: &storage.ProcessSignal{ExecFilePath: "/bin/test"},
					})
				}
				synctest.Wait()

				// No output while offline
				select {
				case <-d.output:
					t.Fatal("expected no output while offline")
				default:
				}

				// Go online
				d.Notify(common.SensorComponentEventCentralReachable)
				synctest.Wait()

				// Drain and count.
				// The legacy queue holds exactly bufferSize items.
				// The PubSub BufferedConsumer holds bufferSize + 1: its run()
				// goroutine pulls one event from the buffer to process (blocking
				// on the paused callback), freeing a slot for one extra TryWrite.
				expectedEvents := bufferSize
				if pubSubEnabled {
					expectedEvents = bufferSize + 1
				}

				var received int
				for {
					select {
					case <-d.output:
						received++
					default:
						assert.Equal(t, expectedEvents, received, "expected %d events, got %d out of %d sent", expectedEvents, received, totalEvents)
						return
					}
				}
			})
		})
	}
}
