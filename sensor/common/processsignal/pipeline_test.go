package processsignal

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stretchr/testify/assert"
)

func TestProcessPipeline(t *testing.T) {
	sensorEvents := make(chan *v1.SensorEvent)
	actualEvents := make(chan *v1.SensorEvent)
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
	signal := v1.ProcessSignal{
		ContainerId: containerID,
	}
	p.Process(&signal)
	time.Sleep(time.Second)
	event := <-actualEvents
	assert.NotNil(t, event)
	assert.Equal(t, deploymentID, event.GetProcessIndicator().GetDeploymentId())
	deleteStore(deploymentID, mockStore)

	// 2. Signal which does not have metadata.
	signal = v1.ProcessSignal{
		ContainerId: containerID,
	}
	p.Process(&signal)
	updateStore(containerID, deploymentID, containerMetadata, mockStore)
	event = <-actualEvents
	assert.NotNil(t, event)
	assert.Equal(t, deploymentID, event.GetProcessIndicator().GetDeploymentId())
	deleteStore(deploymentID, mockStore)

	// 3. back to back signals
	signal = v1.ProcessSignal{
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

func consumeEnrichedSignals(sensorEvents chan *v1.SensorEvent, results chan *v1.SensorEvent, closeChan chan bool) {
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
