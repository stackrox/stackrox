package storagetotype

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cloudsources/discoveredclusters"
	"github.com/stackrox/rox/pkg/protocompat"
)

// DiscoveredCluster converts the given storage.DiscoveredCluster
// to discoveredclusters.DiscoveredCluster.
func DiscoveredCluster(cluster *storage.DiscoveredCluster) *discoveredclusters.DiscoveredCluster {
	discoveredCluster := &discoveredclusters.DiscoveredCluster{
		ID:            cluster.GetMetadata().GetId(),
		Name:          cluster.GetMetadata().GetName(),
		Type:          cluster.GetMetadata().GetType(),
		ProviderType:  cluster.GetMetadata().GetProviderType(),
		Region:        cluster.GetMetadata().GetRegion(),
		Status:        cluster.GetStatus(),
		CloudSourceID: cluster.GetSourceId(),
	}
	if cluster.GetMetadata().GetFirstDiscoveredAt() != nil {
		firstDiscoveredAt, err := protocompat.ConvertTimestampToTimeOrError(cluster.GetMetadata().GetFirstDiscoveredAt())
		if err == nil {
			discoveredCluster.FirstDiscoveredAt = &firstDiscoveredAt
		}
	}
	return discoveredCluster
}

// DiscoveredClusters converts the given list of storage.DiscoveredCluster
// to a list of discoveredclusters.DiscoveredCluster.
func DiscoveredClusters(clusters ...*storage.DiscoveredCluster) []*discoveredclusters.DiscoveredCluster {
	convClusters := make([]*discoveredclusters.DiscoveredCluster, 0, len(clusters))
	for _, cluster := range clusters {
		convClusters = append(convClusters, DiscoveredCluster(cluster))
	}
	return convClusters
}
