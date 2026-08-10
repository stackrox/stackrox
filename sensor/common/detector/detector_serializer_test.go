package detector

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/common"
	detectorEvents "github.com/stackrox/rox/sensor/common/detector/events"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSerializerOlderUpdateIsIgnored(t *testing.T) {
	for _, pubSubEnabled := range []bool{false, true} {
		t.Run(fmt.Sprintf("pubsub=%t", pubSubEnabled), func(t *testing.T) {
			t.Setenv(features.SensorInternalPubSub.EnvVar(), fmt.Sprintf("%t", pubSubEnabled))

			synctest.Test(t, func(t *testing.T) {
				d, _, nps, _ := createTestDetector(t, pubSubEnabled)
				d.unifiedDetector = &fakeUnifiedDetector{
					alerts: []*storage.Alert{{Id: "a1", Policy: &storage.Policy{Id: "p1"}}},
				}
				nps.EXPECT().Find("default", gomock.Any()).Return(nil).AnyTimes()

				require.NoError(t, d.Start())
				defer d.Stop()

				d.Notify(common.SensorComponentEventCentralReachable)

				// CREATE with timestamp 100
				depNew := &storage.Deployment{
					Id: "dep-1", Name: "test", Namespace: "default",
					StateTimestamp: 100,
				}
				d.ProcessDeployment(context.Background(), depNew, central.ResourceAction_CREATE_RESOURCE)
				synctest.Wait()
				<-d.output

				// UPDATE with older timestamp 50 — should be ignored by serializer
				d.deduper.removeDeployment("dep-1")
				depOld := &storage.Deployment{
					Id: "dep-1", Name: "test", Namespace: "default",
					StateTimestamp: 50,
				}
				d.ProcessDeployment(context.Background(), depOld, central.ResourceAction_UPDATE_RESOURCE)
				synctest.Wait()

				select {
				case msg := <-d.output:
					t.Fatalf("expected no output for older update but got: %v", msg)
				default:
				}
			})
		})
	}
}

func TestSerializerCreateEnforces(t *testing.T) {
	for _, pubSubEnabled := range []bool{false, true} {
		t.Run(fmt.Sprintf("pubsub=%t", pubSubEnabled), func(t *testing.T) {
			t.Setenv(features.SensorInternalPubSub.EnvVar(), fmt.Sprintf("%t", pubSubEnabled))

			synctest.Test(t, func(t *testing.T) {
				d, _, nps, _ := createTestDetector(t, pubSubEnabled)
				enforcerFake := &recordingEnforcer{}
				d.enforcer = enforcerFake
				d.unifiedDetector = &fakeUnifiedDetector{
					alerts: []*storage.Alert{{Id: "a1", Policy: &storage.Policy{Id: "p1"}}},
				}
				nps.EXPECT().Find("default", gomock.Any()).Return(nil).AnyTimes()

				require.NoError(t, d.Start())
				defer d.Stop()

				d.Notify(common.SensorComponentEventCentralReachable)

				dep := &storage.Deployment{Id: "dep-1", Name: "test", Namespace: "default"}
				d.ProcessDeployment(context.Background(), dep, central.ResourceAction_CREATE_RESOURCE)
				synctest.Wait()
				<-d.output

				assert.True(t, enforcerFake.called.Load(), "enforcer should be called on CREATE")
			})
		})
	}
}

func TestSerializerRemoveDeletesFromProcessingMap(t *testing.T) {
	for _, pubSubEnabled := range []bool{false, true} {
		t.Run(fmt.Sprintf("pubsub=%t", pubSubEnabled), func(t *testing.T) {
			t.Setenv(features.SensorInternalPubSub.EnvVar(), fmt.Sprintf("%t", pubSubEnabled))

			synctest.Test(t, func(t *testing.T) {
				d, _, nps, _ := createTestDetector(t, pubSubEnabled)
				nps.EXPECT().Find("default", gomock.Any()).Return(nil).AnyTimes()

				require.NoError(t, d.Start())
				defer d.Stop()

				d.Notify(common.SensorComponentEventCentralReachable)

				dep := &storage.Deployment{Id: "dep-1", Name: "test", Namespace: "default"}

				// CREATE to add to processing map
				d.ProcessDeployment(context.Background(), dep, central.ResourceAction_CREATE_RESOURCE)
				synctest.Wait()
				<-d.output

				_, exists := d.deploymentProcessingMap["dep-1"]
				assert.True(t, exists, "deployment should be in processing map after CREATE")

				// REMOVE should delete from processing map and send empty alerts
				d.ProcessDeployment(context.Background(), dep, central.ResourceAction_REMOVE_RESOURCE)
				synctest.Wait()

				select {
				case msg := <-d.output:
					alertResults := msg.GetEvent().GetAlertResults()
					assert.Equal(t, "dep-1", alertResults.GetDeploymentId())
					assert.Empty(t, alertResults.GetAlerts(), "REMOVE should send empty alerts")
				default:
					t.Fatal("expected output for REMOVE")
				}

				_, exists = d.deploymentProcessingMap["dep-1"]
				assert.False(t, exists, "deployment should be removed from processing map after REMOVE")
			})
		})
	}
}

// TestSerializerUpdateAfterRemoveMarksAlertsResolved verifies that when the
// serializer receives an UPDATE for a deployment not in the processing map,
// all alerts are marked RESOLVED. This simulates a race where a late UPDATE
// (still in the enricher/blockingScan goroutine) reaches the serializer after
// a faster REMOVE has already cleared the deployment from the processing map.
//
// We send directly to the serializer instead of going through ProcessDeployment
// because ProcessDeployment calls markDeploymentForProcessing which re-adds
// the deployment to the map, making it impossible to reproduce the race with
// synctest's deterministic scheduling. An empty processing map is equivalent
// to the post-REMOVE state, so no CREATE/REMOVE setup is needed.
func TestSerializerUpdateAfterRemoveMarksAlertsResolved(t *testing.T) {
	for _, pubSubEnabled := range []bool{false, true} {
		t.Run(fmt.Sprintf("pubsub=%t", pubSubEnabled), func(t *testing.T) {
			t.Setenv(features.SensorInternalPubSub.EnvVar(), fmt.Sprintf("%t", pubSubEnabled))

			synctest.Test(t, func(t *testing.T) {
				d, _, _, _ := createTestDetector(t, pubSubEnabled)

				require.NoError(t, d.Start())
				defer d.Stop()

				d.sendDeployAlertOutput(&detectorEvents.DeployAlertOutputEvent{
					Context: context.Background(),
					Action:  central.ResourceAction_UPDATE_RESOURCE,
					Results: &central.AlertResults{
						DeploymentId: "dep-1",
						Alerts: []*storage.Alert{
							{Id: "a1", Policy: &storage.Policy{Id: "p1"}},
						},
					},
				})
				synctest.Wait()

				select {
				case msg := <-d.output:
					alertResults := msg.GetEvent().GetAlertResults()
					require.NotEmpty(t, alertResults.GetAlerts())
					for _, alert := range alertResults.GetAlerts() {
						assert.Equal(t, storage.ViolationState_RESOLVED, alert.GetState(),
							"alerts for UPDATE after REMOVE should be marked RESOLVED")
					}
				default:
					t.Fatal("expected output for UPDATE after REMOVE")
				}
			})
		})
	}
}

func TestSerializerPubSubStopUnblocksOutputSend(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

	synctest.Test(t, func(t *testing.T) {
		d, _, nps, _ := createTestDetector(t, true)
		d.output = make(chan *message.ExpiringMessage)
		d.unifiedDetector = &fakeUnifiedDetector{
			alerts: []*storage.Alert{{Id: "a1", Policy: &storage.Policy{Id: "p1"}}},
		}
		nps.EXPECT().Find("default", gomock.Any()).Return(nil).AnyTimes()

		require.NoError(t, d.Start())

		d.Notify(common.SensorComponentEventCentralReachable)

		dep := &storage.Deployment{Id: "dep-1", Name: "test", Namespace: "default"}
		d.ProcessDeployment(context.Background(), dep, central.ResourceAction_CREATE_RESOURCE)
		synctest.Wait()

		// Don't read from d.output — the consumer callback is blocked on the send.
		// Stop should complete because the send is guarded by alertStopSig.
		done := make(chan struct{})
		go func() {
			d.Stop()
			close(done)
		}()
		synctest.Wait()

		select {
		case <-done:
		default:
			t.Fatal("Stop() did not complete: handleDeployAlertOutputEvent is blocked on d.output without a stop guard")
		}
	})
}

type recordingEnforcer struct {
	fakeEnforcer
	called atomic.Bool
}

func (r *recordingEnforcer) ProcessAlertResults(_ central.ResourceAction, _ storage.LifecycleStage, _ *central.AlertResults) {
	r.called.Store(true)
}
