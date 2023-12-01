//go:build sql_integration

package datastoretest

import (
	"context"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	clusterCVEDataStore "github.com/stackrox/rox/central/cve/cluster/datastore"
	cveConverterV2 "github.com/stackrox/rox/central/cve/converter/v2"
	graphDBTestUtils "github.com/stackrox/rox/central/graphdb/testutils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

var (
	allAccessCtx = sac.WithAllAccess(context.Background())
)

func TestClusterCVEDatastoreSAC(t *testing.T) {
	suite.Run(t, new(clusterCVEDatastoreSACSuite))
}

type clusterCVEDatastoreSACSuite struct {
	suite.Suite

	pgStore            clusterCVEDataStore.DataStore
	testGraphDatastore graphDBTestUtils.TestGraphDataStore
}

func (s *clusterCVEDatastoreSACSuite) SetupSuite() {
	var err error
	s.testGraphDatastore, err = graphDBTestUtils.NewTestGraphDataStore(s.T())
	s.Require().NoError(err)
	s.pgStore, err = clusterCVEDataStore.GetTestPostgresDataStore(s.T(), s.testGraphDatastore.GetPostgresPool())
	s.Require().NoError(err)
}

func (s *clusterCVEDatastoreSACSuite) TearDownSuite() {
	s.testGraphDatastore.Cleanup(s.T())
}

func (s *clusterCVEDatastoreSACSuite) cleanImageToVulnerabilitiesGraph() {
	s.Require().NoError(s.testGraphDatastore.CleanClusterToVulnerabilitiesGraph())
}

func getCveID(vulnerability *storage.EmbeddedVulnerability) string {
	return vulnerability.GetCve()
}

type testCase struct {
	name       string
	ctx        context.Context
	visibleCVE map[string]bool
}

func getClusterCVETestCases(_ *testing.T, validCluster1 string, validCluster2 string, readTest bool) []testCase {
	return []testCase{
		{
			name: "Full read-write access has access to all data",
			ctx: sac.WithGlobalAccessScopeChecker(
				context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster),
				),
			),
			visibleCVE: map[string]bool{
				fixtures.GetEmbeddedClusterCVE1234x0001().GetCve(): true,
				fixtures.GetEmbeddedClusterCVE4567x0002().GetCve(): true,
				fixtures.GetEmbeddedClusterCVE2345x0003().GetCve(): true,
			},
		},
		{
			name: "Full read-only access has access to all data",
			ctx: sac.WithGlobalAccessScopeChecker(
				context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster),
				),
			),
			visibleCVE: map[string]bool{
				fixtures.GetEmbeddedClusterCVE1234x0001().GetCve(): readTest,
				fixtures.GetEmbeddedClusterCVE4567x0002().GetCve(): readTest,
				fixtures.GetEmbeddedClusterCVE2345x0003().GetCve(): readTest,
			},
		},
		{
			name: "Full cluster access has access to all data for the cluster",
			ctx: sac.WithGlobalAccessScopeChecker(
				context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster),
					sac.ClusterScopeKeys(validCluster1),
				),
			),
			visibleCVE: map[string]bool{
				fixtures.GetEmbeddedClusterCVE1234x0001().GetCve(): true,
				fixtures.GetEmbeddedClusterCVE4567x0002().GetCve(): true,
				fixtures.GetEmbeddedClusterCVE2345x0003().GetCve(): false,
			},
		},
		{
			name: "Partial cluster access has access to all data for the cluster",
			ctx: sac.WithGlobalAccessScopeChecker(
				context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster),
					sac.ClusterScopeKeys(validCluster1),
					sac.NamespaceScopeKeys(testconsts.NamespaceA),
				),
			),
			visibleCVE: map[string]bool{
				fixtures.GetEmbeddedClusterCVE1234x0001().GetCve(): true,
				fixtures.GetEmbeddedClusterCVE4567x0002().GetCve(): true,
				fixtures.GetEmbeddedClusterCVE2345x0003().GetCve(): false,
			},
		},
		{
			name: "Full access to other cluster has access to all data for that cluster",
			ctx: sac.WithGlobalAccessScopeChecker(
				context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster),
					sac.ClusterScopeKeys(validCluster2),
				),
			),
			visibleCVE: map[string]bool{
				fixtures.GetEmbeddedClusterCVE1234x0001().GetCve(): false,
				fixtures.GetEmbeddedClusterCVE4567x0002().GetCve(): true,
				fixtures.GetEmbeddedClusterCVE2345x0003().GetCve(): true,
			},
		},
		{
			name: "Partial access to other cluster has access to all data for that cluster",
			ctx: sac.WithGlobalAccessScopeChecker(
				context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster),
					sac.ClusterScopeKeys(validCluster2),
					sac.NamespaceScopeKeys(testconsts.NamespaceB),
				),
			),
			visibleCVE: map[string]bool{
				fixtures.GetEmbeddedClusterCVE1234x0001().GetCve(): false,
				fixtures.GetEmbeddedClusterCVE4567x0002().GetCve(): true,
				fixtures.GetEmbeddedClusterCVE2345x0003().GetCve(): true,
			},
		},
		{
			name: "Full access to wrong cluster has access to no data",
			ctx: sac.WithGlobalAccessScopeChecker(
				context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster),
					sac.ClusterScopeKeys(testconsts.WrongCluster),
				),
			),
			visibleCVE: map[string]bool{
				fixtures.GetEmbeddedClusterCVE1234x0001().GetCve(): false,
				fixtures.GetEmbeddedClusterCVE4567x0002().GetCve(): false,
				fixtures.GetEmbeddedClusterCVE2345x0003().GetCve(): false,
			},
		},
	}
}

func getClusterCVEUpsertTestCases(_ *testing.T, validCluster1 string, validCluster2 string) []testCase {
	return []testCase{
		{
			name: "Full read-write access has access to all data",
			ctx: sac.WithGlobalAccessScopeChecker(
				context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster),
				),
			),
			visibleCVE: map[string]bool{
				fixtures.GetEmbeddedClusterCVE1234x0001().GetCve(): true,
				fixtures.GetEmbeddedClusterCVE4567x0002().GetCve(): true,
				fixtures.GetEmbeddedClusterCVE2345x0003().GetCve(): true,
			},
		},
		{
			name: "Full read-only access cannot modify any data",
			ctx: sac.WithGlobalAccessScopeChecker(
				context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster),
				),
			),
			visibleCVE: map[string]bool{
				fixtures.GetEmbeddedClusterCVE1234x0001().GetCve(): false,
				fixtures.GetEmbeddedClusterCVE4567x0002().GetCve(): false,
				fixtures.GetEmbeddedClusterCVE2345x0003().GetCve(): false,
			},
		},
		{
			name: "Full cluster access access cannot modify any data",
			ctx: sac.WithGlobalAccessScopeChecker(
				context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster),
					sac.ClusterScopeKeys(validCluster1),
				),
			),
			visibleCVE: map[string]bool{
				fixtures.GetEmbeddedClusterCVE1234x0001().GetCve(): false,
				fixtures.GetEmbeddedClusterCVE4567x0002().GetCve(): false,
				fixtures.GetEmbeddedClusterCVE2345x0003().GetCve(): false,
			},
		},
		{
			name: "Partial cluster access cannot modify any data",
			ctx: sac.WithGlobalAccessScopeChecker(
				context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster),
					sac.ClusterScopeKeys(validCluster1),
					sac.NamespaceScopeKeys(testconsts.NamespaceA),
				),
			),
			visibleCVE: map[string]bool{
				fixtures.GetEmbeddedClusterCVE1234x0001().GetCve(): false,
				fixtures.GetEmbeddedClusterCVE4567x0002().GetCve(): false,
				fixtures.GetEmbeddedClusterCVE2345x0003().GetCve(): false,
			},
		},
		{
			name: "Full access to other cluster cannot modify any data",
			ctx: sac.WithGlobalAccessScopeChecker(
				context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster),
					sac.ClusterScopeKeys(validCluster2),
				),
			),
			visibleCVE: map[string]bool{
				fixtures.GetEmbeddedClusterCVE1234x0001().GetCve(): false,
				fixtures.GetEmbeddedClusterCVE4567x0002().GetCve(): false,
				fixtures.GetEmbeddedClusterCVE2345x0003().GetCve(): false,
			},
		},
		{
			name: "Partial access to other cluster cannot modify any data",
			ctx: sac.WithGlobalAccessScopeChecker(
				context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster),
					sac.ClusterScopeKeys(validCluster2),
					sac.NamespaceScopeKeys(testconsts.NamespaceB),
				),
			),
			visibleCVE: map[string]bool{
				fixtures.GetEmbeddedClusterCVE1234x0001().GetCve(): false,
				fixtures.GetEmbeddedClusterCVE4567x0002().GetCve(): false,
				fixtures.GetEmbeddedClusterCVE2345x0003().GetCve(): false,
			},
		},
		{
			name: "Full access to wrong cluster cannot modify any data",
			ctx: sac.WithGlobalAccessScopeChecker(
				context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster),
					sac.ClusterScopeKeys(testconsts.WrongCluster),
				),
			),
			visibleCVE: map[string]bool{
				fixtures.GetEmbeddedClusterCVE1234x0001().GetCve(): false,
				fixtures.GetEmbeddedClusterCVE4567x0002().GetCve(): false,
				fixtures.GetEmbeddedClusterCVE2345x0003().GetCve(): false,
			},
		},
	}
}

func getClusterCVESuppressUnsuppressTestCases(_ *testing.T, validCluster1 string, validCluster2 string) []testCase {
	return []testCase{
		{
			name: "Full read-write access has suppress/unsuppress access to all data",
			ctx: sac.WithGlobalAccessScopeChecker(
				context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster, resources.VulnerabilityManagementRequests, resources.VulnerabilityManagementApprovals),
				),
			),
			visibleCVE: map[string]bool{
				fixtures.GetEmbeddedClusterCVE1234x0001().GetCve(): true,
				fixtures.GetEmbeddedClusterCVE4567x0002().GetCve(): true,
				fixtures.GetEmbeddedClusterCVE2345x0003().GetCve(): true,
			},
		},
		{
			name: "Full read-only access cannot suppress or unsuppress any data",
			ctx: sac.WithGlobalAccessScopeChecker(
				context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster, resources.VulnerabilityManagementRequests, resources.VulnerabilityManagementApprovals),
				),
			),
			visibleCVE: map[string]bool{
				fixtures.GetEmbeddedClusterCVE1234x0001().GetCve(): false,
				fixtures.GetEmbeddedClusterCVE4567x0002().GetCve(): false,
				fixtures.GetEmbeddedClusterCVE2345x0003().GetCve(): false,
			},
		},
		{
			name: "Full cluster access cannot suppress or unsuppress any data",
			ctx: sac.WithGlobalAccessScopeChecker(
				context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster, resources.VulnerabilityManagementRequests, resources.VulnerabilityManagementApprovals),
					sac.ClusterScopeKeys(validCluster1),
				),
			),
			visibleCVE: map[string]bool{
				fixtures.GetEmbeddedClusterCVE1234x0001().GetCve(): false,
				fixtures.GetEmbeddedClusterCVE4567x0002().GetCve(): false,
				fixtures.GetEmbeddedClusterCVE2345x0003().GetCve(): false,
			},
		},
		{
			name: "Partial cluster access cannot suppress or unsuppress any data",
			ctx: sac.WithGlobalAccessScopeChecker(
				context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster, resources.VulnerabilityManagementRequests, resources.VulnerabilityManagementApprovals),
					sac.ClusterScopeKeys(validCluster1),
					sac.NamespaceScopeKeys(testconsts.NamespaceA),
				),
			),
			visibleCVE: map[string]bool{
				fixtures.GetEmbeddedClusterCVE1234x0001().GetCve(): false,
				fixtures.GetEmbeddedClusterCVE4567x0002().GetCve(): false,
				fixtures.GetEmbeddedClusterCVE2345x0003().GetCve(): false,
			},
		},
		{
			name: "Full access to other cluster cannot suppress or unsuppress any data",
			ctx: sac.WithGlobalAccessScopeChecker(
				context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster, resources.VulnerabilityManagementRequests, resources.VulnerabilityManagementApprovals),
					sac.ClusterScopeKeys(validCluster2),
				),
			),
			visibleCVE: map[string]bool{
				fixtures.GetEmbeddedClusterCVE1234x0001().GetCve(): false,
				fixtures.GetEmbeddedClusterCVE4567x0002().GetCve(): false,
				fixtures.GetEmbeddedClusterCVE2345x0003().GetCve(): false,
			},
		},
		{
			name: "Partial access to other cluster cannot suppress or unsuppress any data",
			ctx: sac.WithGlobalAccessScopeChecker(
				context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster, resources.VulnerabilityManagementRequests, resources.VulnerabilityManagementApprovals),
					sac.ClusterScopeKeys(validCluster2),
					sac.NamespaceScopeKeys(testconsts.NamespaceB),
				),
			),
			visibleCVE: map[string]bool{
				fixtures.GetEmbeddedClusterCVE1234x0001().GetCve(): false,
				fixtures.GetEmbeddedClusterCVE4567x0002().GetCve(): false,
				fixtures.GetEmbeddedClusterCVE2345x0003().GetCve(): false,
			},
		},
		{
			name: "Full access to wrong cluster cannot suppress or unsuppress any data",
			ctx: sac.WithGlobalAccessScopeChecker(
				context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster, resources.VulnerabilityManagementRequests, resources.VulnerabilityManagementApprovals),
					sac.ClusterScopeKeys(testconsts.WrongCluster),
				),
			),
			visibleCVE: map[string]bool{
				fixtures.GetEmbeddedClusterCVE1234x0001().GetCve(): false,
				fixtures.GetEmbeddedClusterCVE4567x0002().GetCve(): false,
				fixtures.GetEmbeddedClusterCVE2345x0003().GetCve(): false,
			},
		},
	}
}

func embeddedVulnerabilityToClusterCVE(from *storage.EmbeddedVulnerability) *storage.ClusterCVE {
	ret := &storage.ClusterCVE{
		Id: from.GetCve(),
		CveBaseInfo: &storage.CVEInfo{
			Cve:          from.GetCve(),
			Summary:      from.GetSummary(),
			Link:         from.GetLink(),
			PublishedOn:  from.GetPublishedOn(),
			CreatedAt:    from.GetFirstSystemOccurrence(),
			LastModified: from.GetLastModified(),
			CvssV2:       from.GetCvssV2(),
			CvssV3:       from.GetCvssV3(),
		},
		Cvss:         from.GetCvss(),
		Severity:     from.GetSeverity(),
		Snoozed:      from.GetSuppressed(),
		SnoozeStart:  from.GetSuppressActivation(),
		SnoozeExpiry: from.GetSuppressExpiry(),
	}
	if ret.GetCveBaseInfo().GetCvssV3() != nil {
		ret.CveBaseInfo.ScoreVersion = storage.CVEInfo_V3
		ret.ImpactScore = from.GetCvssV3().GetImpactScore()
	} else if ret.GetCveBaseInfo().GetCvssV2() != nil {
		ret.CveBaseInfo.ScoreVersion = storage.CVEInfo_V2
		ret.ImpactScore = from.GetCvssV2().GetImpactScore()
	}
	return ret
}

func (s *clusterCVEDatastoreSACSuite) checkCVEStored(targetCVE string,
	cve *storage.EmbeddedVulnerability,
	shouldBeStored bool) {
	obj, found, err := s.pgStore.Get(allAccessCtx, targetCVE)
	s.NoError(err)
	if shouldBeStored {
		s.True(found)
		s.NotNil(obj)
		s.Equal(cve.GetCvss(), obj.GetCvss())
		s.Equal(cve.GetCvssV3().GetVector(), obj.GetCveBaseInfo().GetCvssV3().GetVector())
	} else {
		s.False(found)
		s.Nil(obj)
	}
}

func (s *clusterCVEDatastoreSACSuite) TestUpsertClusterCVEData() {
	err := s.testGraphDatastore.PushClusterToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph()
	s.Require().NoError(err)
	validClusters := s.testGraphDatastore.GetStoredClusterIDs()
	s.Require().True(len(validClusters) >= 2)

	testCases := getClusterCVEUpsertTestCases(s.T(), validClusters[0], validClusters[1])
	embeddedClusterCVE1 := fixtures.GetEmbeddedClusterCVE1234x0001()
	cve1FixVersion := embeddedClusterCVE1.GetFixedBy()
	embeddedClusterCVE2 := fixtures.GetEmbeddedClusterCVE4567x0002()
	cve2FixVersion := embeddedClusterCVE2.GetFixedBy()
	embeddedClusterCVE3 := fixtures.GetEmbeddedClusterCVE2345x0003()
	cve3FixVersion := embeddedClusterCVE3.GetFixedBy()
	cve1ID := getCveID(embeddedClusterCVE1)
	cve2ID := getCveID(embeddedClusterCVE2)
	cve3ID := getCveID(embeddedClusterCVE3)
	dummyCluster1 := &storage.Cluster{Id: validClusters[0]}
	dummyCluster2 := &storage.Cluster{Id: validClusters[1]}
	cluster1Only := []*storage.Cluster{dummyCluster1}
	cluster2Only := []*storage.Cluster{dummyCluster2}
	clusterCVE1 := embeddedVulnerabilityToClusterCVE(embeddedClusterCVE1)
	clusterCVE2 := embeddedVulnerabilityToClusterCVE(embeddedClusterCVE2)
	clusterCVE3 := embeddedVulnerabilityToClusterCVE(embeddedClusterCVE3)
	clusterCVEParts1x1 := cveConverterV2.NewClusterCVEParts(clusterCVE1, cluster1Only, cve1FixVersion)
	clusterCVEParts1x2 := cveConverterV2.NewClusterCVEParts(clusterCVE2, cluster1Only, cve2FixVersion)
	clusterCVEParts2x2 := cveConverterV2.NewClusterCVEParts(clusterCVE2, cluster2Only, cve2FixVersion)
	clusterCVEParts2x3 := cveConverterV2.NewClusterCVEParts(clusterCVE3, cluster2Only, cve3FixVersion)

	s.Require().NoError(s.pgStore.DeleteClusterCVEsInternal(allAccessCtx, validClusters[0]))
	s.Require().NoError(s.pgStore.DeleteClusterCVEsInternal(allAccessCtx, validClusters[1]))
	for _, c := range testCases {
		s.Run(c.name, func() {
			ctx := c.ctx
			s.checkCVEStored(cve1ID, embeddedClusterCVE1, false)
			err = s.pgStore.UpsertClusterCVEsInternal(ctx, storage.CVE_OPENSHIFT_CVE, clusterCVEParts1x1)
			if c.visibleCVE[cve1ID] {
				s.NoError(err)
				s.checkCVEStored(cve1ID, embeddedClusterCVE1, true)
			} else {
				s.ErrorIs(err, sac.ErrResourceAccessDenied)
			}
			s.checkCVEStored(cve2ID, embeddedClusterCVE2, false)
			err = s.pgStore.UpsertClusterCVEsInternal(ctx, storage.CVE_OPENSHIFT_CVE, clusterCVEParts1x2)
			if c.visibleCVE[cve2ID] {
				s.NoError(err)
				s.checkCVEStored(cve2ID, embeddedClusterCVE2, true)
			} else {
				s.ErrorIs(err, sac.ErrResourceAccessDenied)
			}
			err = s.pgStore.UpsertClusterCVEsInternal(ctx, storage.CVE_K8S_CVE, clusterCVEParts2x2)
			if c.visibleCVE[cve2ID] {
				s.NoError(err)
			} else {
				s.ErrorIs(err, sac.ErrResourceAccessDenied)
			}
			s.checkCVEStored(cve3ID, embeddedClusterCVE3, false)
			err = s.pgStore.UpsertClusterCVEsInternal(ctx, storage.CVE_K8S_CVE, clusterCVEParts2x3)
			if c.visibleCVE[cve3ID] {
				s.NoError(err)
				s.checkCVEStored(cve3ID, embeddedClusterCVE3, true)
			} else {
				s.ErrorIs(err, sac.ErrResourceAccessDenied)
			}
			s.NoError(s.pgStore.DeleteClusterCVEsInternal(allAccessCtx, validClusters[0]))
			s.NoError(s.pgStore.DeleteClusterCVEsInternal(allAccessCtx, validClusters[1]))
		})
	}
}

func (s *clusterCVEDatastoreSACSuite) TestDeleteClusterCVEData() {
	err := s.testGraphDatastore.PushClusterToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph()
	s.Require().NoError(err)
	validClusters := s.testGraphDatastore.GetStoredClusterIDs()
	s.Require().True(len(validClusters) >= 2)

	testCases := getClusterCVETestCases(s.T(), validClusters[0], validClusters[1], false)

	// Prepare data for re-injection
	embeddedClusterCVE1 := fixtures.GetEmbeddedClusterCVE1234x0001()
	cve1FixVersion := embeddedClusterCVE1.GetFixedBy()
	embeddedClusterCVE2 := fixtures.GetEmbeddedClusterCVE4567x0002()
	cve2FixVersion := embeddedClusterCVE2.GetFixedBy()
	embeddedClusterCVE3 := fixtures.GetEmbeddedClusterCVE2345x0003()
	cve3FixVersion := embeddedClusterCVE3.GetFixedBy()
	cve1ID := getCveID(embeddedClusterCVE1)
	cve2ID := getCveID(embeddedClusterCVE2)
	cve3ID := getCveID(embeddedClusterCVE3)
	dummyCluster1 := &storage.Cluster{Id: validClusters[0]}
	dummyCluster2 := &storage.Cluster{Id: validClusters[1]}
	cluster1Only := []*storage.Cluster{dummyCluster1}
	cluster2Only := []*storage.Cluster{dummyCluster2}
	clusterCVE1 := embeddedVulnerabilityToClusterCVE(embeddedClusterCVE1)
	clusterCVE2 := embeddedVulnerabilityToClusterCVE(embeddedClusterCVE2)
	clusterCVE3 := embeddedVulnerabilityToClusterCVE(embeddedClusterCVE3)
	clusterCVEParts1x1 := cveConverterV2.NewClusterCVEParts(clusterCVE1, cluster1Only, cve1FixVersion)
	clusterCVEParts1x2 := cveConverterV2.NewClusterCVEParts(clusterCVE2, cluster1Only, cve2FixVersion)
	clusterCVEParts2x2 := cveConverterV2.NewClusterCVEParts(clusterCVE2, cluster2Only, cve2FixVersion)
	clusterCVEParts2x3 := cveConverterV2.NewClusterCVEParts(clusterCVE3, cluster2Only, cve3FixVersion)

	s.Require().NoError(s.pgStore.DeleteClusterCVEsInternal(allAccessCtx, validClusters[0]))
	s.Require().NoError(s.pgStore.DeleteClusterCVEsInternal(allAccessCtx, validClusters[1]))
	for _, c := range testCases {
		s.Run(c.name, func() {
			err = s.pgStore.UpsertClusterCVEsInternal(allAccessCtx, storage.CVE_OPENSHIFT_CVE, clusterCVEParts1x1)
			s.Require().NoError(err)
			err = s.pgStore.UpsertClusterCVEsInternal(allAccessCtx, storage.CVE_OPENSHIFT_CVE, clusterCVEParts1x2)
			s.Require().NoError(err)
			err = s.pgStore.UpsertClusterCVEsInternal(allAccessCtx, storage.CVE_K8S_CVE, clusterCVEParts2x2)
			s.Require().NoError(err)
			err = s.pgStore.UpsertClusterCVEsInternal(allAccessCtx, storage.CVE_K8S_CVE, clusterCVEParts2x3)
			s.Require().NoError(err)
			s.checkCVEStored(cve1ID, embeddedClusterCVE1, true)
			s.checkCVEStored(cve2ID, embeddedClusterCVE2, true)
			s.checkCVEStored(cve3ID, embeddedClusterCVE3, true)
			err = s.pgStore.DeleteClusterCVEsInternal(c.ctx, validClusters[0])
			if c.visibleCVE[cve1ID] {
				s.NoError(err)
				s.checkCVEStored(cve1ID, embeddedClusterCVE1, false)
			} else {
				s.ErrorIs(err, sac.ErrResourceAccessDenied)
			}
			err = s.pgStore.DeleteClusterCVEsInternal(c.ctx, validClusters[1])
			if c.visibleCVE[cve3ID] {
				s.NoError(err)
				s.checkCVEStored(cve3ID, embeddedClusterCVE3, false)
			} else {
				s.ErrorIs(err, sac.ErrResourceAccessDenied)
			}
			if c.visibleCVE[cve1ID] && c.visibleCVE[cve3ID] {
				s.checkCVEStored(cve2ID, embeddedClusterCVE2, false)
			}
			s.Require().NoError(s.pgStore.DeleteClusterCVEsInternal(allAccessCtx, validClusters[0]))
			s.Require().NoError(s.pgStore.DeleteClusterCVEsInternal(allAccessCtx, validClusters[1]))
		})
	}
}

func (s *clusterCVEDatastoreSACSuite) runTestExistCVE(targetCVE string) {
	err := s.testGraphDatastore.PushClusterToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph()
	s.Require().NoError(err)
	validClusters := s.testGraphDatastore.GetStoredClusterIDs()
	s.Require().True(len(validClusters) >= 2)

	for _, c := range getClusterCVETestCases(s.T(), validClusters[0], validClusters[1], true) {
		s.Run(c.name, func() {
			ctx := c.ctx
			var exists bool
			exists, err = s.pgStore.Exists(ctx, targetCVE)
			s.NoError(err)
			s.Equal(c.visibleCVE[targetCVE], exists)
		})
	}
}

func (s *clusterCVEDatastoreSACSuite) TestExistsSingleCVE() {
	targetCVE := getCveID(fixtures.GetEmbeddedClusterCVE1234x0001())
	s.runTestExistCVE(targetCVE)
}

func (s *clusterCVEDatastoreSACSuite) TestExistsSharedCVE() {
	targetCVE := getCveID(fixtures.GetEmbeddedClusterCVE4567x0002())
	s.runTestExistCVE(targetCVE)
}

func (s *clusterCVEDatastoreSACSuite) runTestGetCVE(targetCVE string, cveObj *storage.EmbeddedVulnerability) {
	err := s.testGraphDatastore.PushClusterToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph()
	s.Require().NoError(err)
	validClusters := s.testGraphDatastore.GetStoredClusterIDs()
	s.Require().True(len(validClusters) >= 2)

	for _, c := range getClusterCVETestCases(s.T(), validClusters[0], validClusters[1], true) {
		s.Run(c.name, func() {
			ctx := c.ctx
			var v3AttackVector string
			var cvss float32
			obj, exists, err := s.pgStore.Get(ctx, targetCVE)
			if c.visibleCVE[targetCVE] {
				s.NotNil(obj)
			} else {
				s.Nil(obj)
			}
			if exists {
				v3AttackVector = obj.GetCveBaseInfo().GetCvssV3().GetVector()
				cvss = obj.GetCvss()
			}

			if c.visibleCVE[targetCVE] {
				s.Equal(cveObj.GetCvss(), cvss)
				s.Equal(cveObj.GetCvssV3().GetVector(), v3AttackVector)
				s.True(exists)
			} else {
				s.False(exists)
			}
			s.NoError(err)
			s.Equal(c.visibleCVE[targetCVE], exists)
		})
	}
}

func (s *clusterCVEDatastoreSACSuite) TestGetSingleCVE() {
	targetCVE := getCveID(fixtures.GetEmbeddedClusterCVE1234x0001())
	cveObj := fixtures.GetEmbeddedClusterCVE1234x0001()
	s.runTestGetCVE(targetCVE, cveObj)
}

func (s *clusterCVEDatastoreSACSuite) TestGetSharedCVE() {
	targetCVE := getCveID(fixtures.GetEmbeddedClusterCVE4567x0002())
	cveObj := fixtures.GetEmbeddedClusterCVE4567x0002()
	s.runTestGetCVE(targetCVE, cveObj)
}

func (s *clusterCVEDatastoreSACSuite) TestGetBatch() {
	err := s.testGraphDatastore.PushClusterToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph()
	s.Require().NoError(err)
	validClusters := s.testGraphDatastore.GetStoredClusterIDs()
	s.Require().True(len(validClusters) >= 2)

	targetCVE1 := getCveID(fixtures.GetEmbeddedClusterCVE1234x0001())
	targetCVE2 := getCveID(fixtures.GetEmbeddedClusterCVE4567x0002())
	targetCVE3 := getCveID(fixtures.GetEmbeddedClusterCVE2345x0003())
	cve1 := fixtures.GetEmbeddedClusterCVE1234x0001()
	cve2 := fixtures.GetEmbeddedClusterCVE4567x0002()
	cve3 := fixtures.GetEmbeddedClusterCVE2345x0003()

	targetCVEs := []string{targetCVE1, targetCVE2, targetCVE3}

	testCases := getClusterCVETestCases(s.T(), validClusters[0], validClusters[1], true)
	for _, c := range testCases {
		s.Run(c.name, func() {
			ctx := c.ctx
			vectorsPerCVE := make(map[string]string, 0)
			cvssPerCVE := make(map[string]float32, 0)
			visibleCVEs := 0
			for _, visible := range c.visibleCVE {
				if visible {
					visibleCVEs++
				}
			}
			results, err := s.pgStore.GetBatch(ctx, targetCVEs)
			s.NoError(err)
			s.Equal(visibleCVEs, len(results))
			for _, cve := range results {
				cveName := cve.GetCveBaseInfo().GetCve()
				cvss := cve.GetCvss()
				v3AttackVector := cve.GetCveBaseInfo().GetCvssV3().GetVector()
				vectorsPerCVE[cveName] = v3AttackVector
				cvssPerCVE[cveName] = cvss
			}

			if c.visibleCVE[targetCVE1] {
				s.Equal(cve1.GetCvssV3().GetVector(), vectorsPerCVE[targetCVE1])
				s.Equal(cve1.GetCvss(), cvssPerCVE[targetCVE1])
			}
			if c.visibleCVE[targetCVE2] {
				s.Equal(cve2.GetCvssV3().GetVector(), vectorsPerCVE[targetCVE2])
				s.Equal(cve2.GetCvss(), cvssPerCVE[targetCVE2])
			}
			if c.visibleCVE[targetCVE3] {
				s.Equal(cve3.GetCvssV3().GetVector(), vectorsPerCVE[targetCVE3])
				s.Equal(cve3.GetCvss(), cvssPerCVE[targetCVE3])
			}
		})
	}
}

func (s *clusterCVEDatastoreSACSuite) TestCount() {
	err := s.testGraphDatastore.PushClusterToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph()
	s.Require().NoError(err)
	validClusters := s.testGraphDatastore.GetStoredClusterIDs()
	s.Require().True(len(validClusters) >= 2)

	testCases := getClusterCVETestCases(s.T(), validClusters[0], validClusters[1], true)
	for _, c := range testCases {
		s.Run(c.name, func() {
			ctx := c.ctx
			visibleCVEs := 0
			for _, visible := range c.visibleCVE {
				if visible {
					visibleCVEs++
				}
			}
			count, err := s.pgStore.Count(ctx, search.EmptyQuery())
			s.NoError(err)
			s.Equal(visibleCVEs, count)

		})
	}
}

func (s *clusterCVEDatastoreSACSuite) TestSearch() {
	err := s.testGraphDatastore.PushClusterToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph()
	s.Require().NoError(err)
	validClusters := s.testGraphDatastore.GetStoredClusterIDs()
	s.Require().True(len(validClusters) >= 2)

	testCases := getClusterCVETestCases(s.T(), validClusters[0], validClusters[1], true)
	for _, c := range testCases {
		s.Run(c.name, func() {
			ctx := c.ctx
			visibleCVEs := 0
			for _, visible := range c.visibleCVE {
				if visible {
					visibleCVEs++
				}
			}
			foundIDs := make([]string, 0, 3)
			results, err := s.pgStore.Search(ctx, search.EmptyQuery())
			s.NoError(err)
			s.Equal(visibleCVEs, len(results))
			for _, r := range results {
				foundIDs = append(foundIDs, r.ID)
			}

			for _, identifier := range foundIDs {
				s.True(c.visibleCVE[identifier])
			}
		})
	}

}

func (s *clusterCVEDatastoreSACSuite) TestSearchClusterCVEs() {
	err := s.testGraphDatastore.PushClusterToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph()
	s.Require().NoError(err)
	validClusters := s.testGraphDatastore.GetStoredClusterIDs()
	s.Require().True(len(validClusters) >= 2)

	testCases := getClusterCVETestCases(s.T(), validClusters[0], validClusters[1], true)
	for _, c := range testCases {
		s.Run(c.name, func() {
			ctx := c.ctx
			visibleCVEs := 0
			for _, visible := range c.visibleCVE {
				if visible {
					visibleCVEs++
				}
			}
			foundIDs := make([]string, 0, 3)
			results, err := s.pgStore.SearchClusterCVEs(ctx, search.EmptyQuery())
			s.NoError(err)
			s.Equal(visibleCVEs, len(results))
			for _, r := range results {
				foundIDs = append(foundIDs, r.GetId())
			}

			for _, identifier := range foundIDs {
				s.True(c.visibleCVE[identifier])
			}
		})
	}
}

func (s *clusterCVEDatastoreSACSuite) TestSearchRawClusterCVEs() {
	err := s.testGraphDatastore.PushClusterToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph()
	s.Require().NoError(err)
	validClusters := s.testGraphDatastore.GetStoredClusterIDs()
	s.Require().True(len(validClusters) >= 2)

	targetCVE1 := getCveID(fixtures.GetEmbeddedClusterCVE1234x0001())
	targetCVE2 := getCveID(fixtures.GetEmbeddedClusterCVE4567x0002())
	targetCVE3 := getCveID(fixtures.GetEmbeddedClusterCVE2345x0003())
	cve1 := fixtures.GetEmbeddedClusterCVE1234x0001()
	cve2 := fixtures.GetEmbeddedClusterCVE4567x0002()
	cve3 := fixtures.GetEmbeddedClusterCVE2345x0003()

	testCases := getClusterCVETestCases(s.T(), validClusters[0], validClusters[1], true)
	for _, c := range testCases {
		s.Run(c.name, func() {
			ctx := c.ctx
			visibleCVEs := 0
			for _, visible := range c.visibleCVE {
				if visible {
					visibleCVEs++
				}
			}
			vectorsPerCVE := make(map[string]string, 0)
			cvssPerCVE := make(map[string]float32, 0)
			foundIDs := make([]string, 0, 3)

			results, err := s.pgStore.SearchRawCVEs(ctx, search.EmptyQuery())
			s.NoError(err)
			s.Equal(visibleCVEs, len(results))
			for _, r := range results {
				foundIDs = append(foundIDs, r.GetId())
				cveName := r.GetId()
				v3AttackVector := r.GetCveBaseInfo().GetCvssV3().GetVector()
				cvss := r.GetCvss()
				vectorsPerCVE[cveName] = v3AttackVector
				cvssPerCVE[cveName] = cvss
			}

			for _, identifier := range foundIDs {
				s.True(c.visibleCVE[identifier])
			}
			if c.visibleCVE[targetCVE1] {
				s.Equal(cve1.GetCvssV3().GetVector(), vectorsPerCVE[targetCVE1])
				s.Equal(cve1.GetCvss(), cvssPerCVE[targetCVE1])
			}
			if c.visibleCVE[targetCVE2] {
				s.Equal(cve2.GetCvssV3().GetVector(), vectorsPerCVE[targetCVE2])
				s.Equal(cve2.GetCvss(), cvssPerCVE[targetCVE2])
			}
			if c.visibleCVE[targetCVE3] {
				s.Equal(cve3.GetCvssV3().GetVector(), vectorsPerCVE[targetCVE3])
				s.Equal(cve3.GetCvss(), cvssPerCVE[targetCVE3])
			}
		})
	}
}

func addDurationToTimestamp(ts *types.Timestamp, duration *types.Duration) *types.Timestamp {
	nanos := ts.GetNanos() + duration.GetNanos()
	seconds := ts.GetSeconds() + duration.GetSeconds()
	nanosInSecond := int32(1000 * 1000 * 1000)
	if nanos >= nanosInSecond {
		seconds += int64(nanos / nanosInSecond)
	}
	return &types.Timestamp{
		Seconds: seconds,
		Nanos:   int32(0),
	}
}

func (s *clusterCVEDatastoreSACSuite) checkCVESnoozed(targetCVE string,
	snoozeStart *types.Timestamp,
	snoozeDuration *types.Duration,
	shouldBeSnoozed bool) {
	var err error
	var found bool
	var objSnoozed bool
	var objSnoozeStart *types.Timestamp
	var objSnoozeExpiry *types.Timestamp
	expectedSnoozeExpire := addDurationToTimestamp(snoozeStart, snoozeDuration)

	var obj *storage.ClusterCVE
	obj, found, err = s.pgStore.Get(allAccessCtx, targetCVE)
	if found {
		objSnoozed = obj.GetSnoozed()
		objSnoozeStart = obj.GetSnoozeStart()
		objSnoozeExpiry = obj.GetSnoozeExpiry()
	} else {
		objSnoozed = false
		objSnoozeStart = nil
		objSnoozeExpiry = nil
	}

	s.NoError(err)
	s.True(found)
	if shouldBeSnoozed {
		s.True(objSnoozed)
		s.Equal(snoozeStart, objSnoozeStart)
		s.Equal(expectedSnoozeExpire, objSnoozeExpiry)
	} else {
		s.False(objSnoozed)
	}
}

func (s *clusterCVEDatastoreSACSuite) checkCVEUnsnoozed(targetCVE string,
	snoozeStart *types.Timestamp,
	snoozeDuration *types.Duration,
	shouldBeUnsnoozed bool) {

	var objSnoozed bool
	var objSnoozeStart *types.Timestamp
	var objSnoozeExpiry *types.Timestamp
	expectedSnoozeExpire := addDurationToTimestamp(snoozeStart, snoozeDuration)
	obj, found, err := s.pgStore.Get(allAccessCtx, targetCVE)
	if found {
		objSnoozed = obj.GetSnoozed()
		objSnoozeStart = obj.GetSnoozeStart()
		objSnoozeExpiry = obj.GetSnoozeExpiry()
	} else {
		objSnoozed = false
		objSnoozeStart = nil
		objSnoozeExpiry = nil
	}

	s.NoError(err)
	s.True(found)
	if shouldBeUnsnoozed {
		s.False(objSnoozed)
	} else {
		s.True(objSnoozed)
		s.Equal(snoozeStart, objSnoozeStart)
		s.Equal(expectedSnoozeExpire, objSnoozeExpiry)
	}
}

func (s *clusterCVEDatastoreSACSuite) runTestSuppressUnsuppressCVE(targetCVE string) {
	err := s.testGraphDatastore.PushClusterToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph()
	s.Require().NoError(err)
	validClusters := s.testGraphDatastore.GetStoredClusterIDs()
	s.Require().True(len(validClusters) >= 2)

	for _, c := range getClusterCVESuppressUnsuppressTestCases(s.T(), validClusters[0], validClusters[1]) {
		s.Run(c.name, func() {
			ctx := c.ctx
			snoozeStart := types.TimestampNow()
			snoozeStart.Nanos = 0
			snoozeDuration := types.DurationProto(10 * time.Minute)

			err = s.pgStore.Suppress(ctx, snoozeStart, snoozeDuration, targetCVE)
			if c.visibleCVE[targetCVE] {
				s.NoError(err)
			} else {
				s.ErrorIs(err, sac.ErrResourceAccessDenied)
			}
			s.checkCVESnoozed(targetCVE, snoozeStart, snoozeDuration, c.visibleCVE[targetCVE])
			if !c.visibleCVE[targetCVE] {

				err = s.pgStore.Suppress(allAccessCtx, snoozeStart, snoozeDuration, targetCVE)

				s.NoError(err)
			}
			// Ensure the object is now snoozed
			s.checkCVESnoozed(targetCVE, snoozeStart, snoozeDuration, true)
			// Unsuppress
			err = s.pgStore.Unsuppress(ctx, targetCVE)

			if c.visibleCVE[targetCVE] {
				s.NoError(err)
			} else {
				s.ErrorIs(err, sac.ErrResourceAccessDenied)
			}
			// Check unsuppressed worked
			s.checkCVEUnsnoozed(targetCVE, snoozeStart, snoozeDuration, c.visibleCVE[targetCVE])
			if !c.visibleCVE[targetCVE] {
				err = s.pgStore.Unsuppress(allAccessCtx, targetCVE)
				s.NoError(err)
			}
		})
	}
}

func (s *clusterCVEDatastoreSACSuite) TestSuppressUnsuppressSingleCVE() {
	targetCVE := getCveID(fixtures.GetEmbeddedClusterCVE1234x0001())
	s.runTestSuppressUnsuppressCVE(targetCVE)
}

func (s *clusterCVEDatastoreSACSuite) TestSuppressUnsuppressSharedCVE() {
	targetCVE := getCveID(fixtures.GetEmbeddedClusterCVE4567x0002())
	s.runTestSuppressUnsuppressCVE(targetCVE)
}
