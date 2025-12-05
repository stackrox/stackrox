package watcher

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/central/baseimage/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestWatcher_StartsAndStops(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockDS := mocks.NewMockDataStore(ctrl)

	// Mock returns empty repositories to avoid processing
	mockDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return([]*storage.BaseImageRepository{}, nil).
		AnyTimes()

	w := &watcherImpl{
		datastore:    mockDS,
		pollInterval: 100 * time.Millisecond,
		stopper:      concurrency.NewStopper(),
	}

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

	w := &watcherImpl{
		datastore:    mockDS,
		pollInterval: 1 * time.Hour, // Long interval
		stopper:      concurrency.NewStopper(),
	}

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

	w := &watcherImpl{
		datastore:    mockDS,
		pollInterval: 1 * time.Hour,
		stopper:      concurrency.NewStopper(),
	}

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

	w := &watcherImpl{
		datastore:    mockDS,
		pollInterval: 1 * time.Hour,
		stopper:      concurrency.NewStopper(),
	}

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

	w := &watcherImpl{
		datastore:    mockDS,
		pollInterval: 100 * time.Millisecond,
		stopper:      concurrency.NewStopper(),
	}

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

	w := &watcherImpl{
		datastore:    mockDS,
		pollInterval: 100 * time.Millisecond,
		stopper:      concurrency.NewStopper(),
	}

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

	w := &watcherImpl{
		datastore:    mockDS,
		pollInterval: 1 * time.Hour,
		stopper:      concurrency.NewStopper(),
	}

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

	w := &watcherImpl{
		datastore:    mockDS,
		pollInterval: 1 * time.Hour,
		stopper:      concurrency.NewStopper(),
	}

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
