package pipeline

import (
	"context"
	"testing"
	"time"

	sensorAPI "github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/detector/mocks"
	"github.com/stackrox/rox/sensor/common/processsignal"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubDispatcher "github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	testContainerID  = "test-container-1"
	testDeploymentID = "test-deployment-1"
	testSignalID     = "test-signal-1"
)

func newTestDispatcher(t *testing.T) common.PubSubDispatcher {
	t.Helper()
	d, err := pubsubDispatcher.NewDispatcher(pubsubDispatcher.WithLaneConfigs([]pubsub.LaneConfig{
		lane.NewBlockingLane(pubsub.EnrichedProcessIndicatorLane),
		lane.NewBlockingLane(pubsub.UnenrichedProcessIndicatorLane),
	}))
	require.NoError(t, err)
	t.Cleanup(func() { d.Stop() })
	return d
}

func newTestFileActivity(containerID, signalID, path string) *sensorAPI.FileActivity {
	return &sensorAPI.FileActivity{
		Hostname: "test-host",
		Process: &sensorAPI.ProcessSignal{
			Id:          signalID,
			ContainerId: containerID,
			Name:        "test-process",
		},
		File: &sensorAPI.FileActivity_Open{
			Open: &sensorAPI.FileOpen{
				Activity: &sensorAPI.FileActivityBase{
					Path:     path,
					HostPath: "/host" + path,
				},
			},
		},
	}
}

func TestFileSystemPipelinePubSubBufferingAndDrain(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

	mockCtrl := gomock.NewController(t)
	mockDetector := mocks.NewMockDetector(mockCtrl)
	clusterStore := clusterentities.NewStore(0, nil, false)
	dispatcher := newTestDispatcher(t)

	activityChan := make(chan *sensorAPI.FileActivity, 10)
	p := NewFileSystemPipeline(mockDetector, clusterStore, activityChan, dispatcher)
	t.Cleanup(func() { p.Stop() })

	// Send a file activity — in pub/sub mode with no container metadata,
	// it should be buffered and an unenriched event published.
	fa := newTestFileActivity(testContainerID, testSignalID, "/etc/passwd")

	// The enricher subscriber will receive the unenriched event,
	// but we simulate the enrichment by directly publishing an enriched event.
	// First, send the file activity through the pipeline.
	activityChan <- fa

	// Give the pipeline time to process and buffer the activity.
	time.Sleep(100 * time.Millisecond)

	// Verify the activity was buffered.
	p.activityMutex.Lock()
	key := cacheKey(testContainerID, testSignalID)
	entry := p.bufferedActivity[key]
	require.NotNil(t, entry, "file activity should be buffered")
	assert.Len(t, entry.activities, 1)
	p.activityMutex.Unlock()

	// Now simulate the enriched process indicator arriving via pub/sub.
	enrichedIndicator := &storage.ProcessIndicator{
		Id:           "enriched-indicator-1",
		DeploymentId: testDeploymentID,
		Signal: &storage.ProcessSignal{
			Id:          testSignalID,
			ContainerId: testContainerID,
			Name:        "test-process",
		},
	}

	// Expect the detector to receive the file access with the enriched indicator.
	fileAccessReceived := make(chan *storage.FileAccess, 1)
	mockDetector.EXPECT().ProcessFileAccess(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, access *storage.FileAccess) {
			fileAccessReceived <- access
		},
	)

	enrichedEvent := processsignal.NewEnrichedProcessIndicatorEvent(context.Background(), enrichedIndicator)
	require.NoError(t, dispatcher.Publish(enrichedEvent))

	select {
	case access := <-fileAccessReceived:
		assert.Equal(t, testDeploymentID, access.GetProcess().GetDeploymentId())
		assert.Equal(t, "/etc/passwd", access.GetFile().GetEffectivePath())
		assert.Equal(t, storage.FileAccess_OPEN, access.GetOperation())
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for file access to be processed")
	}

	// Verify the buffer was drained.
	p.activityMutex.Lock()
	assert.Empty(t, p.bufferedActivity)
	assert.Equal(t, 0, p.totalBufferedActivity)
	p.activityMutex.Unlock()
}

func TestFileSystemPipelinePubSubMultipleActivitiesSameProcess(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

	mockCtrl := gomock.NewController(t)
	mockDetector := mocks.NewMockDetector(mockCtrl)
	clusterStore := clusterentities.NewStore(0, nil, false)
	dispatcher := newTestDispatcher(t)

	activityChan := make(chan *sensorAPI.FileActivity, 10)
	p := NewFileSystemPipeline(mockDetector, clusterStore, activityChan, dispatcher)
	t.Cleanup(func() { p.Stop() })

	// Send multiple file activities for the same process.
	paths := []string{"/etc/passwd", "/etc/shadow", "/etc/hosts"}
	for _, path := range paths {
		activityChan <- newTestFileActivity(testContainerID, testSignalID, path)
	}

	time.Sleep(100 * time.Millisecond)

	// Verify all activities are buffered under the same key.
	p.activityMutex.Lock()
	key := cacheKey(testContainerID, testSignalID)
	entry := p.bufferedActivity[key]
	require.NotNil(t, entry)
	assert.Len(t, entry.activities, 3)
	p.activityMutex.Unlock()

	// All three should be dispatched when the enriched indicator arrives.
	var received []*storage.FileAccess
	done := make(chan struct{})
	mockDetector.EXPECT().ProcessFileAccess(gomock.Any(), gomock.Any()).Times(3).DoAndReturn(
		func(_ context.Context, access *storage.FileAccess) {
			received = append(received, access)
			if len(received) == 3 {
				close(done)
			}
		},
	)

	enrichedIndicator := &storage.ProcessIndicator{
		Id:           "enriched-indicator-1",
		DeploymentId: testDeploymentID,
		Signal: &storage.ProcessSignal{
			Id:          testSignalID,
			ContainerId: testContainerID,
		},
	}
	require.NoError(t, dispatcher.Publish(processsignal.NewEnrichedProcessIndicatorEvent(context.Background(), enrichedIndicator)))

	select {
	case <-done:
		expectedPaths := map[string]bool{"/etc/passwd": true, "/etc/shadow": true, "/etc/hosts": true}
		for _, access := range received {
			assert.True(t, expectedPaths[access.GetFile().GetEffectivePath()],
				"unexpected path: %s", access.GetFile().GetEffectivePath())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for all file accesses to be processed")
	}
}

func TestFileSystemPipelineHostProcessBypassesPubSub(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

	mockCtrl := gomock.NewController(t)
	mockDetector := mocks.NewMockDetector(mockCtrl)
	clusterStore := clusterentities.NewStore(0, nil, false)
	dispatcher := newTestDispatcher(t)

	activityChan := make(chan *sensorAPI.FileActivity, 10)
	p := NewFileSystemPipeline(mockDetector, clusterStore, activityChan, dispatcher)
	t.Cleanup(func() { p.Stop() })

	// Host process (empty container ID) should bypass pub/sub and process directly.
	fa := newTestFileActivity("", testSignalID, "/etc/passwd")

	fileAccessReceived := make(chan struct{})
	mockDetector.EXPECT().ProcessFileAccess(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, access *storage.FileAccess) {
			assert.Equal(t, "/etc/passwd", access.GetFile().GetEffectivePath())
			close(fileAccessReceived)
		},
	)

	activityChan <- fa

	select {
	case <-fileAccessReceived:
		// Success — host process was handled directly.
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for host process file access")
	}

	// Nothing should be buffered.
	p.activityMutex.Lock()
	assert.Empty(t, p.bufferedActivity)
	p.activityMutex.Unlock()
}

func TestFileSystemPipelineBufferExpiration(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

	mockCtrl := gomock.NewController(t)
	mockDetector := mocks.NewMockDetector(mockCtrl)
	clusterStore := clusterentities.NewStore(0, nil, false)
	dispatcher := newTestDispatcher(t)

	activityChan := make(chan *sensorAPI.FileActivity, 10)
	p := NewFileSystemPipeline(mockDetector, clusterStore, activityChan, dispatcher)
	t.Cleanup(func() { p.Stop() })

	// Send a file activity to get it buffered.
	activityChan <- newTestFileActivity(testContainerID, testSignalID, "/etc/passwd")
	time.Sleep(100 * time.Millisecond)

	// Manually backdate the buffer entry to simulate expiration.
	p.activityMutex.Lock()
	key := cacheKey(testContainerID, testSignalID)
	entry := p.bufferedActivity[key]
	require.NotNil(t, entry)
	entry.timestamp = time.Now().Add(-bufferedActivityTTL - time.Second)
	p.activityMutex.Unlock()

	// Trigger pruning manually.
	p.pruneExpiredBuffers()

	// Verify the buffer was cleaned up.
	p.activityMutex.Lock()
	assert.Empty(t, p.bufferedActivity)
	assert.Equal(t, 0, p.totalBufferedActivity)
	p.activityMutex.Unlock()
}

func TestFileSystemPipelineStopWaitsForAllGoroutines(t *testing.T) {
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

	mockCtrl := gomock.NewController(t)
	mockDetector := mocks.NewMockDetector(mockCtrl)
	clusterStore := clusterentities.NewStore(0, nil, false)
	dispatcher := newTestDispatcher(t)

	activityChan := make(chan *sensorAPI.FileActivity, 10)
	p := NewFileSystemPipeline(mockDetector, clusterStore, activityChan, dispatcher)

	// Stop should return without hanging, proving both goroutines exit.
	done := make(chan struct{})
	go func() {
		p.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Success.
	case <-time.After(5 * time.Second):
		t.Fatal("Stop() did not return in time — goroutine leak suspected")
	}
}
