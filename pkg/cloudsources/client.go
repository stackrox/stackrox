package cloudsources

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cloudsources/discoveredclusters"
	"github.com/stackrox/rox/pkg/cloudsources/paladin"
)

// Client exposes functionality to fetch discovered cluster from cloud sources.
//
//go:generate mockgen-wrapper
type Client interface {
	GetDiscoveredClusters(ctx context.Context) ([]*discoveredclusters.DiscoveredCluster, error)
}

// NewClientForCloudSource creates a new Client based on the cloud source to fetch discovered clusters.
func NewClientForCloudSource(source *storage.CloudSource) Client {
	// For the time being, this only supports paladin cloud clients.
	switch source.GetType() {
	case storage.CloudSource_TYPE_PALADIN_CLOUD:
		return paladin.NewClient(source)
	default:
		return nil
	}
}
