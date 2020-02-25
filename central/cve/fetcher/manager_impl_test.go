package fetcher

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
	"github.com/golang/mock/gomock"
	"github.com/stackrox/k8s-istio-cve-pusher/nvd"
	mockClusterDataStore "github.com/stackrox/rox/central/cluster/datastore/mocks"
	"github.com/stackrox/rox/central/cve/converter"
	mockCVEDataStore "github.com/stackrox/rox/central/cve/datastore/mocks"
	"github.com/stackrox/rox/central/cve/matcher"
	mockImageDataStore "github.com/stackrox/rox/central/image/datastore/mocks"
	mockNSDataStore "github.com/stackrox/rox/central/namespace/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/edges"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	correctCVEFile  = "testdata/correct_cves.json"
	cveChecksumFile = "testdata/cve_checksum"
)

func TestUnmarshalCorrectCVEs(t *testing.T) {
	dat, err := ioutil.ReadFile(correctCVEFile)
	require.Nil(t, err)
	var cveEntries []nvd.CVEEntry
	err = json.Unmarshal(dat, &cveEntries)
	assert.Nil(t, err)
	assert.Len(t, cveEntries, 2)
}

func TestReadChecksum(t *testing.T) {
	data, err := ioutil.ReadFile(cveChecksumFile)
	require.Nil(t, err)
	assert.Equal(t, string(data), "e76a63173f5b1e8bdcc9811faf4a4643266cdcb1e179229e30ffcb0e5d8dbe0c")
}

func TestReconcileCVEsInDB(t *testing.T) {
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
		{
			CVE: &schema.CVEJSON40{
				CVEDataMeta: &schema.CVEJSON40CVEDataMeta{
					ID: "CVE-4",
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
								VersionStartIncluding: "1.11.3",
								VersionEndIncluding:   "1.11.7",
							},
						},
					},
				},
			},
		},
	}

	cvesToUpsert := []converter.ClusterCVEParts{
		{
			CVE: &storage.CVE{
				Id:   "CVE-1",
				Link: "https://nvd.nist.gov/vuln/detail/CVE-1",
				Type: storage.CVE_K8S_CVE,
			},
			Children: []converter.EdgeParts{
				{
					Edge: &storage.ClusterCVEEdge{
						Id:        edges.EdgeID{ParentID: "test_cluster_id1", ChildID: "CVE-1"}.ToString(),
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
			CVE: &storage.CVE{
				Id:   "CVE-2",
				Link: "https://nvd.nist.gov/vuln/detail/CVE-2",
				Type: storage.CVE_K8S_CVE,
			},
			Children: []converter.EdgeParts{
				{
					Edge: &storage.ClusterCVEEdge{
						Id:        edges.EdgeID{ParentID: "test_cluster_id1", ChildID: "CVE-2"}.ToString(),
						IsFixable: false,
					},
					ClusterID: "test_cluster_id1",
				},
			},
		},
		{
			CVE: &storage.CVE{
				Id:   "CVE-3",
				Link: "https://nvd.nist.gov/vuln/detail/CVE-3",
				Type: storage.CVE_K8S_CVE,
			},
			Children: []converter.EdgeParts{
				{
					Edge: &storage.ClusterCVEEdge{
						Id:        edges.EdgeID{ParentID: "test_cluster_id1", ChildID: "CVE-3"}.ToString(),
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

	cveManager := &k8sIstioCVEManagerImpl{

		k8sCVEMgr: &k8sCVEManager{
			nvdCVEs: map[string]*schema.NVDCVEFeedJSON10DefCVEItem{
				"CVE-1": nvdCVEs[0],
				"CVE-2": nvdCVEs[1],
				"CVE-3": nvdCVEs[2],
				"CVE-4": nvdCVEs[3],
			},
			clusterDataStore: mockClusters,
			cveDataStore:     mockCVEs,
			cveMatcher:       cveMatcher,
		},
	}

	mockCVEs.EXPECT().Search(gomock.Any(), gomock.Any()).Return(nil, nil)
	mockClusters.EXPECT().GetClusters(gomock.Any()).Return([]*storage.Cluster{cluster}, nil).AnyTimes()
	mockNamespaces.EXPECT().SearchNamespaces(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

	mockCVEs.EXPECT().UpsertClusterCVEs(gomock.Any(), cvesToUpsert).Return(nil)
	mockCVEs.EXPECT().Delete(gomock.Any(), []*storage.CVE{}).Return(nil)

	err = cveManager.updateCVEs(nvdCVEs, converter.K8s)
	assert.NoError(t, err)
}
