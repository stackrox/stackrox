package storagetov1

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// DiscoveredCluster converts the given storage.DiscoveredCluster to a v1.DiscoveredCluster.
func DiscoveredCluster(discoveredCluster *storage.DiscoveredCluster) *v1.DiscoveredCluster {
	metadata := discoveredCluster.GetMetadata()

	v1Metadata := &v1.DiscoveredCluster_Metadata{}
	v1Metadata.SetId(metadata.GetId())
	v1Metadata.SetName(metadata.GetName())
	v1Metadata.SetType(discoveredClusterToV1TypeEnum(metadata.GetType()))
	v1Metadata.SetProviderType(discoveredClusterToV1ProviderTypeEnum(metadata.GetProviderType()))
	v1Metadata.SetRegion(metadata.GetRegion())
	v1Metadata.SetFirstDiscoveredAt(metadata.GetFirstDiscoveredAt())

	source := &v1.DiscoveredCluster_CloudSource{}
	source.SetId(discoveredCluster.GetSourceId())

	v1DiscoveredCluster := &v1.DiscoveredCluster{}
	v1DiscoveredCluster.SetId(discoveredCluster.GetId())
	v1DiscoveredCluster.SetMetadata(v1Metadata)
	v1DiscoveredCluster.SetStatus(discoveredClusterToV1StatusEnum(discoveredCluster.GetStatus()))
	v1DiscoveredCluster.SetSource(source)

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
