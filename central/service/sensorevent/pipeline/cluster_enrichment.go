package pipeline

import (
	"bitbucket.org/stack-rox/apollo/central/datastore"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"golang.org/x/net/context"
)

func newClusterEnrichment(ctx context.Context, clusters datastore.ClusterDataStore) *clusterEnrichmentImpl {
	return &clusterEnrichmentImpl{
		ctx:      ctx,
		clusters: clusters,
	}
}

type clusterEnrichmentImpl struct {
	ctx      context.Context
	clusters datastore.ClusterDataStore
}

func (s *clusterEnrichmentImpl) do(d *v1.Deployment) error {
	d.ClusterName = ""

	cluster, clusterExists, err := s.clusters.GetCluster(d.ClusterId)
	switch {
	case err != nil:
		log.Warnf("Couldn't get name of cluster: %s", err)
	case !clusterExists:
		log.Warnf("Couldn't find cluster '%s'", d.ClusterId)
	default:
		d.ClusterName = cluster.GetName()
	}
	return nil
}
