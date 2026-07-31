package matcher

import (
	"context"
	"fmt"
	"testing"

	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
	mockClusterDataStore "github.com/stackrox/rox/central/cluster/datastore/mocks"
	"github.com/stackrox/rox/central/cve/converter/utils"
	mockImagesDataStore "github.com/stackrox/rox/central/image/datastore/mocks"
	mockImageV2DataStore "github.com/stackrox/rox/central/imagev2/datastore/mocks"
	mockNamespaceDataStore "github.com/stackrox/rox/central/namespace/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestCVEMatcher(t *testing.T) {
	suite.Run(t, new(cveMatcherTestSuite))
}

type cveMatcherTestSuite struct {
	suite.Suite

	hasReadCtx  context.Context
	hasWriteCtx context.Context

	cveMatcher *CVEMatcher
	clusters   *mockClusterDataStore.MockDataStore
	namespaces *mockNamespaceDataStore.MockDataStore
	images     *mockImagesDataStore.MockDataStore
	imagesV2   *mockImageV2DataStore.MockDataStore

	mockCtrl *gomock.Controller
}

func (s *cveMatcherTestSuite) SetupTest() {
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Cluster, resources.Namespace, resources.Image)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Cluster, resources.Namespace, resources.Image)))

	s.mockCtrl = gomock.NewController(s.T())
	s.clusters = mockClusterDataStore.NewMockDataStore(s.mockCtrl)
	s.namespaces = mockNamespaceDataStore.NewMockDataStore(s.mockCtrl)
	s.images = mockImagesDataStore.NewMockDataStore(s.mockCtrl)
	s.imagesV2 = mockImageV2DataStore.NewMockDataStore(s.mockCtrl)

	var err error
	s.cveMatcher, err = NewCVEMatcher(s.clusters, s.namespaces, s.images, s.imagesV2)
	s.Require().NoError(err)
}

func (s *cveMatcherTestSuite) TestValidGKEVersion() {
	versions := []string{"v1.12.3-gke.5", "1.15.3-gke.11", "v1.15.3", "1.15.3-gke", "v1.15.3-gke-1", "v1.14-gke.1", "10.0.3.4"}
	expected := []bool{true, true, false, false, false, false, false}
	for i, version := range versions {
		ok := s.cveMatcher.IsGKEVersion(version)
		s.Equal(expected[i], ok)
	}
}

func (s *cveMatcherTestSuite) TestValidEKSVersion() {
	versions := []string{"v1.12.3-eks-ba3d77", "1.15.3-eks-re4w32", "v1.15.3", "1.15.3-gke", "10.0.3.4"}
	expected := []bool{true, true, false, false, false}
	for i, version := range versions {
		ok := s.cveMatcher.IsEKSVersion(version)
		s.Equal(expected[i], ok)
	}
}

func (s *cveMatcherTestSuite) TestMatchVersions() {
	// Tests base-version matching (CPE update=*) and exact-version matching
	// (CPE update=specific) through the public MatchVersions API.
	cases := map[string]struct {
		cpeVersion     string // CPE version field
		cpeUpdate      string // CPE update field (* for base match, specific for exact)
		clusterVersion string
		expected       bool
	}{
		"base match same version": {
			cpeVersion: "1.15.5", cpeUpdate: "*", clusterVersion: "1.15.5", expected: true,
		},
		"base match with v-prefix cluster": {
			cpeVersion: "1.15.5", cpeUpdate: "*", clusterVersion: "v1.15.5", expected: true,
		},
		"base match different version": {
			cpeVersion: "1.14.5", cpeUpdate: "*", clusterVersion: "1.15.5", expected: false,
		},
		"base match with prerelease": {
			cpeVersion: "1.15.5", cpeUpdate: "*", clusterVersion: "1.15.5-beta1", expected: true,
		},
		"base match with build metadata": {
			cpeVersion: "1.15.5", cpeUpdate: "*", clusterVersion: "1.15.5+build", expected: true,
		},
		"exact match same prerelease": {
			cpeVersion: "1.15.5", cpeUpdate: "alpha1", clusterVersion: "1.15.5-alpha1", expected: true,
		},
		"exact match prerelease vs release": {
			cpeVersion: "1.15.5", cpeUpdate: "alpha1", clusterVersion: "1.15.5", expected: false,
		},
		"exact match prerelease with build metadata": {
			cpeVersion: "1.15.5", cpeUpdate: "alpha1", clusterVersion: "1.15.5-alpha1+build1", expected: true,
		},
		"4-segment version coerced to 3-segment": {
			cpeVersion: "1.15.5", cpeUpdate: "*", clusterVersion: "1.15.5.4", expected: true,
		},
		"4-segment version not in range": {
			cpeVersion: "1.14.5", cpeUpdate: "*", clusterVersion: "1.15.3.4", expected: false,
		},
		"garbage version skipped": {
			cpeVersion: "1.15.5", cpeUpdate: "*", clusterVersion: "not-a-version", expected: false,
		},
	}

	for name, tc := range cases {
		s.Run(name, func() {
			node := &schema.NVDCVEFeedJSON10DefNode{
				Operator: "OR",
				CPEMatch: []*schema.NVDCVEFeedJSON10DefCPEMatch{
					{
						Vulnerable: true,
						Cpe23Uri:   fmt.Sprintf("cpe:2.3:a:kubernetes:kubernetes:%s:%s:*:*:*:*:*:*", tc.cpeVersion, tc.cpeUpdate),
					},
				},
			}
			matched, err := s.cveMatcher.MatchVersions(node, tc.clusterVersion, utils.K8s)
			s.NoError(err)
			s.Equal(tc.expected, matched)
		})
	}
}

func (s *cveMatcherTestSuite) TestIfSpecificVersionCVEAffectsCluster() {
	cluster := &storage.Cluster{
		Id:   "test_cluster_id1",
		Name: "cluster1",
		Status: &storage.ClusterStatus{
			OrchestratorMetadata: &storage.OrchestratorMetadata{
				Version: "v1.15.3",
			},
		},
	}

	cve1 := &schema.NVDCVEFeedJSON10DefCVEItem{
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
							Vulnerable: true,
							Cpe23Uri:   "cpe:2.3:a:kubernetes:kubernetes:1.15.3:*:*:*:*:*:*:*",
						},
					},
				},
			},
		},
	}

	cve2 := &schema.NVDCVEFeedJSON10DefCVEItem{
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
							Cpe23Uri:   "cpe:2.3:a:kubernetes:kubernetes:1.14.3:*:*:*:*:*:*:*",
						},
					},
				},
			},
		},
	}

	cve3 := &schema.NVDCVEFeedJSON10DefCVEItem{
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
							Vulnerable: true,
							Cpe23Uri:   "cpe:2.3:a:kubernetes:kubernetes:1.15.3:alpha1:*:*:*:*:*:*",
						},
					},
				},
			},
		},
	}

	ret, _ := s.cveMatcher.IsClusterAffectedByK8sCVE(s.hasReadCtx, cluster, cve1)
	s.Equal(true, ret)
	ret, _ = s.cveMatcher.IsClusterAffectedByK8sCVE(s.hasReadCtx, cluster, cve2)
	s.Equal(false, ret)
	ret, _ = s.cveMatcher.IsClusterAffectedByK8sCVE(s.hasReadCtx, cluster, cve3)
	s.Equal(false, ret)

	cluster.Status.OrchestratorMetadata.Version = "v1.15.3+build1"
	ret, _ = s.cveMatcher.IsClusterAffectedByK8sCVE(s.hasReadCtx, cluster, cve1)
	s.Equal(true, ret)
	ret, _ = s.cveMatcher.IsClusterAffectedByK8sCVE(s.hasReadCtx, cluster, cve2)
	s.Equal(false, ret)
	ret, _ = s.cveMatcher.IsClusterAffectedByK8sCVE(s.hasReadCtx, cluster, cve3)
	s.Equal(false, ret)

	cluster.Status.OrchestratorMetadata.Version = "v1.15.3-alpha1"
	ret, _ = s.cveMatcher.IsClusterAffectedByK8sCVE(s.hasReadCtx, cluster, cve1)
	s.Equal(true, ret)
	ret, _ = s.cveMatcher.IsClusterAffectedByK8sCVE(s.hasReadCtx, cluster, cve2)
	s.Equal(false, ret)
	ret, _ = s.cveMatcher.IsClusterAffectedByK8sCVE(s.hasReadCtx, cluster, cve3)
	s.Equal(true, ret)

	cluster.Status.OrchestratorMetadata.Version = "v1.15.3-alpha1+build1"
	ret, _ = s.cveMatcher.IsClusterAffectedByK8sCVE(s.hasReadCtx, cluster, cve1)
	s.Equal(true, ret)
	ret, _ = s.cveMatcher.IsClusterAffectedByK8sCVE(s.hasReadCtx, cluster, cve2)
	s.Equal(false, ret)
	ret, _ = s.cveMatcher.IsClusterAffectedByK8sCVE(s.hasReadCtx, cluster, cve3)
	s.Equal(true, ret)
}

func (s *cveMatcherTestSuite) TestMultipleVersionCVEAffectsCluster() {
	cluster := &storage.Cluster{
		Id:   "test_cluster_id1",
		Name: "cluster1",
		Status: &storage.ClusterStatus{
			OrchestratorMetadata: &storage.OrchestratorMetadata{
				Version: "v1.15.3",
			},
		},
	}

	cve1 := &schema.NVDCVEFeedJSON10DefCVEItem{
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
						{
							Vulnerable:            true,
							Cpe23Uri:              "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
							VersionStartIncluding: "1.11.1",
							VersionEndExcluding:   "1.11.9",
						},
					},
				},
			},
		},
	}

	cve2 := &schema.NVDCVEFeedJSON10DefCVEItem{
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
							Vulnerable:            true,
							Cpe23Uri:              "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
							VersionStartIncluding: "1.14.1",
							VersionEndExcluding:   "1.14.9",
						},
						{
							Vulnerable:            true,
							Cpe23Uri:              "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
							VersionStartIncluding: "1.15.1",
							VersionEndExcluding:   "1.15.9",
						},
					},
				},
			},
		},
	}

	cve3 := &schema.NVDCVEFeedJSON10DefCVEItem{
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
							VersionStartIncluding: "1.16.5",
							VersionEndIncluding:   "1.16.9",
						},
						{
							Vulnerable:            true,
							Cpe23Uri:              "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
							VersionStartIncluding: "1.17.1",
							VersionEndIncluding:   "1.17.9",
						},
					},
				},
			},
		},
	}

	ret, _ := s.cveMatcher.IsClusterAffectedByK8sCVE(s.hasReadCtx, cluster, cve1)
	s.Equal(false, ret)

	ret, _ = s.cveMatcher.IsClusterAffectedByK8sCVE(s.hasReadCtx, cluster, cve2)
	s.Equal(true, ret)

	cluster.Status.OrchestratorMetadata.Version = "v1.15.1-beta1"
	ret, _ = s.cveMatcher.IsClusterAffectedByK8sCVE(s.hasReadCtx, cluster, cve1)
	s.Equal(false, ret)

	ret, _ = s.cveMatcher.IsClusterAffectedByK8sCVE(s.hasReadCtx, cluster, cve2)
	s.Equal(true, ret)

	cluster.Status.OrchestratorMetadata.Version = "v1.15.9"
	ret, _ = s.cveMatcher.IsClusterAffectedByK8sCVE(s.hasReadCtx, cluster, cve1)
	s.Equal(false, ret)

	ret, _ = s.cveMatcher.IsClusterAffectedByK8sCVE(s.hasReadCtx, cluster, cve2)
	s.Equal(false, ret)

	cluster.Status.OrchestratorMetadata.Version = "v1.15.1"
	ret, _ = s.cveMatcher.IsClusterAffectedByK8sCVE(s.hasReadCtx, cluster, cve1)
	s.Equal(false, ret)

	ret, _ = s.cveMatcher.IsClusterAffectedByK8sCVE(s.hasReadCtx, cluster, cve2)
	s.Equal(true, ret)

	cluster.Status.OrchestratorMetadata.Version = "v1.16.4"
	ret, _ = s.cveMatcher.IsClusterAffectedByK8sCVE(s.hasReadCtx, cluster, cve3)
	s.Equal(false, ret)

	cluster.Status.OrchestratorMetadata.Version = "v1.17.4"
	ret, _ = s.cveMatcher.IsClusterAffectedByK8sCVE(s.hasReadCtx, cluster, cve3)
	s.Equal(true, ret)
}

func (s *cveMatcherTestSuite) TestSingleAndMultipleVersionCVEAffectsCluster() {
	cluster := &storage.Cluster{
		Id:   "test_cluster_id1",
		Name: "cluster1",
		Status: &storage.ClusterStatus{
			OrchestratorMetadata: &storage.OrchestratorMetadata{
				Version: "v1.10.6",
			},
		},
	}
	cve := &schema.NVDCVEFeedJSON10DefCVEItem{
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
						{
							Vulnerable:            true,
							Cpe23Uri:              "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
							VersionStartIncluding: "1.11.1",
							VersionEndExcluding:   "1.11.9",
						},
						{
							Vulnerable: true,
							Cpe23Uri:   "cpe:2.3:a:kubernetes:kubernetes:1.10.3:alpha1:*:*:*:*:*:*",
						},
						{
							Vulnerable: true,
							Cpe23Uri:   "cpe:2.3:a:kubernetes:kubernetes:1.10.3:beta1:*:*:*:*:*:*",
						},
					},
				},
			},
		},
	}

	ret, _ := s.cveMatcher.IsClusterAffectedByK8sCVE(s.hasReadCtx, cluster, cve)
	s.Equal(true, ret)

	cluster.Status.OrchestratorMetadata.Version = "v1.10.3-alpha1"
	ret, _ = s.cveMatcher.IsClusterAffectedByK8sCVE(s.hasReadCtx, cluster, cve)
	s.Equal(true, ret)

	cluster.Status.OrchestratorMetadata.Version = "v1.10.3-beta1"
	ret, _ = s.cveMatcher.IsClusterAffectedByK8sCVE(s.hasReadCtx, cluster, cve)
	s.Equal(true, ret)

	cluster.Status.OrchestratorMetadata.Version = "v1.10.3-rc1"
	ret, _ = s.cveMatcher.IsClusterAffectedByK8sCVE(s.hasReadCtx, cluster, cve)
	s.Equal(true, ret)
}

func (s *cveMatcherTestSuite) TestCountCVEsAffectsCluster() {
	cluster := &storage.Cluster{
		Id:   "test_cluster_id1",
		Name: "cluster1",
		Status: &storage.ClusterStatus{
			OrchestratorMetadata: &storage.OrchestratorMetadata{
				Version: "v1.10.6",
			},
		},
	}
	cves := []*schema.NVDCVEFeedJSON10DefCVEItem{
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

	var countAffectedClusters, countFixableCVEs int
	for _, cve := range cves {
		affected, _ := s.cveMatcher.IsClusterAffectedByK8sCVE(s.hasReadCtx, cluster, cve)
		if affected {
			countAffectedClusters++
		}
		if IsClusterCVEFixable(cve) {
			countFixableCVEs++
		}
	}
	s.Equal(countAffectedClusters, 3)
	s.Equal(countFixableCVEs, 1)
}

func (s *cveMatcherTestSuite) TestNonK8sCPEMatch() {
	cluster := &storage.Cluster{
		Id:   "test_cluster_id1",
		Name: "cluster1",
		Status: &storage.ClusterStatus{
			OrchestratorMetadata: &storage.OrchestratorMetadata{
				Version: "v1.10.6",
			},
		},
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
								Vulnerable:            true,
								Cpe23Uri:              "cpe:2.3:a:vendorfoo:projectbar:*:*:*:*:*:*:*:*",
								VersionStartIncluding: "1.10.1",
								VersionEndExcluding:   "1.10.9",
							},
							{
								Vulnerable:            true,
								Cpe23Uri:              "cpe:2.3:a:vendorfoo:projectbar:*:*:*:*:*:*:*:*",
								VersionStartIncluding: "1.11.1",
								VersionEndExcluding:   "1.11.9",
							},
							{
								Vulnerable: true,
								Cpe23Uri:   "cpe:2.3:a:vendorfoo:projectbar:1.13.1:*:*:*:*:*:*:*",
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
								Cpe23Uri:              "cpe:2.3:a:vendorfoo:projectbar:*:*:*:*:*:*:*:*",
								VersionStartIncluding: "1.10.1",
								VersionEndExcluding:   "1.10.9",
							},
							{
								Vulnerable:            true,
								Cpe23Uri:              "cpe:2.3:a:vendorfoo:projectbar:*:*:*:*:*:*:*:*",
								VersionStartIncluding: "1.11.1",
								VersionEndExcluding:   "1.11.9",
							},
							{
								Vulnerable: true,
								Cpe23Uri:   "cpe:2.3:a:vendorfoo:projectbar:1.13.1:*:*:*:*:*:*:*",
							},
						},
					},
					{
						Operator: "OR",
						CPEMatch: []*schema.NVDCVEFeedJSON10DefCPEMatch{
							{
								Vulnerable:            true,
								Cpe23Uri:              "cpe:2.3:a:kubernetes:kubernetes:*:*:*:*:*:*:*:*",
								VersionStartIncluding: "1.10.1",
								VersionEndExcluding:   "1.10.9",
							},
							{
								Vulnerable: true,
								Cpe23Uri:   "cpe:2.3:a:kubernetes:kubernetes:1.13.1:*:*:*:*:*:*:*",
							},
						},
					},
				},
			},
		},
	}

	ret, _ := s.cveMatcher.IsClusterAffectedByK8sCVE(s.hasReadCtx, cluster, cves[0])
	s.Equal(false, ret)
	ret, _ = s.cveMatcher.IsClusterAffectedByK8sCVE(s.hasReadCtx, cluster, cves[1])
	s.Equal(true, ret)
}

func (s *cveMatcherTestSuite) TestFixableCVEs() {
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
	actual := IsClusterCVEFixable(cves[0])
	s.Equal(actual, true)
	actual = IsClusterCVEFixable(cves[1])
	s.Equal(actual, false)
	actual = IsClusterCVEFixable(cves[2])
	s.Equal(actual, false)
}

func (s *cveMatcherTestSuite) TestIstioCVEImpactsCluster() {
	expected := []bool{true, true, true, false}

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
	}

	namespaces := []search.Result{
		{
			ID: "test_namespace1",
		},
	}

	images := []*storage.Image{
		{
			Id: "test_image_id1",
			Name: &storage.ImageName{
				Tag:      "1.2.6",
				Remote:   "istio/proxyv2",
				Registry: "docker.io",
				FullName: "docker.io/istio/proxyv2:1.2.6",
			},
		},
		{
			Id: "test_image_id2",
			Name: &storage.ImageName{
				Tag:      "1.4.4",
				Remote:   "istio/node-agent-k8s",
				Registry: "docker.io",
				FullName: "docker.io/istio/node-agent-k8s:1.4.4",
			},
		},
		{
			Id: "test_image_id3",
			Name: &storage.ImageName{
				Tag:      "v1.13.11-gke.14",
				Remote:   "kube-proxy",
				Registry: "gke.gcr.io",
				FullName: "gke.gcr.io/kube-proxy:v1.13.11-gke.14",
			},
		},
		{
			Id: "test_image_id4",
			Name: &storage.ImageName{
				Tag:      "v0.11.0",
				Remote:   "jetstack/cert-manager-controller",
				Registry: "quay.io",
				FullName: "quay.io/jetstack/cert-manager-controller:v0.11.0",
			},
		},
	}

	s.clusters.EXPECT().GetClusters(gomock.Any()).Return(clusters, nil).AnyTimes()
	s.namespaces.EXPECT().Search(gomock.Any(), gomock.Any()).Return(namespaces, nil).AnyTimes()

	// Matcher uses images or imagesV2 depending on FlattenImageData.
	imagesV2 := make([]*storage.ImageV2, len(images))
	for i, img := range images {
		imagesV2[i] = &storage.ImageV2{Id: img.GetId(), Name: img.GetName()}
	}
	if features.FlattenImageData.Enabled() {
		s.imagesV2.EXPECT().SearchRawImages(gomock.Any(), gomock.Any()).Return(imagesV2, nil).AnyTimes()
	} else {
		s.images.EXPECT().SearchRawImages(gomock.Any(), gomock.Any()).Return(images, nil).AnyTimes()
	}

	ok, err := s.cveMatcher.isIstioControlPlaneRunning(context.Background())
	s.Nil(err)
	s.Equal(ok, true)

	istioCVEs := []*schema.NVDCVEFeedJSON10DefCVEItem{
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
								Vulnerable:            true,
								Cpe23Uri:              "cpe:2.3:a:istio:istio:*:*:*:*:*:*:*:*",
								VersionStartIncluding: "",
								VersionEndIncluding:   "",
								VersionEndExcluding:   "1.1.13",
							},
							{
								Vulnerable:            true,
								Cpe23Uri:              "cpe:2.3:a:istio:istio:*:*:*:*:*:*:*:*",
								VersionStartIncluding: "1.2.0",
								VersionEndIncluding:   "",
								VersionEndExcluding:   "1.2.8",
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
								Cpe23Uri:              "cpe:2.3:a:istio:istio:*:*:*:*:*:*:*:*",
								VersionStartIncluding: "1.4.1",
								VersionEndIncluding:   "",
								VersionEndExcluding:   "1.4.9",
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
								Cpe23Uri:              "cpe:2.3:a:istio:istio:1.4.4:*:*:*:*:*:*:*",
								VersionStartIncluding: "",
								VersionEndIncluding:   "",
								VersionEndExcluding:   "",
							},
						},
					},
				},
			},
		},
		{
			CVE: &schema.CVEJSON40{
				CVEDataMeta: &schema.CVEJSON40CVEDataMeta{
					ID: "CVE-2019-4",
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
								VersionStartIncluding: "1.3.1",
								VersionEndIncluding:   "",
								VersionEndExcluding:   "1.3.9",
							},
						},
					},
				},
			},
		},
	}

	for i, cve := range istioCVEs {
		actual, err := s.cveMatcher.IsClusterAffectedByIstioCVE(context.Background(), clusters[0], cve)
		s.Nil(err)
		s.Equal(expected[i], actual)
	}
}
