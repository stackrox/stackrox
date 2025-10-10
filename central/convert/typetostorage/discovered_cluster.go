package typetostorage

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cloudsources/discoveredclusters"
	"github.com/stackrox/rox/pkg/protocompat"
)

// DiscoveredCluster converts the given discoveredclusters.DiscoveredCluster
// to storage.DiscoveredCluster.
func DiscoveredCluster(cluster *discoveredclusters.DiscoveredCluster) *storage.DiscoveredCluster {
	id := discoveredclusters.GenerateDiscoveredClusterID(cluster)
	metadataId := cluster.GetID()
	name := cluster.GetName()
	clusterType := cluster.GetType()
	providerType := cluster.GetProviderType()
	region := cluster.GetRegion()
	firstDiscoveredAt := protocompat.ConvertTimeToTimestampOrNil(cluster.GetFirstDiscoveredAt())

	metadata := storage.DiscoveredCluster_Metadata_builder{
		Id:                &metadataId,
		Name:              &name,
		Type:              &clusterType,
		ProviderType:      &providerType,
		Region:            &region,
		FirstDiscoveredAt: firstDiscoveredAt,
	}.Build()

	status := cluster.GetStatus()
	sourceId := cluster.GetCloudSourceID()
	lastUpdatedAt := protocompat.TimestampNow()

	storageConfig := storage.DiscoveredCluster_builder{
		Id:            &id,
		Metadata:      metadata,
		Status:        &status,
		SourceId:      &sourceId,
		LastUpdatedAt: lastUpdatedAt,
	}.Build()
	return storageConfig
}
