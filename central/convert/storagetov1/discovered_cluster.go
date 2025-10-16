package storagetov1

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// DiscoveredCluster converts the given storage.DiscoveredCluster to a v1.DiscoveredCluster.
func DiscoveredCluster(discoveredCluster *storage.DiscoveredCluster) *v1.DiscoveredCluster {
	metadata := discoveredCluster.GetMetadata()
	dm := &v1.DiscoveredCluster_Metadata{}
	dm.SetId(metadata.GetId())
	dm.SetName(metadata.GetName())
	dm.SetType(discoveredClusterToV1TypeEnum(metadata.GetType()))
	dm.SetProviderType(discoveredClusterToV1ProviderTypeEnum(metadata.GetProviderType()))
	dm.SetRegion(metadata.GetRegion())
	dm.SetFirstDiscoveredAt(metadata.GetFirstDiscoveredAt())
	dc := &v1.DiscoveredCluster_CloudSource{}
	dc.SetId(discoveredCluster.GetSourceId())
	v1DiscoveredCluster := &v1.DiscoveredCluster{}
	v1DiscoveredCluster.SetId(discoveredCluster.GetId())
	v1DiscoveredCluster.SetMetadata(dm)
	v1DiscoveredCluster.SetStatus(discoveredClusterToV1StatusEnum(discoveredCluster.GetStatus()))
	v1DiscoveredCluster.SetSource(dc)
	return v1DiscoveredCluster
}

// DiscoveredClusterList converts the given ...*storage.DiscoveredCluster to a []*v1.DiscoveredCluster.
func DiscoveredClusterList(discoveredClusters ...*storage.DiscoveredCluster) []*v1.DiscoveredCluster {
	v1DiscoveredClusters := make([]*v1.DiscoveredCluster, 0, len(discoveredClusters))
	for _, dc := range discoveredClusters {
		v1DiscoveredClusters = append(v1DiscoveredClusters, DiscoveredCluster(dc))
	}
	return v1DiscoveredClusters
}

func discoveredClusterToV1TypeEnum(val storage.ClusterMetadata_Type) v1.DiscoveredCluster_Metadata_Type {
	return v1.DiscoveredCluster_Metadata_Type(
		v1.DiscoveredCluster_Metadata_Type_value[val.String()],
	)
}

func discoveredClusterToV1ProviderTypeEnum(val storage.DiscoveredCluster_Metadata_ProviderType) v1.DiscoveredCluster_Metadata_ProviderType {
	return v1.DiscoveredCluster_Metadata_ProviderType(
		v1.DiscoveredCluster_Metadata_ProviderType_value[val.String()],
	)
}

func discoveredClusterToV1StatusEnum(val storage.DiscoveredCluster_Status) v1.DiscoveredCluster_Status {
	return v1.DiscoveredCluster_Status(
		v1.DiscoveredCluster_Status_value[val.String()],
	)
}
