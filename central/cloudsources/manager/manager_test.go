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
			"MC_testing_test-cluster-1_eastus",
			"test-cluster-1",
			storage.ClusterMetadata_AKS,
		),
		createCluster("something-else", "something-else", storage.ClusterMetadata_ROSA),
		createCluster("another-thing", "another-thing", storage.ClusterMetadata_OSD),
		createCluster("1231245342513", "test-cluster-2", storage.ClusterMetadata_GKE),
		createCluster("1231234124123541", "test-cluster-3", storage.ClusterMetadata_EKS),
	}}

	discoveredClusters := []*discoveredclusters.DiscoveredCluster{
		{
			ID:   "MC_testing_test-cluster-1_eastus",
			Name: "test-cluster-1",
			Type: storage.ClusterMetadata_AKS,
		},
		{
			ID:   "1231245342513",
			Name: "test-cluster-2",
			Type: storage.ClusterMetadata_GKE,
		},
		{
			ID:   "1231234124123541",
			Name: "test-cluster-3",
			Type: storage.ClusterMetadata_EKS,
		},
		{
			ID:   "55555555",
			Name: "unsecured",
			Type: storage.ClusterMetadata_AKS,
		},
	}

	securedIds := set.NewFrozenStringSet(discoveredClusters[0].GetID(),
		discoveredClusters[1].GetID(), discoveredClusters[2].GetID())
	unsecuredIds := set.NewFrozenStringSet(discoveredClusters[3].GetID())

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
