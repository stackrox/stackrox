package watcher

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/baseimage/reposcan"
	delegatedRegistryMocks "github.com/stackrox/rox/pkg/delegatedregistry/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestDelegatedScanner_ScanRepository_ReturnsNotImplementedError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockDelegator := delegatedRegistryMocks.NewMockDelegator(ctrl)

	scanner := NewDelegatedScanner(mockDelegator, "cluster-123")

	repo := &storage.BaseImageRepository{
		Id:             "test-id",
		RepositoryPath: "docker.io/library/nginx",
	}

	req := reposcan.ScanRequest{
		Pattern:   "*",
		CheckTags: make(map[string]*storage.BaseImageTag),
		SkipTags:  make(map[string]struct{}),
	}

	// Iterator should yield a fatal error.
	var fatalErr error
	for _, err := range scanner.ScanRepository(context.Background(), repo, req) {
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

func TestDelegatedScanner_Name(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockDelegator := delegatedRegistryMocks.NewMockDelegator(ctrl)

	scanner := NewDelegatedScanner(mockDelegator, "cluster-456")

	assert.Equal(t, "delegated", scanner.Name())
}

func TestDelegatedScanner_ImplementsInterface(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockDelegator := delegatedRegistryMocks.NewMockDelegator(ctrl)

	scanner := NewDelegatedScanner(mockDelegator, "cluster-123")

	// Verify DelegatedScanner implements Scanner interface.
	var _ reposcan.Scanner = scanner
}
