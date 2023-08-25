package processsignal

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/detector/mocks"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestProcessPipelineOffline(t *testing.T) {
	containerMetadata1 := clusterentities.ContainerMetadata{
		DeploymentID: "mock-deployment-1",
		ContainerID:  "1e43ac4f61f9",
	}
	containerMetadata2 := clusterentities.ContainerMetadata{
		DeploymentID: "mock-deployment-2",
		ContainerID:  "2e43ac4f61f9",
	}
	type processIndicatorMessageT struct {
		signal                  *storage.ProcessSignal
		expectDeploymentID      string
		expectContextCancel     func(t assert.TestingT, err error, msgAndArgs ...interface{}) bool
		signalProcessingRoutine func(p *Pipeline, signal *storage.ProcessSignal, store *clusterentities.Store,
			meta clusterentities.ContainerMetadata, wg *sync.WaitGroup)
	}
	cases := []struct {
		name string
		// initialState is the state in which the pipeline will be set immediately after starting
		initialState common.SensorComponentEvent
		// laterState is the state in which the pipeline will be set after handling the `initialSignal`
		laterState common.SensorComponentEvent
		// initialSignal are emitted after transition to `initialState` but before `laterState`
		initialSignal *processIndicatorMessageT
		// laterSignal are emitted after transition to `laterState`
		laterSignal *processIndicatorMessageT
	}{
		{
			name:         "Staying online should yield all messages with valid context",
			initialState: common.SensorComponentEventCentralReachable,
			laterState:   common.SensorComponentEventCentralReachable,
			initialSignal: &processIndicatorMessageT{
				signal:                  &storage.ProcessSignal{ContainerId: containerMetadata1.ContainerID},
				expectDeploymentID:      containerMetadata1.DeploymentID,
				expectContextCancel:     assert.NoError,
				signalProcessingRoutine: processSignal,
			},
			laterSignal: &processIndicatorMessageT{
				signal:                  &storage.ProcessSignal{ContainerId: containerMetadata2.ContainerID},
				expectDeploymentID:      containerMetadata2.DeploymentID,
				expectContextCancel:     assert.NoError,
				signalProcessingRoutine: processSignal,
			},
		},
		{
			name:         "Going offline should yield all second message with canceled context",
			initialState: common.SensorComponentEventCentralReachable,
			laterState:   common.SensorComponentEventOfflineMode,
			initialSignal: &processIndicatorMessageT{
				signal:                  &storage.ProcessSignal{ContainerId: containerMetadata1.ContainerID},
				expectDeploymentID:      containerMetadata1.DeploymentID,
				expectContextCancel:     assert.NoError,
				signalProcessingRoutine: processSignal,
			},
			laterSignal: &processIndicatorMessageT{
				signal:                  &storage.ProcessSignal{ContainerId: containerMetadata2.ContainerID},
				expectDeploymentID:      containerMetadata2.DeploymentID,
				expectContextCancel:     assert.Error,
				signalProcessingRoutine: processSignal,
			},
		},
		{
			name:         "Staying Offline mode should yield all messages with canceled context",
			initialState: common.SensorComponentEventOfflineMode,
			laterState:   common.SensorComponentEventOfflineMode,
			initialSignal: &processIndicatorMessageT{
				signal:                  &storage.ProcessSignal{ContainerId: containerMetadata1.ContainerID},
				expectDeploymentID:      containerMetadata1.DeploymentID,
				expectContextCancel:     assert.Error,
				signalProcessingRoutine: processSignal,
			},
			laterSignal: &processIndicatorMessageT{
				signal:                  &storage.ProcessSignal{ContainerId: containerMetadata2.ContainerID},
				expectDeploymentID:      containerMetadata2.DeploymentID,
				expectContextCancel:     assert.Error,
				signalProcessingRoutine: processSignal,
			},
		},
		{
			name:         "Going online should yield second message with valid context",
			initialState: common.SensorComponentEventOfflineMode,
			laterState:   common.SensorComponentEventCentralReachable,
			initialSignal: &processIndicatorMessageT{
				signal:                  &storage.ProcessSignal{ContainerId: containerMetadata1.ContainerID},
				expectDeploymentID:      containerMetadata1.DeploymentID,
				expectContextCancel:     assert.Error,
				signalProcessingRoutine: processSignal,
			},
			laterSignal: &processIndicatorMessageT{
				signal:                  &storage.ProcessSignal{ContainerId: containerMetadata2.ContainerID},
				expectDeploymentID:      containerMetadata2.DeploymentID,
				expectContextCancel:     assert.NoError,
				signalProcessingRoutine: processSignal,
			},
		},
		{
			name:         "Transitioning through offline should keep the enricher alive",
			initialState: common.SensorComponentEventOfflineMode,
			laterState:   common.SensorComponentEventCentralReachable,
			// initial signal is processed in offline mode without the enricher (processSignal) as metadata is known
			initialSignal: &processIndicatorMessageT{
				signal:                  &storage.ProcessSignal{ContainerId: containerMetadata1.ContainerID},
				expectDeploymentID:      containerMetadata1.DeploymentID,
				expectContextCancel:     assert.Error,
				signalProcessingRoutine: processSignal,
			},
			// initial signal is processed in online using the enricher (processSignalAsync) as metadata will be
			// updated through ticker asynchronously
			laterSignal: &processIndicatorMessageT{
				signal:                  &storage.ProcessSignal{ContainerId: containerMetadata2.ContainerID},
				expectDeploymentID:      containerMetadata2.DeploymentID,
				expectContextCancel:     assert.NoError,
				signalProcessingRoutine: processSignalAsync,
			},
		},
	}

	sensorEvents := make(chan *message.ExpiringMessage)
	defer close(sensorEvents)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	testCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	actualEvents := forwardEvents(testCtx, sensorEvents)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			caseCtx, cancel := context.WithCancel(context.Background())
			defer cancel()
			mockStore := clusterentities.NewStore()
			mockDetector := mocks.NewMockDetector(mockCtrl)
			defer deleteStore(containerMetadata1.DeploymentID, mockStore)
			defer deleteStore(containerMetadata2.DeploymentID, mockStore)

			// Detector can be called in any order, so no assertions regarding the order of events.
			mockDetector.EXPECT().ProcessIndicator(gomock.Any(), gomock.Any()).
				MinTimes(2).
				MaxTimes(2).
				DoAndReturn(func(_ context.Context, ind *storage.ProcessIndicator) {
					assert.Contains(t,
						[]string{tc.initialSignal.expectDeploymentID, tc.laterSignal.expectDeploymentID},
						ind.GetDeploymentId())
				})

			p := NewProcessPipeline(sensorEvents, mockStore,
				filter.NewFilter(5, 5, []int{3, 3, 3}),
				mockDetector)
			defer p.Shutdown()

			metadataWg := &sync.WaitGroup{}
			metadataWg.Add(2)

			p.Notify(tc.initialState)
			tc.initialSignal.signalProcessingRoutine(p, tc.initialSignal.signal, mockStore, containerMetadata1, metadataWg)

			p.Notify(tc.laterState)
			tc.laterSignal.signalProcessingRoutine(p, tc.laterSignal.signal, mockStore, containerMetadata2, metadataWg)

			// Wait for metadata to arrive - either directly or through the ticker
			metadataWg.Wait()

			// Events contains processed signals. They may arrive in any order
			events := collectEventsFor(caseCtx, actualEvents, 500*time.Millisecond)
			// These tests always use two signals that should be processed
			assert.Len(t, events, 2)

			for _, e := range events {
				assert.Contains(t,
					[]string{containerMetadata1.DeploymentID, containerMetadata2.DeploymentID},
					e.GetEvent().GetProcessIndicator().GetDeploymentId())
			}
		})
	}
}

// processSignal calls p.Process and ensures that the stores are in the correct state for the test to make sense
func processSignal(p *Pipeline,
	signal *storage.ProcessSignal,
	store *clusterentities.Store,
	meta clusterentities.ContainerMetadata,
	wg *sync.WaitGroup) {
	defer wg.Done()
	// If PI has metadata in store it will be enriched immediately.
	// If not, then the enrichment happens async based on ticker - see processSignalAsync.
	// Here, we simulate immediate enrichment to not make the test more complex.
	updateStore(meta.ContainerID, meta.DeploymentID, meta, store)
	p.Process(signal)
}

func processSignalAsync(p *Pipeline,
	signal *storage.ProcessSignal,
	store *clusterentities.Store,
	meta clusterentities.ContainerMetadata,
	wg *sync.WaitGroup) {
	defer wg.Done()
	// For the scenario when the enrichment happens async based on ticker -
	// simulating the situation in which we receive a process indicator from a container that is still unknown -
	// this test would need to:
	// 1) Call p.Process
	// 2) Call updateStore(...)
	// 3) Wait for the metadata to be consumed (see `enricher.processLoop` case metadata)
	// 4) Wait for enricher tick (see `enricher.processLoop` case <-ticker.C)
	// 5) Wait for the enriched signal to be written to the channel
	// 6) Assert on the messages from the channel.

	p.Process(signal)
	updateStore(meta.ContainerID, meta.DeploymentID, meta, store)
	// Let enricher tick at least once
	<-time.After(enrichInterval)
}

// collectEventsFor reads events from a channel for a given time and returns them as slice
func collectEventsFor(ctx context.Context, ch <-chan *message.ExpiringMessage, t time.Duration) []*message.ExpiringMessage {
	arr := make([]*message.ExpiringMessage, 0)
	for {
		select {
		case m, ok := <-ch:
			if !ok {
				return arr
			}
			arr = append(arr, m)
		case <-ctx.Done():
			return arr
		case <-time.After(t):
			return arr
		}
	}
}

func TestProcessPipelineOnline(t *testing.T) {
	sensorEvents := make(chan *message.ExpiringMessage)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockStore := clusterentities.NewStore()
	mockDetector := mocks.NewMockDetector(mockCtrl)

	p := NewProcessPipeline(sensorEvents, mockStore, filter.NewFilter(5, 5, []int{10, 10, 10}),
		mockDetector)
	p.Notify(common.SensorComponentEventCentralReachable)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	actualEvents := forwardEvents(ctx, sensorEvents)

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
	mockDetector.EXPECT().ProcessIndicator(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, ind *storage.ProcessIndicator) {
		assert.Equal(t, deploymentID, ind.GetDeploymentId())
	})
	p.Process(&signal)

	msg := <-actualEvents
	assert.NotNil(t, msg)
	assert.Equal(t, deploymentID, msg.GetEvent().GetProcessIndicator().GetDeploymentId())
	deleteStore(deploymentID, mockStore)

	// 2. Signal which does not have metadata.
	signal = storage.ProcessSignal{
		ContainerId: containerID,
	}
	mockDetector.EXPECT().ProcessIndicator(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, ind *storage.ProcessIndicator) {
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
	mockDetector.EXPECT().ProcessIndicator(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, ind *storage.ProcessIndicator) {
		assert.Equal(t, deploymentID, ind.GetDeploymentId())
	})
	p.Process(&signal)
	updateStore(containerID, deploymentID, containerMetadata, mockStore)
	msg = <-actualEvents
	assert.NotNil(t, msg)
	assert.Equal(t, deploymentID, msg.GetEvent().GetProcessIndicator().GetDeploymentId())
	mockDetector.EXPECT().ProcessIndicator(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, ind *storage.ProcessIndicator) {
		assert.Equal(t, deploymentID, ind.GetDeploymentId())
	})
	p.Process(&signal)
	msg = <-actualEvents
	assert.NotNil(t, msg)
	assert.Equal(t, deploymentID, msg.GetEvent().GetProcessIndicator().GetDeploymentId())
	deleteStore(deploymentID, mockStore)
}

func forwardEvents(ctx context.Context, sensorEvents chan *message.ExpiringMessage) <-chan *message.ExpiringMessage {
	results := make(chan *message.ExpiringMessage)
	go func() {
		defer close(results)
		for {
			select {
			case results <- <-sensorEvents:
			case <-ctx.Done():
				return
			}
		}
	}()
	return results
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
