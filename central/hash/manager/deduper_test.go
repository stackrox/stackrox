package manager

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
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
