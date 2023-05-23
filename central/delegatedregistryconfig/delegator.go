package delegatedregistryconfig

import (
	"context"
	"errors"

	"github.com/stackrox/rox/generated/storage"
)

var (
	// ErrInvalidCluster indicates a cluster is invalid or missing
	ErrInvalidCluster = errors.New("invalid cluster")
)

type Delegator interface {
	// GetDelegateClusterID returns the cluster id that should enrich this image (if any)
	//
	// If cluster id is populated and/or ErrInvalidCluster returned then enrichment of this
	// image is meant to be delegated
	//
	// Any other error indicates an issue to obtain the config
	GetDelegateClusterID(ctx context.Context, image *storage.Image) (string, error)

	// DelegateEnrichImage sends an enrichment request to the cluster represented by cluster id
	DelegateEnrichImage(ctx context.Context, image *storage.Image, clusterID string) error
}
