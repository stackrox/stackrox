package delegatedregistry

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Delegator defines an interface for delegating image enrichment requests to secured clusters.
//
//go:generate mockgen-wrapper
type Delegator interface {
	// GetDelegateClusterID returns the cluster id that should enrich this image (if any) and
	// true if enrichment should be delegated to a secured cluster, false otherwise.
	GetDelegateClusterID(ctx context.Context, imgName *storage.ImageName) (string, bool, error)

	// DelegateScanImage sends a scan request to the provided cluster.
	DelegateScanImage(ctx context.Context, imgName *storage.ImageName, clusterID string, force bool) (*storage.Image, error)
}
