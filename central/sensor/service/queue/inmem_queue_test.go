package queue

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	queue := NewQueue()

	dep1 := newDeployment("id", "name1", central.ResourceAction_CREATE_RESOURCE)
	dep2 := newDeployment("id", "name2", central.ResourceAction_UPDATE_RESOURCE)
	dep3 := newDeployment("id", "name3", central.ResourceAction_UPDATE_RESOURCE)

	require.NoError(t, queue.Push(dep1))
	require.NoError(t, queue.Push(dep2))
	require.NoError(t, queue.Push(dep3))

	// This should pull name1 that should not be deduped
	val, err := queue.Pull()
	require.NoError(t, err)
	assert.Equal(t, dep1, val)

	// This should dedupe
	val, err = queue.Pull()
	require.NoError(t, err)
	assert.Equal(t, dep3, val)

	val, err = queue.Pull()
	require.NoError(t, err)
	assert.Nil(t, val)

	// Test that if you remove an item with the ID that its okay
	require.NoError(t, queue.Push(dep1))
	require.NoError(t, queue.Push(dep2))

	val, err = queue.Pull()
	require.NoError(t, err)
	assert.Equal(t, dep1, val)

	require.NoError(t, queue.Push(dep3))

	val, err = queue.Pull()
	require.NoError(t, err)
	assert.Equal(t, dep3, val)

}
