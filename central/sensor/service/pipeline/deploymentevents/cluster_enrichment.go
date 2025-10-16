package deploymentevents

import (
	"context"

	"github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/generated/storage"
)

func newClusterEnrichment(clusters datastore.DataStore) *clusterEnrichmentImpl {
	return &clusterEnrichmentImpl{
		clusters: clusters,
	}
}

type clusterEnrichmentImpl struct {
	clusters datastore.DataStore
}

func (s *clusterEnrichmentImpl) do(ctx context.Context, d *storage.Deployment) error {
	d.SetClusterName("")

	clusterName, clusterExists, err := s.clusters.GetClusterName(ctx, d.GetClusterId())
	switch {
	case err != nil:
		log.Warnf("Couldn't get name of cluster: %s", err)
	case !clusterExists:
		log.Warnf("Couldn't find cluster '%s'", d.GetClusterId())
	default:
		d.SetClusterName(clusterName)
	}
	return nil
}
