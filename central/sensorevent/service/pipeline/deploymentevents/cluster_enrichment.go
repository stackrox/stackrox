package deploymentevents

import (
	"bitbucket.org/stack-rox/apollo/central/cluster/datastore"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

func newClusterEnrichment(clusters datastore.DataStore) *clusterEnrichmentImpl {
	return &clusterEnrichmentImpl{
		clusters: clusters,
	}
}

type clusterEnrichmentImpl struct {
	clusters datastore.DataStore
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
