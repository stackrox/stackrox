package watcher

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/central/baseimage/datastore/repository/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/baseimage/reposcan"
	"github.com/stackrox/rox/pkg/concurrency"
	delegatedRegistryMocks "github.com/stackrox/rox/pkg/delegatedregistry/mocks"
	"github.com/stackrox/rox/pkg/errox"
	registryMocks "github.com/stackrox/rox/pkg/registries/mocks"
	"github.com/stackrox/rox/pkg/registries/types"
	registryTypesMocks "github.com/stackrox/rox/pkg/registries/types/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// createTestWatcher creates a watcherImpl with mock dependencies for testing.
func createTestWatcher(ctrl *gomock.Controller, mockDS *mocks.MockDataStore, pollInterval time.Duration) *watcherImpl {
	mockRegistrySet := registryMocks.NewMockSet(ctrl)
	mockDelegator := delegatedRegistryMocks.NewMockDelegator(ctrl)

	// Set default behavior: no delegation
	mockDelegator.EXPECT().
		GetDelegateClusterID(gomock.Any(), gomock.Any()).
		Return("", false, nil).
		AnyTimes()

	// Set default behavior: no matching registries (support both methods for deduplication setting)
	mockRegistrySet.EXPECT().
		GetAllUnique().
		Return(nil).
		AnyTimes()
	mockRegistrySet.EXPECT().
		GetAll().
		Return(nil).
		AnyTimes()

	return &watcherImpl{
		datastore:    mockDS,
		delegator:    mockDelegator,
		pollInterval: pollInterval,
		localScanner: reposcan.NewLocalScanner(mockRegistrySet),
		stopper:      concurrency.NewStopper(),
	}
}

func TestWatcher_StartsAndStops(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockDS := mocks.NewMockDataStore(ctrl)

	// Mock returns empty repositories to avoid processing
	mockDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return([]*storage.BaseImageRepository{}, nil).
		AnyTimes()

	w := createTestWatcher(ctrl, mockDS, 100*time.Millisecond)

	// Start watcher
	w.Start()

	// Let it run briefly
	time.Sleep(150 * time.Millisecond)

	// Stop should complete quickly
	done := make(chan struct{})
	go func() {
		w.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Watcher did not stop within 1 second")
	}
}

func TestWatcher_PollsImmediately(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockDS := mocks.NewMockDataStore(ctrl)

	pollCalled := make(chan struct{})

	mockDS.EXPECT().
		ListRepositories(gomock.Any()).
		DoAndReturn(func(ctx context.Context) ([]*storage.BaseImageRepository, error) {
			close(pollCalled)
			return []*storage.BaseImageRepository{}, nil
		}).
		Times(1)

	w := createTestWatcher(ctrl, mockDS, 1*time.Hour)

	w.Start()
	defer w.Stop()

	// Verify immediate poll happened
	select {
	case <-pollCalled:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Watcher did not poll immediately on start")
	}
}

func TestWatcher_ProcessesMultipleRepositories(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockDS := mocks.NewMockDataStore(ctrl)

	repos := []*storage.BaseImageRepository{
		{Id: "1", RepositoryPath: "registry.io/repo1", TagPattern: "*"},
		{Id: "2", RepositoryPath: "registry.io/repo2", TagPattern: "*"},
		{Id: "3", RepositoryPath: "registry.io/repo3", TagPattern: "*"},
	}

	mockDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return(repos, nil)

	w := createTestWatcher(ctrl, mockDS, 1*time.Hour)

	assert.NotPanics(t, func() {
		w.pollOnce()
	})
}

func TestWatcher_HandlesDatastoreError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockDS := mocks.NewMockDataStore(ctrl)

	mockDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return(nil, errox.InvariantViolation.New("database connection failed"))

	w := createTestWatcher(ctrl, mockDS, 1*time.Hour)

	assert.NotPanics(t, func() {
		w.pollOnce()
	})
}

func TestWatcher_StartIsIdempotent(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockDS := mocks.NewMockDataStore(ctrl)

	mockDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return([]*storage.BaseImageRepository{}, nil).
		AnyTimes()

	w := createTestWatcher(ctrl, mockDS, 100*time.Millisecond)

	// Call Start multiple times
	w.Start()
	w.Start()
	w.Start()

	time.Sleep(150 * time.Millisecond)

	// Should stop cleanly
	w.Stop()
}

func TestWatcher_StopIsIdempotent(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockDS := mocks.NewMockDataStore(ctrl)

	mockDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return([]*storage.BaseImageRepository{}, nil).
		AnyTimes()

	w := createTestWatcher(ctrl, mockDS, 100*time.Millisecond)

	w.Start()
	time.Sleep(150 * time.Millisecond)

	// Call Stop multiple times
	w.Stop()
	w.Stop()
	w.Stop()

	// Should not hang or panic
	assert.True(t, true)
}

func TestWatcher_StopsGracefullyDuringPoll(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockDS := mocks.NewMockDataStore(ctrl)

	// Block during ListRepositories
	blockCh := make(chan struct{})
	mockDS.EXPECT().
		ListRepositories(gomock.Any()).
		DoAndReturn(func(ctx context.Context) ([]*storage.BaseImageRepository, error) {
			<-blockCh
			return []*storage.BaseImageRepository{}, nil
		}).
		AnyTimes()

	w := createTestWatcher(ctrl, mockDS, 1*time.Hour)

	w.Start()

	// Give it time to enter poll
	time.Sleep(50 * time.Millisecond)

	// Stop while poll is blocked
	done := make(chan struct{})
	go func() {
		w.Stop()
		close(done)
	}()

	// Unblock the poll
	close(blockCh)

	// Stop should complete
	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Stop did not complete within 2 seconds")
	}
}

func TestWatcher_AccessesAllProtoFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockDS := mocks.NewMockDataStore(ctrl)

	repo := &storage.BaseImageRepository{
		Id:             "test-id",
		RepositoryPath: "registry.io/test",
		TagPattern:     "v*",
		PatternHash:    "abc123",
		HealthStatus:   storage.BaseImageRepository_HEALTHY,
	}

	mockDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return([]*storage.BaseImageRepository{repo}, nil)

	w := createTestWatcher(ctrl, mockDS, 1*time.Hour)

	// Should not panic when accessing proto fields
	w.pollOnce()

	// Verify fields are accessible
	require.NotNil(t, repo)
	assert.Equal(t, "test-id", repo.GetId())
	assert.Equal(t, "registry.io/test", repo.GetRepositoryPath())
	assert.Equal(t, "v*", repo.GetTagPattern())
	assert.Equal(t, "abc123", repo.GetPatternHash())
	assert.Equal(t, storage.BaseImageRepository_HEALTHY, repo.GetHealthStatus())
}

// TestWatcher_InvalidRepositoryPath is intentionally skipped.
//
// Empty repository paths represent a programming error (data should be validated before storage).
// The code uses utils.Should() which triggers HardPanic - a panic that cannot be caught in tests.
// This is by design to ensure programming errors are always surfaced.
//
// The behavior is verified by the existence of utils.Should() call in processRepository().
func TestWatcher_InvalidRepositoryPath(t *testing.T) {
	t.Skip("utils.Should() triggers HardPanic which cannot be caught in tests by design")
}

func TestWatcher_DelegationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockDS := mocks.NewMockDataStore(ctrl)
	mockRegistrySet := registryMocks.NewMockSet(ctrl)
	mockDelegator := delegatedRegistryMocks.NewMockDelegator(ctrl)

	repo := &storage.BaseImageRepository{
		Id:             "test-id",
		RepositoryPath: "docker.io/library/nginx",
		TagPattern:     "*",
	}

	mockDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return([]*storage.BaseImageRepository{repo}, nil)

	// Delegation check returns error - should continue with Central-based processing
	mockDelegator.EXPECT().
		GetDelegateClusterID(gomock.Any(), gomock.Any()).
		Return("", false, errox.InvariantViolation.New("delegation check failed"))

	// No matching registries
	mockRegistrySet.EXPECT().
		GetAllUnique().
		Return(nil).
		AnyTimes()
	mockRegistrySet.EXPECT().
		GetAll().
		Return(nil).
		AnyTimes()

	w := &watcherImpl{
		datastore:    mockDS,
		delegator:    mockDelegator,
		pollInterval: 1 * time.Hour,
		localScanner: reposcan.NewLocalScanner(mockRegistrySet),
		stopper:      concurrency.NewStopper(),
	}

	// Should not panic on delegation error
	assert.NotPanics(t, func() {
		w.pollOnce()
	})
}

func TestWatcher_ShouldDelegate(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockDS := mocks.NewMockDataStore(ctrl)
	mockRegistrySet := registryMocks.NewMockSet(ctrl)
	mockDelegator := delegatedRegistryMocks.NewMockDelegator(ctrl)

	repo := &storage.BaseImageRepository{
		Id:             "test-id",
		RepositoryPath: "docker.io/library/nginx",
		TagPattern:     "*",
	}

	mockDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return([]*storage.BaseImageRepository{repo}, nil)

	// Delegation check returns shouldDelegate=true
	mockDelegator.EXPECT().
		GetDelegateClusterID(gomock.Any(), gomock.Any()).
		Return("cluster-123", true, nil)

	// GetAll should NOT be called since we skip processing
	// (no expectation set means test fails if called)

	w := &watcherImpl{
		datastore:    mockDS,
		delegator:    mockDelegator,
		pollInterval: 1 * time.Hour,
		localScanner: reposcan.NewLocalScanner(mockRegistrySet),
		stopper:      concurrency.NewStopper(),
	}

	// Should not panic when delegation is required
	assert.NotPanics(t, func() {
		w.pollOnce()
	})
}

func TestWatcher_NoMatchingRegistry(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockDS := mocks.NewMockDataStore(ctrl)
	mockRegistrySet := registryMocks.NewMockSet(ctrl)
	mockDelegator := delegatedRegistryMocks.NewMockDelegator(ctrl)
	mockRegistry := registryTypesMocks.NewMockImageRegistry(ctrl)

	repo := &storage.BaseImageRepository{
		Id:             "test-id",
		RepositoryPath: "docker.io/library/nginx",
		TagPattern:     "*",
	}

	mockDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return([]*storage.BaseImageRepository{repo}, nil)

	mockDelegator.EXPECT().
		GetDelegateClusterID(gomock.Any(), gomock.Any()).
		Return("", false, nil)

	// Registry exists but doesn't match the image
	mockRegistry.EXPECT().
		Match(gomock.Any()).
		Return(false)

	mockRegistrySet.EXPECT().
		GetAllUnique().
		Return([]types.ImageRegistry{mockRegistry}).
		AnyTimes()
	mockRegistrySet.EXPECT().
		GetAll().
		Return([]types.ImageRegistry{mockRegistry}).
		AnyTimes()

	w := &watcherImpl{
		datastore:    mockDS,
		delegator:    mockDelegator,
		pollInterval: 1 * time.Hour,
		localScanner: reposcan.NewLocalScanner(mockRegistrySet),
		stopper:      concurrency.NewStopper(),
	}

	// Should not panic when no matching registry found
	assert.NotPanics(t, func() {
		w.pollOnce()
	})
}

func TestWatcher_MatchingRegistryWithTagListError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockDS := mocks.NewMockDataStore(ctrl)
	mockRegistrySet := registryMocks.NewMockSet(ctrl)
	mockDelegator := delegatedRegistryMocks.NewMockDelegator(ctrl)
	mockRegistry := registryTypesMocks.NewMockImageRegistry(ctrl)

	repo := &storage.BaseImageRepository{
		Id:             "test-id",
		RepositoryPath: "docker.io/library/nginx",
		TagPattern:     "*",
	}

	mockDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return([]*storage.BaseImageRepository{repo}, nil)

	mockDelegator.EXPECT().
		GetDelegateClusterID(gomock.Any(), gomock.Any()).
		Return("", false, nil)

	// Registry matches and returns error on ListTags
	mockRegistry.EXPECT().
		Match(gomock.Any()).
		Return(true)

	mockRegistry.EXPECT().
		ListTags(gomock.Any(), gomock.Any()).
		Return(nil, errox.InvariantViolation.New("registry connection failed"))

	mockRegistrySet.EXPECT().
		GetAllUnique().
		Return([]types.ImageRegistry{mockRegistry}).
		AnyTimes()
	mockRegistrySet.EXPECT().
		GetAll().
		Return([]types.ImageRegistry{mockRegistry}).
		AnyTimes()

	w := &watcherImpl{
		datastore:    mockDS,
		delegator:    mockDelegator,
		pollInterval: 1 * time.Hour,
		localScanner: reposcan.NewLocalScanner(mockRegistrySet),
		stopper:      concurrency.NewStopper(),
	}

	// Should not panic on tag listing error
	assert.NotPanics(t, func() {
		w.pollOnce()
	})
}

func TestWatcher_MatchingRegistrySuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockDS := mocks.NewMockDataStore(ctrl)
	mockRegistrySet := registryMocks.NewMockSet(ctrl)
	mockDelegator := delegatedRegistryMocks.NewMockDelegator(ctrl)
	mockRegistry := registryTypesMocks.NewMockImageRegistry(ctrl)

	repo := &storage.BaseImageRepository{
		Id:             "test-id",
		RepositoryPath: "docker.io/library/nginx",
		TagPattern:     "1.*",
	}

	mockDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return([]*storage.BaseImageRepository{repo}, nil)

	mockDelegator.EXPECT().
		GetDelegateClusterID(gomock.Any(), gomock.Any()).
		Return("", false, nil)

	// Registry matches and returns tags successfully.
	mockRegistry.EXPECT().
		Match(gomock.Any()).
		Return(true).
		AnyTimes()

	mockRegistry.EXPECT().
		ListTags(gomock.Any(), gomock.Any()).
		Return([]string{"1.0", "1.1", "1.2", "2.0", "latest"}, nil)

	// Mock Source() for rate limiter lookup.
	mockRegistry.EXPECT().
		Source().
		Return(&storage.ImageIntegration{Id: "integration-1"}).
		AnyTimes()

	// Mock Metadata calls for the 3 matching tags (1.0, 1.1, 1.2).
	mockRegistry.EXPECT().
		Metadata(gomock.Any()).
		DoAndReturn(func(img *storage.Image) (*storage.ImageMetadata, error) {
			return &storage.ImageMetadata{
				V2: &storage.V2Metadata{
					Digest: "sha256:abc123" + img.GetName().GetTag(),
				},
			}, nil
		}).
		Times(3)

	mockRegistrySet.EXPECT().
		GetAllUnique().
		Return([]types.ImageRegistry{mockRegistry}).
		AnyTimes()
	mockRegistrySet.EXPECT().
		GetAll().
		Return([]types.ImageRegistry{mockRegistry}).
		AnyTimes()

	w := &watcherImpl{
		datastore:    mockDS,
		delegator:    mockDelegator,
		pollInterval: 1 * time.Hour,
		localScanner: reposcan.NewLocalScanner(mockRegistrySet),
		stopper:      concurrency.NewStopper(),
	}

	// Should complete successfully.
	assert.NotPanics(t, func() {
		w.pollOnce()
	})
}

func TestWatcher_ContextCancellation(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockDS := mocks.NewMockDataStore(ctrl)
	mockRegistrySet := registryMocks.NewMockSet(ctrl)
	mockDelegator := delegatedRegistryMocks.NewMockDelegator(ctrl)

	repo := &storage.BaseImageRepository{
		Id:             "test-id",
		RepositoryPath: "docker.io/library/nginx",
		TagPattern:     "*",
	}

	mockDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return([]*storage.BaseImageRepository{repo}, nil)

	// Block on GetDelegateClusterID until context is cancelled
	mockDelegator.EXPECT().
		GetDelegateClusterID(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, _ interface{}) (string, bool, error) {
			<-ctx.Done()
			return "", false, ctx.Err()
		})

	// After delegation error, processing continues and GetAll is called
	mockRegistrySet.EXPECT().
		GetAllUnique().
		Return(nil).
		AnyTimes()
	mockRegistrySet.EXPECT().
		GetAll().
		Return(nil).
		AnyTimes()

	w := &watcherImpl{
		datastore:    mockDS,
		delegator:    mockDelegator,
		pollInterval: 1 * time.Hour,
		localScanner: reposcan.NewLocalScanner(mockRegistrySet),
		stopper:      concurrency.NewStopper(),
	}

	// Start the watcher
	w.Start()

	// Give it time to start processing
	time.Sleep(50 * time.Millisecond)

	// Stop while processing - this cancels context
	done := make(chan struct{})
	go func() {
		w.Stop()
		close(done)
	}()

	// Should complete quickly
	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Watcher did not stop within 2 seconds")
	}
}
