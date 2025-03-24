//go:build sql_integration

package platformcve

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	clusterCVEDS "github.com/stackrox/rox/central/cve/cluster/datastore"
	"github.com/stackrox/rox/central/cve/converter/v2"
	converterV2 "github.com/stackrox/rox/central/cve/converter/v2"
	"github.com/stackrox/rox/central/views/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pkgCVE "github.com/stackrox/rox/pkg/cve"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type testCase struct {
	desc        string
	ctx         context.Context
	q           *v1.Query
	matchFilter *filterImpl
	expectedErr string
}

type sacTestCase struct {
	desc            string
	ctx             context.Context
	visibleClusters set.StringSet
}

type paginationTestCase struct {
	desc   string
	q      *v1.Query
	offset int
	limit  int
	less   lessFunc
}

type lessFunc func(records []CveCore) func(i, j int) bool

type filterImpl struct {
	matchCluster  func(cluster *storage.Cluster) bool
	matchCVEParts func(cvePart converter.ClusterCVEParts) bool
}

func matchAllFilter() *filterImpl {
	return &filterImpl{
		matchCluster: func(_ *storage.Cluster) bool {
			return true
		},
		matchCVEParts: func(_ converter.ClusterCVEParts) bool {
			return true
		},
	}
}

func matchNoneFilter() *filterImpl {
	return &filterImpl{
		matchCluster: func(_ *storage.Cluster) bool {
			return false
		},
		matchCVEParts: func(_ converter.ClusterCVEParts) bool {
			return false
		},
	}
}

func (f *filterImpl) withClusterFilter(fn func(cluster *storage.Cluster) bool) *filterImpl {
	f.matchCluster = fn
	return f
}

func (f *filterImpl) withCVEPartsFilter(fn func(cveParts converter.ClusterCVEParts) bool) *filterImpl {
	f.matchCVEParts = fn
	return f
}

func TestPlatformCVEView(t *testing.T) {
	suite.Run(t, new(PlatformCVEViewTestSuite))
}

type PlatformCVEViewTestSuite struct {
	suite.Suite

	testDB  *pgtest.TestPostgres
	cveView CveView
	// ClusterID -> *storage.Cluster
	clusterMap map[string]*storage.Cluster
	// ClusterName -> ClusterID
	clusterNameToIDMap map[string]string
	cvePartsList       []converter.ClusterCVEParts
}

func (s *PlatformCVEViewTestSuite) SetupSuite() {
	ctx := sac.WithAllAccess(context.Background())
	s.testDB = pgtest.ForT(s.T())

	// Initialize the datastore.
	clusterDatastore, err := clusterDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.Require().NoError(err)
	clusterCVEDatastore, err := clusterCVEDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.Require().NoError(err)

	clusterNameToIDMap := make(map[string]string)
	// Upsert test data
	clusterMap, cvePartsByType := getTestData()
	for _, c := range clusterMap {
		err = clusterDatastore.UpdateCluster(ctx, c)
		s.Require().NoError(err)
		clusterNameToIDMap[c.GetName()] = c.GetId()
	}

	var allCvePartsList []converter.ClusterCVEParts
	for cveType, cvePartsList := range cvePartsByType {
		err = clusterCVEDatastore.UpsertClusterCVEsInternal(ctx, cveType, cvePartsList...)
		s.Require().NoError(err)
		allCvePartsList = append(allCvePartsList, cvePartsList...)
	}

	for i, cveParts := range allCvePartsList {
		stored, exists, err := clusterCVEDatastore.Get(ctx, cveParts.CVE.GetId())
		s.Require().NoError(err)
		s.Require().True(exists)
		allCvePartsList[i].CVE = stored
	}

	s.clusterMap = clusterMap
	s.clusterNameToIDMap = clusterNameToIDMap
	s.cvePartsList = allCvePartsList
	s.cveView = NewCVEView(s.testDB.DB)
}

func (s *PlatformCVEViewTestSuite) TestGetPlatformCVECore() {
	for _, tc := range s.testCases() {
		s.T().Run(tc.desc, func(t *testing.T) {
			actual, err := s.cveView.Get(sac.WithAllAccess(tc.ctx), tc.q)
			if tc.expectedErr != "" {
				s.ErrorContains(err, tc.expectedErr)
				return
			}
			assert.NoError(t, err)

			expected := s.compileExpectedCVECores(tc.matchFilter)
			assert.Equal(t, len(expected), len(actual))
			assert.ElementsMatch(t, expected, actual)

			for _, record := range actual {
				// The total cluster count should be equal to aggregation of the all platform type counts.
				assert.Equal(t,
					record.GetClusterCountByPlatformType().GetGenericClusterCount()+
						record.GetClusterCountByPlatformType().GetKubernetesClusterCount()+
						record.GetClusterCountByPlatformType().GetOpenshiftClusterCount()+
						record.GetClusterCountByPlatformType().GetOpenshift4ClusterCount(),
					record.GetClusterCount(),
				)
			}
		})
	}
}

func (s *PlatformCVEViewTestSuite) TestGetPlatformCVECoreSAC() {
	for _, tc := range s.testCases() {
		for _, sacTC := range s.sacTestCases(tc.ctx) {
			s.T().Run(fmt.Sprintf("SAC desc: %s; test desc: %s ", sacTC.desc, tc.desc), func(t *testing.T) {
				actual, err := s.cveView.Get(sacTC.ctx, tc.q)
				if tc.expectedErr != "" {
					s.ErrorContains(err, tc.expectedErr)
					return
				}
				assert.NoError(t, err)

				// Wrap cluster filter with sac filter.
				filterWithSAC := matchAllFilter().withClusterFilter(func(cluster *storage.Cluster) bool {
					if sacTC.visibleClusters.Contains(cluster.GetId()) {
						return tc.matchFilter.matchCluster(cluster)
					}
					return false
				})
				filterWithSAC.matchCVEParts = tc.matchFilter.matchCVEParts

				expected := s.compileExpectedCVECores(filterWithSAC)
				assert.Equal(t, len(expected), len(actual))
				assert.ElementsMatch(t, expected, actual)
			})
		}
	}
}

func (s *PlatformCVEViewTestSuite) TestGetPlatformCVECoreWithPagination() {
	for _, paginationTc := range s.paginationTestCases() {
		testCases := s.testCases()
		for i := range testCases {
			tc := &testCases[i]
			applyPaginationProps(tc, paginationTc)
			s.T().Run(tc.desc, func(t *testing.T) {
				actual, err := s.cveView.Get(sac.WithAllAccess(tc.ctx), tc.q)
				if tc.expectedErr != "" {
					s.ErrorContains(err, tc.expectedErr)
					return
				}
				assert.NoError(t, err)

				expected := s.compileExpectedCVECoresWithPagination(tc.matchFilter, paginationTc.less, paginationTc.offset, paginationTc.limit)
				assert.Equal(t, len(expected), len(actual))
				assert.EqualValues(t, expected, actual)

				for _, record := range actual {
					// The total cluster count should be equal to aggregation of the all platform type counts.
					assert.Equal(t,
						record.GetClusterCountByPlatformType().GetGenericClusterCount()+
							record.GetClusterCountByPlatformType().GetKubernetesClusterCount()+
							record.GetClusterCountByPlatformType().GetOpenshiftClusterCount()+
							record.GetClusterCountByPlatformType().GetOpenshift4ClusterCount(),
						record.GetClusterCount(),
					)
				}
			})
		}
	}
}

func (s *PlatformCVEViewTestSuite) TestGetClusterIDs() {
	for _, tc := range s.testCases() {
		s.T().Run(tc.desc, func(t *testing.T) {
			// Such testcases are meant only for Get().
			if tc.expectedErr != "" {
				return
			}

			actualAffectedClusterIDs, err := s.cveView.GetClusterIDs(sac.WithAllAccess(tc.ctx), tc.q)
			assert.NoError(t, err)
			expectedAffectedClusterIDs := s.compileExpectedAffectedClusterIDs(tc.matchFilter)
			assert.ElementsMatch(t, expectedAffectedClusterIDs, actualAffectedClusterIDs)
		})
	}
}

func (s *PlatformCVEViewTestSuite) TestGetClusterIDsSAC() {
	for _, tc := range s.testCases() {
		for _, sacTC := range s.sacTestCases(tc.ctx) {
			s.T().Run(fmt.Sprintf("SAC desc: %s; test desc: %s ", sacTC.desc, tc.desc), func(t *testing.T) {
				// Such testcases are meant only for Get().
				if tc.expectedErr != "" {
					return
				}
				actualAffectedClusterIDs, err := s.cveView.GetClusterIDs(sacTC.ctx, tc.q)
				assert.NoError(t, err)

				// Wrap cluster filter with sac filter.
				filterWithSAC := matchAllFilter().withClusterFilter(func(cluster *storage.Cluster) bool {
					if sacTC.visibleClusters.Contains(cluster.GetId()) {
						return tc.matchFilter.matchCluster(cluster)
					}
					return false
				})
				filterWithSAC.matchCVEParts = tc.matchFilter.matchCVEParts

				expectedAffectedClusterIDs := s.compileExpectedAffectedClusterIDs(filterWithSAC)
				assert.ElementsMatch(t, expectedAffectedClusterIDs, actualAffectedClusterIDs)
			})
		}
	}
}

func (s *PlatformCVEViewTestSuite) TestCountPlatformCVECore() {
	for _, tc := range s.testCases() {
		s.T().Run(tc.desc, func(t *testing.T) {
			actual, err := s.cveView.Count(sac.WithAllAccess(tc.ctx), tc.q)
			if tc.expectedErr != "" {
				s.ErrorContains(err, tc.expectedErr)
				return
			}
			assert.NoError(t, err)

			expected := s.compileExpectedCVECores(tc.matchFilter)
			assert.Equal(t, len(expected), actual)
		})
	}
}

func (s *PlatformCVEViewTestSuite) TestCountPlatformCVECoreSAC() {
	for _, tc := range s.testCases() {
		for _, sacTC := range s.sacTestCases(tc.ctx) {
			s.T().Run(fmt.Sprintf("SAC desc: %s; test desc: %s ", sacTC.desc, tc.desc), func(t *testing.T) {
				actual, err := s.cveView.Count(sacTC.ctx, tc.q)
				if tc.expectedErr != "" {
					s.ErrorContains(err, tc.expectedErr)
					return
				}
				assert.NoError(t, err)

				// Wrap cluster filter with sac filter.
				filterWithSAC := matchAllFilter().withClusterFilter(func(cluster *storage.Cluster) bool {
					if sacTC.visibleClusters.Contains(cluster.GetId()) {
						return tc.matchFilter.matchCluster(cluster)
					}
					return false
				})
				filterWithSAC.matchCVEParts = tc.matchFilter.matchCVEParts

				expected := s.compileExpectedCVECores(filterWithSAC)
				assert.Equal(t, len(expected), actual)
			})
		}
	}
}

func (s *PlatformCVEViewTestSuite) TestCVECountByType() {
	for _, tc := range s.testCases() {
		s.T().Run(tc.desc, func(t *testing.T) {
			actual, err := s.cveView.CVECountByType(sac.WithAllAccess(tc.ctx), tc.q)
			if tc.expectedErr != "" {
				s.ErrorContains(err, tc.expectedErr)
				return
			}
			assert.NoError(t, err)

			expected := s.compileExpectedCVECountByType(tc.matchFilter)
			assert.EqualValues(t, expected, actual)
		})
	}
}

func (s *PlatformCVEViewTestSuite) TestCVECountByTypeSAC() {
	for _, tc := range s.testCases() {
		for _, sacTC := range s.sacTestCases(tc.ctx) {
			s.T().Run(fmt.Sprintf("SAC desc: %s; test desc: %s ", sacTC.desc, tc.desc), func(t *testing.T) {
				actual, err := s.cveView.CVECountByType(sacTC.ctx, tc.q)
				if tc.expectedErr != "" {
					s.ErrorContains(err, tc.expectedErr)
					return
				}
				assert.NoError(t, err)

				// Wrap cluster filter with sac filter.
				filterWithSAC := matchAllFilter().withClusterFilter(func(cluster *storage.Cluster) bool {
					if sacTC.visibleClusters.Contains(cluster.GetId()) {
						return tc.matchFilter.matchCluster(cluster)
					}
					return false
				})
				filterWithSAC.matchCVEParts = tc.matchFilter.matchCVEParts

				expected := s.compileExpectedCVECountByType(filterWithSAC)
				assert.EqualValues(t, expected, actual)
			})
		}
	}
}

func (s *PlatformCVEViewTestSuite) TestCVECountByFixability() {
	for _, tc := range s.testCases() {
		s.T().Run(tc.desc, func(t *testing.T) {
			actual, err := s.cveView.CVECountByFixability(sac.WithAllAccess(tc.ctx), tc.q)
			if tc.expectedErr != "" {
				s.ErrorContains(err, tc.expectedErr)
				return
			}
			assert.NoError(t, err)

			expected := s.compileExpectedCVECountByFixability(tc.matchFilter)
			assert.EqualValues(t, expected, actual)
		})
	}
}

func (s *PlatformCVEViewTestSuite) TestCVECountByFixabilitySAC() {
	for _, tc := range s.testCases() {
		for _, sacTC := range s.sacTestCases(tc.ctx) {
			s.T().Run(fmt.Sprintf("SAC desc: %s; test desc: %s ", sacTC.desc, tc.desc), func(t *testing.T) {
				actual, err := s.cveView.CVECountByFixability(sacTC.ctx, tc.q)
				if tc.expectedErr != "" {
					s.ErrorContains(err, tc.expectedErr)
					return
				}
				assert.NoError(t, err)

				// Wrap cluster filter with sac filter.
				filterWithSAC := matchAllFilter().withClusterFilter(func(cluster *storage.Cluster) bool {
					if sacTC.visibleClusters.Contains(cluster.GetId()) {
						return tc.matchFilter.matchCluster(cluster)
					}
					return false
				})
				filterWithSAC.matchCVEParts = tc.matchFilter.matchCVEParts

				expected := s.compileExpectedCVECountByFixability(filterWithSAC)
				assert.EqualValues(t, expected, actual)
			})
		}
	}
}

func (s *PlatformCVEViewTestSuite) testCases() []testCase {
	return []testCase{
		{
			desc:        "search all",
			ctx:         context.Background(),
			q:           search.EmptyQuery(),
			matchFilter: matchAllFilter(),
		},
		{
			desc: "search one cve",
			ctx:  context.Background(),
			q:    search.NewQueryBuilder().AddExactMatches(search.CVE, "cve-1").ProtoQuery(),
			matchFilter: matchAllFilter().withCVEPartsFilter(func(cveParts converterV2.ClusterCVEParts) bool {
				return cveParts.CVE.GetCveBaseInfo().GetCve() == "cve-1"
			}),
		},
		{
			desc: "search one cluster",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddExactMatches(search.Cluster, "openshift-1").ProtoQuery(),
			matchFilter: matchAllFilter().withClusterFilter(func(cluster *storage.Cluster) bool {
				return cluster.GetName() == "openshift-1"
			}),
		},
		{
			desc: "search one cve + one cluster",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddExactMatches(search.CVE, "cve-2").
				AddExactMatches(search.Cluster, "openshift-2").
				ProtoQuery(),
			matchFilter: matchAllFilter().
				withClusterFilter(func(cluster *storage.Cluster) bool {
					return cluster.GetName() == "openshift-2"
				}).
				withCVEPartsFilter(func(cveParts converterV2.ClusterCVEParts) bool {
					return cveParts.CVE.GetCveBaseInfo().GetCve() == "cve-2"
				}),
		},
		{
			desc: "search cvss > 7.0",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddStrings(search.CVSS, ">7.0").ProtoQuery(),
			matchFilter: matchAllFilter().withCVEPartsFilter(func(cveParts converterV2.ClusterCVEParts) bool {
				return cveParts.CVE.Cvss > 7.0
			}),
		},
		{
			desc: "search fixable",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddBools(search.ClusterCVEFixable, true).ProtoQuery(),
			matchFilter: matchAllFilter().withCVEPartsFilter(func(cveParts converterV2.ClusterCVEParts) bool {
				for _, child := range cveParts.Children {
					if child.Edge.GetIsFixable() {
						return true
					}
				}
				return false
			}),
		},
		{
			desc: "search fixable + cluster type OCP",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddBools(search.ClusterCVEFixable, true).
				AddExactMatches(search.ClusterType, storage.ClusterMetadata_OCP.String()).
				ProtoQuery(),
			matchFilter: matchAllFilter().
				withClusterFilter(func(cluster *storage.Cluster) bool {
					return cluster.GetStatus().GetProviderMetadata().GetCluster().GetType() == storage.ClusterMetadata_OCP
				}).
				withCVEPartsFilter(func(cveParts converterV2.ClusterCVEParts) bool {
					for _, child := range cveParts.Children {
						if child.Edge.GetIsFixable() {
							return true
						}
					}
					return false
				}),
		},
		{
			desc: "search openshift4 platform type",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddExactMatches(search.ClusterPlatformType, storage.ClusterType_OPENSHIFT4_CLUSTER.String()).ProtoQuery(),
			matchFilter: matchAllFilter().withClusterFilter(func(cluster *storage.Cluster) bool {
				return cluster.GetType() == storage.ClusterType_OPENSHIFT4_CLUSTER
			}),
		},
		{
			desc: "search multiple platform types",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddExactMatches(search.ClusterPlatformType,
					storage.ClusterType_KUBERNETES_CLUSTER.String(),
					storage.ClusterType_OPENSHIFT_CLUSTER.String(),
				).
				ProtoQuery(),
			matchFilter: matchAllFilter().withClusterFilter(func(cluster *storage.Cluster) bool {
				return cluster.GetType() == storage.ClusterType_KUBERNETES_CLUSTER ||
					cluster.GetType() == storage.ClusterType_OPENSHIFT_CLUSTER
			}),
		},
		{
			desc: "search by kubernetes version",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddExactMatches(search.ClusterKubernetesVersion, "9.0").ProtoQuery(),
			matchFilter: matchAllFilter().withClusterFilter(func(cluster *storage.Cluster) bool {
				return cluster.GetStatus().GetOrchestratorMetadata().GetVersion() == "9.0"
			}),
		},
		{
			desc: "search by cluster label",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddMapQuery(search.ClusterLabel, "platform-type", "openshift").
				ProtoQuery(),
			matchFilter: matchAllFilter().withClusterFilter(func(cluster *storage.Cluster) bool {
				return cluster.GetLabels()["platform-type"] == "openshift"
			}),
		},
		{
			desc:        "no match",
			ctx:         context.Background(),
			q:           search.NewQueryBuilder().AddExactMatches(search.CVEType, storage.CVE_IMAGE_CVE.String()).ProtoQuery(),
			matchFilter: matchNoneFilter(),
		},
		{
			desc: "with select",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddSelectFields(search.NewQuerySelect(search.CVE)).
				AddExactMatches(search.CVEType, storage.CVE_K8S_CVE.String()).ProtoQuery(),
			expectedErr: "Unexpected select clause in query",
		},
		{
			desc: "with group by",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddExactMatches(search.CVEType, storage.CVE_K8S_CVE.String()).
				AddGroupBy(search.CVE).ProtoQuery(),
			expectedErr: "Unexpected group by clause in query",
		},
		{
			desc: "search one cve w/ cluster scope",
			ctx: scoped.Context(context.Background(), scoped.Scope{
				ID:    s.clusterNameToIDMap["kubernetes-1"],
				Level: v1.SearchCategory_CLUSTERS,
			}),
			q: search.NewQueryBuilder().
				AddExactMatches(search.CVE, "cve-4").
				ProtoQuery(),
			matchFilter: matchAllFilter().
				withClusterFilter(func(cluster *storage.Cluster) bool {
					return cluster.GetId() == s.clusterNameToIDMap["kubernetes-1"]
				}).
				withCVEPartsFilter(func(cveParts converterV2.ClusterCVEParts) bool {
					return cveParts.CVE.GetCveBaseInfo().GetCve() == "cve-4"
				}),
		},
		{
			desc: "search fixable w/ cve & cluster scope",
			ctx: scoped.Context(context.Background(), scoped.Scope{
				ID:    s.clusterNameToIDMap["openshift4-2"],
				Level: v1.SearchCategory_CLUSTERS,
				Parent: &scoped.Scope{
					ID:    pkgCVE.ID("cve-2", storage.CVE_OPENSHIFT_CVE.String()),
					Level: v1.SearchCategory_CLUSTER_VULNERABILITIES,
				},
			}),
			q: search.NewQueryBuilder().
				AddBools(search.ClusterCVEFixable, true).ProtoQuery(),
			matchFilter: matchAllFilter().
				withClusterFilter(func(cluster *storage.Cluster) bool {
					return cluster.GetId() == s.clusterNameToIDMap["openshift4-2"]
				}).
				withCVEPartsFilter(func(cveParts converterV2.ClusterCVEParts) bool {
					if cveParts.CVE.GetId() != pkgCVE.ID("cve-2", storage.CVE_OPENSHIFT_CVE.String()) {
						return false
					}
					for _, child := range cveParts.Children {
						if child.Edge.GetIsFixable() {
							return true
						}
					}
					return false
				}),
		},
	}
}

func (s *PlatformCVEViewTestSuite) sacTestCases(ctx context.Context) []sacTestCase {
	return []sacTestCase{
		{
			desc: "All clusters visible",
			ctx: sac.WithGlobalAccessScopeChecker(ctx,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster))),
			visibleClusters: set.NewStringSet(
				s.clusterNameToIDMap["generic-1"], s.clusterNameToIDMap["generic-2"],
				s.clusterNameToIDMap["kubernetes-1"], s.clusterNameToIDMap["kubernetes-2"],
				s.clusterNameToIDMap["openshift-1"], s.clusterNameToIDMap["openshift-2"],
				s.clusterNameToIDMap["openshift4-1"], s.clusterNameToIDMap["openshift4-2"],
			),
		},
		{
			desc: "generic-1, kubernetes-1, openshift-1 and openshift4-1 clusters visible",
			ctx: sac.WithGlobalAccessScopeChecker(ctx,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster),
					sac.ClusterScopeKeys(
						s.clusterNameToIDMap["generic-1"], s.clusterNameToIDMap["kubernetes-1"],
						s.clusterNameToIDMap["openshift-1"], s.clusterNameToIDMap["openshift4-1"]))),
			visibleClusters: set.NewStringSet(
				s.clusterNameToIDMap["generic-1"],
				s.clusterNameToIDMap["kubernetes-1"],
				s.clusterNameToIDMap["openshift-1"],
				s.clusterNameToIDMap["openshift4-1"],
			),
		},
		{
			desc: "generic-2, kubernetes-2, openshift-2, openshift4-2 clusters visible",
			ctx: sac.WithGlobalAccessScopeChecker(ctx,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster),
					sac.ClusterScopeKeys(
						s.clusterNameToIDMap["generic-2"], s.clusterNameToIDMap["kubernetes-2"],
						s.clusterNameToIDMap["openshift-2"], s.clusterNameToIDMap["openshift4-2"]))),
			visibleClusters: set.NewStringSet(
				s.clusterNameToIDMap["generic-2"],
				s.clusterNameToIDMap["kubernetes-2"],
				s.clusterNameToIDMap["openshift-2"],
				s.clusterNameToIDMap["openshift4-2"],
			),
		},
		{
			desc: "NamespaceA in openshift4-2 cluster visible",
			ctx: sac.WithGlobalAccessScopeChecker(ctx,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster),
					sac.ClusterScopeKeys(s.clusterNameToIDMap["openshift4-2"]),
					sac.NamespaceScopeKeys(testconsts.NamespaceA))),
			visibleClusters: set.NewStringSet(s.clusterNameToIDMap["openshift4-2"]),
		},
	}
}

func (s *PlatformCVEViewTestSuite) paginationTestCases() []paginationTestCase {
	return []paginationTestCase{
		{
			desc: "Offset: 0, Limit: 6, Order By: CVSS descending",
			q: search.NewQueryBuilder().WithPagination(
				search.NewPagination().Limit(6).AddSortOption(
					search.NewSortOption(search.CVSS).Reversed(true),
				).AddSortOption(search.NewSortOption(search.CVEID)),
			).ProtoQuery(),
			offset: 0,
			limit:  6,
			less: func(records []CveCore) func(i int, j int) bool {
				return func(i, j int) bool {
					if records[i].GetCVSS() == records[j].GetCVSS() {
						return records[i].GetCVEID() < records[j].GetCVEID()
					}
					return records[i].GetCVSS() > records[j].GetCVSS()
				}
			},
		},
		{
			desc: "Offset: 6, Limit: 6, Order By: CVEType",
			q: search.NewQueryBuilder().WithPagination(
				search.NewPagination().Offset(6).Limit(6).AddSortOption(
					search.NewSortOption(search.CVEType),
				).AddSortOption(search.NewSortOption(search.CVEID)),
			).ProtoQuery(),
			offset: 6,
			limit:  6,
			less: func(records []CveCore) func(i int, j int) bool {
				return func(i int, j int) bool {
					if records[i].GetCVEType() == records[j].GetCVEType() {
						return records[i].GetCVEID() < records[j].GetCVEID()
					}
					return records[i].GetCVEType() < records[j].GetCVEType()
				}
			},
		},
		{
			desc: "Order By number of affected clusters",
			q: search.NewQueryBuilder().WithPagination(
				search.NewPagination().AddSortOption(
					search.NewSortOption(search.ClusterID).AggregateBy(aggregatefunc.Count, true).Reversed(true),
				).AddSortOption(search.NewSortOption(search.CVEID)),
			).ProtoQuery(),
			offset: 0,
			limit:  0,
			less: func(records []CveCore) func(i int, j int) bool {
				return func(i int, j int) bool {
					if records[i].GetClusterCount() == records[j].GetClusterCount() {
						return records[i].GetCVEID() < records[j].GetCVEID()
					}
					return records[i].GetClusterCount() > records[j].GetClusterCount()
				}
			},
		},
	}
}

func (s *PlatformCVEViewTestSuite) compileExpectedCVECores(filter *filterImpl) []CveCore {
	var expected []CveCore
	for _, cveParts := range s.cvePartsList {
		if !filter.matchCVEParts(cveParts) {
			continue
		}
		clusterCount := 0
		genericClusterCount := 0
		kubernetesClusterCount := 0
		openshiftClusterCount := 0
		openshift4ClusterCount := 0
		fixableCount := 0

		for _, child := range cveParts.Children {
			cluster, exists := s.clusterMap[child.ClusterID]
			if !exists || !filter.matchCluster(cluster) {
				continue
			}
			clusterCount++
			switch cluster.GetType() {
			case storage.ClusterType_GENERIC_CLUSTER:
				genericClusterCount++
			case storage.ClusterType_KUBERNETES_CLUSTER:
				kubernetesClusterCount++
			case storage.ClusterType_OPENSHIFT_CLUSTER:
				openshiftClusterCount++
			case storage.ClusterType_OPENSHIFT4_CLUSTER:
				openshift4ClusterCount++
			}

			if child.Edge.GetIsFixable() {
				fixableCount++
			}
		}
		if clusterCount == 0 {
			// if no clusters matched, then cve is not included in results
			continue
		}

		cveCreatedTime, err := protocompat.ConvertTimestampToTimeOrError(cveParts.CVE.GetCveBaseInfo().GetCreatedAt())
		s.Require().NoError(err)
		cveCreatedTime = cveCreatedTime.Round(time.Microsecond)
		expected = append(expected, &platformCVECoreResponse{
			CVE:                 cveParts.CVE.GetCveBaseInfo().GetCve(),
			CVEID:               cveParts.CVE.GetId(),
			CVEType:             cveParts.CVE.GetType(),
			CVSS:                cveParts.CVE.GetCvss(),
			ClusterCount:        clusterCount,
			GenericClusters:     genericClusterCount,
			KubernetesClusters:  kubernetesClusterCount,
			OpenshiftClusters:   openshiftClusterCount,
			Openshift4Clusters:  openshift4ClusterCount,
			FirstDiscoveredTime: &cveCreatedTime,
			FixableCount:        fixableCount,
		})
	}

	return expected
}

func (s *PlatformCVEViewTestSuite) compileExpectedCVECoresWithPagination(filter *filterImpl, less lessFunc, offset, limit int) []CveCore {
	expected := s.compileExpectedCVECores(filter)
	if less != nil {
		sort.SliceStable(expected, less(expected))
	}
	if offset >= len(expected) {
		return []CveCore{}
	}
	if limit == 0 {
		return expected
	}
	end := offset + limit
	if end > len(expected) {
		end = len(expected)
	}
	return expected[offset:end]
}

func (s *PlatformCVEViewTestSuite) compileExpectedAffectedClusterIDs(filter *filterImpl) []string {
	affectedClusterIDs := set.NewStringSet()
	for _, cveParts := range s.cvePartsList {
		if !filter.matchCVEParts(cveParts) {
			continue
		}
		for _, child := range cveParts.Children {
			cluster, exists := s.clusterMap[child.ClusterID]
			if !exists || !filter.matchCluster(cluster) {
				continue
			}
			affectedClusterIDs.Add(cluster.GetId())
		}
	}
	return affectedClusterIDs.AsSlice()
}

func (s *PlatformCVEViewTestSuite) compileExpectedCVECountByType(filter *filterImpl) CVECountByType {
	k8sCVECount := 0
	openshiftCVECount := 0
	istioCVECount := 0

	for _, cveParts := range s.cvePartsList {
		if !filter.matchCVEParts(cveParts) {
			continue
		}
		clusterCount := 0

		for _, child := range cveParts.Children {
			cluster, exists := s.clusterMap[child.ClusterID]
			if !exists || !filter.matchCluster(cluster) {
				continue
			}
			clusterCount++
		}
		if clusterCount == 0 {
			// if no clusters matched, then cve is not included in results
			continue
		}

		switch cveParts.CVE.GetType() {
		case storage.CVE_K8S_CVE:
			k8sCVECount++
		case storage.CVE_OPENSHIFT_CVE:
			openshiftCVECount++
		case storage.CVE_ISTIO_CVE:
			istioCVECount++
		}
	}

	return &cveCountByTypeResponse{
		KubernetesCVECount: k8sCVECount,
		OpenshiftCVECount:  openshiftCVECount,
		IstioCVECount:      istioCVECount,
	}
}

func (s *PlatformCVEViewTestSuite) compileExpectedCVECountByFixability(filter *filterImpl) common.ResourceCountByFixability {
	totalCVECount := 0
	fixableCVECount := 0

	for _, cveParts := range s.cvePartsList {
		if !filter.matchCVEParts(cveParts) {
			continue
		}
		clusterCount := 0
		fixable := false

		for _, child := range cveParts.Children {
			cluster, exists := s.clusterMap[child.ClusterID]
			if !exists || !filter.matchCluster(cluster) {
				continue
			}
			clusterCount++
			if child.Edge.GetIsFixable() {
				fixable = true
			}
		}
		if clusterCount == 0 {
			// if no clusters matched, then cve is not included in results
			continue
		}

		totalCVECount++
		if fixable {
			fixableCVECount++
		}
	}

	return &cveCountByFixabilityResponse{
		CVECount:     totalCVECount,
		FixableCount: fixableCVECount,
	}
}

func applyPaginationProps(baseTc *testCase, paginationTc paginationTestCase) {
	baseTc.desc = fmt.Sprintf("%s %s", baseTc.desc, paginationTc.desc)
	baseTc.q.Pagination = paginationTc.q.GetPagination()
}

func getTestData() (map[string]*storage.Cluster, map[storage.CVE_CVEType][]converter.ClusterCVEParts) {
	// Clusters
	clusterMap := make(map[string]*storage.Cluster)
	generic1 := generateTestCluster(&testClusterFields{
		Name:         "generic-1",
		PlatformType: storage.ClusterType_GENERIC_CLUSTER,
		ProviderType: storage.ClusterMetadata_AKS,
		Labels:       map[string]string{},
		K8sVersion:   "9.0",
		IsOpenshift:  false,
	})
	clusterMap[generic1.GetId()] = generic1

	generic2 := generateTestCluster(&testClusterFields{
		Name:         "generic-2",
		PlatformType: storage.ClusterType_GENERIC_CLUSTER,
		ProviderType: storage.ClusterMetadata_ARO,
		Labels:       map[string]string{},
		K8sVersion:   "9.0",
		IsOpenshift:  false,
	})
	clusterMap[generic2.GetId()] = generic2

	kubernetes1 := generateTestCluster(&testClusterFields{
		Name:         "kubernetes-1",
		PlatformType: storage.ClusterType_KUBERNETES_CLUSTER,
		ProviderType: storage.ClusterMetadata_EKS,
		Labels:       map[string]string{},
		K8sVersion:   "9.0",
		IsOpenshift:  false,
	})
	clusterMap[kubernetes1.GetId()] = kubernetes1

	kubernetes2 := generateTestCluster(&testClusterFields{
		Name:         "kubernetes-2",
		PlatformType: storage.ClusterType_KUBERNETES_CLUSTER,
		ProviderType: storage.ClusterMetadata_GKE,
		Labels:       map[string]string{},
		K8sVersion:   "9.0",
		IsOpenshift:  false,
	})
	clusterMap[kubernetes2.GetId()] = kubernetes2

	openshift1 := generateTestCluster(&testClusterFields{
		Name:         "openshift-1",
		PlatformType: storage.ClusterType_OPENSHIFT_CLUSTER,
		ProviderType: storage.ClusterMetadata_OCP,
		Labels:       map[string]string{"platform-type": "openshift"},
		K8sVersion:   "8.0",
		IsOpenshift:  true,
	})
	clusterMap[openshift1.GetId()] = openshift1

	openshift2 := generateTestCluster(&testClusterFields{
		Name:         "openshift-2",
		PlatformType: storage.ClusterType_OPENSHIFT_CLUSTER,
		ProviderType: storage.ClusterMetadata_OSD,
		Labels:       map[string]string{"platform-type": "openshift"},
		K8sVersion:   "8.0",
		IsOpenshift:  true,
	})
	clusterMap[openshift2.GetId()] = openshift2

	openshift41 := generateTestCluster(&testClusterFields{
		Name:         "openshift4-1",
		PlatformType: storage.ClusterType_OPENSHIFT4_CLUSTER,
		ProviderType: storage.ClusterMetadata_OCP,
		Labels:       map[string]string{"platform-type": "openshift"},
		K8sVersion:   "8.0",
		IsOpenshift:  true,
	})
	clusterMap[openshift41.GetId()] = openshift41

	openshift42 := generateTestCluster(&testClusterFields{
		Name:         "openshift4-2",
		PlatformType: storage.ClusterType_OPENSHIFT4_CLUSTER,
		ProviderType: storage.ClusterMetadata_OCP,
		Labels:       map[string]string{"platform-type": "openshift"},
		K8sVersion:   "8.0",
		IsOpenshift:  true,
	})
	clusterMap[openshift42.GetId()] = openshift42

	// CVEs and CVEParts
	cve1K8 := generateTestCVE("cve-1", storage.CVE_K8S_CVE, 9.8)
	cve2K8 := generateTestCVE("cve-2", storage.CVE_K8S_CVE, 8.5)
	cve3Openshift := generateTestCVE("cve-3", storage.CVE_OPENSHIFT_CVE, 7.3)
	cve4K8 := generateTestCVE("cve-4", storage.CVE_K8S_CVE, 3.4)
	cve5K8 := generateTestCVE("cve-5", storage.CVE_K8S_CVE, 9.5)
	cve1Openshift := generateTestCVE("cve-1", storage.CVE_OPENSHIFT_CVE, 8.7)
	cve2Openshift := generateTestCVE("cve-2", storage.CVE_OPENSHIFT_CVE, 6.3)
	cve4Openshift := generateTestCVE("cve-4", storage.CVE_OPENSHIFT_CVE, 4.9)
	cve5Openshift := generateTestCVE("cve-5", storage.CVE_OPENSHIFT_CVE, 7.0)
	cve1Istio := generateTestCVE("cve-1", storage.CVE_ISTIO_CVE, 7.2)
	cve5Istio := generateTestCVE("cve-5", storage.CVE_ISTIO_CVE, 4.8)

	// CVEParts
	cvePartsByType := make(map[storage.CVE_CVEType][]converter.ClusterCVEParts)
	for _, cveParts := range []converter.ClusterCVEParts{
		converterV2.NewClusterCVEParts(cve1K8, []*storage.Cluster{generic1}, "9.2"),
		converterV2.NewClusterCVEParts(cve2K8, []*storage.Cluster{generic1, generic2}, "9.3"),
		converterV2.NewClusterCVEParts(cve3Openshift, []*storage.Cluster{generic2}, ""),
		converterV2.NewClusterCVEParts(cve4K8, []*storage.Cluster{kubernetes1, kubernetes2}, "9.3"),
		converterV2.NewClusterCVEParts(cve5K8, []*storage.Cluster{kubernetes1, kubernetes2}, "9.2"),
		converterV2.NewClusterCVEParts(cve1Openshift, []*storage.Cluster{openshift1, openshift41, openshift42}, ""),
		converterV2.NewClusterCVEParts(cve2Openshift, []*storage.Cluster{openshift1, openshift2, openshift42}, "4.15"),
		converterV2.NewClusterCVEParts(cve4Openshift, []*storage.Cluster{openshift2, openshift42}, "4.13"),
		converterV2.NewClusterCVEParts(cve5Openshift, []*storage.Cluster{openshift41, openshift42}, "4.15"),
		converterV2.NewClusterCVEParts(cve1Istio, []*storage.Cluster{generic1}, ""),
		converterV2.NewClusterCVEParts(cve5Istio, []*storage.Cluster{openshift41}, "4.15"),
	} {
		cvePartsByType[cveParts.CVE.GetType()] = append(cvePartsByType[cveParts.CVE.GetType()], cveParts)
	}

	return clusterMap, cvePartsByType
}

type testClusterFields struct {
	Name         string
	PlatformType storage.ClusterType
	ProviderType storage.ClusterMetadata_Type
	Labels       map[string]string
	K8sVersion   string
	IsOpenshift  bool
}

func generateTestCluster(tcf *testClusterFields) *storage.Cluster {
	return &storage.Cluster{
		Id:        uuid.NewV4().String(),
		Name:      tcf.Name,
		Type:      tcf.PlatformType,
		Labels:    tcf.Labels,
		MainImage: "quay.io/stackrox-io/main",
		Status: &storage.ClusterStatus{
			OrchestratorMetadata: &storage.OrchestratorMetadata{
				Version: tcf.K8sVersion,
			},
			ProviderMetadata: &storage.ProviderMetadata{
				Cluster: &storage.ClusterMetadata{
					Type: tcf.ProviderType,
				},
			},
		},
	}
}

func generateTestCVE(cve string, cveType storage.CVE_CVEType, cvss float32) *storage.ClusterCVE {
	return &storage.ClusterCVE{
		Id: pkgCVE.ID(cve, cveType.String()),
		CveBaseInfo: &storage.CVEInfo{
			Cve: cve,
		},
		Type: cveType,
		Cvss: cvss,
	}
}
