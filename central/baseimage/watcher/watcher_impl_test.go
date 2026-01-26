package watcher

import (
	"context"
	"errors"
	"testing"
	"time"

	baseImageDSMocks "github.com/stackrox/rox/central/baseimage/datastore/mocks"
	repoDSMocks "github.com/stackrox/rox/central/baseimage/datastore/repository/mocks"
	tagDSMocks "github.com/stackrox/rox/central/baseimage/datastore/tag/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/baseimage/reposcan"
	"github.com/stackrox/rox/pkg/baseimage/tagfetcher"
	delegatedRegistryMocks "github.com/stackrox/rox/pkg/delegatedregistry/mocks"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protocompat"
	registryMocks "github.com/stackrox/rox/pkg/registries/mocks"
	"github.com/stackrox/rox/pkg/registries/types"
	registryTypesMocks "github.com/stackrox/rox/pkg/registries/types/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// createTestWatcher creates a watcherImpl with mock dependencies for testing.
// Pass nil for mocks that should use default (no-op) implementations.
func createTestWatcher(
	ctrl *gomock.Controller,
	mockRepoDS *repoDSMocks.MockDataStore,
	mockTagDS *tagDSMocks.MockDataStore,
	mockRegistrySet *registryMocks.MockSet,
	mockDelegator *delegatedRegistryMocks.MockDelegator,
	poll time.Duration,
	delegationEnabled bool,
) Watcher {
	if mockTagDS == nil {
		mockTagDS = tagDSMocks.NewMockDataStore(ctrl)
	}
	if mockRegistrySet == nil {
		mockRegistrySet = registryMocks.NewMockSet(ctrl)
	}
	if mockDelegator == nil {
		mockDelegator = delegatedRegistryMocks.NewMockDelegator(ctrl)
	}

	// Create default baseImageDS mock.
	mockBaseImageDS := baseImageDSMocks.NewMockDataStore(ctrl)
	mockBaseImageDS.EXPECT().ReplaceByRepository(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	// Allow tag datastore calls.
	mockTagDS.EXPECT().ListTagsByRepository(gomock.Any(), gomock.Any()).Return([]*storage.BaseImageTag{}, nil).AnyTimes()
	mockTagDS.EXPECT().UpsertMany(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	return New(mockRepoDS, mockTagDS, mockBaseImageDS, mockRegistrySet, mockDelegator, poll, 10, 0, delegationEnabled)
}

func TestWatcher_StartsAndStops(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepoDS := repoDSMocks.NewMockDataStore(ctrl)

	// Polls immediately on start + 1 timed poll after 100ms (with 150ms sleep)
	mockRepoDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return([]*storage.BaseImageRepository{}, nil).
		Times(2)

	w := createTestWatcher(ctrl, mockRepoDS, nil, nil, nil, 100*time.Millisecond, false)

	// Start watcher
	w.Start()

	// Let it run briefly (should trigger 2 polls: immediate + 1 timed)
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
	mockRepoDS := repoDSMocks.NewMockDataStore(ctrl)

	pollCalled := make(chan struct{})

	mockRepoDS.EXPECT().
		ListRepositories(gomock.Any()).
		DoAndReturn(func(ctx context.Context) ([]*storage.BaseImageRepository, error) {
			close(pollCalled)
			return []*storage.BaseImageRepository{}, nil
		}).
		Times(1)

	w := createTestWatcher(ctrl, mockRepoDS, nil, nil, nil, 1*time.Hour, false)

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
	mockRepoDS := repoDSMocks.NewMockDataStore(ctrl)
	mockTagDS := tagDSMocks.NewMockDataStore(ctrl)
	mockRegistrySet := registryMocks.NewMockSet(ctrl)
	mockDelegator := delegatedRegistryMocks.NewMockDelegator(ctrl)

	repos := []*storage.BaseImageRepository{
		{Id: "00000000-0000-0000-0000-000000000001", RepositoryPath: "registry.io/repo1", TagPattern: "*"},
		{Id: "00000000-0000-0000-0000-000000000002", RepositoryPath: "registry.io/repo2", TagPattern: "*"},
		{Id: "00000000-0000-0000-0000-000000000003", RepositoryPath: "registry.io/repo3", TagPattern: "*"},
	}

	mockRepoDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return(repos, nil).
		Times(1)

	// Each repository will be processed: 3 delegation checks
	mockDelegator.EXPECT().
		GetDelegateClusterID(gomock.Any(), gomock.Any()).
		Return("", false, nil).
		Times(3)

	// Each repo lists tags from cache: 3 ListTagsByRepository calls
	mockTagDS.EXPECT().
		ListTagsByRepository(gomock.Any(), gomock.Any()).
		Return([]*storage.BaseImageTag{}, nil).
		Times(3)

	// Each repo tries to find registry: 3 GetAllUnique calls
	mockRegistrySet.EXPECT().
		GetAllUnique().
		Return(nil).
		Times(3)

	// No tags stored (no matching registry), so no UpsertMany/DeleteMany calls expected

	w := createTestWatcher(ctrl, mockRepoDS, mockTagDS, mockRegistrySet, mockDelegator, 1*time.Hour, true)

	assert.NotPanics(t, func() {
		w.(*watcherImpl).pollOnce()
	})
}

func TestWatcher_HandlesDatastoreError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepoDS := repoDSMocks.NewMockDataStore(ctrl)

	mockRepoDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return(nil, errox.InvariantViolation.New("database connection failed")).
		Times(1)

	w := createTestWatcher(ctrl, mockRepoDS, nil, nil, nil, 1*time.Hour, false)

	assert.NotPanics(t, func() {
		w.(*watcherImpl).pollOnce()
	})
}

func TestWatcher_StartIsIdempotent(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepoDS := repoDSMocks.NewMockDataStore(ctrl)

	// Polls immediately + 1 timed poll after 100ms (with 150ms sleep)
	mockRepoDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return([]*storage.BaseImageRepository{}, nil).
		Times(2)

	w := createTestWatcher(ctrl, mockRepoDS, nil, nil, nil, 100*time.Millisecond, false)

	// Call Start multiple times (only first should take effect)
	w.Start()
	w.Start()
	w.Start()

	time.Sleep(150 * time.Millisecond)

	// Should stop cleanly
	w.Stop()
}

func TestWatcher_StopIsIdempotent(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepoDS := repoDSMocks.NewMockDataStore(ctrl)

	// Polls immediately + 1 timed poll after 100ms (with 150ms sleep)
	mockRepoDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return([]*storage.BaseImageRepository{}, nil).
		Times(2)

	w := createTestWatcher(ctrl, mockRepoDS, nil, nil, nil, 100*time.Millisecond, false)

	w.Start()
	time.Sleep(150 * time.Millisecond)

	// Call Stop multiple times (only first should take effect)
	w.Stop()
	w.Stop()
	w.Stop()

	// Should not hang or panic
	assert.True(t, true)
}

func TestWatcher_StopsGracefullyDuringPoll(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepoDS := repoDSMocks.NewMockDataStore(ctrl)

	// Block during ListRepositories
	blockCh := make(chan struct{})
	mockRepoDS.EXPECT().
		ListRepositories(gomock.Any()).
		DoAndReturn(func(ctx context.Context) ([]*storage.BaseImageRepository, error) {
			<-blockCh
			return []*storage.BaseImageRepository{}, nil
		}).
		Times(1)

	w := createTestWatcher(ctrl, mockRepoDS, nil, nil, nil, 1*time.Hour, false)

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
	mockRepoDS := repoDSMocks.NewMockDataStore(ctrl)
	mockTagDS := tagDSMocks.NewMockDataStore(ctrl)
	mockRegistrySet := registryMocks.NewMockSet(ctrl)
	mockDelegator := delegatedRegistryMocks.NewMockDelegator(ctrl)

	repo := &storage.BaseImageRepository{
		Id:             "00000000-0000-0000-0000-0000000000ff",
		RepositoryPath: "registry.io/test",
		TagPattern:     "v*",
		PatternHash:    "abc123",
		HealthStatus:   storage.BaseImageRepository_HEALTHY,
	}

	mockRepoDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return([]*storage.BaseImageRepository{repo}, nil).
		Times(1)

	// One delegation check for the repository
	mockDelegator.EXPECT().
		GetDelegateClusterID(gomock.Any(), gomock.Any()).
		Return("", false, nil).
		Times(1)

	// Fetch existing tags from cache
	mockTagDS.EXPECT().
		ListTagsByRepository(gomock.Any(), gomock.Any()).
		Return([]*storage.BaseImageTag{}, nil).
		Times(1)

	// Scanner calls GetAllUnique
	mockRegistrySet.EXPECT().
		GetAllUnique().
		Return(nil).
		Times(1)

	// No tags stored (no matching registry), so no UpsertMany/DeleteMany calls expected

	w := createTestWatcher(ctrl, mockRepoDS, mockTagDS, mockRegistrySet, mockDelegator, 1*time.Hour, true)

	// Should not panic when accessing proto fields
	w.(*watcherImpl).pollOnce()

	// Verify fields are accessible
	require.NotNil(t, repo)
	assert.Equal(t, "00000000-0000-0000-0000-0000000000ff", repo.GetId())
	assert.Equal(t, "registry.io/test", repo.GetRepositoryPath())
	assert.Equal(t, "v*", repo.GetTagPattern())
	assert.Equal(t, "abc123", repo.GetPatternHash())
	assert.Equal(t, storage.BaseImageRepository_HEALTHY, repo.GetHealthStatus())
}

func TestWatcher_DelegationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepoDS := repoDSMocks.NewMockDataStore(ctrl)
	mockTagDS := tagDSMocks.NewMockDataStore(ctrl)
	mockRegistrySet := registryMocks.NewMockSet(ctrl)
	mockDelegator := delegatedRegistryMocks.NewMockDelegator(ctrl)

	repo := &storage.BaseImageRepository{
		Id:             "00000000-0000-0000-0000-0000000000ff",
		RepositoryPath: "docker.io/library/nginx",
		TagPattern:     "*",
	}

	mockRepoDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return([]*storage.BaseImageRepository{repo}, nil).
		Times(1)

	// Delegation check returns error - should continue with Central-based processing
	mockDelegator.EXPECT().
		GetDelegateClusterID(gomock.Any(), gomock.Any()).
		Return("", false, errox.InvariantViolation.New("delegation check failed")).
		Times(1)

	// Fetch existing tags from cache
	mockTagDS.EXPECT().
		ListTagsByRepository(gomock.Any(), gomock.Any()).
		Return([]*storage.BaseImageTag{}, nil).
		Times(1)

	// No matching registries - scanner calls GetAllUnique
	mockRegistrySet.EXPECT().
		GetAllUnique().
		Return(nil).
		Times(1)

	// No tags stored (no matching registry), so no UpsertMany/DeleteMany calls expected

	w := createTestWatcher(ctrl, mockRepoDS, mockTagDS, mockRegistrySet, mockDelegator, 1*time.Hour, true)

	// Should not panic on delegation error
	assert.NotPanics(t, func() {
		w.(*watcherImpl).pollOnce()
	})
}

func TestWatcher_ShouldDelegate(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepoDS := repoDSMocks.NewMockDataStore(ctrl)
	mockTagDS := tagDSMocks.NewMockDataStore(ctrl)
	mockDelegator := delegatedRegistryMocks.NewMockDelegator(ctrl)

	repo := &storage.BaseImageRepository{
		Id:             "00000000-0000-0000-0000-0000000000ff",
		RepositoryPath: "docker.io/library/nginx",
		TagPattern:     "*",
	}

	mockRepoDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return([]*storage.BaseImageRepository{repo}, nil).
		Times(1)

	// Fetch existing tags from cache
	mockTagDS.EXPECT().
		ListTagsByRepository(gomock.Any(), gomock.Any()).
		Return([]*storage.BaseImageTag{}, nil).
		Times(1)

	// Delegation check returns shouldDelegate=true
	mockDelegator.EXPECT().
		GetDelegateClusterID(gomock.Any(), gomock.Any()).
		Return("cluster-123", true, nil).
		Times(1)

	// Delegated scanner returns error (not implemented)
	// No tags stored (delegated scanner not implemented), so no UpsertMany/DeleteMany calls expected

	w := createTestWatcher(ctrl, mockRepoDS, mockTagDS, nil, mockDelegator, 1*time.Hour, true)

	// Should not panic when delegation is required
	assert.NotPanics(t, func() {
		w.(*watcherImpl).pollOnce()
	})
}

func TestWatcher_NoMatchingRegistry(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepoDS := repoDSMocks.NewMockDataStore(ctrl)
	mockTagDS := tagDSMocks.NewMockDataStore(ctrl)
	mockRegistrySet := registryMocks.NewMockSet(ctrl)
	mockDelegator := delegatedRegistryMocks.NewMockDelegator(ctrl)
	mockRegistry := registryTypesMocks.NewMockImageRegistry(ctrl)

	repo := &storage.BaseImageRepository{
		Id:             "00000000-0000-0000-0000-0000000000ff",
		RepositoryPath: "docker.io/library/nginx",
		TagPattern:     "*",
	}

	mockRepoDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return([]*storage.BaseImageRepository{repo}, nil).
		Times(1)

	mockDelegator.EXPECT().
		GetDelegateClusterID(gomock.Any(), gomock.Any()).
		Return("", false, nil).
		Times(1)

	// Fetch existing tags from cache
	mockTagDS.EXPECT().
		ListTagsByRepository(gomock.Any(), gomock.Any()).
		Return([]*storage.BaseImageTag{}, nil).
		Times(1)

	// Registry exists but doesn't match the image
	mockRegistry.EXPECT().
		Match(gomock.Any()).
		Return(false).
		Times(1)

	// Scanner calls GetAllUnique
	mockRegistrySet.EXPECT().
		GetAllUnique().
		Return([]types.ImageRegistry{mockRegistry}).
		Times(1)

	// No tags stored (registry doesn't match), so no UpsertMany/DeleteMany calls expected

	w := createTestWatcher(ctrl, mockRepoDS, mockTagDS, mockRegistrySet, mockDelegator, 1*time.Hour, true)

	// Should not panic when no matching registry found
	assert.NotPanics(t, func() {
		w.(*watcherImpl).pollOnce()
	})
}

func TestWatcher_MatchingRegistryWithTagListError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepoDS := repoDSMocks.NewMockDataStore(ctrl)
	mockTagDS := tagDSMocks.NewMockDataStore(ctrl)
	mockRegistrySet := registryMocks.NewMockSet(ctrl)
	mockDelegator := delegatedRegistryMocks.NewMockDelegator(ctrl)
	mockRegistry := registryTypesMocks.NewMockImageRegistry(ctrl)

	repo := &storage.BaseImageRepository{
		Id:             "00000000-0000-0000-0000-0000000000ff",
		RepositoryPath: "docker.io/library/nginx",
		TagPattern:     "*",
	}

	mockRepoDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return([]*storage.BaseImageRepository{repo}, nil).
		Times(1)

	mockDelegator.EXPECT().
		GetDelegateClusterID(gomock.Any(), gomock.Any()).
		Return("", false, nil).
		Times(1)

	// Fetch existing tags from cache
	mockTagDS.EXPECT().
		ListTagsByRepository(gomock.Any(), gomock.Any()).
		Return([]*storage.BaseImageTag{}, nil).
		Times(1)

	// Registry matches and returns error on ListTags
	mockRegistry.EXPECT().
		Match(gomock.Any()).
		Return(true).
		Times(1)

	mockRegistry.EXPECT().
		ListTags(gomock.Any(), gomock.Any()).
		Return(nil, errox.InvariantViolation.New("registry connection failed")).
		Times(1)

	mockRegistrySet.EXPECT().
		GetAllUnique().
		Return([]types.ImageRegistry{mockRegistry}).
		Times(1)

	// No tags stored (ListTags failed), so no UpsertMany/DeleteMany calls expected

	w := createTestWatcher(ctrl, mockRepoDS, mockTagDS, mockRegistrySet, mockDelegator, 1*time.Hour, true)

	// Should not panic on tag listing error
	assert.NotPanics(t, func() {
		w.(*watcherImpl).pollOnce()
	})
}

func TestWatcher_MatchingRegistrySuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepoDS := repoDSMocks.NewMockDataStore(ctrl)
	mockTagDS := tagDSMocks.NewMockDataStore(ctrl)
	mockRegistrySet := registryMocks.NewMockSet(ctrl)
	mockDelegator := delegatedRegistryMocks.NewMockDelegator(ctrl)
	mockRegistry := registryTypesMocks.NewMockImageRegistry(ctrl)

	repo := &storage.BaseImageRepository{
		Id:             "00000000-0000-0000-0000-0000000000ff",
		RepositoryPath: "docker.io/library/nginx",
		TagPattern:     "1.*",
	}

	mockRepoDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return([]*storage.BaseImageRepository{repo}, nil).
		Times(1)

	mockDelegator.EXPECT().
		GetDelegateClusterID(gomock.Any(), gomock.Any()).
		Return("", false, nil).
		Times(1)

	// Fetch existing tags from cache
	mockTagDS.EXPECT().
		ListTagsByRepository(gomock.Any(), gomock.Any()).
		Return([]*storage.BaseImageTag{}, nil).
		Times(1)

	// Registry matches and returns tags successfully
	mockRegistry.EXPECT().
		Match(gomock.Any()).
		Return(true).
		Times(1)

	mockRegistry.EXPECT().
		ListTags(gomock.Any(), gomock.Any()).
		Return([]string{"1.0", "1.1", "1.2", "2.0", "latest"}, nil).
		Times(1)

	// Mock Source() for rate limiter lookup (called once for rate limiter creation)
	mockRegistry.EXPECT().
		Source().
		Return(&storage.ImageIntegration{Id: "integration-1"}).
		Times(1)

	// Mock Metadata calls - 3 matching tags (1.0, 1.1, 1.2), but all return nil V1
	// which causes validation errors
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
		Times(1)

	// No tags stored (all metadata calls failed V1 validation), so no UpsertMany/DeleteMany calls expected

	w := createTestWatcher(ctrl, mockRepoDS, mockTagDS, mockRegistrySet, mockDelegator, 1*time.Hour, true)

	// Should complete successfully
	assert.NotPanics(t, func() {
		w.(*watcherImpl).pollOnce()
	})
}

func TestWatcher_ContextCancellation(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepoDS := repoDSMocks.NewMockDataStore(ctrl)
	mockTagDS := tagDSMocks.NewMockDataStore(ctrl)
	mockRegistrySet := registryMocks.NewMockSet(ctrl)
	mockDelegator := delegatedRegistryMocks.NewMockDelegator(ctrl)

	repo := &storage.BaseImageRepository{
		Id:             "00000000-0000-0000-0000-0000000000ff",
		RepositoryPath: "docker.io/library/nginx",
		TagPattern:     "*",
	}

	mockRepoDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return([]*storage.BaseImageRepository{repo}, nil).
		Times(1)

	// Fetch existing tags from cache (happens before delegation check)
	mockTagDS.EXPECT().
		ListTagsByRepository(gomock.Any(), gomock.Any()).
		Return([]*storage.BaseImageTag{}, nil).
		Times(1)

	// Block on GetDelegateClusterID until context is cancelled
	mockDelegator.EXPECT().
		GetDelegateClusterID(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, _ interface{}) (string, bool, error) {
			<-ctx.Done()
			return "", false, ctx.Err()
		}).
		Times(1)

	// After delegation error, processing continues and scanner calls GetAllUnique
	mockRegistrySet.EXPECT().
		GetAllUnique().
		Return(nil).
		Times(1)

	// No tags stored (no matching registry), so no UpsertMany/DeleteMany calls expected

	w := createTestWatcher(ctrl, mockRepoDS, mockTagDS, mockRegistrySet, mockDelegator, 1*time.Hour, true)

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

// TestWatcher_IncrementalUpdate_CheckTagsConstruction verifies that cached tags
// are correctly loaded and split into CheckTags and SkipTags based on the tag limit.
func TestWatcher_IncrementalUpdate_CheckTagsConstruction(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepoDS := repoDSMocks.NewMockDataStore(ctrl)
	mockTagDS := tagDSMocks.NewMockDataStore(ctrl)
	mockRegistrySet := registryMocks.NewMockSet(ctrl)
	mockDelegator := delegatedRegistryMocks.NewMockDelegator(ctrl)
	mockRegistry := registryTypesMocks.NewMockImageRegistry(ctrl)

	repo := &storage.BaseImageRepository{
		Id:             "11111111-1111-1111-1111-111111111111",
		RepositoryPath: "docker.io/library/nginx",
		TagPattern:     "1.*",
	}

	// Create cached tags with known creation times (newest first after sorting)
	now := time.Now()
	cachedTags := []*storage.BaseImageTag{
		{
			Id:                    "tag-1",
			BaseImageRepositoryId: "11111111-1111-1111-1111-111111111111",
			Tag:                   "1.25",
			ManifestDigest:        "sha256:digest-25",
			Created:               timestamppb.New(now.Add(-1 * time.Hour)), // Newest
		},
		{
			Id:                    "tag-2",
			BaseImageRepositoryId: "11111111-1111-1111-1111-111111111111",
			Tag:                   "1.24",
			ManifestDigest:        "sha256:digest-24",
			Created:               timestamppb.New(now.Add(-2 * time.Hour)),
		},
		{
			Id:                    "tag-3",
			BaseImageRepositoryId: "11111111-1111-1111-1111-111111111111",
			Tag:                   "1.23",
			ManifestDigest:        "sha256:digest-23",
			Created:               timestamppb.New(now.Add(-3 * time.Hour)), // Oldest
		},
	}

	mockRepoDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return([]*storage.BaseImageRepository{repo}, nil).
		Times(1)

	mockDelegator.EXPECT().
		GetDelegateClusterID(gomock.Any(), gomock.Any()).
		Return("", false, nil).
		Times(1)

	// Return cached tags - watcher should load these and build CheckTags/SkipTags
	mockTagDS.EXPECT().
		ListTagsByRepository(gomock.Any(), repo.GetId()).
		Return(cachedTags, nil).
		Times(1)

	// Registry matches
	mockRegistry.EXPECT().
		Match(gomock.Any()).
		Return(true).
		Times(1)

	// Registry returns matching tags (including the cached ones)
	mockRegistry.EXPECT().
		ListTags(gomock.Any(), gomock.Any()).
		Return([]string{"1.23", "1.24", "1.25"}, nil).
		Times(1)

	mockRegistry.EXPECT().
		Source().
		Return(&storage.ImageIntegration{Id: "integration-1"}).
		Times(1)

	// Metadata calls - return same digest for cached tags (no change)
	// The scanner should skip refetching metadata for unchanged digests
	mockRegistry.EXPECT().
		Metadata(gomock.Any()).
		DoAndReturn(func(img *storage.Image) (*storage.ImageMetadata, error) {
			tag := img.GetName().GetTag()
			// Return existing digest for cached tags - no update needed
			var digest string
			switch tag {
			case "1.25":
				digest = "sha256:digest-25"
			case "1.24":
				digest = "sha256:digest-24"
			case "1.23":
				digest = "sha256:digest-23"
			}
			return &storage.ImageMetadata{
				V1: &storage.V1Metadata{},
				V2: &storage.V2Metadata{
					Digest: digest,
				},
			}, nil
		}).
		Times(3)

	mockRegistrySet.EXPECT().
		GetAllUnique().
		Return([]types.ImageRegistry{mockRegistry}).
		Times(1)

	// Since digests haven't changed, no updates should be batched
	// But Flush is always called, so we need UpsertMany expectation
	mockTagDS.EXPECT().
		UpsertMany(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, tags []*storage.BaseImageTag) error {
			// Could verify tags here if needed
			return nil
		}).
		AnyTimes()

	mockTagDS.EXPECT().
		DeleteMany(gomock.Any(), gomock.Any()).
		AnyTimes()

	w := createTestWatcher(ctrl, mockRepoDS, mockTagDS, mockRegistrySet, mockDelegator, 1*time.Hour, true)

	// Execute poll
	require.NotPanics(t, func() {
		w.(*watcherImpl).pollOnce()
	})
}

// TestWatcher_IncrementalUpdate_SkipTagsWithLargeCache tests that when there are
// more cached tags than the tag limit, the excess tags go into SkipTags.
func TestWatcher_IncrementalUpdate_SkipTagsWithLargeCache(t *testing.T) {
	// Set a low limit for testing (automatically restored after test)
	t.Setenv(env.BaseImageWatcherPerRepoTagLimit.EnvVar(), "2")

	ctrl := gomock.NewController(t)
	mockRepoDS := repoDSMocks.NewMockDataStore(ctrl)
	mockTagDS := tagDSMocks.NewMockDataStore(ctrl)
	mockRegistrySet := registryMocks.NewMockSet(ctrl)
	mockDelegator := delegatedRegistryMocks.NewMockDelegator(ctrl)
	mockRegistry := registryTypesMocks.NewMockImageRegistry(ctrl)

	repo := &storage.BaseImageRepository{
		Id:             "11111111-1111-1111-1111-111111111111",
		RepositoryPath: "docker.io/library/nginx",
		TagPattern:     "1.*",
	}

	// Create 5 cached tags, but limit is 2, so only first 2 should be in CheckTags
	now := time.Now()
	cachedTags := []*storage.BaseImageTag{
		{Tag: "1.25", Created: timestamppb.New(now.Add(-1 * time.Hour)), ManifestDigest: "sha256:25"},
		{Tag: "1.24", Created: timestamppb.New(now.Add(-2 * time.Hour)), ManifestDigest: "sha256:24"},
		{Tag: "1.23", Created: timestamppb.New(now.Add(-3 * time.Hour)), ManifestDigest: "sha256:23"}, // SkipTag
		{Tag: "1.22", Created: timestamppb.New(now.Add(-4 * time.Hour)), ManifestDigest: "sha256:22"}, // SkipTag
		{Tag: "1.21", Created: timestamppb.New(now.Add(-5 * time.Hour)), ManifestDigest: "sha256:21"}, // SkipTag
	}

	for _, tag := range cachedTags {
		tag.BaseImageRepositoryId = repo.GetId()
		tag.Id = "id-" + tag.GetTag()
	}

	mockRepoDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return([]*storage.BaseImageRepository{repo}, nil).
		Times(1)

	mockDelegator.EXPECT().
		GetDelegateClusterID(gomock.Any(), gomock.Any()).
		Return("", false, nil).
		Times(1)

	mockTagDS.EXPECT().
		ListTagsByRepository(gomock.Any(), repo.GetId()).
		Return(cachedTags, nil).
		Times(1)

	mockRegistry.EXPECT().
		Match(gomock.Any()).
		Return(true).
		Times(1)

	// Registry returns all tags
	mockRegistry.EXPECT().
		ListTags(gomock.Any(), gomock.Any()).
		Return([]string{"1.25", "1.24", "1.23", "1.22", "1.21"}, nil).
		Times(1)

	mockRegistry.EXPECT().
		Source().
		Return(&storage.ImageIntegration{Id: "integration-1"}).
		Times(1)

	// Only first 2 tags should have metadata fetched (CheckTags), rest are skipped
	mockRegistry.EXPECT().
		Metadata(gomock.Any()).
		DoAndReturn(func(img *storage.Image) (*storage.ImageMetadata, error) {
			tag := img.GetName().GetTag()
			// Should only see 1.25 and 1.24
			require.Contains(t, []string{"1.25", "1.24"}, tag, "Unexpected metadata fetch for tag outside CheckTags")
			return &storage.ImageMetadata{
				V1: &storage.V1Metadata{},
				V2: &storage.V2Metadata{
					Digest: "sha256:" + tag[2:], // Extract version number
				},
			}, nil
		}).
		Times(2) // Only CheckTags (first 2)

	mockRegistrySet.EXPECT().
		GetAllUnique().
		Return([]types.ImageRegistry{mockRegistry}).
		Times(1)

	mockTagDS.EXPECT().
		UpsertMany(gomock.Any(), gomock.Any()).
		AnyTimes()

	mockTagDS.EXPECT().
		DeleteMany(gomock.Any(), gomock.Any()).
		AnyTimes()

	w := createTestWatcher(ctrl, mockRepoDS, mockTagDS, mockRegistrySet, mockDelegator, 1*time.Hour, true)

	require.NotPanics(t, func() {
		w.(*watcherImpl).pollOnce()
	})
}

// TestWatcher_TagBatch_FlushAfterScan verifies that the tag batch is always
// flushed after a scan completes, even if batch size wasn't reached.
func TestWatcher_TagBatch_FlushAfterScan(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepoDS := repoDSMocks.NewMockDataStore(ctrl)
	mockTagDS := tagDSMocks.NewMockDataStore(ctrl)
	mockRegistrySet := registryMocks.NewMockSet(ctrl)
	mockDelegator := delegatedRegistryMocks.NewMockDelegator(ctrl)
	mockRegistry := registryTypesMocks.NewMockImageRegistry(ctrl)

	repo := &storage.BaseImageRepository{
		Id:             "11111111-1111-1111-1111-111111111111",
		RepositoryPath: "docker.io/library/nginx",
		TagPattern:     "1.*",
	}

	mockRepoDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return([]*storage.BaseImageRepository{repo}, nil).
		Times(1)

	mockDelegator.EXPECT().
		GetDelegateClusterID(gomock.Any(), gomock.Any()).
		Return("", false, nil).
		Times(1)

	mockTagDS.EXPECT().
		ListTagsByRepository(gomock.Any(), repo.GetId()).
		Return([]*storage.BaseImageTag{}, nil).
		Times(1)

	mockRegistry.EXPECT().
		Match(gomock.Any()).
		Return(true).
		Times(1)

	// Return 3 tags (less than typical batch size of 100)
	mockRegistry.EXPECT().
		ListTags(gomock.Any(), gomock.Any()).
		Return([]string{"1.0", "1.1", "1.2"}, nil).
		Times(1)

	mockRegistry.EXPECT().
		Source().
		Return(&storage.ImageIntegration{Id: "integration-1"}).
		Times(1)

	now := time.Now()
	mockRegistry.EXPECT().
		Metadata(gomock.Any()).
		DoAndReturn(func(img *storage.Image) (*storage.ImageMetadata, error) {
			return &storage.ImageMetadata{
				V1: &storage.V1Metadata{
					Created: protocompat.ConvertTimeToTimestampOrNil(&now),
				},
				V2: &storage.V2Metadata{
					Digest: "sha256:digest-" + img.GetName().GetTag(),
				},
				LayerShas: []string{"sha256:layer1", "sha256:layer2"},
			}, nil
		}).
		Times(3)

	mockRegistrySet.EXPECT().
		GetAllUnique().
		Return([]types.ImageRegistry{mockRegistry}).
		Times(1)

	// Verify UpsertMany is called at least once (during Flush)
	upsertCalled := false
	mockTagDS.EXPECT().
		UpsertMany(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, tags []*storage.BaseImageTag) error {
			upsertCalled = true
			require.NotEmpty(t, tags, "UpsertMany should receive tags")
			require.LessOrEqual(t, len(tags), 3, "Should have at most 3 tags")
			return nil
		}).
		Times(1)

	// DeleteMany might be called with empty batch during Flush
	mockTagDS.EXPECT().
		DeleteMany(gomock.Any(), gomock.Any()).
		AnyTimes()

	w := createTestWatcher(ctrl, mockRepoDS, mockTagDS, mockRegistrySet, mockDelegator, 1*time.Hour, true)

	require.NotPanics(t, func() {
		w.(*watcherImpl).pollOnce()
	})

	require.True(t, upsertCalled, "UpsertMany should have been called during Flush")
}

// TestWatcher_TagBatch_DeleteEvent verifies that DeleteMany is called when
// tags are deleted (present in cache but not in registry).
func TestWatcher_TagBatch_DeleteEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepoDS := repoDSMocks.NewMockDataStore(ctrl)
	mockTagDS := tagDSMocks.NewMockDataStore(ctrl)
	mockRegistrySet := registryMocks.NewMockSet(ctrl)
	mockDelegator := delegatedRegistryMocks.NewMockDelegator(ctrl)
	mockRegistry := registryTypesMocks.NewMockImageRegistry(ctrl)

	repo := &storage.BaseImageRepository{
		Id:             "11111111-1111-1111-1111-111111111111",
		RepositoryPath: "docker.io/library/nginx",
		TagPattern:     "1.*",
	}

	now := time.Now()
	// Cache has tags 1.0, 1.1, 1.2
	cachedTags := []*storage.BaseImageTag{
		{
			Id:                    "tag-1",
			BaseImageRepositoryId: "11111111-1111-1111-1111-111111111111",
			Tag:                   "1.0",
			ManifestDigest:        "sha256:digest-1.0",
			Created:               timestamppb.New(now.Add(-1 * time.Hour)),
		},
		{
			Id:                    "tag-2",
			BaseImageRepositoryId: "11111111-1111-1111-1111-111111111111",
			Tag:                   "1.1",
			ManifestDigest:        "sha256:digest-1.1",
			Created:               timestamppb.New(now.Add(-2 * time.Hour)),
		},
		{
			Id:                    "tag-3",
			BaseImageRepositoryId: "11111111-1111-1111-1111-111111111111",
			Tag:                   "1.2",
			ManifestDigest:        "sha256:digest-1.2",
			Created:               timestamppb.New(now.Add(-3 * time.Hour)),
		},
	}

	mockRepoDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return([]*storage.BaseImageRepository{repo}, nil).
		Times(1)

	mockDelegator.EXPECT().
		GetDelegateClusterID(gomock.Any(), gomock.Any()).
		Return("", false, nil).
		Times(1)

	mockTagDS.EXPECT().
		ListTagsByRepository(gomock.Any(), repo.GetId()).
		Return(cachedTags, nil).
		Times(1)

	mockRegistry.EXPECT().
		Match(gomock.Any()).
		Return(true).
		Times(1)

	// Registry only returns 1.0 and 1.1 - tag 1.2 was deleted
	mockRegistry.EXPECT().
		ListTags(gomock.Any(), gomock.Any()).
		Return([]string{"1.0", "1.1"}, nil).
		Times(1)

	mockRegistry.EXPECT().
		Source().
		Return(&storage.ImageIntegration{Id: "integration-1"}).
		Times(1)

	mockRegistry.EXPECT().
		Metadata(gomock.Any()).
		DoAndReturn(func(img *storage.Image) (*storage.ImageMetadata, error) {
			return &storage.ImageMetadata{
				V1: &storage.V1Metadata{
					Created: protocompat.ConvertTimeToTimestampOrNil(&now),
				},
				V2: &storage.V2Metadata{
					Digest: "sha256:digest-" + img.GetName().GetTag(),
				},
				LayerShas: []string{"sha256:layer1", "sha256:layer2"},
			}, nil
		}).
		Times(2) // Only for 1.0 and 1.1

	mockRegistrySet.EXPECT().
		GetAllUnique().
		Return([]types.ImageRegistry{mockRegistry}).
		Times(1)

	mockTagDS.EXPECT().
		UpsertMany(gomock.Any(), gomock.Any()).
		AnyTimes()

	// Verify DeleteMany is called for the deleted tag
	deleteCalled := false
	mockTagDS.EXPECT().
		DeleteMany(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, ids []string) error {
			deleteCalled = true
			require.NotEmpty(t, ids, "DeleteMany should receive tag IDs")
			// Should delete tag-3 (1.2)
			return nil
		}).
		Times(1)

	w := createTestWatcher(ctrl, mockRepoDS, mockTagDS, mockRegistrySet, mockDelegator, 1*time.Hour, true)

	require.NotPanics(t, func() {
		w.(*watcherImpl).pollOnce()
	})

	require.True(t, deleteCalled, "DeleteMany should have been called for deleted tag")
}

func TestValidate_EmptyTag(t *testing.T) {
	event := reposcan.TagEvent{
		Tag:  "",
		Type: reposcan.TagEventMetadata,
	}

	err := validate(event)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tag is empty")
}

func TestValidate_TagEventError_Valid(t *testing.T) {
	event := reposcan.TagEvent{
		Tag:   "1.0",
		Type:  reposcan.TagEventError,
		Error: errors.New("some error"),
	}

	err := validate(event)
	assert.NoError(t, err)
}

func TestValidate_TagEventError_MissingError(t *testing.T) {
	event := reposcan.TagEvent{
		Tag:  "1.0",
		Type: reposcan.TagEventError,
	}

	err := validate(event)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error event without error")
}

func TestValidate_TagEventError_WithMetadata(t *testing.T) {
	now := time.Now()
	event := reposcan.TagEvent{
		Tag:   "1.0",
		Type:  reposcan.TagEventError,
		Error: errors.New("some error"),
		Metadata: &tagfetcher.TagMetadata{
			Tag:            "1.0",
			ManifestDigest: "sha256:abc",
			Created:        &now,
			LayerDigests:   []string{"sha256:layer1"},
		},
	}

	err := validate(event)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error event containing metadata")
}

func TestValidate_TagEventDeleted_Valid(t *testing.T) {
	event := reposcan.TagEvent{
		Tag:  "1.0",
		Type: reposcan.TagEventDeleted,
	}

	err := validate(event)
	assert.NoError(t, err)
}

func TestValidate_TagEventDeleted_WithMetadata(t *testing.T) {
	now := time.Now()
	event := reposcan.TagEvent{
		Tag:  "1.0",
		Type: reposcan.TagEventDeleted,
		Metadata: &tagfetcher.TagMetadata{
			Tag:            "1.0",
			ManifestDigest: "sha256:abc",
			Created:        &now,
			LayerDigests:   []string{"sha256:layer1"},
		},
	}

	err := validate(event)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "deletion event containing metadata")
}

func TestValidate_TagEventDeleted_WithError(t *testing.T) {
	event := reposcan.TagEvent{
		Tag:   "1.0",
		Type:  reposcan.TagEventDeleted,
		Error: errors.New("some error"),
	}

	err := validate(event)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "deletion event containing error")
}

func TestValidate_TagEventMetadata_Valid(t *testing.T) {
	now := time.Now()
	event := reposcan.TagEvent{
		Tag:  "1.0",
		Type: reposcan.TagEventMetadata,
		Metadata: &tagfetcher.TagMetadata{
			Tag:            "1.0",
			ManifestDigest: "sha256:abc123",
			Created:        &now,
			LayerDigests:   []string{"sha256:layer1", "sha256:layer2"},
		},
	}

	err := validate(event)
	assert.NoError(t, err)
}

func TestValidate_TagEventMetadata_WithError(t *testing.T) {
	now := time.Now()
	event := reposcan.TagEvent{
		Tag:   "1.0",
		Type:  reposcan.TagEventMetadata,
		Error: errors.New("should not have error"),
		Metadata: &tagfetcher.TagMetadata{
			Tag:            "1.0",
			ManifestDigest: "sha256:abc",
			Created:        &now,
			LayerDigests:   []string{"sha256:layer1"},
		},
	}

	err := validate(event)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "metadata event containing error")
}

func TestValidate_TagEventMetadata_MissingMetadata(t *testing.T) {
	event := reposcan.TagEvent{
		Tag:  "1.0",
		Type: reposcan.TagEventMetadata,
	}

	err := validate(event)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "metadata is empty")
}

func TestValidate_TagEventMetadata_TagMismatch(t *testing.T) {
	now := time.Now()
	event := reposcan.TagEvent{
		Tag:  "1.0",
		Type: reposcan.TagEventMetadata,
		Metadata: &tagfetcher.TagMetadata{
			Tag:            "2.0", // Different tag
			ManifestDigest: "sha256:abc",
			Created:        &now,
			LayerDigests:   []string{"sha256:layer1"},
		},
	}

	err := validate(event)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is different from event tag")
}

func TestValidate_TagEventMetadata_EmptyManifestDigest(t *testing.T) {
	now := time.Now()
	event := reposcan.TagEvent{
		Tag:  "1.0",
		Type: reposcan.TagEventMetadata,
		Metadata: &tagfetcher.TagMetadata{
			Tag:            "1.0",
			ManifestDigest: "", // Empty
			Created:        &now,
			LayerDigests:   []string{"sha256:layer1"},
		},
	}

	err := validate(event)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "manifest digest is empty")
}

func TestValidate_TagEventMetadata_EmptyLayerDigests(t *testing.T) {
	now := time.Now()
	event := reposcan.TagEvent{
		Tag:  "1.0",
		Type: reposcan.TagEventMetadata,
		Metadata: &tagfetcher.TagMetadata{
			Tag:            "1.0",
			ManifestDigest: "sha256:abc",
			Created:        &now,
			LayerDigests:   []string{}, // Empty
		},
	}

	err := validate(event)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "layers are empty")
}

func TestValidate_TagEventMetadata_NilLayerDigests(t *testing.T) {
	now := time.Now()
	event := reposcan.TagEvent{
		Tag:  "1.0",
		Type: reposcan.TagEventMetadata,
		Metadata: &tagfetcher.TagMetadata{
			Tag:            "1.0",
			ManifestDigest: "sha256:abc",
			Created:        &now,
			LayerDigests:   nil, // Nil
		},
	}

	err := validate(event)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "layers are empty")
}

func TestValidate_TagEventMetadata_NilCreated(t *testing.T) {
	event := reposcan.TagEvent{
		Tag:  "1.0",
		Type: reposcan.TagEventMetadata,
		Metadata: &tagfetcher.TagMetadata{
			Tag:            "1.0",
			ManifestDigest: "sha256:abc",
			Created:        nil, // Nil
			LayerDigests:   []string{"sha256:layer1"},
		},
	}

	err := validate(event)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "created timestamp is empty")
}

func TestValidate_UnknownEventType(t *testing.T) {
	event := reposcan.TagEvent{
		Tag:  "1.0",
		Type: 999, // Unknown type
	}

	err := validate(event)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown event type: 999")
}

func TestTagUUID_Deterministic(t *testing.T) {
	repoID := "11111111-1111-1111-1111-111111111111"
	tag := "1.25"

	// Generate UUID multiple times
	id1, err1 := tagUUID(repoID, tag)
	id2, err2 := tagUUID(repoID, tag)
	id3, err3 := tagUUID(repoID, tag)

	// All should succeed
	require.NoError(t, err1)
	require.NoError(t, err2)
	require.NoError(t, err3)

	// All IDs should be identical (deterministic)
	assert.Equal(t, id1, id2)
	assert.Equal(t, id2, id3)

	// Different tags should produce different IDs
	id4, err4 := tagUUID(repoID, "1.24")
	require.NoError(t, err4)
	assert.NotEqual(t, id1, id4)
}

func TestTagUUID_InvalidRepoID(t *testing.T) {
	invalidRepoID := "not-a-uuid"
	tag := "1.25"

	id, err := tagUUID(invalidRepoID, tag)
	require.Error(t, err)
	assert.Empty(t, id)
	assert.Contains(t, err.Error(), "invalid UUID")
}

func TestWatcher_DelegatedFeatureFlag_Disabled(t *testing.T) {
	// When delegation is disabled, GetDelegateClusterID should not be called.
	ctrl := gomock.NewController(t)
	mockRepoDS := repoDSMocks.NewMockDataStore(ctrl)
	mockTagDS := tagDSMocks.NewMockDataStore(ctrl)
	mockRegistrySet := registryMocks.NewMockSet(ctrl)
	mockDelegator := delegatedRegistryMocks.NewMockDelegator(ctrl)

	repo := &storage.BaseImageRepository{
		Id:             "00000000-0000-0000-0000-0000000000ff",
		RepositoryPath: "docker.io/library/nginx",
		TagPattern:     "*",
	}

	mockRepoDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return([]*storage.BaseImageRepository{repo}, nil).
		Times(1)

	// GetDelegateClusterID should NOT be called when delegation is disabled.
	// If it were called, this test would fail due to missing expectation.

	// Fetch existing tags from cache.
	mockTagDS.EXPECT().
		ListTagsByRepository(gomock.Any(), gomock.Any()).
		Return([]*storage.BaseImageTag{}, nil).
		Times(1)

	// Scanner calls GetAllUnique (local scanner is always used).
	mockRegistrySet.EXPECT().
		GetAllUnique().
		Return(nil).
		Times(1)

	w := createTestWatcher(ctrl, mockRepoDS, mockTagDS, mockRegistrySet, mockDelegator, 1*time.Hour, false)

	assert.NotPanics(t, func() {
		w.(*watcherImpl).pollOnce()
	})
}

func TestWatcher_DelegatedFeatureFlag_Enabled(t *testing.T) {
	// When delegation is enabled, GetDelegateClusterID should be called.
	ctrl := gomock.NewController(t)
	mockRepoDS := repoDSMocks.NewMockDataStore(ctrl)
	mockTagDS := tagDSMocks.NewMockDataStore(ctrl)
	mockRegistrySet := registryMocks.NewMockSet(ctrl)
	mockDelegator := delegatedRegistryMocks.NewMockDelegator(ctrl)

	repo := &storage.BaseImageRepository{
		Id:             "00000000-0000-0000-0000-0000000000ff",
		RepositoryPath: "docker.io/library/nginx",
		TagPattern:     "*",
	}

	mockRepoDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return([]*storage.BaseImageRepository{repo}, nil).
		Times(1)

	// GetDelegateClusterID should be called when feature is enabled.
	mockDelegator.EXPECT().
		GetDelegateClusterID(gomock.Any(), gomock.Any()).
		Return("", false, nil).
		Times(1)

	// Fetch existing tags from cache.
	mockTagDS.EXPECT().
		ListTagsByRepository(gomock.Any(), gomock.Any()).
		Return([]*storage.BaseImageTag{}, nil).
		Times(1)

	// Scanner calls GetAllUnique.
	mockRegistrySet.EXPECT().
		GetAllUnique().
		Return(nil).
		Times(1)

	w := createTestWatcher(ctrl, mockRepoDS, mockTagDS, mockRegistrySet, mockDelegator, 1*time.Hour, true)

	assert.NotPanics(t, func() {
		w.(*watcherImpl).pollOnce()
	})
}
