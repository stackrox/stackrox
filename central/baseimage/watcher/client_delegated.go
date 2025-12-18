package watcher

import (
	"context"
	"fmt"
	"iter"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/delegatedregistry"
)

// DelegatedRepositoryClient delegates scanning to a secured cluster.
type DelegatedRepositoryClient struct {
	delegator delegatedregistry.Delegator
	clusterID string
}

// NewDelegatedRepositoryClient creates a DelegatedRepositoryClient.
func NewDelegatedRepositoryClient(delegator delegatedregistry.Delegator, clusterID string) *DelegatedRepositoryClient {
	return &DelegatedRepositoryClient{delegator: delegator, clusterID: clusterID}
}

// Name implements RepositoryClient.
func (c *DelegatedRepositoryClient) Name() string {
	return "delegated"
}

// ScanRepository implements RepositoryClient.
func (c *DelegatedRepositoryClient) ScanRepository(
	ctx context.Context,
	repo *storage.BaseImageRepository,
	req ScanRequest,
) iter.Seq2[TagEvent, error] {
	return func(yield func(TagEvent, error) bool) {
		yield(TagEvent{}, fmt.Errorf(
			"delegated repository scanning not implemented for cluster %s (ROX-31926/31927)",
			c.clusterID))
	}
}
