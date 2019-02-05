package streamer

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func newDeployment(id, name string, action central.ResourceAction) *central.MsgFromSensor {
	return &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Resource: &central.SensorEvent_Deployment{
					Deployment: &storage.Deployment{
						Id:   id,
						Name: name,
					},
				},
				Action: action,
			},
		},
	}
}

func TestDedupingLogic(t *testing.T) {
	queue := newQueue()

	dep1 := newDeployment("id", "name1", central.ResourceAction_CREATE_RESOURCE)
	dep2 := newDeployment("id", "name2", central.ResourceAction_UPDATE_RESOURCE)
	dep3 := newDeployment("id", "name3", central.ResourceAction_UPDATE_RESOURCE)

	queue.push(dep1)
	queue.push(dep2)
	queue.push(dep3)

	// This should pull name1 that should not be deduped
	val := queue.pull()
	assert.Equal(t, dep1, val)

	// This should dedupe
	val = queue.pull()
	assert.Equal(t, dep3, val)

	val = queue.pull()
	assert.Nil(t, val)

	// Test that if you remove an item with the ID that its okay
	queue.push(dep1)
	queue.push(dep2)

	val = queue.pull()
	assert.Equal(t, dep1, val)

	queue.push(dep3)

	val = queue.pull()
	assert.Equal(t, dep3, val)

}
