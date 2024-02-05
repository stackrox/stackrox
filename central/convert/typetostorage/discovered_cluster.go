package typetostorage

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/discoveredclusters"
)

// DiscoveredCluster converts the given discoveredclusters.DiscoveredCluster
// to storage.DiscoveredCluster.
func DiscoveredCluster(cluster *discoveredclusters.DiscoveredCluster) *storage.DiscoveredCluster {
	storageConfig := &storage.DiscoveredCluster{
		Id: discoveredclusters.GenerateDiscoveredClusterID(cluster),
		Metadata: &storage.DiscoveredCluster_Metadata{
			Id:                cluster.GetID(),
			Name:              cluster.GetName(),
			Type:              cluster.GetType(),
			ProviderType:      cluster.GetProviderType(),
			Region:            cluster.GetRegion(),
			FirstDiscoveredAt: cluster.GetFirstDiscoveredAt(),
		},
		Status:        cluster.GetStatus(),
		SourceId:      cluster.GetCloudSourceID(),
		LastUpdatedAt: types.TimestampNow(),
	}

	return storageConfig
}
