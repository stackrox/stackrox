package processsignal

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/detector/mocks"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestProcessPipeline(t *testing.T) {
	sensorEvents := make(chan *message.ExpiringMessage)
	actualEvents := make(chan *message.ExpiringMessage)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockStore := clusterentities.NewStore()
	mockDetector := mocks.NewMockDetector(mockCtrl)

	p := NewProcessPipeline(sensorEvents, mockStore, filter.NewFilter(5, 5, []int{10, 10, 10}),
		mockDetector)
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
	mockDetector.EXPECT().ProcessIndicator(gomock.Any()).DoAndReturn(func(ind *storage.ProcessIndicator) {
		assert.Equal(t, deploymentID, ind.GetDeploymentId())
	})
	p.Process(&signal)
	time.Sleep(time.Second)
	msg := <-actualEvents
	assert.NotNil(t, msg)
	assert.Equal(t, deploymentID, msg.GetEvent().GetProcessIndicator().GetDeploymentId())
	deleteStore(deploymentID, mockStore)

	// 2. Signal which does not have metadata.
	signal = storage.ProcessSignal{
		ContainerId: containerID,
	}
	mockDetector.EXPECT().ProcessIndicator(gomock.Any()).DoAndReturn(func(ind *storage.ProcessIndicator) {
		assert.Equal(t, deploymentID, ind.GetDeploymentId())
	})
	p.Process(&signal)
	updateStore(containerID, deploymentID, containerMetadata, mockStore)
	msg = <-actualEvents
	assert.NotNil(t, msg)
	assert.Equal(t, deploymentID, msg.GetEvent().GetProcessIndicator().GetDeploymentId())
	deleteStore(deploymentID, mockStore)

	// 3. back to back signals
	signal = storage.ProcessSignal{
		ContainerId: containerID,
	}
	mockDetector.EXPECT().ProcessIndicator(gomock.Any()).DoAndReturn(func(ind *storage.ProcessIndicator) {
		assert.Equal(t, deploymentID, ind.GetDeploymentId())
	})
	p.Process(&signal)
	updateStore(containerID, deploymentID, containerMetadata, mockStore)
	msg = <-actualEvents
	assert.NotNil(t, msg)
	assert.Equal(t, deploymentID, msg.GetEvent().GetProcessIndicator().GetDeploymentId())
	mockDetector.EXPECT().ProcessIndicator(gomock.Any()).DoAndReturn(func(ind *storage.ProcessIndicator) {
		assert.Equal(t, deploymentID, ind.GetDeploymentId())
	})
	p.Process(&signal)
	msg = <-actualEvents
	assert.NotNil(t, msg)
	assert.Equal(t, deploymentID, msg.GetEvent().GetProcessIndicator().GetDeploymentId())
	deleteStore(deploymentID, mockStore)

	closeChan <- true
}

func consumeEnrichedSignals(sensorEvents chan *message.ExpiringMessage, results chan *message.ExpiringMessage, closeChan chan bool) {
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
