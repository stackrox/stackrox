package manager

import (
	"context"
	"testing"

	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cloudsources/discoveredclusters"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
)

type mockClusterStore struct {
	clusters []*storage.Cluster

	clusterDS.DataStore
}

func (m *mockClusterStore) WalkClusters(_ context.Context, fn func(obj *storage.Cluster) error) error {
	for _, cluster := range m.clusters {
		_ = fn(cluster)
	}
	return nil
}

func TestMatchDiscoveredClusters(t *testing.T) {
	clusterStore := &mockClusterStore{clusters: []*storage.Cluster{
		createCluster(
			"123123213123_MC_testing_test-cluster-1_eastus",
			"test-cluster-1",
			storage.ClusterMetadata_AKS,
		),
		createCluster(
			"2c507da1-b882-48cc-8143-b74e14c5cd4f",
			"rosa-cluster",
			storage.ClusterMetadata_ROSA,
		),
		createCluster(
			"460c8808-9f70-51e7-9f3a-973f44ab8595",
			"osd-cluster",
			storage.ClusterMetadata_OSD,
		),
		createCluster("1231245342513", "test-cluster-2", storage.ClusterMetadata_GKE),
		createCluster("arn:aws:eks:us-east-1:test-account:cluster/test-cluster-3",
			"test-cluster-3", storage.ClusterMetadata_EKS),
		nil,
		{
			Status: &storage.ClusterStatus{
				ProviderMetadata: &storage.ProviderMetadata{
					Provider: &storage.ProviderMetadata_Aws{Aws: &storage.AWSProviderMetadata{
						AccountId: "666666666666",
					}},
				},
			},
		},
	}}

	discoveredClusters := []*discoveredclusters.DiscoveredCluster{
		{
			ID:           "123123213123_MC_testing_test-cluster-1_eastus",
			Name:         "test-cluster-1",
			Type:         storage.ClusterMetadata_AKS,
			ProviderType: storage.DiscoveredCluster_Metadata_PROVIDER_TYPE_AZURE,
		},
		{
			ID:           "1231245342513",
			Name:         "test-cluster-2",
			Type:         storage.ClusterMetadata_GKE,
			ProviderType: storage.DiscoveredCluster_Metadata_PROVIDER_TYPE_GCP,
		},
		{
			ID:           "arn:aws:eks:us-east-1:test-account:cluster/test-cluster-3",
			Name:         "test-cluster-3",
			Type:         storage.ClusterMetadata_EKS,
			ProviderType: storage.DiscoveredCluster_Metadata_PROVIDER_TYPE_AWS,
		},
		{
			ID:           "5553424234234_MC_testing_unsecured_eastus",
			Name:         "unsecured",
			Type:         storage.ClusterMetadata_AKS,
			ProviderType: storage.DiscoveredCluster_Metadata_PROVIDER_TYPE_AZURE,
		},
		{
			ID:           "arn:aws:eks:us-east-1:666666666666:cluster/test-cluster-5",
			Name:         "test-cluster-5",
			Type:         storage.ClusterMetadata_EKS,
			ProviderType: storage.DiscoveredCluster_Metadata_PROVIDER_TYPE_AWS,
		},
	}

	securedIds := set.NewFrozenStringSet(discoveredClusters[0].GetID(),
		discoveredClusters[1].GetID(), discoveredClusters[2].GetID())
	unsecuredIds := set.NewFrozenStringSet(discoveredClusters[3].GetID())
	unspecifiedIds := set.NewFrozenStringSet(discoveredClusters[4].GetID())

	m := &managerImpl{
		clusterDataStore: clusterStore,
	}

	m.matchDiscoveredClusters(discoveredClusters)

	for _, cluster := range discoveredClusters {
		if securedIds.Contains(cluster.GetID()) {
			assert.Equal(t, storage.DiscoveredCluster_STATUS_SECURED, cluster.GetStatus())
		}
		if unsecuredIds.Contains(cluster.GetID()) {
			assert.Equal(t, storage.DiscoveredCluster_STATUS_UNSECURED, cluster.GetStatus())
		}
		if unspecifiedIds.Contains(cluster.GetID()) {
			assert.Equal(t, storage.DiscoveredCluster_STATUS_UNSPECIFIED, cluster.GetStatus())
		}
	}
}

func createCluster(id, name string, clusterType storage.ClusterMetadata_Type) *storage.Cluster {
	return &storage.Cluster{
		Status: &storage.ClusterStatus{
			ProviderMetadata: &storage.ProviderMetadata{
				Cluster: &storage.ClusterMetadata{
					Type: clusterType,
					Name: name,
					Id:   id,
				},
			},
		},
	}
}
