package resolvers

import (
	"context"
	"testing"

	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	"github.com/stackrox/rox/central/cve/converter/utils"
	"github.com/stackrox/rox/central/cve/matcher"
	imageMocks "github.com/stackrox/rox/central/image/datastore/mocks"
	nsMocks "github.com/stackrox/rox/central/namespace/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestK8sCVEEnvImpact(t *testing.T) {
	expected := []float64{0.6, 0.4, 0.4}

	clusters := []*storage.Cluster{
		{
			Id:   "test_cluster_id1",
			Name: "cluster1",
			Status: &storage.ClusterStatus{
				OrchestratorMetadata: &storage.OrchestratorMetadata{
					Version: "v1.14.2",
				},
			},
		},
		{
			Id:   "test_cluster_id2",
			Name: "cluster2",
			Status: &storage.ClusterStatus{
				OrchestratorMetadata: &storage.OrchestratorMetadata{
					Version: "v1.14.5+build1",
				},
			},
		},
		{
			Id:   "test_cluster_id3",
			Name: "cluster3",
			Status: &storage.ClusterStatus{
				OrchestratorMetadata: &storage.OrchestratorMetadata{
					Version: "v1.15.4-beta1",
				},
			},
		},
		{
			Id:   "test_cluster_id4",
			Name: "cluster4",
			Status: &storage.ClusterStatus{
				OrchestratorMetadata: &storage.OrchestratorMetadata{
					Version: "v1.16.3-alpha1+build2",
				},
			},
		},
		{
			Id:   "test_cluster_id5",
			Name: "cluster4",
			Status: &storage.ClusterStatus{
				OrchestratorMetadata: &storage.OrchestratorMetadata{
					Version: "v1.17.5",
				},
			},
		},
	}

	ctrl := gomock.NewController(t)
	clusterDataStore := clusterMocks.NewMockDataStore(ctrl)
	nsDataStore := nsMocks.NewMockDataStore(ctrl)
	imageDataStore := imageMocks.NewMockDataStore(ctrl)
	cveMatcher, err := matcher.NewCVEMatcher(clusterDataStore, nsDataStore, imageDataStore)
	require.NoError(t, err)

	clusterDataStore.EXPECT().GetClusters(gomock.Any()).Return(clusters, nil).AnyTimes()
	nsDataStore.EXPECT().Search(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	clusterDataStore.EXPECT().GetClusters(gomock.Any()).Return(clusters, nil).AnyTimes()

	resolver := &Resolver{
		ClusterDataStore: clusterDataStore,
		cveMatcher:       cveMatcher,
	}

	cves := []*schema.NVDCVEFeedJSON10DefCVEItem{
		{
			CVE: &schema.CVEJSON40{
				CVEDataMeta: &schema.CVEJSON40CVEDataMeta{
					ID: "CVE-2019-1",
				},
			},
			Configurations: &schema.NVDCVEFeedJSON10DefConfigurations{
				Nodes: []*schema.NVDCVEFeedJSON10DefNode{
					{
						Operator: "OR",
						CPEMatch: []*schema.NVDCVEFeedJSON10DefCPEMatch{
							{
								Vulnerable: true,
								Cpe23Uri:   "cpe:2.3:a:kubernetes:kubernetes:1.15.4:*:*:*:*:*:*:*",
							},
							{
								Vulnerable:            true,
								Cpe23Uri:              "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
								VersionStartIncluding: "1.16.1",
								VersionEndIncluding:   "1.16.9",
							},
							{
								Vulnerable:            true,
								Cpe23Uri:              "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
								VersionStartIncluding: "1.17.1",
								VersionEndExcluding:   "1.17.7",
							},
						},
					},
				},
			},
		},
		{
			CVE: &schema.CVEJSON40{
				CVEDataMeta: &schema.CVEJSON40CVEDataMeta{
					ID: "CVE-2019-2",
				},
			},
			Configurations: &schema.NVDCVEFeedJSON10DefConfigurations{
				Nodes: []*schema.NVDCVEFeedJSON10DefNode{
					{
						Operator: "OR",
						CPEMatch: []*schema.NVDCVEFeedJSON10DefCPEMatch{
							{
								Vulnerable:            true,
								Cpe23Uri:              "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
								VersionStartIncluding: "1.14.1",
								VersionEndExcluding:   "1.14.9",
							},
						},
					},
				},
			},
		},
		{
			CVE: &schema.CVEJSON40{
				CVEDataMeta: &schema.CVEJSON40CVEDataMeta{
					ID: "CVE-2019-3",
				},
			},

			Configurations: &schema.NVDCVEFeedJSON10DefConfigurations{
				Nodes: []*schema.NVDCVEFeedJSON10DefNode{
					{
						Operator: "OR",
						CPEMatch: []*schema.NVDCVEFeedJSON10DefCPEMatch{
							{
								Vulnerable:            true,
								Cpe23Uri:              "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
								VersionStartIncluding: "1.10.1",
								VersionEndIncluding:   "1.10.9",
							},
							{
								Vulnerable:            true,
								Cpe23Uri:              "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
								VersionStartIncluding: "1.11.1",
								VersionEndExcluding:   "1.11.7",
							},
							{
								Vulnerable: true,
								Cpe23Uri:   "cpe:2.3:a:kubernetes:kubernetes:1.14.5:*:*:*:*:*:*:*",
							},
							{
								Vulnerable: true,
								Cpe23Uri:   "cpe:2.3:a:kubernetes:kubernetes:1.15.4:alpha1:*:*:*:*:*:*",
							},
							{
								Vulnerable: true,
								Cpe23Uri:   "cpe:2.3:a:kubernetes:kubernetes:1.15.4:beta1:*:*:*:*:*:*",
							},
						},
					},
				},
			},
		},
	}

	for i, cve := range cves {
		numerator, denominator, err := resolver.getComponentsForAffectedCluster(context.Background(), cve, utils.K8s)
		assert.Nil(t, err)

		assert.Equal(t, float64(numerator)/float64(denominator), expected[i])
	}
	for i, cve := range cves {
		numerator, denominator, err := resolver.getComponentsForAffectedCluster(context.Background(), cve, utils.OpenShift)
		assert.Nil(t, err)

		assert.Equal(t, float64(numerator)/float64(denominator), expected[i])
	}
}
