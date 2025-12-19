package watcher

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/baseimage/reposcan"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protocompat"
	registryMocks "github.com/stackrox/rox/pkg/registries/mocks"
	"github.com/stackrox/rox/pkg/registries/types"
	registryTypesMocks "github.com/stackrox/rox/pkg/registries/types/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestLocalScanner_ScanRepository_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRegistrySet := registryMocks.NewMockSet(ctrl)
	mockRegistry := registryTypesMocks.NewMockImageRegistry(ctrl)

	repo := &storage.BaseImageRepository{
		Id:             "test-id",
		RepositoryPath: "docker.io/library/nginx",
		TagPattern:     "1.*",
	}

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

	createdTime := time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC)
	protoTime := protocompat.ConvertTimeToTimestampOrNil(&createdTime)

	// Mock Metadata calls for the 3 matching tags (1.0, 1.1, 1.2).
	mockRegistry.EXPECT().
		Metadata(gomock.Any()).
		DoAndReturn(func(img *storage.Image) (*storage.ImageMetadata, error) {
			return &storage.ImageMetadata{
				V1: &storage.V1Metadata{
					Created: protoTime,
				},
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

	scanner := reposcan.NewLocalScanner(mockRegistrySet)

	req := reposcan.ScanRequest{
		Pattern:   "1.*",
		CheckTags: make(map[string]*storage.BaseImageTag),
		SkipTags:  make(map[string]struct{}),
	}

	var metadataEvents []reposcan.TagEvent
	for event, err := range scanner.ScanRepository(context.Background(), repo, req) {
		require.NoError(t, err)
		if event.Type == reposcan.TagEventMetadata {
			metadataEvents = append(metadataEvents, event)
		}
	}

	// Should have 3 metadata events for matching tags.
	require.Len(t, metadataEvents, 3)

	// Verify digests are set.
	for _, event := range metadataEvents {
		assert.NotEmpty(t, event.Metadata.ManifestDigest)
		assert.Contains(t, []string{"1.0", "1.1", "1.2"}, event.Tag)
	}
}

func TestLocalScanner_ScanRepository_NoMatchingRegistry(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRegistrySet := registryMocks.NewMockSet(ctrl)
	mockRegistry := registryTypesMocks.NewMockImageRegistry(ctrl)

	repo := &storage.BaseImageRepository{
		Id:             "test-id",
		RepositoryPath: "docker.io/library/nginx",
	}

	mockRegistry.EXPECT().
		Match(gomock.Any()).
		Return(false)

	mockRegistrySet.EXPECT().
		GetAllUnique().
		Return([]types.ImageRegistry{mockRegistry})

	scanner := reposcan.NewLocalScanner(mockRegistrySet)

	req := reposcan.ScanRequest{
		Pattern:   "*",
		CheckTags: make(map[string]*storage.BaseImageTag),
		SkipTags:  make(map[string]struct{}),
	}

	var fatalErr error
	for _, err := range scanner.ScanRepository(context.Background(), repo, req) {
		if err != nil {
			fatalErr = err
			break
		}
	}

	require.Error(t, fatalErr)
	assert.Contains(t, fatalErr.Error(), "no matching image integration found")
}

func TestLocalScanner_ScanRepository_ListTagsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRegistrySet := registryMocks.NewMockSet(ctrl)
	mockRegistry := registryTypesMocks.NewMockImageRegistry(ctrl)

	repo := &storage.BaseImageRepository{
		Id:             "test-id",
		RepositoryPath: "docker.io/library/nginx",
	}

	mockRegistry.EXPECT().
		Match(gomock.Any()).
		Return(true)

	mockRegistry.EXPECT().
		ListTags(gomock.Any(), gomock.Any()).
		Return(nil, errox.InvariantViolation.New("connection failed"))

	mockRegistrySet.EXPECT().
		GetAllUnique().
		Return([]types.ImageRegistry{mockRegistry})

	scanner := reposcan.NewLocalScanner(mockRegistrySet)

	req := reposcan.ScanRequest{
		Pattern:   "*",
		CheckTags: make(map[string]*storage.BaseImageTag),
		SkipTags:  make(map[string]struct{}),
	}

	var fatalErr error
	for _, err := range scanner.ScanRepository(context.Background(), repo, req) {
		if err != nil {
			fatalErr = err
			break
		}
	}

	require.Error(t, fatalErr)
	assert.Contains(t, fatalErr.Error(), "connection failed")
}

func TestLocalScanner_ScanRepository_InvalidRepositoryPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRegistrySet := registryMocks.NewMockSet(ctrl)

	repo := &storage.BaseImageRepository{
		Id:             "test-id",
		RepositoryPath: "", // Invalid empty path.
	}

	scanner := reposcan.NewLocalScanner(mockRegistrySet)

	req := reposcan.ScanRequest{
		Pattern:   "*",
		CheckTags: make(map[string]*storage.BaseImageTag),
		SkipTags:  make(map[string]struct{}),
	}

	var fatalErr error
	for _, err := range scanner.ScanRepository(context.Background(), repo, req) {
		if err != nil {
			fatalErr = err
			break
		}
	}

	require.Error(t, fatalErr)
	assert.Contains(t, fatalErr.Error(), "parsing repository path")
}

func TestLocalScanner_ScanRepository_EmptyResult(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRegistrySet := registryMocks.NewMockSet(ctrl)
	mockRegistry := registryTypesMocks.NewMockImageRegistry(ctrl)

	repo := &storage.BaseImageRepository{
		Id:             "test-id",
		RepositoryPath: "docker.io/library/nginx",
	}

	mockRegistry.EXPECT().
		Match(gomock.Any()).
		Return(true)

	mockRegistry.EXPECT().
		ListTags(gomock.Any(), gomock.Any()).
		Return([]string{}, nil)

	mockRegistrySet.EXPECT().
		GetAllUnique().
		Return([]types.ImageRegistry{mockRegistry})

	scanner := reposcan.NewLocalScanner(mockRegistrySet)

	req := reposcan.ScanRequest{
		Pattern:   "*",
		CheckTags: make(map[string]*storage.BaseImageTag),
		SkipTags:  make(map[string]struct{}),
	}

	var eventCount int
	for _, err := range scanner.ScanRepository(context.Background(), repo, req) {
		require.NoError(t, err)
		eventCount++
	}

	// No tags means no metadata events.
	assert.Equal(t, 0, eventCount)
}

func TestLocalScanner_Name(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRegistrySet := registryMocks.NewMockSet(ctrl)

	scanner := reposcan.NewLocalScanner(mockRegistrySet)

	assert.Equal(t, "local", scanner.Name())
}

func TestLocalScanner_ImplementsInterface(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRegistrySet := registryMocks.NewMockSet(ctrl)

	scanner := reposcan.NewLocalScanner(mockRegistrySet)

	// Verify LocalScanner implements Scanner interface.
	var _ reposcan.Scanner = scanner
}

func TestLocalScanner_ScanRepository_DeletionEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRegistrySet := registryMocks.NewMockSet(ctrl)
	mockRegistry := registryTypesMocks.NewMockImageRegistry(ctrl)

	repo := &storage.BaseImageRepository{
		Id:             "test-id",
		RepositoryPath: "docker.io/library/nginx",
		TagPattern:     "*",
	}

	// Registry returns only "1.0" - the other cached tags are "deleted".
	mockRegistry.EXPECT().
		Match(gomock.Any()).
		Return(true).
		AnyTimes()

	mockRegistry.EXPECT().
		ListTags(gomock.Any(), gomock.Any()).
		Return([]string{"1.0"}, nil)

	mockRegistry.EXPECT().
		Source().
		Return(&storage.ImageIntegration{Id: "integration-1"}).
		AnyTimes()

	createdTime := time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC)
	protoTime := protocompat.ConvertTimeToTimestampOrNil(&createdTime)

	mockRegistry.EXPECT().
		Metadata(gomock.Any()).
		Return(&storage.ImageMetadata{
			V1: &storage.V1Metadata{
				Created: protoTime,
			},
			V2: &storage.V2Metadata{Digest: "sha256:abc123"},
		}, nil)

	mockRegistrySet.EXPECT().
		GetAllUnique().
		Return([]types.ImageRegistry{mockRegistry}).
		AnyTimes()

	scanner := reposcan.NewLocalScanner(mockRegistrySet)

	// CheckTags contains tags that were previously cached.
	req := reposcan.ScanRequest{
		Pattern: "*",
		CheckTags: map[string]*storage.BaseImageTag{
			"1.0": {Tag: "1.0", ManifestDigest: "sha256:old"},
			"1.1": {Tag: "1.1", ManifestDigest: "sha256:deleted1"},
			"1.2": {Tag: "1.2", ManifestDigest: "sha256:deleted2"},
		},
		SkipTags: map[string]struct{}{
			"1.3": {}, // Also deleted from registry.
		},
	}

	var metadataEvents, deletedEvents []reposcan.TagEvent
	for event, err := range scanner.ScanRepository(context.Background(), repo, req) {
		require.NoError(t, err)
		switch event.Type {
		case reposcan.TagEventMetadata:
			metadataEvents = append(metadataEvents, event)
		case reposcan.TagEventDeleted:
			deletedEvents = append(deletedEvents, event)
		}
	}

	// Should have 1 metadata event for "1.0".
	require.Len(t, metadataEvents, 1)
	assert.Equal(t, "1.0", metadataEvents[0].Tag)

	// Should have 3 deletion events for "1.1", "1.2" (from CheckTags), "1.3" (from SkipTags).
	require.Len(t, deletedEvents, 3)
	deletedTags := make(map[string]bool)
	for _, event := range deletedEvents {
		deletedTags[event.Tag] = true
	}
	assert.True(t, deletedTags["1.1"], "expected 1.1 to be deleted")
	assert.True(t, deletedTags["1.2"], "expected 1.2 to be deleted")
	assert.True(t, deletedTags["1.3"], "expected 1.3 to be deleted")
}

func TestLocalScanner_ScanRepository_SkipTags(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRegistrySet := registryMocks.NewMockSet(ctrl)
	mockRegistry := registryTypesMocks.NewMockImageRegistry(ctrl)

	repo := &storage.BaseImageRepository{
		Id:             "test-id",
		RepositoryPath: "docker.io/library/nginx",
		TagPattern:     "*",
	}

	mockRegistry.EXPECT().
		Match(gomock.Any()).
		Return(true).
		AnyTimes()

	// Registry returns 3 tags, but we skip 2 of them.
	mockRegistry.EXPECT().
		ListTags(gomock.Any(), gomock.Any()).
		Return([]string{"1.0", "1.1", "1.2"}, nil)

	mockRegistry.EXPECT().
		Source().
		Return(&storage.ImageIntegration{Id: "integration-1"}).
		AnyTimes()

	createdTime := time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC)
	protoTime := protocompat.ConvertTimeToTimestampOrNil(&createdTime)

	// Only 1 metadata call for "1.0" - other tags are skipped.
	mockRegistry.EXPECT().
		Metadata(gomock.Any()).
		DoAndReturn(func(img *storage.Image) (*storage.ImageMetadata, error) {
			assert.Equal(t, "1.0", img.GetName().GetTag(), "only 1.0 should be fetched")
			return &storage.ImageMetadata{
				V1: &storage.V1Metadata{
					Created: protoTime,
				},
				V2: &storage.V2Metadata{Digest: "sha256:abc123"},
			}, nil
		})

	mockRegistrySet.EXPECT().
		GetAllUnique().
		Return([]types.ImageRegistry{mockRegistry}).
		AnyTimes()

	scanner := reposcan.NewLocalScanner(mockRegistrySet)

	req := reposcan.ScanRequest{
		Pattern:   "*",
		CheckTags: make(map[string]*storage.BaseImageTag),
		SkipTags: map[string]struct{}{
			"1.1": {}, // Skip fetching metadata.
			"1.2": {}, // Skip fetching metadata.
		},
	}

	var metadataEvents []reposcan.TagEvent
	for event, err := range scanner.ScanRepository(context.Background(), repo, req) {
		require.NoError(t, err)
		if event.Type == reposcan.TagEventMetadata {
			metadataEvents = append(metadataEvents, event)
		}
	}

	// Should only have 1 metadata event for "1.0".
	require.Len(t, metadataEvents, 1)
	assert.Equal(t, "1.0", metadataEvents[0].Tag)
}

func TestLocalScanner_ScanRepository_MetadataFetchError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRegistrySet := registryMocks.NewMockSet(ctrl)
	mockRegistry := registryTypesMocks.NewMockImageRegistry(ctrl)

	repo := &storage.BaseImageRepository{
		Id:             "test-id",
		RepositoryPath: "docker.io/library/nginx",
		TagPattern:     "*",
	}

	mockRegistry.EXPECT().
		Match(gomock.Any()).
		Return(true).
		AnyTimes()

	mockRegistry.EXPECT().
		ListTags(gomock.Any(), gomock.Any()).
		Return([]string{"1.0", "1.1"}, nil)

	mockRegistry.EXPECT().
		Source().
		Return(&storage.ImageIntegration{Id: "integration-1"}).
		AnyTimes()

	createdTime := time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC)
	protoTime := protocompat.ConvertTimeToTimestampOrNil(&createdTime)

	// First tag succeeds, second tag fails.
	mockRegistry.EXPECT().
		Metadata(gomock.Any()).
		DoAndReturn(func(img *storage.Image) (*storage.ImageMetadata, error) {
			if img.GetName().GetTag() == "1.1" {
				return nil, errox.InvariantViolation.New("manifest not found")
			}
			return &storage.ImageMetadata{
				V1: &storage.V1Metadata{
					Created: protoTime,
				},
				V2: &storage.V2Metadata{Digest: "sha256:abc123"},
			}, nil
		}).
		Times(2)

	mockRegistrySet.EXPECT().
		GetAllUnique().
		Return([]types.ImageRegistry{mockRegistry}).
		AnyTimes()

	scanner := reposcan.NewLocalScanner(mockRegistrySet)

	req := reposcan.ScanRequest{
		Pattern:   "*",
		CheckTags: make(map[string]*storage.BaseImageTag),
		SkipTags:  make(map[string]struct{}),
	}

	var metadataEvents, errorEvents []reposcan.TagEvent
	for event, err := range scanner.ScanRepository(context.Background(), repo, req) {
		require.NoError(t, err, "iterator error should be nil, errors come via events")
		switch event.Type {
		case reposcan.TagEventMetadata:
			metadataEvents = append(metadataEvents, event)
		case reposcan.TagEventError:
			errorEvents = append(errorEvents, event)
		}
	}

	// Should have 1 metadata event and 1 error event.
	require.Len(t, metadataEvents, 1)
	require.Len(t, errorEvents, 1)

	assert.Equal(t, "1.0", metadataEvents[0].Tag)
	assert.Equal(t, "1.1", errorEvents[0].Tag)
	assert.NotNil(t, errorEvents[0].Error)
	assert.Contains(t, errorEvents[0].Error.Error(), "manifest not found")
}
