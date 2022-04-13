package deploymentevents

import (
	"context"

	"github.com/stackrox/stackrox/central/cluster/datastore"
	"github.com/stackrox/stackrox/generated/storage"
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
	d.ClusterName = ""

	clusterName, clusterExists, err := s.clusters.GetClusterName(ctx, d.ClusterId)
	switch {
	case err != nil:
		log.Warnf("Couldn't get name of cluster: %s", err)
	case !clusterExists:
		log.Warnf("Couldn't find cluster '%s'", d.ClusterId)
	default:
		d.ClusterName = clusterName
	}
	return nil
}
