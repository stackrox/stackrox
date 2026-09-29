package deduper

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/require"
)

func TestRuntimeAlertRemovalIsForwarded(t *testing.T) {
	cases := map[string]bool{
		"runtime alert only":                 false,
		"deploy result before runtime alert": true,
	}

	for name, sendDeployResult := range cases {
		t.Run(name, func(t *testing.T) {
			downstream := new(fakeStream)
			stream := NewDedupingMessageStream(downstream, nil, false)

			makeMessage := func(action central.ResourceAction, stage storage.LifecycleStage) *central.MsgFromSensor {
				return &central.MsgFromSensor{
					Msg: &central.MsgFromSensor_Event{
						Event: &central.SensorEvent{
							Id:     "deployment-1",
							Action: action,
							Resource: &central.SensorEvent_AlertResults{
								AlertResults: &central.AlertResults{
									DeploymentId: "deployment-1",
									Stage:        stage,
								},
							},
						},
					},
				}
			}

			if sendDeployResult {
				msg := makeMessage(central.ResourceAction_CREATE_RESOURCE, storage.LifecycleStage_DEPLOY)
				require.NoError(t, stream.Send(msg))
				require.Len(t, downstream.orderedMessages, 1)
			}

			runtime := makeMessage(central.ResourceAction_CREATE_RESOURCE, storage.LifecycleStage_RUNTIME)
			runtime.GetEvent().GetAlertResults().Alerts = []*storage.Alert{
				{
					Id:             "runtime-alert-1",
					LifecycleStage: storage.LifecycleStage_RUNTIME,
					State:          storage.ViolationState_ACTIVE,
				},
			}
			beforeRuntime := len(downstream.orderedMessages)
			require.NoError(t, stream.Send(runtime))
			require.Len(t, downstream.orderedMessages, beforeRuntime+1)

			// Sensor's removal message is empty and has the default lifecycle
			// stage, DEPLOY, even when the deployment has runtime alerts.
			removal := makeMessage(central.ResourceAction_REMOVE_RESOURCE, storage.LifecycleStage_DEPLOY)
			beforeRemoval := len(downstream.orderedMessages)
			require.NoError(t, stream.Send(removal))
			require.Len(t, downstream.orderedMessages, beforeRemoval+1,
				"alert removal must reach Central even without a prior deploy-time result")
			require.Same(t, removal, downstream.orderedMessages[len(downstream.orderedMessages)-1])

			// Repeated alert removals must also reach Central.
			require.NoError(t, stream.Send(removal))
			require.Len(t, downstream.orderedMessages, beforeRemoval+2)
			require.Same(t, removal, downstream.orderedMessages[len(downstream.orderedMessages)-1])

			// Removal must clear any cached deploy-time result.
			if sendDeployResult {
				msg := makeMessage(central.ResourceAction_CREATE_RESOURCE, storage.LifecycleStage_DEPLOY)
				require.NoError(t, stream.Send(msg))
				require.Len(t, downstream.orderedMessages, beforeRemoval+3)
			}
		})
	}
}

func TestDeploymentRemovalDeduplication(t *testing.T) {
	cases := map[string]bool{
		"previously sent": true,
		"never sent":      false,
	}
	for name, previouslySent := range cases {
		t.Run(name, func(t *testing.T) {
			downstream := new(fakeStream)
			stream := NewDedupingMessageStream(downstream, nil, false)
			msg := &central.MsgFromSensor{
				Msg: &central.MsgFromSensor_Event{
					Event: &central.SensorEvent{
						Id:     "deployment-1",
						Action: central.ResourceAction_CREATE_RESOURCE,
						Resource: &central.SensorEvent_Deployment{
							Deployment: &storage.Deployment{Id: "deployment-1"},
						},
					},
				},
			}
			if previouslySent {
				require.NoError(t, stream.Send(msg))
				require.Len(t, downstream.orderedMessages, 1)
			}
			removal := msg.CloneVT()
			removal.GetEvent().Action = central.ResourceAction_REMOVE_RESOURCE
			require.NoError(t, stream.Send(removal))
			require.NoError(t, stream.Send(removal))
			if previouslySent {
				require.Len(t, downstream.orderedMessages, 2)
				require.Same(t, removal, downstream.orderedMessages[1])
			} else {
				require.Empty(t, downstream.orderedMessages)
			}
		})
	}
}
