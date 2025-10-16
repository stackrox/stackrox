package typetostorage

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cloudsources/discoveredclusters"
	"github.com/stackrox/rox/pkg/protocompat"
)

// DiscoveredCluster converts the given discoveredclusters.DiscoveredCluster
// to storage.DiscoveredCluster.
func DiscoveredCluster(cluster *discoveredclusters.DiscoveredCluster) *storage.DiscoveredCluster {
	dm := &storage.DiscoveredCluster_Metadata{}
	dm.SetId(cluster.GetID())
	dm.SetName(cluster.GetName())
	dm.SetType(cluster.GetType())
	dm.SetProviderType(cluster.GetProviderType())
	dm.SetRegion(cluster.GetRegion())
	dm.SetFirstDiscoveredAt(protocompat.ConvertTimeToTimestampOrNil(cluster.GetFirstDiscoveredAt()))
	storageConfig := &storage.DiscoveredCluster{}
	storageConfig.SetId(discoveredclusters.GenerateDiscoveredClusterID(cluster))
	storageConfig.SetMetadata(dm)
	storageConfig.SetStatus(cluster.GetStatus())
	storageConfig.SetSourceId(cluster.GetCloudSourceID())
	storageConfig.SetLastUpdatedAt(protocompat.TimestampNow())
	return storageConfig
}
