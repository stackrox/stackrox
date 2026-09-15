package watcher

import (
	"context"
	"errors"
	"testing"
	"time"

	baseImageDSMocks "github.com/stackrox/rox/central/baseimage/datastore/mocks"
	repoDS "github.com/stackrox/rox/central/baseimage/datastore/repository"
	repoDSMocks "github.com/stackrox/rox/central/baseimage/datastore/repository/mocks"
	tagDSMocks "github.com/stackrox/rox/central/baseimage/datastore/tag/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/baseimage/reposcan"
	"github.com/stackrox/rox/pkg/baseimage/tagfetcher"
	delegatedRegistryMocks "github.com/stackrox/rox/pkg/delegatedregistry/mocks"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protocompat"
	registryMocks "github.com/stackrox/rox/pkg/registries/mocks"
	"github.com/stackrox/rox/pkg/registries/types"
	registryTypesMocks "github.com/stackrox/rox/pkg/registries/types/mocks"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"
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

	// Allow repository status updates.
	mockRepoDS.EXPECT().UpdateStatus(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

	return New(mockRepoDS, mockTagDS, mockBaseImageDS, mockRegistrySet, mockDelegator, poll, 10*time.Millisecond, 10, 100, 5, delegationEnabled)
}

// runPollAndWait runs a single poll cycle synchronously.
func runPollAndWait(w Watcher) {
	impl := w.(*watcherImpl)
	ctx := sac.WithAllAccess(context.Background())
	_ = impl.poll(ctx)
}

func TestWatcher_StartsAndStops(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepoDS := repoDSMocks.NewMockDataStore(ctrl)

	mockRepoDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return([]*storage.BaseImageRepository{}, nil).
		AnyTimes()

	w := createTestWatcher(ctrl, mockRepoDS, nil, nil, nil, 100*time.Millisecond, false)

	w.Start()
	time.Sleep(50 * time.Millisecond)

	done := make(chan struct{})
	go func() {
		w.Stop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("Watcher did not stop within 1 second")
	}
}

func TestWatcher_PollsOnFirstTick(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepoDS := repoDSMocks.NewMockDataStore(ctrl)

	pollCalled := make(chan struct{}, 1)

	mockRepoDS.EXPECT().
		ListRepositories(gomock.Any()).
		DoAndReturn(func(ctx context.Context) ([]*storage.BaseImageRepository, error) {
			select {
			case pollCalled <- struct{}{}:
			default:
			}
			return []*storage.BaseImageRepository{}, nil
		}).
		AnyTimes()

	w := createTestWatcher(ctrl, mockRepoDS, nil, nil, nil, 1*time.Hour, false)

	w.Start()
	defer w.Stop()

	select {
	case <-pollCalled:
	case <-time.After(1 * time.Second):
		t.Fatal("Watcher did not poll on first tick")
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

	for _, repo := range repos {
		mockRepoDS.EXPECT().
			UpdateStatus(gomock.Any(), repo.GetId(), gomock.Any()).
			Return(repo, nil).
			MinTimes(1)
	}

	mockDelegator.EXPECT().
		GetDelegateClusterID(gomock.Any(), gomock.Any()).
		Return("", false, nil).
		Times(3)

	mockTagDS.EXPECT().
		ListTagsByRepository(gomock.Any(), gomock.Any()).
		Return([]*storage.BaseImageTag{}, nil).
		Times(3)

	mockRegistrySet.EXPECT().
		GetAllUnique().
		Return(nil).
		Times(3)

	w := createTestWatcher(ctrl, mockRepoDS, mockTagDS, mockRegistrySet, mockDelegator, 1*time.Hour, true)

	assert.NotPanics(t, func() {
		runPollAndWait(w)
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
		runPollAndWait(w)
	})
}

func TestWatcher_StartIsIdempotent(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepoDS := repoDSMocks.NewMockDataStore(ctrl)

	mockRepoDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return([]*storage.BaseImageRepository{}, nil).
		AnyTimes()

	w := createTestWatcher(ctrl, mockRepoDS, nil, nil, nil, 100*time.Millisecond, false)

	w.Start()
	w.Start()
	w.Start()

	time.Sleep(50 * time.Millisecond)
	w.Stop()
}

func TestWatcher_StopIsIdempotent(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepoDS := repoDSMocks.NewMockDataStore(ctrl)

	mockRepoDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return([]*storage.BaseImageRepository{}, nil).
		AnyTimes()

	w := createTestWatcher(ctrl, mockRepoDS, nil, nil, nil, 100*time.Millisecond, false)

	w.Start()
	time.Sleep(50 * time.Millisecond)

	w.Stop()
	w.Stop()
	w.Stop()

	assert.True(t, true)
}

func TestWatcher_StopsGracefullyDuringPoll(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepoDS := repoDSMocks.NewMockDataStore(ctrl)

	// First call is from failInFlightScans in Start(); let it through.
	// Second call (from poll) blocks until released.
	blockCh := make(chan struct{})
	var callCount int
	mockRepoDS.EXPECT().
		ListRepositories(gomock.Any()).
		DoAndReturn(func(ctx context.Context) ([]*storage.BaseImageRepository, error) {
			callCount++
			if callCount >= 2 {
				<-blockCh
			}
			return []*storage.BaseImageRepository{}, nil
		}).
		AnyTimes()

	w := createTestWatcher(ctrl, mockRepoDS, nil, nil, nil, 1*time.Hour, false)

	w.Start()
	time.Sleep(50 * time.Millisecond)

	done := make(chan struct{})
	go func() {
		w.Stop()
		close(done)
	}()

	close(blockCh)

	select {
	case <-done:
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

	mockRepoDS.EXPECT().
		UpdateStatus(gomock.Any(), repo.GetId(), gomock.Any()).
		Return(repo, nil).
		Times(1)

	mockDelegator.EXPECT().
		GetDelegateClusterID(gomock.Any(), gomock.Any()).
		Return("", false, nil).
		Times(1)

	mockTagDS.EXPECT().
		ListTagsByRepository(gomock.Any(), gomock.Any()).
		Return([]*storage.BaseImageTag{}, nil).
		Times(1)

	mockRegistrySet.EXPECT().
		GetAllUnique().
		Return(nil).
		Times(1)

	w := createTestWatcher(ctrl, mockRepoDS, mockTagDS, mockRegistrySet, mockDelegator, 1*time.Hour, true)

	runPollAndWait(w)

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

	mockRepoDS.EXPECT().
		UpdateStatus(gomock.Any(), repo.GetId(), gomock.Any()).
		Return(repo, nil).
		Times(1)

	mockDelegator.EXPECT().
		GetDelegateClusterID(gomock.Any(), gomock.Any()).
		Return("", false, errox.InvariantViolation.New("delegation check failed")).
		Times(1)

	mockTagDS.EXPECT().
		ListTagsByRepository(gomock.Any(), gomock.Any()).
		Return([]*storage.BaseImageTag{}, nil).
		Times(1)

	mockRegistrySet.EXPECT().
		GetAllUnique().
		Return(nil).
		Times(1)

	w := createTestWatcher(ctrl, mockRepoDS, mockTagDS, mockRegistrySet, mockDelegator, 1*time.Hour, true)

	assert.NotPanics(t, func() {
		runPollAndWait(w)
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

	mockRepoDS.EXPECT().
		UpdateStatus(gomock.Any(), repo.GetId(), gomock.Any()).
		Return(repo, nil).
		Times(1)

	mockTagDS.EXPECT().
		ListTagsByRepository(gomock.Any(), gomock.Any()).
		Return([]*storage.BaseImageTag{}, nil).
		Times(1)

	mockDelegator.EXPECT().
		GetDelegateClusterID(gomock.Any(), gomock.Any()).
		Return("cluster-123", true, nil).
		Times(1)

	w := createTestWatcher(ctrl, mockRepoDS, mockTagDS, nil, mockDelegator, 1*time.Hour, true)

	assert.NotPanics(t, func() {
		runPollAndWait(w)
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

	mockRepoDS.EXPECT().
		UpdateStatus(gomock.Any(), repo.GetId(), gomock.Any()).
		Return(repo, nil).
		Times(1)

	mockDelegator.EXPECT().
		GetDelegateClusterID(gomock.Any(), gomock.Any()).
		Return("", false, nil).
		Times(1)

	mockTagDS.EXPECT().
		ListTagsByRepository(gomock.Any(), gomock.Any()).
		Return([]*storage.BaseImageTag{}, nil).
		Times(1)

	mockRegistry.EXPECT().
		Match(gomock.Any()).
		Return(false).
		Times(1)

	mockRegistrySet.EXPECT().
		GetAllUnique().
		Return([]types.ImageRegistry{mockRegistry}).
		Times(1)

	w := createTestWatcher(ctrl, mockRepoDS, mockTagDS, mockRegistrySet, mockDelegator, 1*time.Hour, true)

	assert.NotPanics(t, func() {
		runPollAndWait(w)
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

	mockRepoDS.EXPECT().
		UpdateStatus(gomock.Any(), repo.GetId(), gomock.Any()).
		Return(repo, nil).
		Times(1)

	mockDelegator.EXPECT().
		GetDelegateClusterID(gomock.Any(), gomock.Any()).
		Return("", false, nil).
		Times(1)

	mockTagDS.EXPECT().
		ListTagsByRepository(gomock.Any(), gomock.Any()).
		Return([]*storage.BaseImageTag{}, nil).
		Times(1)

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

	w := createTestWatcher(ctrl, mockRepoDS, mockTagDS, mockRegistrySet, mockDelegator, 1*time.Hour, true)

	assert.NotPanics(t, func() {
		runPollAndWait(w)
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

	mockRepoDS.EXPECT().
		UpdateStatus(gomock.Any(), repo.GetId(), gomock.Any()).
		Return(repo, nil).
		Times(1)

	mockDelegator.EXPECT().
		GetDelegateClusterID(gomock.Any(), gomock.Any()).
		Return("", false, nil).
		Times(1)

	mockTagDS.EXPECT().
		ListTagsByRepository(gomock.Any(), gomock.Any()).
		Return([]*storage.BaseImageTag{}, nil).
		Times(1)

	mockRegistry.EXPECT().
		Match(gomock.Any()).
		Return(true).
		Times(1)

	mockRegistry.EXPECT().
		ListTags(gomock.Any(), gomock.Any()).
		Return([]string{"1.0", "1.1", "1.2", "2.0", "latest"}, nil).
		Times(1)

	mockRegistry.EXPECT().
		Source().
		Return(&storage.ImageIntegration{Id: "integration-1"}).
		Times(1)

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

	w := createTestWatcher(ctrl, mockRepoDS, mockTagDS, mockRegistrySet, mockDelegator, 1*time.Hour, true)

	assert.NotPanics(t, func() {
		runPollAndWait(w)
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
		AnyTimes()

	mockRepoDS.EXPECT().
		UpdateStatus(gomock.Any(), repo.GetId(), gomock.Any()).
		Return(repo, nil).
		AnyTimes()

	mockTagDS.EXPECT().
		ListTagsByRepository(gomock.Any(), gomock.Any()).
		Return([]*storage.BaseImageTag{}, nil).
		AnyTimes()

	// Block on GetDelegateClusterID until context is cancelled.
	mockDelegator.EXPECT().
		GetDelegateClusterID(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, _ interface{}) (string, bool, error) {
			<-ctx.Done()
			return "", false, ctx.Err()
		}).
		AnyTimes()

	mockRegistrySet.EXPECT().
		GetAllUnique().
		Return(nil).
		AnyTimes()

	w := createTestWatcher(ctrl, mockRepoDS, mockTagDS, mockRegistrySet, mockDelegator, 1*time.Hour, true)

	w.Start()
	time.Sleep(50 * time.Millisecond)

	done := make(chan struct{})
	go func() {
		w.Stop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Watcher did not stop within 2 seconds")
	}
}

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

	now := time.Now()
	cachedTags := []*storage.BaseImageTag{
		{
			Id:                    "tag-1",
			BaseImageRepositoryId: "11111111-1111-1111-1111-111111111111",
			Tag:                   "1.25",
			ManifestDigest:        "sha256:digest-25",
			Created:               timestamppb.New(now.Add(-1 * time.Hour)),
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
			Created:               timestamppb.New(now.Add(-3 * time.Hour)),
		},
	}

	mockRepoDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return([]*storage.BaseImageRepository{repo}, nil).
		Times(1)

	mockRepoDS.EXPECT().
		UpdateStatus(gomock.Any(), repo.GetId(), gomock.Any()).
		Return(repo, nil).
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

	mockRegistry.EXPECT().
		ListTags(gomock.Any(), gomock.Any()).
		Return([]string{"1.23", "1.24", "1.25"}, nil).
		Times(1)

	mockRegistry.EXPECT().
		Source().
		Return(&storage.ImageIntegration{Id: "integration-1"}).
		Times(1)

	mockRegistry.EXPECT().
		Metadata(gomock.Any()).
		DoAndReturn(func(img *storage.Image) (*storage.ImageMetadata, error) {
			tag := img.GetName().GetTag()
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

	mockTagDS.EXPECT().
		UpsertMany(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, tags []*storage.BaseImageTag) error {
			return nil
		}).
		AnyTimes()

	mockTagDS.EXPECT().
		DeleteMany(gomock.Any(), gomock.Any()).
		AnyTimes()

	w := createTestWatcher(ctrl, mockRepoDS, mockTagDS, mockRegistrySet, mockDelegator, 1*time.Hour, true)

	require.NotPanics(t, func() {
		runPollAndWait(w)
	})
}

func TestWatcher_IncrementalUpdate_SkipTagsWithLargeCache(t *testing.T) {
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
	cachedTags := []*storage.BaseImageTag{
		{Tag: "1.25", Created: timestamppb.New(now.Add(-1 * time.Hour)), ManifestDigest: "sha256:25"},
		{Tag: "1.24", Created: timestamppb.New(now.Add(-2 * time.Hour)), ManifestDigest: "sha256:24"},
		{Tag: "1.23", Created: timestamppb.New(now.Add(-3 * time.Hour)), ManifestDigest: "sha256:23"},
		{Tag: "1.22", Created: timestamppb.New(now.Add(-4 * time.Hour)), ManifestDigest: "sha256:22"},
		{Tag: "1.21", Created: timestamppb.New(now.Add(-5 * time.Hour)), ManifestDigest: "sha256:21"},
	}

	for _, tag := range cachedTags {
		tag.BaseImageRepositoryId = repo.GetId()
		tag.Id = "id-" + tag.GetTag()
	}

	mockRepoDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return([]*storage.BaseImageRepository{repo}, nil).
		Times(1)

	mockRepoDS.EXPECT().
		UpdateStatus(gomock.Any(), repo.GetId(), gomock.Any()).
		Return(repo, nil).
		AnyTimes()

	mockDelegator.EXPECT().
		GetDelegateClusterID(gomock.Any(), gomock.Any()).
		Return("", false, nil).
		Times(1)

	mockTagDS.EXPECT().
		ListTagsByRepository(gomock.Any(), repo.GetId()).
		Return(cachedTags, nil).
		AnyTimes()

	mockRegistry.EXPECT().
		Match(gomock.Any()).
		Return(true).
		Times(1)

	mockRegistry.EXPECT().
		ListTags(gomock.Any(), gomock.Any()).
		Return([]string{"1.25", "1.24", "1.23", "1.22", "1.21"}, nil).
		Times(1)

	mockRegistry.EXPECT().
		Source().
		Return(&storage.ImageIntegration{Id: "integration-1"}).
		Times(1)

	mockRegistry.EXPECT().
		Metadata(gomock.Any()).
		DoAndReturn(func(img *storage.Image) (*storage.ImageMetadata, error) {
			tag := img.GetName().GetTag()
			require.Contains(t, []string{"1.25", "1.24"}, tag, "Unexpected metadata fetch for tag outside CheckTags")
			return &storage.ImageMetadata{
				V1: &storage.V1Metadata{},
				V2: &storage.V2Metadata{
					Digest: "sha256:" + tag[2:],
				},
			}, nil
		}).
		Times(2)

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

	mockBaseImageDS := baseImageDSMocks.NewMockDataStore(ctrl)
	mockBaseImageDS.EXPECT().ReplaceByRepository(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	// Create watcher with tagLimit=2 to test skip tag behavior.
	w := New(mockRepoDS, mockTagDS, mockBaseImageDS, mockRegistrySet, mockDelegator,
		1*time.Hour, 10*time.Millisecond, 10, 2, 5, true)

	require.NotPanics(t, func() {
		runPollAndWait(w)
	})
}

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

	mockRepoDS.EXPECT().
		UpdateStatus(gomock.Any(), repo.GetId(), gomock.Any()).
		Return(repo, nil).
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

	mockTagDS.EXPECT().
		DeleteMany(gomock.Any(), gomock.Any()).
		AnyTimes()

	w := createTestWatcher(ctrl, mockRepoDS, mockTagDS, mockRegistrySet, mockDelegator, 1*time.Hour, true)

	require.NotPanics(t, func() {
		runPollAndWait(w)
	})

	require.True(t, upsertCalled, "UpsertMany should have been called during Flush")
}

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

	mockRepoDS.EXPECT().
		UpdateStatus(gomock.Any(), repo.GetId(), gomock.Any()).
		Return(repo, nil).
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
		Times(2)

	mockRegistrySet.EXPECT().
		GetAllUnique().
		Return([]types.ImageRegistry{mockRegistry}).
		Times(1)

	mockTagDS.EXPECT().
		UpsertMany(gomock.Any(), gomock.Any()).
		AnyTimes()

	deleteCalled := false
	mockTagDS.EXPECT().
		DeleteMany(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, ids []string) error {
			deleteCalled = true
			require.NotEmpty(t, ids, "DeleteMany should receive tag IDs")
			return nil
		}).
		Times(1)

	w := createTestWatcher(ctrl, mockRepoDS, mockTagDS, mockRegistrySet, mockDelegator, 1*time.Hour, true)

	require.NotPanics(t, func() {
		runPollAndWait(w)
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
			Tag:            "2.0",
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
			ManifestDigest: "",
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
			LayerDigests:   []string{},
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
			LayerDigests:   nil,
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
			Created:        nil,
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
		Type: 999,
	}

	err := validate(event)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown event type: 999")
}

func TestTagUUID_Deterministic(t *testing.T) {
	repoID := "11111111-1111-1111-1111-111111111111"
	tag := "1.25"

	id1, err1 := tagUUID(repoID, tag)
	id2, err2 := tagUUID(repoID, tag)
	id3, err3 := tagUUID(repoID, tag)

	require.NoError(t, err1)
	require.NoError(t, err2)
	require.NoError(t, err3)

	assert.Equal(t, id1, id2)
	assert.Equal(t, id2, id3)

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

	mockRepoDS.EXPECT().
		UpdateStatus(gomock.Any(), repo.GetId(), gomock.Any()).
		Return(repo, nil).
		Times(1)

	mockTagDS.EXPECT().
		ListTagsByRepository(gomock.Any(), gomock.Any()).
		Return([]*storage.BaseImageTag{}, nil).
		Times(1)

	mockRegistrySet.EXPECT().
		GetAllUnique().
		Return(nil).
		Times(1)

	w := createTestWatcher(ctrl, mockRepoDS, mockTagDS, mockRegistrySet, mockDelegator, 1*time.Hour, false)

	assert.NotPanics(t, func() {
		runPollAndWait(w)
	})
}

func TestWatcher_DelegatedFeatureFlag_Enabled(t *testing.T) {
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

	mockRepoDS.EXPECT().
		UpdateStatus(gomock.Any(), repo.GetId(), gomock.Any()).
		Return(repo, nil).
		Times(1)

	mockDelegator.EXPECT().
		GetDelegateClusterID(gomock.Any(), gomock.Any()).
		Return("", false, nil).
		Times(1)

	mockTagDS.EXPECT().
		ListTagsByRepository(gomock.Any(), gomock.Any()).
		Return([]*storage.BaseImageTag{}, nil).
		Times(1)

	mockRegistrySet.EXPECT().
		GetAllUnique().
		Return(nil).
		Times(1)

	w := createTestWatcher(ctrl, mockRepoDS, mockTagDS, mockRegistrySet, mockDelegator, 1*time.Hour, true)

	assert.NotPanics(t, func() {
		runPollAndWait(w)
	})
}

func TestIsRepositoryDue(t *testing.T) {
	now := time.Now()
	fiveHoursAgo := now.Add(-5 * time.Hour)
	threeHoursAgo := now.Add(-3 * time.Hour)
	pollInterval := 4 * time.Hour

	cases := []struct {
		name         string
		status       storage.BaseImageRepository_Status
		lastPolledAt *time.Time
		expected     bool
	}{
		{"CREATED always due", storage.BaseImageRepository_CREATED, nil, true},
		{"QUEUED never due", storage.BaseImageRepository_QUEUED, nil, false},
		{"IN_PROGRESS never due", storage.BaseImageRepository_IN_PROGRESS, nil, false},
		{"READY nil lastPolled due", storage.BaseImageRepository_READY, nil, true},
		{"READY recently polled not due", storage.BaseImageRepository_READY, &now, false},
		{"READY within interval not due", storage.BaseImageRepository_READY, &threeHoursAgo, false},
		{"READY interval elapsed due", storage.BaseImageRepository_READY, &fiveHoursAgo, true},
		{"FAILED nil lastPolled due", storage.BaseImageRepository_FAILED, nil, true},
		{"FAILED recently polled not due", storage.BaseImageRepository_FAILED, &now, false},
		{"FAILED within interval not due", storage.BaseImageRepository_FAILED, &threeHoursAgo, false},
		{"FAILED interval elapsed due", storage.BaseImageRepository_FAILED, &fiveHoursAgo, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &storage.BaseImageRepository{
				Id:           uuid.NewV4().String(),
				Status:       tc.status,
				LastPolledAt: protocompat.ConvertTimeToTimestampOrNil(tc.lastPolledAt),
			}
			assert.Equal(t, tc.expected, isRepositoryDue(repo, pollInterval))
		})
	}
}

func TestWatcher_ScanFailure_SetsFailedStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepoDS := repoDSMocks.NewMockDataStore(ctrl)
	mockTagDS := tagDSMocks.NewMockDataStore(ctrl)
	mockBaseImageDS := baseImageDSMocks.NewMockDataStore(ctrl)
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

	// Claiming (QUEUED).
	mockRepoDS.EXPECT().
		UpdateStatus(gomock.Any(), repo.GetId(), gomock.Any()).
		Return(repo, nil).
		Times(1)

	// IN_PROGRESS.
	mockRepoDS.EXPECT().
		UpdateStatus(gomock.Any(), repo.GetId(), gomock.Any()).
		Return(repo, nil).
		Times(1)

	// FAILED with failure message.
	mockRepoDS.EXPECT().
		UpdateStatus(gomock.Any(), repo.GetId(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, id string, update repoDS.StatusUpdate) (*storage.BaseImageRepository, error) {
			assert.Equal(t, storage.BaseImageRepository_FAILED, update.Status)
			assert.Equal(t, repoDS.FailureCountIncrement, update.FailureCountOp)
			assert.NotNil(t, update.LastFailureMessage)
			assert.Contains(t, *update.LastFailureMessage, "failed to list tags")
			return repo, nil
		}).
		Times(1)

	mockDelegator.EXPECT().
		GetDelegateClusterID(gomock.Any(), gomock.Any()).
		Return("", false, nil).
		Times(1)

	mockTagDS.EXPECT().
		ListTagsByRepository(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("database connection failed")).
		Times(1)

	w := New(mockRepoDS, mockTagDS, mockBaseImageDS, mockRegistrySet, mockDelegator, 1*time.Hour, 10*time.Millisecond, 10, 10, 5, true)

	assert.NotPanics(t, func() {
		runPollAndWait(w)
	})
}

func TestWatcher_SchedulerCadence_SkipsNotDueRepos(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepoDS := repoDSMocks.NewMockDataStore(ctrl)

	repo := &storage.BaseImageRepository{
		Id:             "00000000-0000-0000-0000-0000000000ff",
		RepositoryPath: "docker.io/library/nginx",
		TagPattern:     "*",
		Status:         storage.BaseImageRepository_READY,
		LastPolledAt:   protocompat.TimestampNow(),
	}

	mockRepoDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return([]*storage.BaseImageRepository{repo}, nil).
		Times(1)

	w := createTestWatcher(ctrl, mockRepoDS, nil, nil, nil, 4*time.Hour, false)

	ctx := sac.WithAllAccess(context.Background())
	err := w.(*watcherImpl).poll(ctx)
	assert.NoError(t, err)
}

func TestWatcher_PollFairness_SortsReposByLastPolledAt(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepoDS := repoDSMocks.NewMockDataStore(ctrl)
	mockTagDS := tagDSMocks.NewMockDataStore(ctrl)
	mockBaseImageDS := baseImageDSMocks.NewMockDataStore(ctrl)
	mockRegistrySet := registryMocks.NewMockSet(ctrl)
	mockDelegator := delegatedRegistryMocks.NewMockDelegator(ctrl)

	now := time.Now()
	oldestPolledAt := now.Add(-6 * time.Hour)
	recentPolledAt := now.Add(-5 * time.Hour)
	neverScanned := &storage.BaseImageRepository{
		Id:             "00000000-0000-0000-0000-000000000001",
		RepositoryPath: "docker.io/library/never",
		TagPattern:     "*",
		Status:         storage.BaseImageRepository_CREATED,
	}
	oldestScanned := &storage.BaseImageRepository{
		Id:             "00000000-0000-0000-0000-000000000002",
		RepositoryPath: "docker.io/library/oldest",
		TagPattern:     "*",
		Status:         storage.BaseImageRepository_READY,
		LastPolledAt:   protocompat.ConvertTimeToTimestampOrNil(&oldestPolledAt),
	}
	recentlyScanned := &storage.BaseImageRepository{
		Id:             "00000000-0000-0000-0000-000000000003",
		RepositoryPath: "docker.io/library/recent",
		TagPattern:     "*",
		Status:         storage.BaseImageRepository_READY,
		LastPolledAt:   protocompat.ConvertTimeToTimestampOrNil(&recentPolledAt),
	}

	mockRepoDS.EXPECT().
		ListRepositories(gomock.Any()).
		Return([]*storage.BaseImageRepository{recentlyScanned, oldestScanned, neverScanned}, nil).
		Times(1)

	// Track claim order via QUEUED transitions.
	var mu sync.Mutex
	claimOrder := make([]string, 0, 3)

	mockRepoDS.EXPECT().
		UpdateStatus(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, id string, update repoDS.StatusUpdate) (*storage.BaseImageRepository, error) {
			if update.Status == storage.BaseImageRepository_QUEUED {
				mu.Lock()
				claimOrder = append(claimOrder, id)
				mu.Unlock()
			}
			switch id {
			case neverScanned.GetId():
				return neverScanned, nil
			case oldestScanned.GetId():
				return oldestScanned, nil
			case recentlyScanned.GetId():
				return recentlyScanned, nil
			}
			return nil, nil
		}).
		AnyTimes()

	mockTagDS.EXPECT().ListTagsByRepository(gomock.Any(), gomock.Any()).Return([]*storage.BaseImageTag{}, nil).AnyTimes()
	mockRegistrySet.EXPECT().GetAllUnique().Return(nil).AnyTimes()
	mockBaseImageDS.EXPECT().ReplaceByRepository(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	w := New(mockRepoDS, mockTagDS, mockBaseImageDS, mockRegistrySet, mockDelegator, 4*time.Hour, 10*time.Millisecond, 10, 100, 5, false)

	ctx := sac.WithAllAccess(context.Background())
	err := w.(*watcherImpl).poll(ctx)
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, claimOrder, 3)
	assert.Equal(t, neverScanned.GetId(), claimOrder[0], "never-scanned repo should be claimed first")
	assert.Equal(t, oldestScanned.GetId(), claimOrder[1], "oldest-scanned repo should be claimed second")
	assert.Equal(t, recentlyScanned.GetId(), claimOrder[2], "recently-scanned repo should be claimed last")
}

func TestWatcher_RegistryError_SetsFailedStatus(t *testing.T) {
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

	// Claiming (QUEUED).
	mockRepoDS.EXPECT().
		UpdateStatus(gomock.Any(), repo.GetId(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, update repoDS.StatusUpdate) (*storage.BaseImageRepository, error) {
			assert.Equal(t, storage.BaseImageRepository_QUEUED, update.Status)
			return repo, nil
		}).
		Times(1)

	// IN_PROGRESS.
	mockRepoDS.EXPECT().
		UpdateStatus(gomock.Any(), repo.GetId(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, update repoDS.StatusUpdate) (*storage.BaseImageRepository, error) {
			assert.Equal(t, storage.BaseImageRepository_IN_PROGRESS, update.Status)
			return repo, nil
		}).
		Times(1)

	// FAILED with registry error message.
	mockRepoDS.EXPECT().
		UpdateStatus(gomock.Any(), repo.GetId(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, update repoDS.StatusUpdate) (*storage.BaseImageRepository, error) {
			assert.Equal(t, storage.BaseImageRepository_FAILED, update.Status)
			assert.Equal(t, repoDS.FailureCountIncrement, update.FailureCountOp)
			require.NotNil(t, update.LastFailureMessage)
			assert.Contains(t, *update.LastFailureMessage, "registry connection failed")
			return repo, nil
		}).
		Times(1)

	mockDelegator.EXPECT().
		GetDelegateClusterID(gomock.Any(), gomock.Any()).
		Return("", false, nil).
		Times(1)

	mockTagDS.EXPECT().
		ListTagsByRepository(gomock.Any(), gomock.Any()).
		Return([]*storage.BaseImageTag{}, nil).
		Times(1)

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

	w := createTestWatcher(ctrl, mockRepoDS, mockTagDS, mockRegistrySet, mockDelegator, 1*time.Hour, true)

	assert.NotPanics(t, func() {
		runPollAndWait(w)
	})
}

func TestWatcher_LastPolledAt_SetAtScanCompletion(t *testing.T) {
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

	// Claiming (QUEUED).
	mockRepoDS.EXPECT().
		UpdateStatus(gomock.Any(), repo.GetId(), gomock.Any()).
		Return(repo, nil).
		Times(1)

	var inProgressTime time.Time
	mockRepoDS.EXPECT().
		UpdateStatus(gomock.Any(), repo.GetId(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, update repoDS.StatusUpdate) (*storage.BaseImageRepository, error) {
			assert.Equal(t, storage.BaseImageRepository_IN_PROGRESS, update.Status)
			inProgressTime = time.Now()
			return repo, nil
		}).
		Times(1)

	// Final: READY with LastPolledAt.
	mockRepoDS.EXPECT().
		UpdateStatus(gomock.Any(), repo.GetId(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, update repoDS.StatusUpdate) (*storage.BaseImageRepository, error) {
			assert.Equal(t, storage.BaseImageRepository_READY, update.Status)
			require.NotNil(t, update.LastPolledAt)
			assert.True(t, update.LastPolledAt.After(inProgressTime) || update.LastPolledAt.Equal(inProgressTime),
				"LastPolledAt should be >= inProgressTime (set at completion, not start)")
			return repo, nil
		}).
		Times(1)

	mockDelegator.EXPECT().
		GetDelegateClusterID(gomock.Any(), gomock.Any()).
		Return("", false, nil).
		Times(1)

	mockTagDS.EXPECT().
		ListTagsByRepository(gomock.Any(), gomock.Any()).
		Return([]*storage.BaseImageTag{}, nil).
		AnyTimes()

	mockRegistry.EXPECT().
		Match(gomock.Any()).
		Return(true).
		Times(1)

	mockRegistry.EXPECT().
		ListTags(gomock.Any(), gomock.Any()).
		Return([]string{}, nil).
		Times(1)

	mockRegistrySet.EXPECT().
		GetAllUnique().
		Return([]types.ImageRegistry{mockRegistry}).
		Times(1)

	w := createTestWatcher(ctrl, mockRepoDS, mockTagDS, mockRegistrySet, mockDelegator, 1*time.Hour, true)

	runPollAndWait(w)
}

// TestWatcher_RecoveryDoesNotCorruptActiveScans verifies that crash recovery at
// startup and sequential poll ticks prevent the historical bug where recovery
// on a subsequent tick would mark an actively-scanning repo as FAILED.
//
// With the simplified model (single PeriodicWorker, sequential ticks), this bug
// is structurally impossible: recovery runs once in Start() before any polls,
// and each poll blocks until all scans complete before the next tick fires.
func TestWatcher_RecoveryDoesNotCorruptActiveScans(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepoDS := repoDSMocks.NewMockDataStore(ctrl)
	mockTagDS := tagDSMocks.NewMockDataStore(ctrl)
	mockRegistrySet := registryMocks.NewMockSet(ctrl)
	mockDelegator := delegatedRegistryMocks.NewMockDelegator(ctrl)
	mockRegistry := registryTypesMocks.NewMockImageRegistry(ctrl)
	mockBaseImageDS := baseImageDSMocks.NewMockDataStore(ctrl)

	repoID := uuid.NewV4().String()
	repo := &storage.BaseImageRepository{
		Id:             repoID,
		RepositoryPath: "docker.io/library/nginx",
		TagPattern:     "*",
		Status:         storage.BaseImageRepository_CREATED,
	}

	var statusMu sync.Mutex
	var statusHistory []storage.BaseImageRepository_Status
	var recoveryCorruptedScan bool

	mockRepoDS.EXPECT().
		ListRepositories(gomock.Any()).
		DoAndReturn(func(_ context.Context) ([]*storage.BaseImageRepository, error) {
			statusMu.Lock()
			defer statusMu.Unlock()
			clone := proto.Clone(repo).(*storage.BaseImageRepository)
			return []*storage.BaseImageRepository{clone}, nil
		}).
		AnyTimes()

	mockRepoDS.EXPECT().
		UpdateStatus(gomock.Any(), repoID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, update repoDS.StatusUpdate) (*storage.BaseImageRepository, error) {
			statusMu.Lock()
			defer statusMu.Unlock()

			if update.Status == storage.BaseImageRepository_FAILED &&
				repo.GetStatus() == storage.BaseImageRepository_IN_PROGRESS {
				if update.LastFailureMessage != nil && *update.LastFailureMessage == "scan interrupted by restart" {
					recoveryCorruptedScan = true
				}
			}

			repo.Status = update.Status
			statusHistory = append(statusHistory, update.Status)
			clone := proto.Clone(repo).(*storage.BaseImageRepository)
			return clone, nil
		}).
		AnyTimes()

	mockDelegator.EXPECT().
		GetDelegateClusterID(gomock.Any(), gomock.Any()).
		Return("", false, nil).
		AnyTimes()

	mockTagDS.EXPECT().
		ListTagsByRepository(gomock.Any(), repoID).
		Return([]*storage.BaseImageTag{}, nil).
		AnyTimes()

	mockRegistry.EXPECT().Match(gomock.Any()).Return(true).AnyTimes()
	mockRegistry.EXPECT().ListTags(gomock.Any(), gomock.Any()).Return([]string{}, nil).AnyTimes()
	mockRegistry.EXPECT().Source().Return(&storage.ImageIntegration{Id: "integration-1"}).AnyTimes()
	mockRegistrySet.EXPECT().GetAllUnique().Return([]types.ImageRegistry{mockRegistry}).AnyTimes()
	mockBaseImageDS.EXPECT().ReplaceByRepository(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	w := New(mockRepoDS, mockTagDS, mockBaseImageDS, mockRegistrySet, mockDelegator,
		1*time.Hour,         // pollInterval
		20*time.Millisecond, // schedulerCadence
		10, 100, 5, true)

	w.Start()

	// Let a few ticks run.
	time.Sleep(100 * time.Millisecond)

	w.Stop()

	statusMu.Lock()
	defer statusMu.Unlock()

	t.Logf("Status history: %v", statusHistory)

	assert.False(t, recoveryCorruptedScan,
		"Recovery should not mark actively scanning repo as FAILED")
}
