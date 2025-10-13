package ocm

import (
	"os"
	"testing"
	"time"

	accountsmgmtv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cloudsources/discoveredclusters"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapToDiscoveredClusters(t *testing.T) {
	c := &ocmClient{
		cloudSourceID: "12345",
	}

	cluster1, err := time.Parse(time.RFC3339, "2024-02-13T21:34:35.11432Z")
	require.NoError(t, err)

	cluster2, err := time.Parse(time.RFC3339, "2024-02-13T21:34:03.763759Z")
	require.NoError(t, err)

	cluster3, err := time.Parse(time.RFC3339, "2024-02-13T21:30:57.000508Z")
	require.NoError(t, err)

	cluster4, err := time.Parse(time.RFC3339, "2024-02-13T21:28:54.916088Z")
	require.NoError(t, err)

	expectedClusters := []*discoveredclusters.DiscoveredCluster{
		{
			ID:                "11b0e5a3-0a38-4484-8272-1fd690bd65b5",
			Name:              "rosa-cluster",
			Type:              storage.ClusterMetadata_ROSA,
			ProviderType:      storage.DiscoveredCluster_Metadata_PROVIDER_TYPE_AWS,
			FirstDiscoveredAt: &cluster1,
			Region:            "us-east1",
			CloudSourceID:     "12345",
		},
		{
			ID:                "22b0e5a3-0a38-4484-8272-1fd690bd65b5",
			Name:              "aro-cluster",
			Type:              storage.ClusterMetadata_ARO,
			ProviderType:      storage.DiscoveredCluster_Metadata_PROVIDER_TYPE_AZURE,
			Region:            "us-east1",
			FirstDiscoveredAt: &cluster2,
			CloudSourceID:     "12345",
		},
		{
			ID:                "338d4973-7a37-4559-afad-f3d75d96c7fd",
			Name:              "osd-cluster",
			Type:              storage.ClusterMetadata_OSD,
			ProviderType:      storage.DiscoveredCluster_Metadata_PROVIDER_TYPE_GCP,
			Region:            "us-central1",
			FirstDiscoveredAt: &cluster3,
			CloudSourceID:     "12345",
		},
		{
			ID:                "44a6254c-8bc4-4724-abfe-c510747742b8",
			Name:              "ocp-gcp-cluster",
			Type:              storage.ClusterMetadata_OCP,
			ProviderType:      storage.DiscoveredCluster_Metadata_PROVIDER_TYPE_GCP,
			Region:            "us-central1",
			FirstDiscoveredAt: &cluster4,
			CloudSourceID:     "12345",
		},
	}

	data, err := os.ReadFile("testdata/response.json")
	require.NoError(t, err)

	subs, err := accountsmgmtv1.UnmarshalSubscriptionList(data)
	require.NoError(t, err)

	clusters, err := c.mapToDiscoveredClusters(subs)
	assert.NoError(t, err)
	assert.ElementsMatch(t, expectedClusters, clusters)
}
