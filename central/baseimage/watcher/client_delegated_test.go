package watcher

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	delegatedRegistryMocks "github.com/stackrox/rox/pkg/delegatedregistry/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestDelegatedRepositoryClient_ScanRepository_ReturnsNotImplementedError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockDelegator := delegatedRegistryMocks.NewMockDelegator(ctrl)

	client := NewDelegatedRepositoryClient(mockDelegator, "cluster-123")

	repo := &storage.BaseImageRepository{
		Id:             "test-id",
		RepositoryPath: "docker.io/library/nginx",
	}

	req := ScanRequest{
		Pattern:   "*",
		CheckTags: make(map[string]*storage.BaseImageTag),
		SkipTags:  make(map[string]struct{}),
	}

	// Iterator should yield a fatal error.
	var fatalErr error
	for _, err := range client.ScanRepository(context.Background(), repo, req) {
		if err != nil {
			fatalErr = err
			break
		}
	}

	require.Error(t, fatalErr)
	assert.Contains(t, fatalErr.Error(), "delegated repository scanning not implemented")
	assert.Contains(t, fatalErr.Error(), "cluster-123")
	assert.Contains(t, fatalErr.Error(), "ROX-31926/31927")
}

func TestDelegatedRepositoryClient_Name(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockDelegator := delegatedRegistryMocks.NewMockDelegator(ctrl)

	client := NewDelegatedRepositoryClient(mockDelegator, "cluster-456")

	assert.Equal(t, "delegated", client.Name())
}

func TestDelegatedRepositoryClient_ImplementsInterface(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockDelegator := delegatedRegistryMocks.NewMockDelegator(ctrl)

	client := NewDelegatedRepositoryClient(mockDelegator, "cluster-123")

	// Verify DelegatedRepositoryClient implements RepositoryClient interface.
	var _ RepositoryClient = client
}
