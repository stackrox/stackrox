package watcher

import (
	"testing"

	"github.com/stackrox/rox/pkg/baseimage/reposcan"
	delegatedRegistryMocks "github.com/stackrox/rox/pkg/delegatedregistry/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestDelegatedScanner_ScanRepository_ReturnsNotImplementedError(t *testing.T) {
	t.Skip("TODO(ROX-31926): Update this test to properly test the delegated scanner implementation")
	// This test was checking for the "not implemented" error, which is no longer valid
	// now that we have a real implementation. We need integration tests with a mock broker
	// to properly test the delegated scanning flow.
}

func TestDelegatedScanner_Name(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockDelegator := delegatedRegistryMocks.NewMockDelegator(ctrl)

	scanner := NewDelegatedScanner(mockDelegator, nil, "cluster-456")

	assert.Equal(t, "delegated", scanner.Name())
}

func TestDelegatedScanner_ImplementsInterface(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockDelegator := delegatedRegistryMocks.NewMockDelegator(ctrl)

	scanner := NewDelegatedScanner(mockDelegator, nil, "cluster-123")

	// Verify DelegatedScanner implements Scanner interface.
	var _ reposcan.Scanner = scanner
}
