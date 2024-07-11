package manager

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	eventPkg "github.com/stackrox/rox/pkg/sensor/event"
	"github.com/stretchr/testify/assert"
)

func getDeploymentEvent(action central.ResourceAction, id, name string, processingAttempt int32) *central.MsgFromSensor {
	return &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id: id,
				Resource: &central.SensorEvent_Deployment{
					Deployment: &storage.Deployment{
						Id:   id,
						Name: name,
					},
				},
				Action: action,
			}},
		ProcessingAttempt: processingAttempt,
	}
}

func TestDeduper(t *testing.T) {
	type testEvents struct {
		event  *central.MsgFromSensor
		result bool
	}
	cases := []struct {
		testName   string
		testEvents []testEvents
	}{
		{
			testName: "empty event",
			testEvents: []testEvents{
				{
					event:  &central.MsgFromSensor{},
					result: true,
				},
			},
		},
		{
			testName: "network flow",
			testEvents: []testEvents{
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_NetworkFlowUpdate{
							NetworkFlowUpdate: &central.NetworkFlowUpdate{},
						},
					},
					result: true,
				},
			},
		},
		{
			testName: "duplicate runtime alerts",
			testEvents: []testEvents{
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "abc",
								Resource: &central.SensorEvent_AlertResults{
									AlertResults: &central.AlertResults{
										Stage: storage.LifecycleStage_RUNTIME,
									},
								},
							},
						},
					},
					result: true,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "abc",
								Resource: &central.SensorEvent_AlertResults{
									AlertResults: &central.AlertResults{
										Stage: storage.LifecycleStage_RUNTIME,
									},
								},
							},
						},
					},
					result: true,
				},
			},
		},
		{
			testName: "attempted alert",
			testEvents: []testEvents{
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "abc",
								Resource: &central.SensorEvent_AlertResults{
									AlertResults: &central.AlertResults{
										Stage: storage.LifecycleStage_DEPLOY,
										Alerts: []*storage.Alert{
											{
												State: storage.ViolationState_ATTEMPTED,
											},
										},
									},
								},
							},
						},
					},
					result: true,
				},
			},
		},
		{
			testName: "process indicator",
			testEvents: []testEvents{
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Resource: &central.SensorEvent_ProcessIndicator{
									ProcessIndicator: &storage.ProcessIndicator{},
								},
							}},
					},
					result: true,
				},
			},
		},
		{
			testName: "deployment create",
			testEvents: []testEvents{
				{
					event:  getDeploymentEvent(central.ResourceAction_CREATE_RESOURCE, "1", "dep1", 0),
					result: true,
				},
				{
					event:  getDeploymentEvent(central.ResourceAction_REMOVE_RESOURCE, "1", "dep1", 0),
					result: true,
				},
			},
		},
		{
			testName: "deployment update",
			testEvents: []testEvents{
				{
					event:  getDeploymentEvent(central.ResourceAction_UPDATE_RESOURCE, "1", "dep1", 0),
					result: true,
				},
				{
					event:  getDeploymentEvent(central.ResourceAction_REMOVE_RESOURCE, "1", "dep1", 0),
					result: true,
				},
			},
		},
		{
			testName: "deployment sync",
			testEvents: []testEvents{
				{
					event:  getDeploymentEvent(central.ResourceAction_REMOVE_RESOURCE, "1", "dep1", 0),
					result: true,
				},
				{
					event:  getDeploymentEvent(central.ResourceAction_REMOVE_RESOURCE, "1", "dep1", 0),
					result: true,
				},
			},
		},
		{
			testName: "deployment flow",
			testEvents: []testEvents{
				{
					event:  getDeploymentEvent(central.ResourceAction_CREATE_RESOURCE, "1", "dep1", 0),
					result: true,
				},
				{
					event:  getDeploymentEvent(central.ResourceAction_UPDATE_RESOURCE, "1", "dep1", 0),
					result: false,
				},
				{
					event:  getDeploymentEvent(central.ResourceAction_REMOVE_RESOURCE, "1", "dep1", 0),
					result: true,
				},
				{
					event:  getDeploymentEvent(central.ResourceAction_REMOVE_RESOURCE, "1", "dep1", 0),
					result: true,
				},
			},
		},
		{
			testName: "deployment processing attempt flow",
			testEvents: []testEvents{
				{
					event:  getDeploymentEvent(central.ResourceAction_CREATE_RESOURCE, "1", "dep1", 0),
					result: true,
				},
				{
					event:  getDeploymentEvent(central.ResourceAction_UPDATE_RESOURCE, "1", "dep1", 1),
					result: true,
				},
				{
					event:  getDeploymentEvent(central.ResourceAction_UPDATE_RESOURCE, "1", "dep2", 0),
					result: true,
				},
				{
					event:  getDeploymentEvent(central.ResourceAction_UPDATE_RESOURCE, "1", "dep1", 2),
					result: false,
				},
				{
					event:  getDeploymentEvent(central.ResourceAction_REMOVE_RESOURCE, "1", "dep2", 0),
					result: true,
				},
				{
					event:  getDeploymentEvent(central.ResourceAction_REMOVE_RESOURCE, "1", "dep1", 1),
					result: true,
				},
				{
					event:  getDeploymentEvent(central.ResourceAction_UPDATE_RESOURCE, "1", "dep2", 1),
					result: false,
				},
			},
		},
	}
	t.Parallel()
	for _, c := range cases {
		testCase := c
		t.Run(c.testName, func(t *testing.T) {
			deduper := NewDeduper(make(map[string]uint64)).(*deduperImpl)
			for _, testEvent := range testCase.testEvents {
				assert.Equal(t, testEvent.result, deduper.ShouldProcess(testEvent.event))
			}
			assert.Len(t, deduper.successfullyProcessed, 0)
			assert.Len(t, deduper.received, 0)
		})
	}
}

func TestReconciliation(t *testing.T) {
	deduper := NewDeduper(make(map[string]uint64)).(*deduperImpl)

	d1 := getDeploymentEvent(central.ResourceAction_SYNC_RESOURCE, "1", "1", 0)
	d2 := getDeploymentEvent(central.ResourceAction_SYNC_RESOURCE, "2", "2", 0)
	d3 := getDeploymentEvent(central.ResourceAction_UPDATE_RESOURCE, "3", "3", 0)
	d4 := getDeploymentEvent(central.ResourceAction_SYNC_RESOURCE, "4", "4", 0)
	d5 := getDeploymentEvent(central.ResourceAction_SYNC_RESOURCE, "5", "5", 0)

	d1Alert := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id: d1.GetEvent().GetId(),
				Resource: &central.SensorEvent_AlertResults{
					AlertResults: &central.AlertResults{
						DeploymentId: d1.GetEvent().GetId(),
						Stage:        storage.LifecycleStage_DEPLOY,
					},
				},
				Action: central.ResourceAction_SYNC_RESOURCE,
			},
		},
	}

	// Basic case
	deduper.StartSync()
	deduper.ShouldProcess(d1)
	deduper.MarkSuccessful(d1)
	deduper.ShouldProcess(d2)
	deduper.MarkSuccessful(d2)
	deduper.ProcessSync()
	assert.Len(t, deduper.successfullyProcessed, 2)
	assert.Contains(t, deduper.successfullyProcessed, eventPkg.GetKeyFromMessage(d1))
	assert.Contains(t, deduper.successfullyProcessed, eventPkg.GetKeyFromMessage(d2))

	// Values in successfully processed that should be removed
	deduper.ShouldProcess(d3)
	deduper.MarkSuccessful(d3)

	deduper.StartSync()
	deduper.ShouldProcess(d4)
	deduper.ShouldProcess(d5)
	deduper.MarkSuccessful(d4)
	deduper.MarkSuccessful(d5)
	deduper.ProcessSync()
	assert.Len(t, deduper.successfullyProcessed, 2)
	assert.Contains(t, deduper.successfullyProcessed, eventPkg.GetKeyFromMessage(d4))
	assert.Contains(t, deduper.successfullyProcessed, eventPkg.GetKeyFromMessage(d5))

	// Should clear out successfully processed
	deduper.StartSync()
	deduper.ProcessSync()
	assert.Len(t, deduper.successfullyProcessed, 0)

	// Add d1 to successfully processed map, call start sync again, and only put d1 in the received map
	// and not in successfully processed. Ensure it is not reconciled away
	deduper.StartSync()
	deduper.ShouldProcess(d1)
	deduper.MarkSuccessful(d1)
	deduper.StartSync()
	deduper.ShouldProcess(d1)
	deduper.ProcessSync()
	assert.Len(t, deduper.successfullyProcessed, 1)
	assert.Contains(t, deduper.successfullyProcessed, eventPkg.GetKeyFromMessage(d1))

	deduper.StartSync()
	deduper.ProcessSync()
	assert.Len(t, deduper.successfullyProcessed, 0)

	// Ensure alert is removed when reconcile occurs
	deduper.StartSync()
	deduper.ShouldProcess(d1)
	deduper.MarkSuccessful(d1)
	deduper.ShouldProcess(d1Alert)
	deduper.MarkSuccessful(d1Alert)
	assert.Len(t, deduper.successfullyProcessed, 2)
	deduper.StartSync()
	deduper.ProcessSync()
	assert.Len(t, deduper.successfullyProcessed, 0)
}
