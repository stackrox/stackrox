package watcher

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	registryMocks "github.com/stackrox/rox/pkg/registries/mocks"
	"github.com/stackrox/rox/pkg/registries/types"
	registryTypesMocks "github.com/stackrox/rox/pkg/registries/types/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestLocalRepositoryClient_ScanRepository_Success(t *testing.T) {
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

	mockRegistrySet.EXPECT().
		GetAllUnique().
		Return([]types.ImageRegistry{mockRegistry}).
		AnyTimes()

	client := NewLocalRepositoryClient(mockRegistrySet)

	req := ScanRequest{
		Pattern:   "1.*",
		CheckTags: make(map[string]struct{}),
		SkipTags:  make(map[string]struct{}),
	}

	var tags []string
	for event, err := range client.ScanRepository(context.Background(), repo, req) {
		require.NoError(t, err)
		tags = append(tags, event.Tag)
	}

	// Should have 3 matching tags.
	require.Len(t, tags, 3)
	assert.Contains(t, tags, "1.0")
	assert.Contains(t, tags, "1.1")
	assert.Contains(t, tags, "1.2")
}

func TestLocalRepositoryClient_ScanRepository_NoMatchingRegistry(t *testing.T) {
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

	client := NewLocalRepositoryClient(mockRegistrySet)

	req := ScanRequest{
		Pattern:   "*",
		CheckTags: make(map[string]struct{}),
		SkipTags:  make(map[string]struct{}),
	}

	var fatalErr error
	for _, err := range client.ScanRepository(context.Background(), repo, req) {
		if err != nil {
			fatalErr = err
			break
		}
	}

	require.Error(t, fatalErr)
	assert.Contains(t, fatalErr.Error(), "no matching image integration found")
}

func TestLocalRepositoryClient_ScanRepository_ListTagsError(t *testing.T) {
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

	client := NewLocalRepositoryClient(mockRegistrySet)

	req := ScanRequest{
		Pattern:   "*",
		CheckTags: make(map[string]struct{}),
		SkipTags:  make(map[string]struct{}),
	}

	var fatalErr error
	for _, err := range client.ScanRepository(context.Background(), repo, req) {
		if err != nil {
			fatalErr = err
			break
		}
	}

	require.Error(t, fatalErr)
	assert.Contains(t, fatalErr.Error(), "connection failed")
}

func TestLocalRepositoryClient_ScanRepository_InvalidRepositoryPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRegistrySet := registryMocks.NewMockSet(ctrl)

	repo := &storage.BaseImageRepository{
		Id:             "test-id",
		RepositoryPath: "", // Invalid empty path.
	}

	client := NewLocalRepositoryClient(mockRegistrySet)

	req := ScanRequest{
		Pattern:   "*",
		CheckTags: make(map[string]struct{}),
		SkipTags:  make(map[string]struct{}),
	}

	var fatalErr error
	for _, err := range client.ScanRepository(context.Background(), repo, req) {
		if err != nil {
			fatalErr = err
			break
		}
	}

	require.Error(t, fatalErr)
	assert.Contains(t, fatalErr.Error(), "parsing repository path")
}

func TestLocalRepositoryClient_ScanRepository_EmptyResult(t *testing.T) {
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

	client := NewLocalRepositoryClient(mockRegistrySet)

	req := ScanRequest{
		Pattern:   "*",
		CheckTags: make(map[string]struct{}),
		SkipTags:  make(map[string]struct{}),
	}

	var tags []string
	for event, err := range client.ScanRepository(context.Background(), repo, req) {
		require.NoError(t, err)
		tags = append(tags, event.Tag)
	}

	assert.Empty(t, tags)
}

func TestLocalRepositoryClient_Name(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRegistrySet := registryMocks.NewMockSet(ctrl)

	client := NewLocalRepositoryClient(mockRegistrySet)

	assert.Equal(t, "local", client.Name())
}

func TestLocalRepositoryClient_ImplementsInterface(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRegistrySet := registryMocks.NewMockSet(ctrl)

	client := NewLocalRepositoryClient(mockRegistrySet)

	// Verify LocalRepositoryClient implements RepositoryClient interface.
	var _ RepositoryClient = client
}
