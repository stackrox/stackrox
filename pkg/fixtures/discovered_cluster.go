package fixtures

import (
	"fmt"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cloudsources/discoveredclusters"
)

// GetDiscoveredCluster returns a mock discovered cluster.
func GetDiscoveredCluster() *discoveredclusters.DiscoveredCluster {
	now := time.Now()
	return &discoveredclusters.DiscoveredCluster{
		ID:                "my-id",
		Name:              "my-cluster",
		Type:              storage.ClusterMetadata_GKE,
		ProviderType:      storage.DiscoveredCluster_Metadata_PROVIDER_TYPE_GCP,
		Region:            "us-east-1",
		FirstDiscoveredAt: &now,
		Status:            storage.DiscoveredCluster_STATUS_SECURED,
		CloudSourceID:     "fb28231c-54d1-41e1-9551-ede4c0e15c6c",
	}
}

// GetManyDiscoveredClusters returns the given number of discovered clusters.
func GetManyDiscoveredClusters(num int) []*discoveredclusters.DiscoveredCluster {
	res := make([]*discoveredclusters.DiscoveredCluster, 0, num)
	for i := 0; i < num; i++ {
		discoveredCluster := GetDiscoveredCluster()
		discoveredCluster.ID = fmt.Sprintf("my-cluster-%02d", i)
		discoveredCluster.Name = fmt.Sprintf("my-cluster-%02d", i)
		if i < num/2 {
			discoveredCluster.Type = storage.ClusterMetadata_GKE
			discoveredCluster.Status = storage.DiscoveredCluster_STATUS_SECURED
			discoveredCluster.CloudSourceID = "fb28231c-54d1-41e1-9551-ede4c0e15c6c"
		} else {
			discoveredCluster.Type = storage.ClusterMetadata_EKS
			discoveredCluster.Status = storage.DiscoveredCluster_STATUS_UNSECURED
			discoveredCluster.CloudSourceID = "4f026c43-8b8a-465a-8c09-664f16c9e8e3"
		}
		res = append(res, discoveredCluster)
	}
	return res
}
