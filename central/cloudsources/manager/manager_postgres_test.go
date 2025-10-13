//go:build sql_integration

package manager

import (
	"testing"

	cloudSourcesDS "github.com/stackrox/rox/central/cloudsources/datastore"
	"github.com/stackrox/rox/central/convert/typetostorage"
	discoveredClustersDS "github.com/stackrox/rox/central/discoveredclusters/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cloudsources/discoveredclusters"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChangeDiscoveredClustersStatus(t *testing.T) {
	pool := pgtest.ForT(t)
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
	}

	m.MarkClusterUnsecured("2c507da1-b882-48cc-8143-b74e14c5cd4f")

	clusters, err := dcDS.ListDiscoveredClusters(discoveredClusterCtx, search.EmptyQuery())
	require.NoError(t, err)
	require.Len(t, clusters, 1)
	storedDiscoveredCluster := clusters[0]
	expectedDiscoveredCluster := typetostorage.DiscoveredCluster(discoveredCluster)
	expectedDiscoveredCluster.Status = storage.DiscoveredCluster_STATUS_UNSECURED

	assert.Equal(t, expectedDiscoveredCluster.GetId(), storedDiscoveredCluster.GetId())
	assert.Equal(t, expectedDiscoveredCluster.GetStatus(), storedDiscoveredCluster.GetStatus())
}
