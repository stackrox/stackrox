package cloudsources

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cloudsources/discoveredclusters"
	"github.com/stackrox/rox/pkg/cloudsources/ocm"
	"github.com/stackrox/rox/pkg/cloudsources/opts"
	"github.com/stackrox/rox/pkg/cloudsources/paladin"
	"github.com/stackrox/rox/pkg/errox"
)

// Client exposes functionality to fetch discovered cluster from cloud sources.
//
//go:generate mockgen-wrapper
type Client interface {
	GetDiscoveredClusters(ctx context.Context) ([]*discoveredclusters.DiscoveredCluster, error)

	Ping(ctx context.Context) error
}

// NewClientForCloudSource creates a new Client based on the cloud source to fetch discovered clusters.
func NewClientForCloudSource(ctx context.Context, source *storage.CloudSource, options ...opts.ClientOpts) (Client, error) {
	switch source.GetType() {
	case storage.CloudSource_TYPE_PALADIN_CLOUD:
		return paladin.NewClient(source, options...), nil
	case storage.CloudSource_TYPE_OCM:
		return ocm.NewClient(ctx, source, options...)
	default:
		return nil, errox.InvalidArgs.Newf("unsupported type %q given", source.GetType().String())
	}
}
