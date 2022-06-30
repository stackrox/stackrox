package fetcher

import (
	"testing"

	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
	"github.com/golang/mock/gomock"
	mockClusterDataStore "github.com/stackrox/rox/central/cluster/datastore/mocks"
	mockCVEDataStore "github.com/stackrox/rox/central/cve/cluster/datastore/mocks"
	"github.com/stackrox/rox/central/cve/converter/utils"
	"github.com/stackrox/rox/central/cve/converter/v2"
	"github.com/stackrox/rox/central/cve/matcher"
	mockImageDataStore "github.com/stackrox/rox/central/image/datastore/mocks"
	mockNSDataStore "github.com/stackrox/rox/central/namespace/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cve"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReconcileCVEsInPostgres(t *testing.T) {
	envIsolator := envisolator.NewEnvIsolator(t)
	envIsolator.Setenv(features.PostgresDatastore.EnvVar(), "true")

	if !features.PostgresDatastore.Enabled() {
		t.Skip("Skip postgres store tests")
		t.SkipNow()
	}

	cluster := &storage.Cluster{
		Id:   "test_cluster_id1",
		Name: "cluster1",
		Status: &storage.ClusterStatus{
			OrchestratorMetadata: &storage.OrchestratorMetadata{
				Version: "v1.10.6",
			},
		},
	}

	nvdCVEs := []*schema.NVDCVEFeedJSON10DefCVEItem{
		{
			CVE: &schema.CVEJSON40{
				CVEDataMeta: &schema.CVEJSON40CVEDataMeta{
					ID: "CVE-1",
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
								VersionEndExcluding:   "1.10.9",
							},
						},
					},
				},
			},
		},
		{
			CVE: &schema.CVEJSON40{
				CVEDataMeta: &schema.CVEJSON40CVEDataMeta{
					ID: "CVE-2",
				},
			},
			Configurations: &schema.NVDCVEFeedJSON10DefConfigurations{
				Nodes: []*schema.NVDCVEFeedJSON10DefNode{
					{
						Operator: "OR",
						CPEMatch: []*schema.NVDCVEFeedJSON10DefCPEMatch{
							{
								Vulnerable: true,
								Cpe23Uri:   "cpe:2.3:a:kubernetes:kubernetes:1.10.6:*:*:*:*:*:*:*",
							},
						},
					},
				},
			},
		},
		{
			CVE: &schema.CVEJSON40{
				CVEDataMeta: &schema.CVEJSON40CVEDataMeta{
					ID: "CVE-3",
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
								VersionStartIncluding: "1.10.3",
								VersionEndIncluding:   "1.10.7",
							},
						},
					},
				},
			},
		},
	}

	embeddedCVEs, err := utils.NvdCVEsToEmbeddedCVEs(nvdCVEs, utils.K8s)
	require.NoError(t, err)

	embeddedCVEToClusters := map[string][]*storage.Cluster{
		"CVE-1": {
			cluster,
		},
		"CVE-2": {
			cluster,
		},
		"CVE-3": {
			cluster,
		},
	}

	cvesToUpsert := []converter.ClusterCVEParts{
		{
			CVE: &storage.ClusterCVE{
				Id: cve.ID("CVE-1", storage.CVE_K8S_CVE.String()),
				CveBaseInfo: &storage.CVEInfo{
					Cve:  "CVE-1",
					Link: "https://nvd.nist.gov/vuln/detail/CVE-1",
				},
				Type: storage.CVE_K8S_CVE,
			},
			Children: []converter.EdgeParts{
				{
					Edge: &storage.ClusterCVEEdge{
						Id:        postgres.IDFromPks([]string{"test_cluster_id1", cve.ID("CVE-1", storage.CVE_K8S_CVE.String())}),
						IsFixable: true,
						HasFixedBy: &storage.ClusterCVEEdge_FixedBy{
							FixedBy: "1.10.9",
						},
					},
					ClusterID: "test_cluster_id1",
				},
			},
		},
		{
			CVE: &storage.ClusterCVE{
				Id: cve.ID("CVE-2", storage.CVE_K8S_CVE.String()),
				CveBaseInfo: &storage.CVEInfo{
					Cve:  "CVE-2",
					Link: "https://nvd.nist.gov/vuln/detail/CVE-2",
				},
				Type: storage.CVE_K8S_CVE,
			},
			Children: []converter.EdgeParts{
				{
					Edge: &storage.ClusterCVEEdge{
						Id:        postgres.IDFromPks([]string{"test_cluster_id1", cve.ID("CVE-2", storage.CVE_K8S_CVE.String())}),
						IsFixable: false,
					},
					ClusterID: "test_cluster_id1",
				},
			},
		},
		{
			CVE: &storage.ClusterCVE{
				Id: cve.ID("CVE-3", storage.CVE_K8S_CVE.String()),
				CveBaseInfo: &storage.CVEInfo{
					Cve:  "CVE-3",
					Link: "https://nvd.nist.gov/vuln/detail/CVE-3",
				},
				Type: storage.CVE_K8S_CVE,
			},
			Children: []converter.EdgeParts{
				{
					Edge: &storage.ClusterCVEEdge{
						Id:        postgres.IDFromPks([]string{"test_cluster_id1", cve.ID("CVE-3", storage.CVE_K8S_CVE.String())}),
						IsFixable: false,
					},
					ClusterID: "test_cluster_id1",
				},
			},
		},
	}

	ctrl := gomock.NewController(t)
	mockClusters := mockClusterDataStore.NewMockDataStore(ctrl)
	mockNamespaces := mockNSDataStore.NewMockDataStore(ctrl)
	mockImages := mockImageDataStore.NewMockDataStore(ctrl)
	mockCVEs := mockCVEDataStore.NewMockDataStore(ctrl)

	cveMatcher, err := matcher.NewCVEMatcher(mockClusters, mockNamespaces, mockImages)
	require.NoError(t, err)

	cveManager := &orchestratorIstioCVEManagerImpl{
		orchestratorCVEMgr: &orchestratorCVEManager{
			clusterDataStore:    mockClusters,
			clusterCVEDataStore: mockCVEs,
			cveMatcher:          cveMatcher,
		},
	}

	mockClusters.EXPECT().GetClusters(gomock.Any()).Return([]*storage.Cluster{cluster}, nil).AnyTimes()
	mockNamespaces.EXPECT().Search(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

	mockCVEs.EXPECT().UpsertClusterCVEsInternal(gomock.Any(), storage.CVE_K8S_CVE, cvesToUpsert).Return(nil)
	err = cveManager.orchestratorCVEMgr.updateCVEs(embeddedCVEs, embeddedCVEToClusters, utils.K8s)
	assert.NoError(t, err)
}
