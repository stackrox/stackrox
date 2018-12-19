package processsignal

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stretchr/testify/assert"
)

func TestProcessPipeline(t *testing.T) {
	sensorEvents := make(chan *central.SensorEvent)
	actualEvents := make(chan *central.SensorEvent)
	mockStore := clusterentities.NewStore()
	p := NewProcessPipeline(sensorEvents, mockStore)
	closeChan := make(chan bool)

	go consumeEnrichedSignals(sensorEvents, actualEvents, closeChan)

	containerID := "fe43ac4f61f9"
	deploymentID := "mock-deployment"
	containerMetadata := clusterentities.ContainerMetadata{
		DeploymentID: deploymentID,
		ContainerID:  containerID,
	}

	// 1. Signal which has metadata present in store
	updateStore(containerID, deploymentID, containerMetadata, mockStore)
	signal := storage.ProcessSignal{
		ContainerId: containerID,
	}
	p.Process(&signal)
	time.Sleep(time.Second)
	event := <-actualEvents
	assert.NotNil(t, event)
	assert.Equal(t, deploymentID, event.GetProcessIndicator().GetDeploymentId())
	deleteStore(deploymentID, mockStore)

	// 2. Signal which does not have metadata.
	signal = storage.ProcessSignal{
		ContainerId: containerID,
	}
	p.Process(&signal)
	updateStore(containerID, deploymentID, containerMetadata, mockStore)
	event = <-actualEvents
	assert.NotNil(t, event)
	assert.Equal(t, deploymentID, event.GetProcessIndicator().GetDeploymentId())
	deleteStore(deploymentID, mockStore)

	// 3. back to back signals
	signal = storage.ProcessSignal{
		ContainerId: containerID,
	}
	p.Process(&signal)
	updateStore(containerID, deploymentID, containerMetadata, mockStore)
	event = <-actualEvents
	assert.NotNil(t, event)
	assert.Equal(t, deploymentID, event.GetProcessIndicator().GetDeploymentId())
	p.Process(&signal)
	event = <-actualEvents
	assert.NotNil(t, event)
	assert.Equal(t, deploymentID, event.GetProcessIndicator().GetDeploymentId())
	deleteStore(deploymentID, mockStore)

	closeChan <- true
}

func consumeEnrichedSignals(sensorEvents chan *central.SensorEvent, results chan *central.SensorEvent, closeChan chan bool) {
	for {
		select {
		case event := <-sensorEvents:
			results <- event
		case <-closeChan:
			return
		}
	}
}

func updateStore(containerID, deploymentID string, containerMetadata clusterentities.ContainerMetadata, mockStore *clusterentities.Store) {
	entityData := &clusterentities.EntityData{}
	entityData.AddContainerID(containerID, containerMetadata)
	updates := map[string]*clusterentities.EntityData{
		deploymentID: entityData,
	}
	mockStore.Apply(updates, false)
}

func deleteStore(deploymentID string, mockStore *clusterentities.Store) {
	entityData := &clusterentities.EntityData{}
	updates := map[string]*clusterentities.EntityData{
		deploymentID: entityData,
	}
	mockStore.Apply(updates, false)
}
