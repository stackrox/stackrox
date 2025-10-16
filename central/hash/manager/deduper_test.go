package manager

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	eventPkg "github.com/stackrox/rox/pkg/sensor/event"
	"github.com/stackrox/rox/pkg/sensor/hash"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
)

func getDeploymentEvent(action central.ResourceAction, id, name string, processingAttempt int32) *central.MsgFromSensor {
	return central.MsgFromSensor_builder{
		Event: central.SensorEvent_builder{
			Id: id,
			Deployment: storage.Deployment_builder{
				Id:   id,
				Name: name,
			}.Build(),
			Action: action,
		}.Build(),
		ProcessingAttempt: processingAttempt,
	}.Build()
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
					event: central.MsgFromSensor_builder{
						NetworkFlowUpdate: &central.NetworkFlowUpdate{},
					}.Build(),
					result: true,
				},
			},
		},
		{
			testName: "duplicate runtime alerts",
			testEvents: []testEvents{
				{
					event: central.MsgFromSensor_builder{
						Event: central.SensorEvent_builder{
							Id: "abc",
							AlertResults: central.AlertResults_builder{
								Stage: storage.LifecycleStage_RUNTIME,
							}.Build(),
						}.Build(),
					}.Build(),
					result: true,
				},
				{
					event: central.MsgFromSensor_builder{
						Event: central.SensorEvent_builder{
							Id: "abc",
							AlertResults: central.AlertResults_builder{
								Stage: storage.LifecycleStage_RUNTIME,
							}.Build(),
						}.Build(),
					}.Build(),
					result: true,
				},
			},
		},
		{
			testName: "duplicate node indexes should not be deduped",
			testEvents: []testEvents{
				{
					event: central.MsgFromSensor_builder{
						Event: central.SensorEvent_builder{
							Id: "1",
							IndexReport: v4.IndexReport_builder{
								HashId:   "a",
								State:    "7",
								Success:  true,
								Err:      "",
								Contents: nil,
							}.Build(),
						}.Build(),
					}.Build(),
					result: true,
				},
				{
					event: central.MsgFromSensor_builder{
						Event: central.SensorEvent_builder{
							Id: "1",
							IndexReport: v4.IndexReport_builder{
								HashId:   "a",
								State:    "7",
								Success:  true,
								Err:      "",
								Contents: nil,
							}.Build(),
						}.Build(),
					}.Build(),
					result: true,
				},
			},
		},
		{
			testName: "attempted alert",
			testEvents: []testEvents{
				{
					event: central.MsgFromSensor_builder{
						Event: central.SensorEvent_builder{
							Id: "abc",
							AlertResults: central.AlertResults_builder{
								Stage: storage.LifecycleStage_DEPLOY,
								Alerts: []*storage.Alert{
									storage.Alert_builder{
										State: storage.ViolationState_ATTEMPTED,
									}.Build(),
								},
							}.Build(),
						}.Build(),
					}.Build(),
					result: true,
				},
			},
		},
		{
			testName: "process indicator",
			testEvents: []testEvents{
				{
					event: central.MsgFromSensor_builder{
						Event: central.SensorEvent_builder{
							ProcessIndicator: &storage.ProcessIndicator{},
						}.Build(),
					}.Build(),
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
	for _, c := range cases {
		testCase := c
		t.Run(c.testName, func(t *testing.T) {
			deduper := NewDeduper(make(map[string]uint64), uuid.NewV4().String()).(*deduperImpl)
			for _, testEvent := range testCase.testEvents {
				assert.Equal(t, testEvent.result, deduper.ShouldProcess(testEvent.event))
			}
			assert.Len(t, deduper.successfullyProcessed, 0)
			assert.Len(t, deduper.received, 0)
		})
	}
}

func TestReconciliation(t *testing.T) {
	deduper := NewDeduper(make(map[string]uint64), uuid.NewV4().String()).(*deduperImpl)

	d1 := getDeploymentEvent(central.ResourceAction_SYNC_RESOURCE, "1", "1", 0)
	d2 := getDeploymentEvent(central.ResourceAction_SYNC_RESOURCE, "2", "2", 0)
	d3 := getDeploymentEvent(central.ResourceAction_UPDATE_RESOURCE, "3", "3", 0)
	d4 := getDeploymentEvent(central.ResourceAction_SYNC_RESOURCE, "4", "4", 0)
	d5 := getDeploymentEvent(central.ResourceAction_SYNC_RESOURCE, "5", "5", 0)

	d1Alert := central.MsgFromSensor_builder{
		Event: central.SensorEvent_builder{
			Id: d1.GetEvent().GetId(),
			AlertResults: central.AlertResults_builder{
				DeploymentId: d1.GetEvent().GetId(),
				Stage:        storage.LifecycleStage_DEPLOY,
			}.Build(),
			Action: central.ResourceAction_SYNC_RESOURCE,
		}.Build(),
	}.Build()

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

type testEvents func(*testing.T, **deduperImpl)

func TestReconciliationOnDisconnection(t *testing.T) {
	d1 := getDeploymentEvent(central.ResourceAction_SYNC_RESOURCE, "1", "d1", 0)
	d2 := getDeploymentEvent(central.ResourceAction_SYNC_RESOURCE, "2", "d2", 0)
	d3 := getDeploymentEvent(central.ResourceAction_SYNC_RESOURCE, "3", "d3", 0)
	cases := map[string]struct {
		events []testEvents
	}{
		"normal sync": {
			events: []testEvents{
				// Simulated new connection
				newConnection(nil),
				// Sync resources d1 d2
				syncEventSuccessfully(d1),
				syncEventSuccessfully(d2),
				// Sync event
				syncEvent,
				// Assert d1 and d2
				assertEvents([]*central.MsgFromSensor{d1, d2}),
			},
		},
		"normal sync with initial deduper state (sensor cannot handle the deduper state)": {
			events: []testEvents{
				// Simulated new connection
				newConnection(getHashesFromEvents([]*central.MsgFromSensor{d1, d2})),
				// Sync resources d1 d2 as sensor cannot handle the deduper state
				syncEventShouldNotProcess(d1),
				syncEventShouldNotProcess(d2),
				// Sync event
				syncEvent,
				// Assert d1 and d2
				assertEvents([]*central.MsgFromSensor{d1, d2}),
			},
		},
		"normal sync with initial deduper state (sensor can handle the deduper state)": {
			events: []testEvents{
				// Simulated new connection
				newConnection(getHashesFromEvents([]*central.MsgFromSensor{d1, d2})),
				// Sensor does not send the sync resources d1 d2 as it can handle the deduper state
				syncEventSuccessfully(d3),
				// Sync event
				syncEvent,
				// Assert d1, d2, and d3
				assertEvents([]*central.MsgFromSensor{ /* d1, d2, */ d3}),
			},
		},
		"reconnection (sensor cannot handle the deduper state)": {
			events: []testEvents{
				// Simulated new connection
				newConnection(nil),
				// Sync resources d1 d2
				syncEventSuccessfully(d1),
				syncEventSuccessfully(d2),
				// Sync event
				syncEvent,
				// Assert d1 and d2
				assertEvents([]*central.MsgFromSensor{d1, d2}),
				// Simulated reconnection
				newConnection(nil),
				// Sensor sends the sync resources d1 d2 as it cannot handle the deduper state
				// Both event should not be processed as they are already in the successfullyProcessed map
				syncEventShouldNotProcess(d1),
				syncEventShouldNotProcess(d2),
				// Sync event
				syncEvent,
				// Assert d1 and d2
				assertEvents([]*central.MsgFromSensor{d1, d2}),
			},
		},
		"reconnection with unsuccessful events (sensor cannot handle the deduper state)": {
			events: []testEvents{
				// Simulated new connection
				newConnection(nil),
				// Sync resources d1 d2
				syncEventSuccessfully(d1),
				syncEventUnsuccessfully(d2),
				// Simulated reconnection
				newConnection(nil),
				// Sensor sends the sync resources d1 d2 again as it cannot handle the deduper state
				// d1 should not be processed as it is already in the successfully processed map
				syncEventShouldNotProcess(d1),
				// d2 should be processed
				syncEventSuccessfully(d2),
				// Sync event
				syncEvent,
				// Assert d1 and d2
				assertEvents([]*central.MsgFromSensor{d1, d2}),
			},
		},
		"reconnection (sensor can handle the deduper state)": {
			events: []testEvents{
				// Simulated new connection
				newConnection(nil),
				// Sync resources d1 d2
				syncEventSuccessfully(d1),
				syncEventSuccessfully(d2),
				// Sync event
				syncEvent,
				// Assert d1 and d2
				assertEvents([]*central.MsgFromSensor{d1, d2}),
				// Simulated reconnection
				newConnection(nil),
				// Sensor does not send the sync resources d1 d2 as it can handle the deduper state
				syncEventSuccessfully(d3),
				// Sync event
				syncEvent,
				// Assert d1, d2, and d3
				assertEvents([]*central.MsgFromSensor{ /* d1, d2, */ d3}), // FIXME: we should have d1 and d2
			},
		},
		"reconnection with unsuccessful events (sensor can handle the deduper state)": {
			events: []testEvents{
				// Simulated new connection
				newConnection(nil),
				// Sync resources d1 d2
				syncEventSuccessfully(d1),
				syncEventUnsuccessfully(d2),
				// Simulated reconnection
				newConnection(nil),
				// Sensor does not send the sync resources d1 as it can handle the deduper state
				syncEventSuccessfully(d2),
				syncEventSuccessfully(d3),
				// Sync event
				syncEvent,
				// Assert d1, d2, and d3
				assertEvents([]*central.MsgFromSensor{ /* d1, */ d2, d3}),
			},
		},
	}
	for tname, tc := range cases {
		t.Run(tname, func(tt *testing.T) {
			var deduper *deduperImpl
			for _, event := range tc.events {
				event(tt, &deduper)
			}
		})
	}
}

func newConnection(initialHashes map[string]uint64) testEvents {
	return func(_ *testing.T, deduper **deduperImpl) {
		if *deduper == nil {
			*deduper = NewDeduper(initialHashes, uuid.NewV4().String()).(*deduperImpl)
		}
		(*deduper).StartSync()
	}
}

func syncEventSuccessfully(event *central.MsgFromSensor) testEvents {
	return func(t *testing.T, deduper **deduperImpl) {
		assert.True(t, (*deduper).ShouldProcess(event))
		(*deduper).MarkSuccessful(event)
	}
}

func syncEventShouldNotProcess(event *central.MsgFromSensor) testEvents {
	return func(t *testing.T, deduper **deduperImpl) {
		assert.False(t, (*deduper).ShouldProcess(event))
	}
}

func syncEventUnsuccessfully(event *central.MsgFromSensor) testEvents {
	return func(t *testing.T, deduper **deduperImpl) {
		assert.True(t, (*deduper).ShouldProcess(event))
	}
}

func syncEvent(_ *testing.T, deduper **deduperImpl) {
	(*deduper).ProcessSync()
}

func assertEvents(events []*central.MsgFromSensor) testEvents {
	return func(t *testing.T, deduper **deduperImpl) {
		assert.Len(t, (*deduper).successfullyProcessed, len(events))
		for _, event := range events {
			_, ok := (*deduper).successfullyProcessed[eventPkg.GetKeyFromMessage(event)]
			assert.True(t, ok)
		}
	}
}

func getHashesFromEvents(events []*central.MsgFromSensor) map[string]uint64 {
	ret := make(map[string]uint64)
	hasher := hash.NewHasher()
	for _, event := range events {
		hashValue, _ := hasher.HashEvent(event.GetEvent())
		ret[eventPkg.GetKeyFromMessage(event)] = hashValue
	}
	return ret
}
