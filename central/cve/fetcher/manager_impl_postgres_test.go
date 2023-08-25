//go:build sql_integration

package fetcher

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
	"github.com/gogo/protobuf/types"
	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	mockClusterDataStore "github.com/stackrox/rox/central/cluster/datastore/mocks"
	clusterCVEDataStore "github.com/stackrox/rox/central/cve/cluster/datastore"
	mockCVEDataStore "github.com/stackrox/rox/central/cve/cluster/datastore/mocks"
	"github.com/stackrox/rox/central/cve/converter/utils"
	"github.com/stackrox/rox/central/cve/converter/v2"
	"github.com/stackrox/rox/central/cve/matcher"
	mockImageDataStore "github.com/stackrox/rox/central/image/datastore/mocks"
	mockNSDataStore "github.com/stackrox/rox/central/namespace/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cve"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestReconcileIstioCVEsInPostgres(t *testing.T) {
	cluster := &storage.Cluster{
		Id:   "test_cluster_id1",
		Name: "cluster1",
		Status: &storage.ClusterStatus{
			OrchestratorMetadata: &storage.OrchestratorMetadata{
				Version: "v1.10.6",
			},
		},
	}

	istioNvdCVEs := []*schema.NVDCVEFeedJSON10DefCVEItem{
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
								Cpe23Uri:              "cpe:2.3:a:istio:istio:*:*:*:*:*:*:*:*",
								VersionStartIncluding: "1.13.12",
								VersionEndIncluding:   "1.13.17",
							},
						},
					},
				},
			},
			Impact: &schema.NVDCVEFeedJSON10DefImpact{
				BaseMetricV3: &schema.NVDCVEFeedJSON10DefImpactBaseMetricV3{
					CVSSV3: &schema.CVSSV30{
						BaseScore:    6.1,
						VectorString: "AV:L/AC:L/PR:L/UI:N/S:U/C:N/I:L/A:H",
						Version:      "3.0",
					},
					ExploitabilityScore: 1.8,
					ImpactScore:         4.2,
				},
			},
		},
	}

	istioEmbeddedCVEs, err := utils.NVDCVEsToEmbeddedCVEs(istioNvdCVEs, utils.Istio)
	require.NoError(t, err)

	istioEmbeddedCVEToClusters := map[string][]*storage.Cluster{
		"CVE-4": {
			cluster,
		},
	}

	istioCvesToUpsert := []converter.ClusterCVEParts{
		{
			CVE: &storage.ClusterCVE{
				Id: cve.ID("CVE-4", storage.CVE_ISTIO_CVE.String()),
				CveBaseInfo: &storage.CVEInfo{
					Cve:          "CVE-4",
					Link:         "https://nvd.nist.gov/vuln/detail/CVE-4",
					ScoreVersion: storage.CVEInfo_V3,
					CvssV3: &storage.CVSSV3{
						Vector:              "AV:L/AC:L/PR:L/UI:N/S:U/C:N/I:L/A:H",
						ExploitabilityScore: 1.8,
						ImpactScore:         4.2,
						AttackVector:        storage.CVSSV3_ATTACK_LOCAL,
						AttackComplexity:    storage.CVSSV3_COMPLEXITY_LOW,
						PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_LOW,
						UserInteraction:     storage.CVSSV3_UI_NONE,
						Scope:               storage.CVSSV3_UNCHANGED,
						Confidentiality:     storage.CVSSV3_IMPACT_NONE,
						Integrity:           storage.CVSSV3_IMPACT_LOW,
						Availability:        storage.CVSSV3_IMPACT_HIGH,
						Score:               6.1,
					},
				},
				Cvss:        6.1,
				ImpactScore: 4.2,
				Type:        storage.CVE_ISTIO_CVE,
			},
			Children: []converter.EdgeParts{
				{
					Edge: &storage.ClusterCVEEdge{
						Id:        pgSearch.IDFromPks([]string{"test_cluster_id1", cve.ID("CVE-4", storage.CVE_ISTIO_CVE.String())}),
						IsFixable: false,
						ClusterId: "test_cluster_id1",
						CveId:     cve.ID("CVE-4", storage.CVE_ISTIO_CVE.String()),
					},
					ClusterID: "test_cluster_id1",
				},
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

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

	mockCVEs.EXPECT().UpsertClusterCVEsInternal(gomock.Any(), storage.CVE_ISTIO_CVE, istioCvesToUpsert).Return(nil)
	err = cveManager.orchestratorCVEMgr.updateCVEs(istioEmbeddedCVEs, istioEmbeddedCVEToClusters, utils.Istio)
	assert.NoError(t, err)
}

func TestReconcileCVEsInPostgres(t *testing.T) {

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
			Impact: &schema.NVDCVEFeedJSON10DefImpact{
				BaseMetricV3: &schema.NVDCVEFeedJSON10DefImpactBaseMetricV3{
					CVSSV3: &schema.CVSSV30{
						BaseScore:    6.1,
						VectorString: "AV:L/AC:L/PR:L/UI:N/S:U/C:N/I:L/A:H",
						Version:      "3.0",
					},
					ExploitabilityScore: 1.8,
					ImpactScore:         4.2,
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
			Impact: &schema.NVDCVEFeedJSON10DefImpact{
				BaseMetricV3: &schema.NVDCVEFeedJSON10DefImpactBaseMetricV3{
					CVSSV3: &schema.CVSSV30{
						BaseScore:    6.1,
						VectorString: "AV:L/AC:L/PR:L/UI:N/S:U/C:N/I:L/A:H",
						Version:      "3.0",
					},
					ExploitabilityScore: 1.8,
					ImpactScore:         4.2,
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
			Impact: &schema.NVDCVEFeedJSON10DefImpact{
				BaseMetricV3: &schema.NVDCVEFeedJSON10DefImpactBaseMetricV3{
					CVSSV3: &schema.CVSSV30{
						BaseScore:    6.1,
						VectorString: "AV:L/AC:L/PR:L/UI:N/S:U/C:N/I:L/A:H",
						Version:      "3.0",
					},
					ExploitabilityScore: 1.8,
					ImpactScore:         4.2,
				},
			},
		},
	}

	embeddedCVEs, err := utils.NVDCVEsToEmbeddedCVEs(nvdCVEs, utils.K8s)
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
					Cve:          "CVE-1",
					Link:         "https://nvd.nist.gov/vuln/detail/CVE-1",
					ScoreVersion: storage.CVEInfo_V3,
					CvssV3: &storage.CVSSV3{
						Vector:              "AV:L/AC:L/PR:L/UI:N/S:U/C:N/I:L/A:H",
						ExploitabilityScore: 1.8,
						ImpactScore:         4.2,
						AttackVector:        storage.CVSSV3_ATTACK_LOCAL,
						AttackComplexity:    storage.CVSSV3_COMPLEXITY_LOW,
						PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_LOW,
						UserInteraction:     storage.CVSSV3_UI_NONE,
						Scope:               storage.CVSSV3_UNCHANGED,
						Confidentiality:     storage.CVSSV3_IMPACT_NONE,
						Integrity:           storage.CVSSV3_IMPACT_LOW,
						Availability:        storage.CVSSV3_IMPACT_HIGH,
						Score:               6.1,
					},
				},
				Cvss:        6.1,
				ImpactScore: 4.2,
				Type:        storage.CVE_K8S_CVE,
			},
			Children: []converter.EdgeParts{
				{
					Edge: &storage.ClusterCVEEdge{
						Id:        pgSearch.IDFromPks([]string{"test_cluster_id1", cve.ID("CVE-1", storage.CVE_K8S_CVE.String())}),
						IsFixable: true,
						HasFixedBy: &storage.ClusterCVEEdge_FixedBy{
							FixedBy: "1.10.9",
						},
						ClusterId: "test_cluster_id1",
						CveId:     cve.ID("CVE-1", storage.CVE_K8S_CVE.String()),
					},
					ClusterID: "test_cluster_id1",
				},
			},
		},
		{
			CVE: &storage.ClusterCVE{
				Id: cve.ID("CVE-2", storage.CVE_K8S_CVE.String()),
				CveBaseInfo: &storage.CVEInfo{
					Cve:          "CVE-2",
					Link:         "https://nvd.nist.gov/vuln/detail/CVE-2",
					ScoreVersion: storage.CVEInfo_V3,
					CvssV3: &storage.CVSSV3{
						Vector:              "AV:L/AC:L/PR:L/UI:N/S:U/C:N/I:L/A:H",
						ExploitabilityScore: 1.8,
						ImpactScore:         4.2,
						AttackVector:        storage.CVSSV3_ATTACK_LOCAL,
						AttackComplexity:    storage.CVSSV3_COMPLEXITY_LOW,
						PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_LOW,
						UserInteraction:     storage.CVSSV3_UI_NONE,
						Scope:               storage.CVSSV3_UNCHANGED,
						Confidentiality:     storage.CVSSV3_IMPACT_NONE,
						Integrity:           storage.CVSSV3_IMPACT_LOW,
						Availability:        storage.CVSSV3_IMPACT_HIGH,
						Score:               6.1,
					},
				},
				Cvss:        6.1,
				ImpactScore: 4.2,
				Type:        storage.CVE_K8S_CVE,
			},
			Children: []converter.EdgeParts{
				{
					Edge: &storage.ClusterCVEEdge{
						Id:        pgSearch.IDFromPks([]string{"test_cluster_id1", cve.ID("CVE-2", storage.CVE_K8S_CVE.String())}),
						IsFixable: false,
						ClusterId: "test_cluster_id1",
						CveId:     cve.ID("CVE-2", storage.CVE_K8S_CVE.String()),
					},
					ClusterID: "test_cluster_id1",
				},
			},
		},
		{
			CVE: &storage.ClusterCVE{
				Id: cve.ID("CVE-3", storage.CVE_K8S_CVE.String()),
				CveBaseInfo: &storage.CVEInfo{
					Cve:          "CVE-3",
					Link:         "https://nvd.nist.gov/vuln/detail/CVE-3",
					ScoreVersion: storage.CVEInfo_V3,
					CvssV3: &storage.CVSSV3{
						Vector:              "AV:L/AC:L/PR:L/UI:N/S:U/C:N/I:L/A:H",
						ExploitabilityScore: 1.8,
						ImpactScore:         4.2,
						AttackVector:        storage.CVSSV3_ATTACK_LOCAL,
						AttackComplexity:    storage.CVSSV3_COMPLEXITY_LOW,
						PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_LOW,
						UserInteraction:     storage.CVSSV3_UI_NONE,
						Scope:               storage.CVSSV3_UNCHANGED,
						Confidentiality:     storage.CVSSV3_IMPACT_NONE,
						Integrity:           storage.CVSSV3_IMPACT_LOW,
						Availability:        storage.CVSSV3_IMPACT_HIGH,
						Score:               6.1,
					},
				},
				Cvss:        6.1,
				ImpactScore: 4.2,
				Type:        storage.CVE_K8S_CVE,
			},
			Children: []converter.EdgeParts{
				{
					Edge: &storage.ClusterCVEEdge{
						Id:        pgSearch.IDFromPks([]string{"test_cluster_id1", cve.ID("CVE-3", storage.CVE_K8S_CVE.String())}),
						IsFixable: false,
						ClusterId: "test_cluster_id1",
						CveId:     cve.ID("CVE-3", storage.CVE_K8S_CVE.String()),
					},
					ClusterID: "test_cluster_id1",
				},
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

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

	mockCVEs.EXPECT().UpsertClusterCVEsInternal(gomock.Any(), storage.CVE_K8S_CVE, cvesToUpsert).Return(nil)
	err = cveManager.orchestratorCVEMgr.updateCVEs(embeddedCVEs, embeddedCVEToClusters, utils.K8s)
	assert.NoError(t, err)
}

func TestClusterCVEOpsInPostgres(t *testing.T) {
	suite.Run(t, new(TestClusterCVEOpsInPostgresTestSuite))
}

type TestClusterCVEOpsInPostgresTestSuite struct {
	suite.Suite

	mockCtrl            *gomock.Controller
	ctx                 context.Context
	testPostgres        *pgtest.TestPostgres
	clusterDataStore    clusterDS.DataStore
	clusterCVEDatastore clusterCVEDataStore.DataStore
	mockNamespaces      *mockNSDataStore.MockDataStore
	mockImages          *mockImageDataStore.MockDataStore
	cveManager          *orchestratorCVEManager
}

func (s *TestClusterCVEOpsInPostgresTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
	s.testPostgres = pgtest.ForT(s.T())
	s.mockCtrl = gomock.NewController(s.T())

	// Create cluster datastore
	s.mockNamespaces = mockNSDataStore.NewMockDataStore(s.mockCtrl)
	s.mockImages = mockImageDataStore.NewMockDataStore(s.mockCtrl)

	// Create cluster cve datastore
	clusterCVEDatastore, err := clusterCVEDataStore.GetTestPostgresDataStore(s.T(), s.testPostgres.DB)
	s.NoError(err)
	s.clusterCVEDatastore = clusterCVEDatastore

	clusterDataStore, err := clusterDS.GetTestPostgresDataStore(s.T(), s.testPostgres.DB)
	s.NoError(err)
	s.clusterDataStore = clusterDataStore

	// Create cve manager
	cveMatcher, err := matcher.NewCVEMatcher(clusterDataStore, s.mockNamespaces, s.mockImages)
	s.NoError(err)

	s.cveManager = &orchestratorCVEManager{
		clusterDataStore:    clusterDataStore,
		clusterCVEDataStore: clusterCVEDatastore,
		cveMatcher:          cveMatcher,
	}
}

func (s *TestClusterCVEOpsInPostgresTestSuite) TearDownSuite() {
	s.testPostgres.Teardown(s.T())
}

func (s *TestClusterCVEOpsInPostgresTestSuite) TestBasicOps() {
	// Upsert cluster.
	c1ID, err := s.clusterDataStore.AddCluster(s.ctx, &storage.Cluster{
		Name:               "c1",
		Labels:             map[string]string{"env": "prod", "team": "team"},
		MainImage:          "docker.io/stackrox/rox:latest",
		CentralApiEndpoint: "central.stackrox:443",
	})
	s.NoError(err)

	// Upsert cluster.
	c2ID, err := s.clusterDataStore.AddCluster(s.ctx, &storage.Cluster{
		Name:               "c2",
		Labels:             map[string]string{"env": "test", "team": "team"},
		MainImage:          "docker.io/stackrox/rox:latest",
		CentralApiEndpoint: "central.stackrox:443",
	})
	s.NoError(err)

	// Upsert cluster.
	c3ID, err := s.clusterDataStore.AddCluster(s.ctx, &storage.Cluster{
		Name:               "c3",
		Labels:             map[string]string{"env": "test", "team": "team"},
		MainImage:          "docker.io/stackrox/rox:latest",
		CentralApiEndpoint: "central.stackrox:443",
	})
	s.NoError(err)

	// Upsert K8s CVEs.

	vulns, clusterMap := getTestClusterCVEParts(10, c1ID, c2ID)
	s.NoError(s.cveManager.updateCVEs(vulns, clusterMap, utils.K8s))
	count, err := s.clusterCVEDatastore.Count(s.ctx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(10, count)

	// Search by matching type.
	results, err := s.clusterCVEDatastore.Search(s.ctx, search.NewQueryBuilder().AddExactMatches(search.CVEType, storage.CVE_K8S_CVE.String()).ProtoQuery())
	s.NoError(err)
	s.Len(results, 10)

	// Search by non-matching type.
	results, err = s.clusterCVEDatastore.Search(s.ctx, search.NewQueryBuilder().AddExactMatches(search.CVEType, storage.CVE_OPENSHIFT_CVE.String()).ProtoQuery())
	s.NoError(err)
	s.Len(results, 0)

	// Search by non-matching type.
	results, err = s.clusterCVEDatastore.Search(s.ctx, search.NewQueryBuilder().AddExactMatches(search.CVEType, storage.CVE_ISTIO_CVE.String()).ProtoQuery())
	s.NoError(err)
	s.Len(results, 0)

	// Search by cluster.
	results, err = s.clusterCVEDatastore.Search(s.ctx, search.NewQueryBuilder().AddExactMatches(search.Cluster, "c1").ProtoQuery())
	s.NoError(err)
	s.Len(results, 10)

	// Suppress CVEs
	start := types.TimestampNow()
	duration := types.DurationProto(10 * time.Minute)
	clusterCVE := utils.EmbeddedVulnerabilityToClusterCVE(storage.CVE_K8S_CVE, vulns[0])
	err = s.clusterCVEDatastore.Suppress(s.ctx, start, duration, vulns[0].GetCve())
	s.NoError(err)

	storedCVE, found, err := s.clusterCVEDatastore.Get(s.ctx, clusterCVE.GetId())
	s.NoError(err)
	s.True(found)
	s.True(storedCVE.GetSnoozed())

	// Reconcile
	s.NoError(s.cveManager.updateCVEs(vulns, clusterMap, utils.K8s))
	count, err = s.clusterCVEDatastore.Count(s.ctx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(10, count)

	// Ensure that snoozed state is persisted.
	storedCVE, found, err = s.clusterCVEDatastore.Get(s.ctx, clusterCVE.GetId())
	s.NoError(err)
	s.True(found)
	s.True(storedCVE.GetSnoozed())

	// Upsert OpenShift CVEs.
	vulns, clusterMap = getTestClusterCVEParts(10, c2ID)
	s.NoError(s.cveManager.updateCVEs(vulns, clusterMap, utils.OpenShift))
	count, err = s.clusterCVEDatastore.Count(s.ctx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(20, count)

	// Search by cluster.
	results, err = s.clusterCVEDatastore.Search(s.ctx, search.NewQueryBuilder().AddExactMatches(search.Cluster, "c2").ProtoQuery())
	s.NoError(err)
	s.Len(results, 20)
	results, err = s.clusterCVEDatastore.Search(s.ctx, search.NewQueryBuilder().AddExactMatches(search.Cluster, "c1").ProtoQuery())
	s.NoError(err)
	s.Len(results, 10)

	// Upsert Istio CVEs.
	vulns, clusterMap = getTestClusterCVEParts(10, c3ID)
	s.NoError(s.cveManager.updateCVEs(vulns, clusterMap, utils.Istio))
	count, err = s.clusterCVEDatastore.Count(s.ctx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(30, count)

	// Search by cluster.
	results, err = s.clusterCVEDatastore.Search(s.ctx, search.NewQueryBuilder().AddExactMatches(search.Cluster, "c3").ProtoQuery())
	s.NoError(err)
	s.Len(results, 10)

	// Upsert more cves and ensure that they are reconciled.
	vulns, clusterMap = getTestClusterCVEParts(20, c1ID)
	s.NoError(s.cveManager.updateCVEs(vulns, clusterMap, utils.K8s))
	count, err = s.clusterCVEDatastore.Count(s.ctx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(40, count)

	// Search by cluster.
	results, err = s.clusterCVEDatastore.Search(s.ctx, search.NewQueryBuilder().AddExactMatches(search.Cluster, "c2").ProtoQuery())
	s.NoError(err)
	s.Len(results, 20)
	results, err = s.clusterCVEDatastore.Search(s.ctx, search.NewQueryBuilder().AddExactMatches(search.Cluster, "c1").ProtoQuery())
	s.NoError(err)
	s.Len(results, 20)

	// Search by type.
	results, err = s.clusterCVEDatastore.Search(s.ctx, search.NewQueryBuilder().AddExactMatches(search.CVEType, storage.CVE_K8S_CVE.String()).ProtoQuery())
	s.NoError(err)
	s.Len(results, 20)
	results, err = s.clusterCVEDatastore.Search(s.ctx, search.NewQueryBuilder().AddExactMatches(search.CVEType, storage.CVE_OPENSHIFT_CVE.String()).ProtoQuery())
	s.NoError(err)
	s.Len(results, 10)
	results, err = s.clusterCVEDatastore.Search(s.ctx, search.NewQueryBuilder().AddExactMatches(search.CVEType, storage.CVE_ISTIO_CVE.String()).ProtoQuery())
	s.NoError(err)
	s.Len(results, 10)
	results, err = s.clusterCVEDatastore.Search(s.ctx, search.ConjunctionQuery(
		search.NewQueryBuilder().AddExactMatches(search.CVEType, storage.CVE_K8S_CVE.String()).ProtoQuery(),
		search.NewQueryBuilder().AddExactMatches(search.Cluster, "c2").ProtoQuery(),
	))
	s.NoError(err)
	s.Len(results, 10)

	// Upsert less cves and ensure that they are reconciled.
	vulns, clusterMap = getTestClusterCVEParts(5, c2ID)
	s.NoError(s.cveManager.updateCVEs(vulns, clusterMap, utils.OpenShift))
	count, err = s.clusterCVEDatastore.Count(s.ctx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(35, count)
	results, err = s.clusterCVEDatastore.Search(s.ctx, search.ConjunctionQuery(
		search.NewQueryBuilder().AddExactMatches(search.CVEType, storage.CVE_OPENSHIFT_CVE.String()).ProtoQuery(),
		search.NewQueryBuilder().AddExactMatches(search.Cluster, "c2").ProtoQuery(),
	))
	s.NoError(err)
	s.Len(results, 5)
	results, err = s.clusterCVEDatastore.Search(s.ctx, search.ConjunctionQuery(
		search.NewQueryBuilder().AddExactMatches(search.CVEType, storage.CVE_OPENSHIFT_CVE.String()).ProtoQuery(),
		search.NewQueryBuilder().AddExactMatches(search.Cluster, "c1").ProtoQuery(),
	))
	s.NoError(err)
	s.Len(results, 0)

	s.NoError(s.clusterCVEDatastore.DeleteClusterCVEsInternal(s.ctx, c2ID))
	count, err = s.clusterCVEDatastore.Count(s.ctx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(30, count)

	s.NoError(s.clusterCVEDatastore.DeleteClusterCVEsInternal(s.ctx, c1ID))
	count, err = s.clusterCVEDatastore.Count(s.ctx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(10, count)

	s.NoError(s.clusterCVEDatastore.DeleteClusterCVEsInternal(s.ctx, c3ID))
	count, err = s.clusterCVEDatastore.Count(s.ctx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(0, count)
}

func getTestClusterCVEParts(numCVEs int, clusters ...string) ([]*storage.EmbeddedVulnerability, map[string][]*storage.Cluster) {
	vulns := make([]*storage.EmbeddedVulnerability, 0, numCVEs)
	vulnToClustersMap := make(map[string][]*storage.Cluster)
	for i := 0; i < numCVEs; i++ {
		cve := fmt.Sprintf("cve-%d", i)
		vulns = append(vulns, &storage.EmbeddedVulnerability{
			Cve: cve,
		})
		for _, cluster := range clusters {
			vulnToClustersMap[cve] = append(vulnToClustersMap[cve], &storage.Cluster{Id: cluster})
		}
	}
	return vulns, vulnToClustersMap
}
