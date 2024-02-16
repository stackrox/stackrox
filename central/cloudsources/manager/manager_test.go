//go:build sql_integration

package manager

import (
	"context"
	"testing"

	cloudSourcesDS "github.com/stackrox/rox/central/cloudsources/datastore"
	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/convert/typetostorage"
	discoveredClustersDS "github.com/stackrox/rox/central/discoveredclusters/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cloudsources/discoveredclusters"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func (m *mockClusterStore) GetCluster(_ context.Context, id string) (*storage.Cluster, bool, error) {
	for _, cluster := range m.clusters {
		if cluster.GetId() == id {
			return cluster, true, nil
		}
	}
	return nil, false, nil
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
			HealthStatus: &storage.ClusterHealthStatus{OverallHealthStatus: storage.ClusterHealthStatus_HEALTHY},
			Status: &storage.ClusterStatus{
				ProviderMetadata: &storage.ProviderMetadata{
					Provider: &storage.ProviderMetadata_Aws{Aws: &storage.AWSProviderMetadata{
						AccountId: "666666666666",
					}},
				},
			},
		},
		{
			HealthStatus: &storage.ClusterHealthStatus{OverallHealthStatus: storage.ClusterHealthStatus_UNHEALTHY},
			Status: &storage.ClusterStatus{
				ProviderMetadata: &storage.ProviderMetadata{
					Cluster: &storage.ClusterMetadata{Id: "5553424234234_MC_testing_unsecured_eastus"},
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

func TestChangeDiscoveredClustersStatus(t *testing.T) {
	pool := pgtest.ForT(t)
	defer func() {
		pool.Teardown(t)
		pool.Close()
	}()
	dcDS := discoveredClustersDS.GetTestPostgresDataStore(t, pool)

	csDS := cloudSourcesDS.GetTestPostgresDataStore(t, pool)

	cloudSource := fixtures.GetStorageCloudSource()
	err := csDS.UpsertCloudSource(cloudSourceCtx, cloudSource)
	require.NoError(t, err)

	clusterStore := &mockClusterStore{clusters: []*storage.Cluster{
		createCluster(
			"2c507da1-b882-48cc-8143-b74e14c5cd4f",
			"test-cluster-1",
			storage.ClusterMetadata_ROSA,
		),
	}}

	discoveredCluster := &discoveredclusters.DiscoveredCluster{
		ID:            "2c507da1-b882-48cc-8143-b74e14c5cd4f",
		Name:          "test-cluster-1",
		Type:          storage.ClusterMetadata_ROSA,
		ProviderType:  storage.DiscoveredCluster_Metadata_PROVIDER_TYPE_AWS,
		Status:        storage.DiscoveredCluster_STATUS_SECURED,
		CloudSourceID: cloudSource.GetId(),
	}

	require.NoError(t, dcDS.UpsertDiscoveredClusters(discoveredClusterCtx, discoveredCluster))

	m := &managerImpl{
		cloudSourcesDataStore:       csDS,
		discoveredClustersDataStore: dcDS,
		clusterDataStore:            clusterStore,
		unspecifiedProviderTypes:    set.NewStringSet(storage.DiscoveredCluster_Metadata_PROVIDER_TYPE_AWS.String()),
	}

	m.MarkClusterUnsecured("2c507da1-b882-48cc-8143-b74e14c5cd4f")

	clusters, err := dcDS.ListDiscoveredClusters(discoveredClusterCtx, search.EmptyQuery())
	require.NoError(t, err)
	require.Len(t, clusters, 1)
	storedDiscoveredCluster := clusters[0]
	expectedDiscoveredCluster := typetostorage.DiscoveredCluster(discoveredCluster)
	expectedDiscoveredCluster.Status = storage.DiscoveredCluster_STATUS_UNSPECIFIED

	assert.Equal(t, expectedDiscoveredCluster.GetId(), storedDiscoveredCluster.GetId())
	assert.Equal(t, expectedDiscoveredCluster.GetStatus(), storedDiscoveredCluster.GetStatus())
}

func createCluster(id, name string, clusterType storage.ClusterMetadata_Type) *storage.Cluster {
	return &storage.Cluster{
		Id:           id,
		HealthStatus: &storage.ClusterHealthStatus{OverallHealthStatus: storage.ClusterHealthStatus_HEALTHY},
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
