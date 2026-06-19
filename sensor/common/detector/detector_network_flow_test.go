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

var (
	srcDeployment = &storage.Deployment{
		Id:        "dep-src",
		Name:      "src-deployment",
		Namespace: "default",
	}
	dstDeployment = &storage.Deployment{
		Id:        "dep-dst",
		Name:      "dst-deployment",
		Namespace: "default",
	}
)

func newTestFlow(srcID, dstID string) *storage.NetworkFlow {
	return &storage.NetworkFlow{
		Props: &storage.NetworkFlowProperties{
			SrcEntity: &storage.NetworkEntityInfo{
				Type: storage.NetworkEntityInfo_DEPLOYMENT,
				Id:   srcID,
			},
			DstEntity: &storage.NetworkEntityInfo{
				Type: storage.NetworkEntityInfo_DEPLOYMENT,
				Id:   dstID,
			},
			DstPort:    80,
			L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		},
	}
}

func TestNetworkFlowPipeline(t *testing.T) {
	tests := map[string]struct {
		setupMocks    func(*mockStore.MockDeploymentStore, *mockStore.MockNetworkPolicyStore)
		setupDetector func(*detectorImpl)
		flow          *storage.NetworkFlow
		expectOutputs int
	}{
		"flow with alerts reaches output for both entities": {
			flow: newTestFlow("dep-src", "dep-dst"),
			setupMocks: func(ds *mockStore.MockDeploymentStore, nps *mockStore.MockNetworkPolicyStore) {
				ds.EXPECT().GetSnapshot("dep-src").Return(srcDeployment).AnyTimes()
				ds.EXPECT().GetSnapshot("dep-dst").Return(dstDeployment).AnyTimes()
				nps.EXPECT().Find("default", gomock.Any()).Return(nil).AnyTimes()
			},
			setupDetector: func(d *detectorImpl) {
				d.unifiedDetector = &fakeUnifiedDetector{
					alerts: []*storage.Alert{{
						Id:     "alert-1",
						Policy: &storage.Policy{Id: "policy-1"},
					}},
				}
			},
			expectOutputs: 2,
		},
		"flow with no alerts produces no output": {
			flow: newTestFlow("dep-src", "dep-dst"),
			setupMocks: func(ds *mockStore.MockDeploymentStore, nps *mockStore.MockNetworkPolicyStore) {
				ds.EXPECT().GetSnapshot("dep-src").Return(srcDeployment).AnyTimes()
				ds.EXPECT().GetSnapshot("dep-dst").Return(dstDeployment).AnyTimes()
				nps.EXPECT().Find("default", gomock.Any()).Return(nil).AnyTimes()
			},
			expectOutputs: 0,
		},
		"flow with missing src deployment produces no output": {
			// buildFlowDetails fails when src entity details cannot be resolved,
			// dropping the entire flow before per-entity enrichment.
			flow: newTestFlow("dep-missing", "dep-dst"),
			setupMocks: func(ds *mockStore.MockDeploymentStore, nps *mockStore.MockNetworkPolicyStore) {
				ds.EXPECT().GetSnapshot("dep-missing").Return(nil).AnyTimes()
				ds.EXPECT().GetSnapshot("dep-dst").Return(dstDeployment).AnyTimes()
				nps.EXPECT().Find("default", gomock.Any()).Return(nil).AnyTimes()
			},
			setupDetector: func(d *detectorImpl) {
				d.unifiedDetector = &fakeUnifiedDetector{
					alerts: []*storage.Alert{{Id: "alert-1", Policy: &storage.Policy{Id: "p1"}}},
				}
			},
			expectOutputs: 0,
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

					d.ProcessNetworkFlow(context.Background(), tc.flow)
					synctest.Wait()

					if tc.expectOutputs > 0 {
						for i := range tc.expectOutputs {
							select {
							case msg := <-d.output:
								require.NotNil(t, msg, "expected output %d", i)
								assert.NotEmpty(t, msg.GetEvent().GetAlertResults().GetAlerts())
							default:
								t.Fatalf("expected %d outputs but only got %d", tc.expectOutputs, i)
							}
						}
					}

					select {
					case msg := <-d.output:
						t.Fatalf("unexpected extra output: %v", msg)
					default:
					}
				})
			})
		}
	}
}

func TestNetworkFlowPipelineOfflineBlocks(t *testing.T) {
	for _, pubSubEnabled := range []bool{false, true} {
		t.Run(fmt.Sprintf("pubsub=%t", pubSubEnabled), func(t *testing.T) {
			t.Setenv(features.SensorInternalPubSub.EnvVar(), fmt.Sprintf("%t", pubSubEnabled))

			synctest.Test(t, func(t *testing.T) {
				d, ds, nps := createTestDetector(t, pubSubEnabled)
				d.unifiedDetector = &fakeUnifiedDetector{
					alerts: []*storage.Alert{{Id: "alert-1", Policy: &storage.Policy{Id: "p1"}}},
				}
				ds.EXPECT().GetSnapshot("dep-src").Return(srcDeployment).AnyTimes()
				ds.EXPECT().GetSnapshot("dep-dst").Return(dstDeployment).AnyTimes()
				nps.EXPECT().Find("default", gomock.Any()).Return(nil).AnyTimes()

				require.NoError(t, d.Start())
				defer d.Stop()

				// Pipeline starts offline
				d.ProcessNetworkFlow(context.Background(), newTestFlow("dep-src", "dep-dst"))
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

func TestNetworkFlowPipelineDropsWhenFull(t *testing.T) {
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
				d, ds, nps := createTestDetectorWithBufferSize(t, pubSubEnabled, bufferSize)
				d.unifiedDetector = &fakeUnifiedDetector{
					alerts: []*storage.Alert{{Id: "alert-1", Policy: &storage.Policy{Id: "p1"}}},
				}
				// Each flow: src=deployment, dst=internet → only src produces an event
				ds.EXPECT().GetSnapshot("dep-src").Return(srcDeployment).AnyTimes()
				nps.EXPECT().Find("default", gomock.Any()).Return(nil).AnyTimes()

				require.NoError(t, d.Start())
				defer d.Stop()

				for i := range totalEvents {
					d.ProcessNetworkFlow(context.Background(), &storage.NetworkFlow{
						Props: &storage.NetworkFlowProperties{
							SrcEntity: &storage.NetworkEntityInfo{
								Type: storage.NetworkEntityInfo_DEPLOYMENT,
								Id:   "dep-src",
							},
							DstEntity: &storage.NetworkEntityInfo{
								Type: storage.NetworkEntityInfo_INTERNET,
								Id:   fmt.Sprintf("internet-%d", i),
							},
							DstPort:    80,
							L4Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
						},
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
