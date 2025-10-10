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
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	containerID1      = "1e43ac4f61f9"
	containerID2      = "2e43ac4f61f9"
	containerID3      = "3e43ac4f61f9"
	deploymentID1     = "mock-deployment-1"
	deploymentID2     = "mock-deployment-2"
	deploymentID3     = "mock-deployment-3"
	outputChannelSize = 2
)

func TestProcessPipelineOfflineV3(t *testing.T) {
	// With event buffering enabled, going from online to offline and vice-versa won't do anything.
	// The tests add the functions online and offline to illustrate how the pipeline would be called in a real scenario.
	cases := map[string]struct {
		entities []clusterentities.ContainerMetadata
		events   []func(*testing.T, *Pipeline)
	}{
		"Online -> signal -> read -> offline -> signal -> online -> read": {
			entities: []clusterentities.ContainerMetadata{
				newEntity(containerID1, deploymentID1),
				newEntity(containerID2, deploymentID2),
			},
			events: []func(*testing.T, *Pipeline){
				online,
				signal(&storage.ProcessSignal{ContainerId: containerID1}, false),
				assertSize(1),
				read(containerID1, deploymentID1),
				assertSize(0),
				offline,
				signal(&storage.ProcessSignal{ContainerId: containerID2}, false),
				assertSize(1),
				online,
				read(containerID2, deploymentID2),
				assertSize(0),
			},
		},
		"Offline -> signal -> signal -> online -> read -> read": {
			entities: []clusterentities.ContainerMetadata{
				newEntity(containerID1, deploymentID1),
				newEntity(containerID2, deploymentID2),
			},
			events: []func(*testing.T, *Pipeline){
				offline,
				signal(&storage.ProcessSignal{ContainerId: containerID1}, false),
				signal(&storage.ProcessSignal{ContainerId: containerID2}, false),
				assertSize(2),
				online,
				read(containerID1, deploymentID1),
				read(containerID2, deploymentID2),
				assertSize(0),
			},
		},
		"Offline -> signal -> signal -> signal -> Online -> read -> read": {
			entities: []clusterentities.ContainerMetadata{
				newEntity(containerID1, deploymentID1),
				newEntity(containerID2, deploymentID2),
				newEntity(containerID3, deploymentID3),
			},
			events: []func(*testing.T, *Pipeline){
				offline,
				signal(&storage.ProcessSignal{ContainerId: containerID1}, false),
				signal(&storage.ProcessSignal{ContainerId: containerID2}, false),
				signal(&storage.ProcessSignal{ContainerId: containerID3}, true),
				assertSize(2), // The third signal should be dropped
				online,
				read(containerID1, deploymentID1),
				read(containerID2, deploymentID2),
				assertSize(0), // The buffer should be empty at this point
			},
		},
	}
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			sensorEvents := make(chan *message.ExpiringMessage, outputChannelSize)
			mockStore := clusterentities.NewStore(0, nil, false)
			mockDetector := mocks.NewMockDetector(mockCtrl)
			pipeline := NewProcessPipeline(sensorEvents, mockStore,
				filter.NewFilter(5, 5, []int{3, 3, 3}),
				mockDetector)
			t.Cleanup(func() {
				pipeline.Shutdown()
				for _, entity := range tc.entities {
					deleteStore(entity.DeploymentID, mockStore)
				}
				close(sensorEvents)
			})
			mockDetector.EXPECT().ProcessIndicator(gomock.Any(), gomock.Any()).AnyTimes()
			for _, entity := range tc.entities {
				updateStore(entity.ContainerID, entity.DeploymentID, entity, mockStore)
			}
			for _, fn := range tc.events {
				fn(t, pipeline)
			}
		})
	}
}

func newEntity(containerID, deploymentID string) clusterentities.ContainerMetadata {
	return clusterentities.ContainerMetadata{
		DeploymentID: deploymentID,
		ContainerID:  containerID,
	}
}

func online(_ *testing.T, pipeline *Pipeline) {
	pipeline.Notify(common.SensorComponentEventCentralReachable)
}

func offline(_ *testing.T, pipeline *Pipeline) {
	pipeline.Notify(common.SensorComponentEventOfflineMode)
}

func signal(signal *storage.ProcessSignal, shouldBeDropped bool) func(*testing.T, *Pipeline) {
	return func(t *testing.T, pipeline *Pipeline) {
		previousLen := len(pipeline.indicators)
		pipeline.Process(signal)
		if shouldBeDropped {
			assert.Never(t, func() bool {
				return previousLen < len(pipeline.indicators)
			}, 500*time.Millisecond, 10*time.Millisecond, "the indicator should be dropped")
		} else {
			assert.Eventually(t, func() bool {
				return previousLen < len(pipeline.indicators)
			}, 500*time.Millisecond, 10*time.Millisecond, "timeout waiting for indicator")
		}
	}
}

func read(containerID, deploymentID string) func(*testing.T, *Pipeline) {
	return func(t *testing.T, pipeline *Pipeline) {
		select {
		case msg, ok := <-pipeline.indicators:
			if !ok {
				t.Error("The indicators channel should not be closed")
			}
			assert.False(t, msg.IsExpired())
			require.NotNil(t, msg.GetEvent().GetProcessIndicator())
			assert.Equal(t, deploymentID, msg.GetEvent().GetProcessIndicator().GetDeploymentId())
			assert.Equal(t, containerID, msg.GetEvent().GetProcessIndicator().GetSignal().GetContainerId())
		case <-time.After(500 * time.Millisecond):
			t.Fatal("Timeout waiting for indicator")
		}
	}
}

func assertSize(size int) func(*testing.T, *Pipeline) {
	return func(t *testing.T, pipeline *Pipeline) {
		assert.Len(t, pipeline.indicators, size)
	}
}

func TestProcessPipelineOfflineV1(t *testing.T) {
	t.Skip("Test skipped: feature is now always enabled with event buffering")
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

	sensorEvents := make(chan *message.ExpiringMessage, 10)
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
			mockStore := clusterentities.NewStore(0, nil, false)
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
			metadataWg.Add(1)

			p.Notify(tc.initialState)
			tc.initialSignal.signalProcessingRoutine(p, tc.initialSignal.signal, mockStore, containerMetadata1, metadataWg)
			// Wait for metadata to arrive - either directly or through the ticker
			metadataWg.Wait()

			events := collectEventsFor(caseCtx, actualEvents, 500*time.Millisecond)

			// At this point we should have only one message
			assert.Len(t, events, 1)
			assert.Equal(t, events[0].GetEvent().GetProcessIndicator().GetDeploymentId(), containerMetadata1.DeploymentID)
			tc.initialSignal.expectContextCancel(t, events[0].Context.Err())

			// Process second part of the test
			metadataWg.Add(1)
			p.Notify(tc.laterState)
			tc.laterSignal.signalProcessingRoutine(p, tc.laterSignal.signal, mockStore, containerMetadata2, metadataWg)

			// Wait for metadata to arrive - either directly or through the ticker
			metadataWg.Wait()

			// Events contains processed signals. They may arrive in any order
			events = collectEventsFor(caseCtx, actualEvents, 500*time.Millisecond)

			// At this point we should have only one message
			assert.Len(t, events, 1)
			assert.Equal(t, events[0].GetEvent().GetProcessIndicator().GetDeploymentId(), containerMetadata2.DeploymentID)
			tc.laterSignal.expectContextCancel(t, events[0].Context.Err())
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
	sensorEvents := make(chan *message.ExpiringMessage, 10)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockStore := clusterentities.NewStore(0, nil, false)
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
