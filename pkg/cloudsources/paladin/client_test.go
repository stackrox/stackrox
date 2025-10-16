package paladin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cloudsources/discoveredclusters"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
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

	testCluster1FirstDiscoveredAt, err := time.Parse(timeFormat, "2023-11-28 08:00:00+0000")
	require.NoError(t, err)
	testCluster2FirstDiscoveredAt, err := time.Parse(timeFormat, "2024-02-01 13:52:00+0000")
	require.NoError(t, err)
	testCluster3FirstDiscoveredAt, err := time.Parse(timeFormat, "2024-02-01 13:52:00+0000")
	require.NoError(t, err)
	testCluster4FirstDiscoveredAt, err := time.Parse(timeFormat, "2024-02-09 08:00:00+0000")
	require.NoError(t, err)

	expectedDiscoveredClusters := []*discoveredclusters.DiscoveredCluster{
		{
			ID:                "123123213123_MC_testing_test-cluster-1_eastus",
			Name:              "test-cluster-1",
			Type:              storage.ClusterMetadata_AKS,
			ProviderType:      storage.DiscoveredCluster_Metadata_PROVIDER_TYPE_AZURE,
			Region:            "eastus",
			CloudSourceID:     "id",
			FirstDiscoveredAt: &testCluster1FirstDiscoveredAt,
		},
		{
			ID:                "1231245342513",
			Name:              "test-cluster-2",
			Type:              storage.ClusterMetadata_GKE,
			ProviderType:      storage.DiscoveredCluster_Metadata_PROVIDER_TYPE_GCP,
			Region:            "us-central1-c",
			CloudSourceID:     "id",
			FirstDiscoveredAt: &testCluster2FirstDiscoveredAt,
		},
		{
			ID:                "1231234124123541",
			Name:              "test-cluster-3",
			Type:              storage.ClusterMetadata_EKS,
			ProviderType:      storage.DiscoveredCluster_Metadata_PROVIDER_TYPE_AWS,
			Region:            "us-central1",
			CloudSourceID:     "id",
			FirstDiscoveredAt: &testCluster3FirstDiscoveredAt,
		},
		{
			ID:                "arn:aws:eks:us-east-1:test-account:cluster/test-cluster-4",
			Name:              "test-cluster-4",
			Type:              storage.ClusterMetadata_EKS,
			ProviderType:      storage.DiscoveredCluster_Metadata_PROVIDER_TYPE_AWS,
			Region:            "us-east-1",
			CloudSourceID:     "id",
			FirstDiscoveredAt: &testCluster4FirstDiscoveredAt,
		},
	}

	cc := &storage.CloudSource_Credentials{}
	cc.SetSecret("testing")
	pcc := &storage.PaladinCloudConfig{}
	pcc.SetEndpoint(server.URL)
	cs := &storage.CloudSource{}
	cs.SetId("id")
	cs.SetCredentials(cc)
	cs.SetPaladinCloud(proto.ValueOrDefault(pcc))
	client := NewClient(cs)

	resp, err := client.GetDiscoveredClusters(context.Background())
	require.NoError(t, err)
	assert.Len(t, resp, 4)

	assert.ElementsMatch(t, resp, expectedDiscoveredClusters)
}
