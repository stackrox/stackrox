package fixtures

import (
	"fmt"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/pkg/discoveredclusters"
	"github.com/stackrox/rox/generated/storage"
)

// GetDiscoveredCluster returns a mock discovered cluster.
func GetDiscoveredCluster() *discoveredclusters.DiscoveredCluster {
	return &discoveredclusters.DiscoveredCluster{
		ID:                "my-id",
		Name:              "my-cluster",
		Type:              storage.ClusterMetadata_GKE,
		ProviderType:      storage.DiscoveredCluster_Metadata_PROVIDER_TYPE_GCP,
		Region:            "us-east-1",
		FirstDiscoveredAt: types.TimestampNow(),
		Status:            storage.DiscoveredCluster_STATUS_SECURED,
		CloudSourceID:     "1234",
	}
}

// GetManyDiscoveredClusters returns the given number of discovered clusters.
func GetManyDiscoveredClusters(num int) []*discoveredclusters.DiscoveredCluster {
	res := make([]*discoveredclusters.DiscoveredCluster, 0, num)
	for i := 0; i < num; i++ {
		discoveredCluster := GetDiscoveredCluster()
		discoveredCluster.ID = fmt.Sprintf("my-cluster-%d", i)
		if i < num/2 {
			discoveredCluster.Type = storage.ClusterMetadata_GKE
		} else {
			discoveredCluster.Type = storage.ClusterMetadata_EKS
		}
		res = append(res, discoveredCluster)
	}
	return res
}
