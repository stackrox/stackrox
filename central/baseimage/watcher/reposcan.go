package watcher

import (
	"context"
	"fmt"
	"iter"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/baseimage/reposcan"
	"github.com/stackrox/rox/pkg/delegatedregistry"
)

// DelegatedScanner delegates scanning to a secured cluster.
type DelegatedScanner struct {
	delegator delegatedregistry.Delegator
	clusterID string
}

// NewDelegatedScanner creates a DelegatedScanner.
func NewDelegatedScanner(delegator delegatedregistry.Delegator, clusterID string) *DelegatedScanner {
	return &DelegatedScanner{delegator: delegator, clusterID: clusterID}
}

// Name implements reposcan.Scanner.
func (c *DelegatedScanner) Name() string {
	return "delegated"
}

// ScanRepository implements reposcan.Scanner.
func (c *DelegatedScanner) ScanRepository(
	ctx context.Context,
	repo *storage.BaseImageRepository,
	req reposcan.ScanRequest,
) iter.Seq2[reposcan.TagEvent, error] {
	return func(yield func(reposcan.TagEvent, error) bool) {
		yield(reposcan.TagEvent{}, fmt.Errorf(
			"delegated repository scanning not implemented for cluster %s (ROX-31926/31927)",
			c.clusterID))
	}
}
