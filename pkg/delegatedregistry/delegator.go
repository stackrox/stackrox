package delegatedregistry

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
)

// ErrNoClusterSpecified is returned when an ad-hoc scanning request is missing a cluster ID
// in the delegated scanning configuration.
var ErrNoClusterSpecified = errox.InvalidArgs.New("no ad-hoc cluster ID specified in the delegated scanning config")

// Delegator defines an interface for delegating image enrichment requests to secured clusters.
//
//go:generate mockgen-wrapper
type Delegator interface {
	// GetDelegateClusterID returns the cluster id that should enrich this image (if any) and
	// true if enrichment should be delegated to a secured cluster, false otherwise.
	GetDelegateClusterID(ctx context.Context, imgName *storage.ImageName) (string, bool, error)

	// DelegateScanImage sends a scan request to the provided cluster.
	DelegateScanImage(ctx context.Context, imgName *storage.ImageName, clusterID string, namespace string, force bool) (*storage.Image, error)

	// ValidateCluster returns nil if a cluster is a valid target for delegation, returns an
	// error otherwise.
	ValidateCluster(clusterID string) error
}
