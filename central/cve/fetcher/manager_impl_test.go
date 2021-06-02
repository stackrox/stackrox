package fetcher

import (
	"context"
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
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/set"
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
	}

	embeddedCVEs, err := converter.NvdCVEsToEmbeddedCVEs(nvdCVEs, converter.K8s)
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

	cveManager := &orchestratorIstioCVEManagerImpl{
		orchestratorCVEMgr: &orchestratorCVEManager{
			embeddedCVEIdToClusters: make(map[converter.CVEType]map[string][]*storage.Cluster),
			clusterDataStore:        mockClusters,
			cveDataStore:            mockCVEs,
			cveMatcher:              cveMatcher,
		},
	}

	mockCVEs.EXPECT().Search(gomock.Any(), gomock.Any()).Return(nil, nil)
	mockClusters.EXPECT().GetClusters(gomock.Any()).Return([]*storage.Cluster{cluster}, nil).AnyTimes()
	mockNamespaces.EXPECT().Search(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

	mockCVEs.EXPECT().UpsertClusterCVEs(gomock.Any(), cvesToUpsert).Return(nil)
	mockCVEs.EXPECT().Delete(gomock.Any(), []*storage.CVE{}).Return(nil)

	err = cveManager.orchestratorCVEMgr.updateCVEs(embeddedCVEs, embeddedCVEToClusters, converter.K8s)
	assert.NoError(t, err)
}

func TestOrchestratorManager_ReconcileCVEs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClusters := mockClusterDataStore.NewMockDataStore(ctrl)
	mockNamespaces := mockNSDataStore.NewMockDataStore(ctrl)
	mockImages := mockImageDataStore.NewMockDataStore(ctrl)
	mockCVEs := mockCVEDataStore.NewMockDataStore(ctrl)
	clusters := []*storage.Cluster{
		{
			Id:   "test_cluster_id1",
			Name: "cluster1",
			Status: &storage.ClusterStatus{
				OrchestratorMetadata: &storage.OrchestratorMetadata{
					Version: "v1.10.6",
				},
			},
		},
		{
			Id:   "test_cluster_id2",
			Name: "cluster2",
			Status: &storage.ClusterStatus{
				OrchestratorMetadata: &storage.OrchestratorMetadata{
					Version: "v1.10.9",
				},
			},
		},
		{
			Id:   "test_cluster_id3",
			Name: "cluster3",
			Status: &storage.ClusterStatus{
				OrchestratorMetadata: &storage.OrchestratorMetadata{
					Version: "v1.10.10",
					IsOpenshift: &storage.OrchestratorMetadata_OpenshiftVersion{
						OpenshiftVersion: "4.7.7",
					},
				},
			},
		},
	}

	mockCVEs.EXPECT().Search(gomock.Any(), gomock.Any()).Return(nil, nil).Times(4)

	mockCVEs.EXPECT().UpsertClusterCVEs(gomock.Any(), gomock.Any()).Return(nil).Times(1).Do(func(arg0 context.Context, cves ...converter.ClusterCVEParts) {
		assert.Equal(t, len(cves), 3)
		for _, cve := range cves {
			switch cve.CVE.GetId() {
			case "CVE-1":
				assert.Equal(t, len(cve.Children), 2) // Cluster 1, 2
				assert.Contains(t, []string{cve.Children[0].ClusterID, cve.Children[1].ClusterID}, clusters[0].GetId())
				assert.Contains(t, []string{cve.Children[0].ClusterID, cve.Children[1].ClusterID}, clusters[1].GetId())
			case "CVE-2":
				assert.Equal(t, len(cve.Children), 1) // Cluster 2
				assert.Equal(t, cve.Children[0].ClusterID, clusters[0].GetId())
			case "CVE-3":
				assert.Equal(t, len(cve.Children), 3) // Cluster 1, 2, 3
				ss := set.StringSet{}
				ss.AddAll(cve.Children[0].ClusterID, cve.Children[1].ClusterID, cve.Children[2].ClusterID)
				assert.Equal(t, len(ss), 3)
			}
		}
	})

	cvesWithComponents := []*nvdCVEWithComponents{
		{
			nvdCVE: &schema.NVDCVEFeedJSON10DefCVEItem{
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
									VersionEndExcluding:   "1.10.10",
								},
							},
						},
					},
				},
			},
			components: []string{
				kubernetes.KubeControllerManager,
			},
		},
		{
			nvdCVE: &schema.NVDCVEFeedJSON10DefCVEItem{
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
			components: []string{
				kubernetes.KubeAPIServer,
				kubernetes.KubeControllerManager,
				kubernetes.KubeAggregator,
			},
		},
		{
			nvdCVE: &schema.NVDCVEFeedJSON10DefCVEItem{
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
									VersionEndIncluding:   "1.10.11",
								},
							},
						},
					},
				},
			},
			components: []string{
				kubernetes.KubeAggregator,
			},
		},
		{
			nvdCVE: &schema.NVDCVEFeedJSON10DefCVEItem{
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
									Cpe23Uri:              "cpe:2.3:a:openshift:openshift:*:*:*:*:*:*:*:*",
									VersionStartIncluding: "4.7.3",
									VersionEndIncluding:   "4.7.10",
								},
							},
						},
					},
				},
			},
			components: []string{
				"openshift",
			},
		},
		{
			nvdCVE: &schema.NVDCVEFeedJSON10DefCVEItem{
				CVE: &schema.CVEJSON40{
					CVEDataMeta: &schema.CVEJSON40CVEDataMeta{
						ID: "CVE-5",
					},
				},
				Configurations: &schema.NVDCVEFeedJSON10DefConfigurations{
					Nodes: []*schema.NVDCVEFeedJSON10DefNode{
						{
							Operator: "OR",
							CPEMatch: []*schema.NVDCVEFeedJSON10DefCPEMatch{
								{
									Vulnerable:            true,
									Cpe23Uri:              "cpe:2.3:a:openshift:openshift:*:*:*:*:*:*:*:*",
									VersionStartIncluding: "4.7.8",
									VersionEndIncluding:   "4.7.12",
								},
							},
						},
					},
				},
			},
			components: []string{
				"openshift",
			},
		},
	}

	cveMatcher, err := matcher.NewCVEMatcher(mockClusters, mockNamespaces, mockImages)
	require.NoError(t, err)

	scanner := mockScanner{
		cveMatcher: cveMatcher,
		nvdCVEs:    cvesWithComponents,
	}

	orchestratorCVEMgr := &orchestratorCVEManager{
		embeddedCVEIdToClusters: make(map[converter.CVEType]map[string][]*storage.Cluster),
		clusterDataStore:        mockClusters,
		cveDataStore:            mockCVEs,
		cveMatcher:              cveMatcher,
		scanners:                make(map[string]types.OrchestratorScanner),
	}
	orchestratorCVEMgr.scanners["someName"] = &scanner

	err = orchestratorCVEMgr.reconcileCVEs(clusters, converter.K8s)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(orchestratorCVEMgr.embeddedCVEIdToClusters[converter.K8s]))

	mockCVEs.EXPECT().UpsertClusterCVEs(gomock.Any(), gomock.Any()).Return(nil).Times(1).Do(func(arg0 context.Context, cves ...converter.ClusterCVEParts) {
		assert.Equal(t, 1, len(cves))
		assert.Equal(t, "CVE-4", cves[0].CVE.GetId())
		assert.Equal(t, 1, len(cves[0].Children))
		assert.Contains(t, clusters[2].GetId(), cves[0].Children[0].ClusterID)
	})
	err = orchestratorCVEMgr.reconcileCVEs(clusters, converter.OpenShift)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(orchestratorCVEMgr.embeddedCVEIdToClusters[converter.OpenShift]))

	mockCVEs.EXPECT().UpsertClusterCVEs(gomock.Any(), gomock.Any()).Return(nil).Times(1).Do(func(arg0 context.Context, cves ...converter.ClusterCVEParts) {
		assert.Equal(t, 2, len(cves)) // CVE 1, 3
	})

	clusters = clusters[1:2]
	err = orchestratorCVEMgr.reconcileCVEs(clusters, converter.K8s)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(orchestratorCVEMgr.embeddedCVEIdToClusters[converter.K8s]))

	mockCVEs.EXPECT().UpsertClusterCVEs(gomock.Any(), gomock.Any()).Return(nil).Times(1).Do(func(arg0 context.Context, cves ...converter.ClusterCVEParts) {
		assert.Equal(t, 0, len(cves)) // CVE 1, 3
	})

	err = orchestratorCVEMgr.reconcileCVEs(clusters, converter.OpenShift)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(orchestratorCVEMgr.embeddedCVEIdToClusters[converter.OpenShift]))
}
