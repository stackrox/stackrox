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

func TestFileAccessPipeline(t *testing.T) {
	deployment := &storage.Deployment{
		Id:        "dep-1",
		Name:      "test-deployment",
		Namespace: "default",
	}

	tests := map[string]struct {
		setupMocks    func(*mockStore.MockDeploymentStore, *mockStore.MockNetworkPolicyStore, *mockStore.MockNodeStore)
		setupDetector func(*detectorImpl)
		access        *storage.FileAccess
		expectOutput  bool
	}{
		"deployment file access with alerts reaches output": {
			access: &storage.FileAccess{
				Process:   &storage.ProcessIndicator{DeploymentId: "dep-1"},
				File:      &storage.FileAccess_File{EffectivePath: "/etc/passwd"},
				Operation: storage.FileAccess_OPEN,
			},
			setupMocks: func(ds *mockStore.MockDeploymentStore, nps *mockStore.MockNetworkPolicyStore, _ *mockStore.MockNodeStore) {
				ds.EXPECT().GetSnapshot("dep-1").Return(deployment)
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
		},
		"deployment file access with no alerts produces no output": {
			access: &storage.FileAccess{
				Process:   &storage.ProcessIndicator{DeploymentId: "dep-1"},
				File:      &storage.FileAccess_File{EffectivePath: "/tmp/safe"},
				Operation: storage.FileAccess_OPEN,
			},
			setupMocks: func(ds *mockStore.MockDeploymentStore, nps *mockStore.MockNetworkPolicyStore, _ *mockStore.MockNodeStore) {
				ds.EXPECT().GetSnapshot("dep-1").Return(deployment)
				nps.EXPECT().Find("default", gomock.Any()).Return(nil)
			},
			expectOutput: false,
		},
		"deployment file access for missing deployment produces no output": {
			access: &storage.FileAccess{
				Process:   &storage.ProcessIndicator{DeploymentId: "dep-missing"},
				File:      &storage.FileAccess_File{EffectivePath: "/etc/passwd"},
				Operation: storage.FileAccess_OPEN,
			},
			setupMocks: func(ds *mockStore.MockDeploymentStore, _ *mockStore.MockNetworkPolicyStore, _ *mockStore.MockNodeStore) {
				ds.EXPECT().GetSnapshot("dep-missing").Return(nil)
			},
			expectOutput: false,
		},
		"node file access with alerts reaches output": {
			access: &storage.FileAccess{
				Hostname:  "node-1",
				Process:   &storage.ProcessIndicator{},
				File:      &storage.FileAccess_File{EffectivePath: "/etc/shadow"},
				Operation: storage.FileAccess_OPEN,
			},
			setupMocks: func(_ *mockStore.MockDeploymentStore, _ *mockStore.MockNetworkPolicyStore, ns *mockStore.MockNodeStore) {
				ns.EXPECT().GetNodeByHostname("node-1").Return(&storage.Node{Id: "node-1", Name: "node-1"})
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
		},
		"node file access for missing node produces no output": {
			access: &storage.FileAccess{
				Hostname:  "node-missing",
				Process:   &storage.ProcessIndicator{},
				File:      &storage.FileAccess_File{EffectivePath: "/etc/shadow"},
				Operation: storage.FileAccess_OPEN,
			},
			setupMocks: func(_ *mockStore.MockDeploymentStore, _ *mockStore.MockNetworkPolicyStore, ns *mockStore.MockNodeStore) {
				ns.EXPECT().GetNodeByHostname("node-missing").Return(nil)
			},
			expectOutput: false,
		},
	}

	for _, pubSubEnabled := range []bool{false, true} {
		for name, tc := range tests {
			t.Run(fmt.Sprintf("%s/pubsub=%t", name, pubSubEnabled), func(t *testing.T) {
				t.Setenv(features.SensorInternalPubSub.EnvVar(), fmt.Sprintf("%t", pubSubEnabled))

				synctest.Test(t, func(t *testing.T) {
					d, ds, nps, ns := createTestDetector(t, pubSubEnabled)
					tc.setupMocks(ds, nps, ns)
					if tc.setupDetector != nil {
						tc.setupDetector(d)
					}

					require.NoError(t, d.Start())
					defer d.Stop()

					d.Notify(common.SensorComponentEventCentralReachable)

					d.ProcessFileAccess(context.Background(), tc.access)
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

func TestFileAccessPipelineOfflineBlocks(t *testing.T) {
	deployment := &storage.Deployment{
		Id:        "dep-1",
		Name:      "test-deployment",
		Namespace: "default",
	}

	for _, pubSubEnabled := range []bool{false, true} {
		t.Run(fmt.Sprintf("pubsub=%t", pubSubEnabled), func(t *testing.T) {
			t.Setenv(features.SensorInternalPubSub.EnvVar(), fmt.Sprintf("%t", pubSubEnabled))

			synctest.Test(t, func(t *testing.T) {
				d, ds, nps, _ := createTestDetector(t, pubSubEnabled)
				d.unifiedDetector = &fakeUnifiedDetector{
					alerts: []*storage.Alert{{Id: "alert-1", Policy: &storage.Policy{Id: "p1"}}},
				}
				ds.EXPECT().GetSnapshot("dep-1").Return(deployment).AnyTimes()
				nps.EXPECT().Find("default", gomock.Any()).Return(nil).AnyTimes()

				require.NoError(t, d.Start())
				defer d.Stop()

				// Pipeline starts offline
				d.ProcessFileAccess(context.Background(), &storage.FileAccess{
					Process:   &storage.ProcessIndicator{DeploymentId: "dep-1"},
					File:      &storage.FileAccess_File{EffectivePath: "/etc/passwd"},
					Operation: storage.FileAccess_OPEN,
				})
				synctest.Wait()

				select {
				case <-d.output:
					t.Fatal("expected no output while offline")
				default:
				}

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

func TestFileAccessPipelineDropsWhenFull(t *testing.T) {
	deployment := &storage.Deployment{
		Id:        "dep-1",
		Name:      "test-deployment",
		Namespace: "default",
	}
	bufferSize := 5
	totalEvents := bufferSize + 20

	// Initialize the rate-limited logger before entering synctest bubbles.
	// It uses a hashicorp LRU that spawns a background timer goroutine;
	// if created inside the bubble, synctest panics on exit because the
	// goroutine remains blocked.
	logging.GetRateLimitedLogger()

	for _, pubSubEnabled := range []bool{false, true} {
		t.Run(fmt.Sprintf("pubsub=%t", pubSubEnabled), func(t *testing.T) {
			t.Setenv(features.SensorInternalPubSub.EnvVar(), fmt.Sprintf("%t", pubSubEnabled))

			synctest.Test(t, func(t *testing.T) {
				d, ds, nps, _ := createTestDetectorWithBufferSize(t, pubSubEnabled, bufferSize)
				d.unifiedDetector = &fakeUnifiedDetector{
					alerts: []*storage.Alert{{Id: "alert-1", Policy: &storage.Policy{Id: "p1"}}},
				}
				ds.EXPECT().GetSnapshot("dep-1").Return(deployment).AnyTimes()
				nps.EXPECT().Find("default", gomock.Any()).Return(nil).AnyTimes()

				require.NoError(t, d.Start())
				defer d.Stop()

				for i := range totalEvents {
					d.ProcessFileAccess(context.Background(), &storage.FileAccess{
						Process:   &storage.ProcessIndicator{Id: fmt.Sprintf("pi-%d", i), DeploymentId: "dep-1"},
						File:      &storage.FileAccess_File{EffectivePath: "/etc/passwd"},
						Operation: storage.FileAccess_OPEN,
					})
				}
				synctest.Wait()

				select {
				case <-d.output:
					t.Fatal("expected no output while offline")
				default:
				}

				d.Notify(common.SensorComponentEventCentralReachable)
				synctest.Wait()

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
