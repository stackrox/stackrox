package delegatedregistryconfig

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

type Delegator interface {
	// GetDelegateClusterID returns the cluster id that should enrich this image (if any) and
	// true if enrichment should be delegated to a secured cluster, false otherwise
	GetDelegateClusterID(ctx context.Context, image *storage.Image) (string, bool, error)

	// DelegateEnrichImage sends an enrichment request the provided cluster
	DelegateEnrichImage(ctx context.Context, image *storage.Image, clusterID string) error
}
