package paladin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cloudsources/discoveredclusters"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_GetAssets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "testing", request.Header.Get("Authorization"))
		assert.NotEmpty(t, request.Header.Get("User-Agent"))

		data, err := os.ReadFile("testdata/response.json")
		require.NoError(t, err)

		_, err = writer.Write(data)
		require.NoError(t, err)
	}))
	defer server.Close()

	expectedDiscoveredClusters := []*discoveredclusters.DiscoveredCluster{
		{
			ID:            "MC_testing_test-cluster-1_eastus",
			Name:          "test-cluster-1",
			Type:          storage.ClusterMetadata_AKS,
			ProviderType:  storage.DiscoveredCluster_Metadata_PROVIDER_TYPE_AZURE,
			Region:        "eastus",
			CloudSourceID: "id",
		},
		{
			ID:            "1231245342513",
			Name:          "test-cluster-2",
			Type:          storage.ClusterMetadata_GKE,
			ProviderType:  storage.DiscoveredCluster_Metadata_PROVIDER_TYPE_GCP,
			Region:        "us-central1-c",
			CloudSourceID: "id",
		},
		{
			ID:            "1231234124123541",
			Name:          "test-cluster-3",
			Type:          storage.ClusterMetadata_EKS,
			ProviderType:  storage.DiscoveredCluster_Metadata_PROVIDER_TYPE_AWS,
			Region:        "us-central1",
			CloudSourceID: "id",
		},
	}

	client := NewClient(&storage.CloudSource{
		Id:          "id",
		Credentials: &storage.CloudSource_Credentials{Secret: "testing"},
		Config: &storage.CloudSource_PaladinCloud{PaladinCloud: &storage.PaladinCloudConfig{
			Endpoint: server.URL,
		}},
	})

	resp, err := client.GetDiscoveredClusters(context.Background())
	require.NoError(t, err)
	assert.Len(t, resp, 3)

	for i, cluster := range resp {
		assert.Equal(t, expectedDiscoveredClusters[i].GetID(), cluster.GetID())
		assert.Equal(t, expectedDiscoveredClusters[i].GetName(), cluster.GetName())
		assert.Equal(t, expectedDiscoveredClusters[i].GetType(), cluster.GetType())
		assert.Equal(t, expectedDiscoveredClusters[i].GetProviderType(), cluster.GetProviderType())
		assert.Equal(t, expectedDiscoveredClusters[i].GetRegion(), cluster.GetRegion())
		assert.Equal(t, expectedDiscoveredClusters[i].GetCloudSourceID(), cluster.GetCloudSourceID())
	}
}
