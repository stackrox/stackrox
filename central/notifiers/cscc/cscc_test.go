package cscc

import (
	"context"
	"testing"

	"cloud.google.com/go/securitycenter/apiv2/securitycenterpb"
	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
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
		Notifier: &storage.Notifier{
			Name:       "FakeSCC",
			UiEndpoint: "https://central.stackrox",
			Type:       "scc",
			Config: &storage.Notifier_Cscc{
				Cscc: &storage.CSCC{
					ServiceAccount: "test_service_account",
					SourceId:       sourceID,
				},
			},
		},
	}

	clusterStore := clusterMocks.NewMockDataStore(gomock.NewController(t))
	alert := &storage.Alert{
		Id: alertID,
		Policy: &storage.Policy{
			Id:             "myPolicyID",
			Name:           "myPolicy",
			Description:    "Fake policy",
			PolicySections: []*storage.PolicySection{},
			Severity:       storage.Severity_HIGH_SEVERITY,
		},
		Entity: &storage.Alert_Deployment_{Deployment: &storage.Alert_Deployment{
			Name:      "myDeployment",
			Id:        "myDeploymentID",
			ClusterId: clusterID,
		}},
		Time: protocompat.TimestampNow(),
	}

	cases := map[string]struct {
		mockCluster  *storage.Cluster
		resourceName string
	}{
		"alert associated with a GKE cluster": {
			mockCluster: &storage.Cluster{
				Id:   "test_id",
				Name: "test_cluster",
				Status: &storage.ClusterStatus{
					ProviderMetadata: &storage.ProviderMetadata{
						Region: "test_region",
						Provider: &storage.ProviderMetadata_Google{
							Google: &storage.GoogleProviderMetadata{
								Project:     "test_project",
								ClusterName: "test_cluster",
							},
						},
						Cluster: &storage.ClusterMetadata{
							Type: storage.ClusterMetadata_GKE,
						},
					},
				},
			},
			resourceName: "//container.googleapis.com/projects/test_project/locations/test_region/clusters/test_cluster",
		},
		"alert associated with an OpenShift cluster running on GCP": {
			mockCluster: &storage.Cluster{
				Id:   "test_id",
				Name: "test_cluster",
				Status: &storage.ClusterStatus{
					ProviderMetadata: &storage.ProviderMetadata{
						Region: "test_region",
						Provider: &storage.ProviderMetadata_Google{
							Google: &storage.GoogleProviderMetadata{
								Project:     "test_project",
								ClusterName: "test_cluster",
							},
						},
						Cluster: &storage.ClusterMetadata{
							Type: storage.ClusterMetadata_OSD,
						},
					},
				},
			},
			resourceName: "//cloudresourcemanager.googleapis.com/projects/test_project",
		},
		"alert associated with a cluster not deployed on GCP": {
			mockCluster: &storage.Cluster{
				Id:   "test_id",
				Name: "test_cluster",
				Status: &storage.ClusterStatus{
					ProviderMetadata: &storage.ProviderMetadata{
						Region: "test_region",
						Provider: &storage.ProviderMetadata_Aws{
							Aws: &storage.AWSProviderMetadata{
								AccountId: "some-account",
							},
						},
						Cluster: &storage.ClusterMetadata{
							Type: storage.ClusterMetadata_OSD,
							Name: "aws-cluster",
						},
					},
				},
			},
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
			assert.Equal(t, securitycenterpb.Finding_HIGH, finding.Severity)
			assert.Equal(t, sourceID, finding.Parent)
			assert.Contains(t, finding.ExternalUri, alertID)
			assert.Equal(t, tc.resourceName, finding.ResourceName)
		})
	}
}
