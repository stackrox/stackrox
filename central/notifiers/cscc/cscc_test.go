package cscc

import (
	"context"
	"testing"

	"cloud.google.com/go/securitycenter/apiv1/securitycenterpb"
	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"
)

func TestCSCC(t *testing.T) {
	var (
		sourceID  = "organizations/0000000000/sources/0000000000"
		alertID   = "myAlertID"
		clusterID = "test_cluster"
	)

	notifier := &cscc{
		config: &config{
			SourceID: sourceID,
		},
		Notifier: storage.Notifier_builder{
			Name:       "FakeSCC",
			UiEndpoint: "https://central.stackrox",
			Type:       "scc",
			Cscc: storage.CSCC_builder{
				ServiceAccount: "test_service_account",
				SourceId:       sourceID,
			}.Build(),
		}.Build(),
	}

	clusterStore := clusterMocks.NewMockDataStore(gomock.NewController(t))
	policy := &storage.Policy{}
	policy.SetId("myPolicyID")
	policy.SetName("myPolicy")
	policy.SetDescription("Fake policy")
	policy.SetPolicySections([]*storage.PolicySection{})
	policy.SetSeverity(storage.Severity_HIGH_SEVERITY)
	ad := &storage.Alert_Deployment{}
	ad.SetName("myDeployment")
	ad.SetId("myDeploymentID")
	ad.SetClusterId(clusterID)
	alert := &storage.Alert{}
	alert.SetId(alertID)
	alert.SetPolicy(policy)
	alert.SetDeployment(proto.ValueOrDefault(ad))
	alert.SetTime(protocompat.TimestampNow())

	cases := map[string]struct {
		mockCluster  *storage.Cluster
		resourceName string
	}{
		"alert associated with a GKE cluster": {
			mockCluster: storage.Cluster_builder{
				Id:   "test_id",
				Name: "test_cluster",
				Status: storage.ClusterStatus_builder{
					ProviderMetadata: storage.ProviderMetadata_builder{
						Region: "test_region",
						Google: storage.GoogleProviderMetadata_builder{
							Project:     "test_project",
							ClusterName: "test_cluster",
						}.Build(),
						Cluster: storage.ClusterMetadata_builder{
							Type: storage.ClusterMetadata_GKE,
						}.Build(),
					}.Build(),
				}.Build(),
			}.Build(),
			resourceName: "//container.googleapis.com/projects/test_project/locations/test_region/clusters/test_cluster",
		},
		"alert associated with an OpenShift cluster running on GCP": {
			mockCluster: storage.Cluster_builder{
				Id:   "test_id",
				Name: "test_cluster",
				Status: storage.ClusterStatus_builder{
					ProviderMetadata: storage.ProviderMetadata_builder{
						Region: "test_region",
						Google: storage.GoogleProviderMetadata_builder{
							Project:     "test_project",
							ClusterName: "test_cluster",
						}.Build(),
						Cluster: storage.ClusterMetadata_builder{
							Type: storage.ClusterMetadata_OSD,
						}.Build(),
					}.Build(),
				}.Build(),
			}.Build(),
			resourceName: "//cloudresourcemanager.googleapis.com/projects/test_project",
		},
		"alert associated with a cluster not deployed on GCP": {
			mockCluster: storage.Cluster_builder{
				Id:   "test_id",
				Name: "test_cluster",
				Status: storage.ClusterStatus_builder{
					ProviderMetadata: storage.ProviderMetadata_builder{
						Region: "test_region",
						Aws: storage.AWSProviderMetadata_builder{
							AccountId: "some-account",
						}.Build(),
						Cluster: storage.ClusterMetadata_builder{
							Type: storage.ClusterMetadata_OSD,
							Name: "aws-cluster",
						}.Build(),
					}.Build(),
				}.Build(),
			}.Build(),
			resourceName: "OSD/aws-cluster",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			clusterStore.EXPECT().GetCluster(gomock.Any(), clusterID).Return(tc.mockCluster, true, nil)
			findingID, finding, err := notifier.initFinding(context.Background(), alert, clusterStore)
			assert.NoError(t, err)
			assert.Equal(t, alertID, findingID)
			assert.NotEmpty(t, finding)
			assert.Equal(t, securitycenterpb.Finding_HIGH, finding.GetSeverity())
			assert.Equal(t, sourceID, finding.GetParent())
			assert.Contains(t, finding.GetExternalUri(), alertID)
			assert.Equal(t, tc.resourceName, finding.GetResourceName())
		})
	}
}
