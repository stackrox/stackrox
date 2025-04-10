//go:build sql_integration

package nodecve

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	nodeCVEDataStore "github.com/stackrox/rox/central/cve/node/datastore"
	nodeDS "github.com/stackrox/rox/central/node/datastore"
	"github.com/stackrox/rox/central/views/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pkgCVE "github.com/stackrox/rox/pkg/cve"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	nodeConverter "github.com/stackrox/rox/pkg/nodes/converter"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/set"
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
	desc         string
	ctx          context.Context
	visibleNodes set.StringSet
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
	matchNode    func(node *storage.Node) bool
	matchCluster func(cluster *storage.Cluster) bool
	matchVuln    func(vuln *storage.NodeVulnerability) bool
}

func matchAllFilter() *filterImpl {
	return &filterImpl{
		matchCluster: func(_ *storage.Cluster) bool {
			return true
		},
		matchNode: func(_ *storage.Node) bool {
			return true
		},
		matchVuln: func(_ *storage.NodeVulnerability) bool {
			return true
		},
	}
}

func matchNoneFilter() *filterImpl {
	return &filterImpl{
		matchCluster: func(_ *storage.Cluster) bool {
			return false
		},
		matchNode: func(_ *storage.Node) bool {
			return false
		},
		matchVuln: func(_ *storage.NodeVulnerability) bool {
			return false
		},
	}
}

func (f *filterImpl) withNodeFilter(fn func(node *storage.Node) bool) *filterImpl {
	f.matchNode = fn
	return f
}

func (f *filterImpl) withClusterFilter(fn func(cluster *storage.Cluster) bool) *filterImpl {
	f.matchCluster = fn
	return f
}

func (f *filterImpl) withVulnFilter(fn func(vuln *storage.NodeVulnerability) bool) *filterImpl {
	f.matchVuln = fn
	return f
}

func TestNodeCVEView(t *testing.T) {
	suite.Run(t, new(NodeCVEViewTestSuite))
}

type NodeCVEViewTestSuite struct {
	suite.Suite

	ctx    context.Context
	testDB *pgtest.TestPostgres

	nameToClusters map[string]*storage.Cluster

	nodeMap      map[string]*storage.Node
	cveView      CveView
	cveCreateMap map[string]*storage.NodeCVE
}

func (s *NodeCVEViewTestSuite) createTestClusters() {
	clusterDatastore, err := clusterDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.Require().NoError(err)
	cluster1 := fixtures.GetCluster(fixtureconsts.ClusterName1)
	cluster1.Id = fixtureconsts.Cluster1
	cluster2 := fixtures.GetCluster(fixtureconsts.ClusterName2)
	cluster2.Id = fixtureconsts.Cluster2
	s.Require().NoError(clusterDatastore.UpdateCluster(s.ctx, cluster1))
	s.Require().NoError(clusterDatastore.UpdateCluster(s.ctx, cluster2))
	s.nameToClusters = map[string]*storage.Cluster{cluster1.GetName(): cluster1, cluster2.GetName(): cluster2}
}

func (s *NodeCVEViewTestSuite) SetupSuite() {
	s.T().Setenv(env.OrphanedCVEsKeepAlive.EnvVar(), "true")
	if !env.OrphanedCVEsKeepAlive.BooleanSetting() {
		s.T().Skip("Skip tests when ROX_ORPHANED_CVES_KEEP_ALIVE disabled")
		s.T().SkipNow()
	}

	s.ctx = sac.WithAllAccess(context.Background())
	s.testDB = pgtest.ForT(s.T())

	// Initialize the datastore.
	s.createTestClusters()

	nodeDatastore := nodeDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.nodeMap = getTestNodes()
	for _, node := range s.nodeMap {
		s.Require().NoError(nodeDatastore.UpsertNode(s.ctx, node))
	}

	cveStore, err := nodeCVEDataStore.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.Require().NoError(err)
	storedCves, err := cveStore.SearchRawCVEs(s.ctx, nil)
	s.Require().NoError(err)
	s.cveCreateMap = make(map[string]*storage.NodeCVE)
	for _, c := range storedCves {
		s.cveCreateMap[c.GetId()] = c
	}
	s.cveView = NewCVEView(s.testDB.DB)
}

func (s *NodeCVEViewTestSuite) TestGetNodeCVECore() {
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
		})
	}
}

func (s *NodeCVEViewTestSuite) TestGetNodeCVECoreSAC() {
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
				matchFilter := *tc.matchFilter
				baseNodeMatchFilter := matchFilter.matchNode
				matchFilter.withNodeFilter(func(node *storage.Node) bool {
					if sacTC.visibleNodes.Contains(node.GetId()) {
						return baseNodeMatchFilter(node)
					}
					return false
				})

				expected := s.compileExpectedCVECores(&matchFilter)
				assert.Equal(t, len(expected), len(actual))
				assert.ElementsMatch(t, expected, actual)
			})
		}
	}
}

func (s *NodeCVEViewTestSuite) TestGetNodeCVECoreWithPagination() {
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
					// The total cve count should be equal to aggregation of the all severity cve counts.
					assert.Equal(t,
						record.GetNodeCount(),
						record.GetNodeCountBySeverity().GetCriticalSeverityCount().GetTotal()+
							record.GetNodeCountBySeverity().GetImportantSeverityCount().GetTotal()+
							record.GetNodeCountBySeverity().GetModerateSeverityCount().GetTotal()+
							record.GetNodeCountBySeverity().GetLowSeverityCount().GetTotal(),
					)
				}
			})
		}
	}
}

func (s *NodeCVEViewTestSuite) TestCountNodeCVECore() {
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

func (s *NodeCVEViewTestSuite) TestCountNodeCVECoreSAC() {
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
				matchFilter := *tc.matchFilter
				baseClusterMatchFilter := matchFilter.matchNode
				matchFilter.withNodeFilter(func(node *storage.Node) bool {
					if sacTC.visibleNodes.Contains(node.GetId()) {
						return baseClusterMatchFilter(node)
					}
					return false
				})

				expected := s.compileExpectedCVECores(&matchFilter)
				assert.Equal(t, len(expected), actual)
			})
		}
	}
}

func (s *NodeCVEViewTestSuite) TestGetNodeIDs() {
	for _, tc := range s.testCases() {
		s.T().Run(tc.desc, func(t *testing.T) {
			// Such testcases are meant only for Get().
			if tc.expectedErr != "" {
				return
			}

			actualAffectedNodeIDs, err := s.cveView.GetNodeIDs(sac.WithAllAccess(tc.ctx), tc.q)
			assert.NoError(t, err)
			expectedAffectedNodeIDs := s.compileExpectedAffectedNodeIDs(tc.matchFilter)
			assert.ElementsMatch(t, expectedAffectedNodeIDs, actualAffectedNodeIDs)
		})
	}
}

func (s *NodeCVEViewTestSuite) TestGetNodeIDsSAC() {
	for _, tc := range s.testCases() {
		for _, sacTC := range s.sacTestCases(tc.ctx) {
			s.T().Run(fmt.Sprintf("SAC desc: %s; test desc: %s ", sacTC.desc, tc.desc), func(t *testing.T) {
				// Such testcases are meant only for Get().
				if tc.expectedErr != "" {
					return
				}
				actualAffectedNodeIDs, err := s.cveView.GetNodeIDs(sacTC.ctx, tc.q)
				assert.NoError(t, err)

				// Wrap cluster filter with sac filter.
				filterWithSAC := matchAllFilter().
					withNodeFilter(func(node *storage.Node) bool {
						if sacTC.visibleNodes.Contains(node.GetId()) {
							return tc.matchFilter.matchNode(node)
						}
						return false
					}).
					withVulnFilter(tc.matchFilter.matchVuln).
					withClusterFilter(tc.matchFilter.matchCluster)

				expectedAffectedClusterIDs := s.compileExpectedAffectedNodeIDs(filterWithSAC)
				assert.ElementsMatch(t, expectedAffectedClusterIDs, actualAffectedNodeIDs)
			})
		}
	}
}

func (s *NodeCVEViewTestSuite) TestCountBySeverity() {
	for _, tc := range s.testCases() {
		s.T().Run(tc.desc, func(t *testing.T) {
			// Such testcases are meant only for Get().
			if tc.expectedErr != "" {
				return
			}

			actualCountBySeverity, err := s.cveView.CountBySeverity(sac.WithAllAccess(tc.ctx), tc.q)
			assert.NoError(t, err)
			expectedCountBySeverity := s.compileExpectedCountBySeverity(tc.matchFilter)
			assert.Equal(t, expectedCountBySeverity, actualCountBySeverity)
		})
	}
}

func (s *NodeCVEViewTestSuite) TestCountBySeveritySAC() {
	for _, tc := range s.testCases() {
		for _, sacTC := range s.sacTestCases(tc.ctx) {
			s.T().Run(fmt.Sprintf("SAC desc: %s; test desc: %s ", sacTC.desc, tc.desc), func(t *testing.T) {
				// Such testcases are meant only for Get().
				if tc.expectedErr != "" {
					return
				}
				actualCountBySeverity, err := s.cveView.CountBySeverity(sacTC.ctx, tc.q)
				assert.NoError(t, err)

				// Wrap cluster filter with sac filter.
				filterWithSAC := matchAllFilter().
					withNodeFilter(func(node *storage.Node) bool {
						if sacTC.visibleNodes.Contains(node.GetId()) {
							return tc.matchFilter.matchNode(node)
						}
						return false
					}).
					withVulnFilter(tc.matchFilter.matchVuln).
					withClusterFilter(tc.matchFilter.matchCluster)
				expectedCountBySeverity := s.compileExpectedCountBySeverity(filterWithSAC)
				assert.Equal(t, expectedCountBySeverity, actualCountBySeverity)

			})
		}
	}
}

func (s *NodeCVEViewTestSuite) testCases() []testCase {
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
			q:    search.NewQueryBuilder().AddExactMatches(search.CVE, "CVE-2014-6200").ProtoQuery(),
			matchFilter: matchAllFilter().withVulnFilter(func(vuln *storage.NodeVulnerability) bool {
				return vuln.GetCveBaseInfo().GetCve() == "CVE-2014-6200"
			}),
		},
		{
			desc: "search one cluster",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddExactMatches(search.Cluster, fixtureconsts.ClusterName2).ProtoQuery(),
			matchFilter: matchAllFilter().withClusterFilter(func(cluster *storage.Cluster) bool {
				return cluster.GetName() == fixtureconsts.ClusterName2
			}),
		},
		{
			desc: "search one node",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddExactMatches(search.Node, "Node-1").
				ProtoQuery(),
			matchFilter: matchAllFilter().
				withNodeFilter(func(node *storage.Node) bool {
					return node.GetName() == "Node-1"
				}),
		},
		{
			desc: "search one cve + one node",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddExactMatches(search.CVE, "CVE-2014-6200").
				AddExactMatches(search.Node, "Node-1").
				ProtoQuery(),
			matchFilter: matchAllFilter().
				withNodeFilter(func(node *storage.Node) bool {
					return node.GetName() == "Node-1"
				}).
				withVulnFilter(func(vuln *storage.NodeVulnerability) bool {
					return vuln.GetCveBaseInfo().GetCve() == "CVE-2014-6200"
				}),
		},
		{
			desc: "search cvss > 7.0",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddStrings(search.CVSS, ">7.0").ProtoQuery(),
			matchFilter: matchAllFilter().
				withVulnFilter(func(vuln *storage.NodeVulnerability) bool {
					return vuln.GetCvss() > 7.0
				}),
		},
		{
			desc: "search one operating system",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddExactMatches(search.OperatingSystem, "os1").
				ProtoQuery(),
			matchFilter: matchAllFilter().
				withNodeFilter(func(node *storage.Node) bool {
					return node.GetOperatingSystem() == "os1"
				}),
		},
		{
			desc: "search fixable",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddBools(search.Fixable, true).
				ProtoQuery(),
			matchFilter: matchAllFilter().
				withVulnFilter(func(vuln *storage.NodeVulnerability) bool {
					return vuln.GetFixedBy() != ""
				}),
		},
		{
			desc: "search one cve + fixable",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddExactMatches(search.CVE, "CVE-2014-6230").
				AddBools(search.Fixable, true).
				ProtoQuery(),
			matchFilter: matchAllFilter().
				withVulnFilter(func(vuln *storage.NodeVulnerability) bool {
					return vuln.GetCveBaseInfo().GetCve() == "CVE-2014-6230" && vuln.GetFixedBy() != ""
				}),
		},
		{
			desc: "search one cve + not fixable",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddExactMatches(search.CVE, "CVE-2014-6230").
				AddBools(search.Fixable, false).
				ProtoQuery(),
			matchFilter: matchAllFilter().
				withVulnFilter(func(vuln *storage.NodeVulnerability) bool {
					return vuln.GetCveBaseInfo().GetCve() == "CVE-2014-6230" && vuln.GetFixedBy() == ""
				}),
		},
		{
			desc: "search critical severity",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddExactMatches(search.Severity, storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY.String()).ProtoQuery(),
			matchFilter: matchAllFilter().
				withVulnFilter(func(vuln *storage.NodeVulnerability) bool {
					return vuln.GetSeverity() == storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY
				}),
		},
		{
			desc: "search low severity",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddExactMatches(search.Severity, storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY.String()).ProtoQuery(),
			matchFilter: matchAllFilter().
				withVulnFilter(func(vuln *storage.NodeVulnerability) bool {
					return vuln.GetSeverity() == storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY
				}),
		},
		{
			desc: "search multiple severities",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddExactMatches(search.Severity,
					storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY.String(),
					storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY.String(),
				).ProtoQuery(),
			matchFilter: matchAllFilter().
				withVulnFilter(func(vuln *storage.NodeVulnerability) bool {
					return vuln.GetSeverity() == storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY ||
						vuln.GetSeverity() == storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY
				}),
		},
		{
			desc: "search by node label",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddMapQuery(search.NodeLabel, "searchLabel", "something").
				ProtoQuery(),
			matchFilter: matchAllFilter().withNodeFilter(func(node *storage.Node) bool {
				return node.GetLabels()["searchLabel"] == "something"
			}),
		},
		{
			desc:        "no match",
			ctx:         context.Background(),
			q:           search.NewQueryBuilder().AddExactMatches(search.OperatingSystem, "os3").ProtoQuery(),
			matchFilter: matchNoneFilter(),
		},
		{
			desc: "with select",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddSelectFields(search.NewQuerySelect(search.CVE)).
				AddExactMatches(search.OperatingSystem, "os1").
				ProtoQuery(),
			expectedErr: "Unexpected select clause in query",
		},
		{
			desc: "with group by",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddExactMatches(search.OperatingSystem, "os1").
				AddGroupBy(search.CVE).ProtoQuery(),
			expectedErr: "Unexpected group by clause in query",
		},
		{
			desc: "search one cve w/ cluster scope",
			ctx: scoped.Context(context.Background(), scoped.Scope{
				IDs:   []string{s.nodeMap[fixtureconsts.Node1].ClusterId},
				Level: v1.SearchCategory_CLUSTERS,
			}),
			q: search.NewQueryBuilder().
				AddExactMatches(search.CVE, "CVE-2014-6210").
				ProtoQuery(),
			matchFilter: matchAllFilter().
				withClusterFilter(func(cluster *storage.Cluster) bool {
					return cluster.GetName() == s.nodeMap[fixtureconsts.Node1].ClusterName
				}).
				withVulnFilter(func(vuln *storage.NodeVulnerability) bool {
					return vuln.GetCveBaseInfo().GetCve() == "CVE-2014-6210"
				}),
		},
		{
			desc: "search one cve w/ node scope",
			ctx: scoped.Context(context.Background(), scoped.Scope{
				IDs:   []string{s.nodeMap[fixtureconsts.Node2].Id},
				Level: v1.SearchCategory_NODES,
			}),
			q: search.NewQueryBuilder().
				AddExactMatches(search.CVE, "CVE-2014-6210").
				ProtoQuery(),
			matchFilter: matchAllFilter().
				withNodeFilter(func(node *storage.Node) bool {
					return node.GetName() == s.nodeMap[fixtureconsts.Node2].Name
				}).
				withVulnFilter(func(vuln *storage.NodeVulnerability) bool {
					return vuln.GetCveBaseInfo().GetCve() == "CVE-2014-6210"
				}),
		},
	}
}

func (s *NodeCVEViewTestSuite) sacTestCases(ctx context.Context) []sacTestCase {
	return []sacTestCase{
		{
			desc: "All nodeMap visible",
			ctx: sac.WithGlobalAccessScopeChecker(ctx,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resources.Node))),
			visibleNodes: set.NewStringSet(
				fixtureconsts.Node1, fixtureconsts.Node2, fixtureconsts.Node3,
			),
		},
		{
			desc: "Nodes in cluster 1 visible",
			ctx: sac.WithGlobalAccessScopeChecker(ctx,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resources.Node),
					sac.ClusterScopeKeys(s.nameToClusters[fixtureconsts.ClusterName1].GetId())),
			),
			visibleNodes: set.NewStringSet(
				fixtureconsts.Node1, fixtureconsts.Node2,
			),
		},
		{
			desc: "Nodes in cluster 2 visible",
			ctx: sac.WithGlobalAccessScopeChecker(ctx,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resources.Node),
					sac.ClusterScopeKeys(s.nameToClusters[fixtureconsts.ClusterName2].GetId())),
			),
			visibleNodes: set.NewStringSet(
				fixtureconsts.Node3,
			),
		},
		{
			desc: "Nodes in all clusters visible",
			ctx: sac.WithGlobalAccessScopeChecker(ctx,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resources.Node),
					sac.ClusterScopeKeys(
						s.nameToClusters[fixtureconsts.ClusterName1].GetId(),
						s.nameToClusters[fixtureconsts.ClusterName2].GetId()),
				),
			),
			visibleNodes: set.NewStringSet(
				fixtureconsts.Node1, fixtureconsts.Node2, fixtureconsts.Node3,
			),
		},
		{
			desc: "No node visible",
			ctx: sac.WithGlobalAccessScopeChecker(ctx,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resources.Node),
					sac.ClusterScopeKeys(fixtureconsts.Cluster3)),
			),
			visibleNodes: set.NewStringSet(),
		},
		{
			desc: "Namespace scope has no impact",
			ctx: sac.WithGlobalAccessScopeChecker(ctx,
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resources.Node),
					sac.ClusterScopeKeys(s.nameToClusters[fixtureconsts.ClusterName2].GetId()),
					sac.NamespaceScopeKeys(fixtureconsts.Namespace1)),
			),
			visibleNodes: set.NewStringSet(
				fixtureconsts.Node3,
			),
		},
	}
}

func (s *NodeCVEViewTestSuite) paginationTestCases() []paginationTestCase {
	return []paginationTestCase{
		{
			desc: "Offset: 0, Limit: 6, Order By: Top CVSS descending",
			q: search.NewQueryBuilder().WithPagination(
				search.NewPagination().
					Limit(6).
					AddSortOption(search.NewSortOption(search.CVSS).
						AggregateBy(aggregatefunc.Max, false).
						Reversed(true)).
					AddSortOption(search.NewSortOption(search.CVE)),
			).ProtoQuery(),
			offset: 0,
			limit:  6,
			less: func(records []CveCore) func(i int, j int) bool {
				return func(i, j int) bool {
					if records[i].GetTopCVSS() == records[j].GetTopCVSS() {
						return records[i].GetCVE() < records[j].GetCVE()
					}
					return records[i].GetTopCVSS() > records[j].GetTopCVSS()
				}
			},
		},
		{
			desc: "Offset: 6, Limit: 6, Order By: CVE",
			q: search.NewQueryBuilder().WithPagination(
				search.NewPagination().Offset(6).Limit(6).AddSortOption(
					search.NewSortOption(search.CVE),
				),
			).ProtoQuery(),
			offset: 6,
			limit:  6,
			less: func(records []CveCore) func(i int, j int) bool {
				return func(i int, j int) bool {
					return records[i].GetCVE() < records[j].GetCVE()
				}
			},
		},
		{
			desc: "Order By the number of affected nodeMap",
			q: search.NewQueryBuilder().WithPagination(
				search.NewPagination().AddSortOption(
					search.NewSortOption(search.NodeID).AggregateBy(aggregatefunc.Count, true).Reversed(true),
				).AddSortOption(search.NewSortOption(search.CVE)),
			).ProtoQuery(),
			offset: 0,
			limit:  0,
			less: func(records []CveCore) func(i int, j int) bool {
				return func(i int, j int) bool {
					if records[i].GetNodeCount() == records[j].GetNodeCount() {
						return records[i].GetCVE() < records[j].GetCVE()
					}
					return records[i].GetNodeCount() > records[j].GetNodeCount()
				}
			},
		},
		{
			desc: "sort by first discovered time",
			q: search.NewQueryBuilder().WithPagination(
				search.NewPagination().AddSortOption(
					search.NewSortOption(search.CVECreatedTime).AggregateBy(aggregatefunc.Min, false),
				).AddSortOption(search.NewSortOption(search.CVE)),
			).ProtoQuery(),
			offset: 0,
			limit:  0,
			less: func(records []CveCore) func(i, j int) bool {
				return func(i, j int) bool {
					recordI, recordJ := records[i], records[j]
					if recordJ == nil {
						recordJ = &nodeCVECoreResponse{}
					}
					if recordI.GetFirstDiscoveredInSystem().Equal(*recordJ.GetFirstDiscoveredInSystem()) {
						return records[i].GetCVE() < records[j].GetCVE()
					}
					return recordI.GetFirstDiscoveredInSystem().Before(*recordJ.GetFirstDiscoveredInSystem())
				}
			},
		},
	}
}

type coreWithStats struct {
	response *nodeCVECoreResponse

	operatingSystems set.StringSet

	severityToNodes map[storage.VulnerabilitySeverity]set.StringSet

	severityToFixableNodes map[storage.VulnerabilitySeverity]set.StringSet
}

func (s *NodeCVEViewTestSuite) compileExpectedCVECores(filter *filterImpl) []CveCore {
	cveMap := make(map[string]*coreWithStats)
	for _, n := range s.nodeMap {
		if !filter.matchNode(n) {
			continue
		}
		if !filter.matchCluster(s.nameToClusters[n.ClusterName]) {
			continue
		}
		for _, c := range n.GetScan().GetComponents() {
			for _, v := range c.GetVulnerabilities() {
				if !filter.matchVuln(v) {
					continue
				}
				cve := v.GetCveBaseInfo().GetCve()
				id := pkgCVE.ID(cve, n.GetScan().GetOperatingSystem())
				if _, ok := s.cveCreateMap[id]; !ok {
					s.Require().Contains(s.cveCreateMap, id)
				}
				cveCreatedTime, err := protocompat.ConvertTimestampToTimeOrError(s.cveCreateMap[id].GetCveBaseInfo().CreatedAt)
				s.Require().NoError(err)
				cveCreatedTime = cveCreatedTime.Round(time.Microsecond)
				withStats, ok := cveMap[cve]
				if !ok {
					withStats = &coreWithStats{
						response: &nodeCVECoreResponse{
							CVE:                     cve,
							FirstDiscoveredInSystem: &cveCreatedTime,
						},
						severityToNodes:        make(map[storage.VulnerabilitySeverity]set.StringSet),
						severityToFixableNodes: make(map[storage.VulnerabilitySeverity]set.StringSet),
					}
					cveMap[v.GetCveBaseInfo().GetCve()] = withStats
				}
				core := withStats.response
				core.CVEIDs = append(core.CVEIDs, id)
				if core.GetTopCVSS() < v.Cvss {
					core.TopCVSS = v.Cvss
				}
				core.NodeIDs = append(core.NodeIDs, n.GetId())
				if core.GetFirstDiscoveredInSystem().Compare(cveCreatedTime) > 0 {
					core.FirstDiscoveredInSystem = &cveCreatedTime
				}
				withStats.operatingSystems.Add(n.GetOperatingSystem())
				nSet, ok := withStats.severityToNodes[v.GetSeverity()]
				nSet.Add(n.Id)
				if !ok {
					withStats.severityToNodes[v.GetSeverity()] = nSet
				}
				if v.GetFixedBy() != "" {
					fixableNSet, ok := withStats.severityToFixableNodes[v.GetSeverity()]
					fixableNSet.Add(n.Id)
					if !ok {
						withStats.severityToFixableNodes[v.GetSeverity()] = fixableNSet
					}
				}
			}
		}
	}
	var expected []CveCore
	for _, withStats := range cveMap {
		core := withStats.response
		core.CVEIDs = set.NewStringSet(core.CVEIDs...).AsSlice()
		sort.SliceStable(core.CVEIDs, func(i, j int) bool {
			return core.CVEIDs[i] < core.CVEIDs[j]
		})
		core.NodeIDs = set.NewStringSet(core.NodeIDs...).AsSlice()
		sort.SliceStable(core.NodeIDs, func(i, j int) bool {
			return core.NodeIDs[i] < core.NodeIDs[j]
		})
		core.NodeCount = len(core.GetNodeIDs())
		core.OperatingSystemCount = withStats.operatingSystems.Cardinality()
		core.NodesWithLowSeverity = withStats.severityToNodes[storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY].Cardinality()
		core.FixableNodesWithLowSeverity = withStats.severityToFixableNodes[storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY].Cardinality()

		core.NodesWithModerateSeverity = withStats.severityToNodes[storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY].Cardinality()
		core.FixableNodesWithModerateSeverity = withStats.severityToFixableNodes[storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY].Cardinality()

		core.NodesWithImportantSeverity = withStats.severityToNodes[storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY].Cardinality()
		core.FixableNodesWithImportantSeverity = withStats.severityToFixableNodes[storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY].Cardinality()

		core.NodesWithCriticalSeverity = withStats.severityToNodes[storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY].Cardinality()
		core.FixableNodesWithCriticalSeverity = withStats.severityToNodes[storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY].Cardinality()
		expected = append(expected, core)
	}
	return expected
}

func (s *NodeCVEViewTestSuite) compileExpectedCountBySeverity(filter *filterImpl) common.ResourceCountByCVESeverity {
	var expected countByNodeCVESeverity
	severityToCVEs := make(map[storage.VulnerabilitySeverity]set.StringSet)
	severityToFixableCVEs := make(map[storage.VulnerabilitySeverity]set.StringSet)
	for _, n := range s.nodeMap {
		if !filter.matchNode(n) {
			continue
		}
		if !filter.matchCluster(s.nameToClusters[n.ClusterName]) {
			continue
		}
		for _, c := range n.GetScan().GetComponents() {
			for _, v := range c.GetVulnerabilities() {
				if !filter.matchVuln(v) {
					continue
				}

				cve := v.GetCveBaseInfo().GetCve()
				cveSet, ok := severityToCVEs[v.GetSeverity()]
				cveSet.Add(cve)
				if !ok {
					severityToCVEs[v.GetSeverity()] = cveSet
				}
				if v.GetFixedBy() != "" {
					fixableCVESet, ok := severityToFixableCVEs[v.GetSeverity()]
					fixableCVESet.Add(cve)
					if !ok {
						severityToFixableCVEs[v.GetSeverity()] = fixableCVESet
					}
				}
			}
		}
	}

	expected.LowSeverityCount = severityToCVEs[storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY].Cardinality()
	expected.FixableLowSeverityCount = severityToFixableCVEs[storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY].Cardinality()

	expected.ModerateSeverityCount = severityToCVEs[storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY].Cardinality()
	expected.FixableModerateSeverityCount = severityToFixableCVEs[storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY].Cardinality()

	expected.ImportantSeverityCount = severityToCVEs[storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY].Cardinality()
	expected.FixableImportantSeverityCount = severityToFixableCVEs[storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY].Cardinality()

	expected.CriticalSeverityCount = severityToCVEs[storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY].Cardinality()
	expected.FixableCriticalSeverityCount = severityToFixableCVEs[storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY].Cardinality()

	return &expected
}

func (s *NodeCVEViewTestSuite) compileExpectedCVECoresWithPagination(filter *filterImpl, less lessFunc, offset, limit int) []CveCore {
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

func (s *NodeCVEViewTestSuite) compileExpectedAffectedNodeIDs(filter *filterImpl) []string {
	affectedNodeIDs := set.NewStringSet()
	for _, n := range s.nodeMap {
		if !filter.matchNode(n) {
			continue
		}
		if !filter.matchCluster(s.nameToClusters[n.ClusterName]) {
			continue
		}
		for _, c := range n.GetScan().GetComponents() {
			for _, v := range c.GetVulnerabilities() {
				if !filter.matchVuln(v) {
					continue
				}
				affectedNodeIDs.Add(n.GetId())
			}
		}
	}
	return affectedNodeIDs.AsSlice()
}

func applyPaginationProps(baseTc *testCase, paginationTc paginationTestCase) {
	baseTc.desc = fmt.Sprintf("%s %s", baseTc.desc, paginationTc.desc)
	baseTc.q.Pagination = paginationTc.q.GetPagination()
}

func getTestNodes() map[string]*storage.Node {
	os1NodeTemplate := fixtures.GetNodeWithUniqueComponents(20, 20)
	nodeConverter.MoveNodeVulnsToNewField(os1NodeTemplate)
	os1NodeTemplate.OperatingSystem = "os1"
	os1NodeTemplate.Scan.Components[0].Vulnerabilities[0].Cvss = 9.8
	os1NodeTemplate.Scan.Components[0].Vulnerabilities[0].Severity = storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY
	os1NodeTemplate.ClusterId = fixtureconsts.Cluster1
	os1NodeTemplate.ClusterName = fixtureconsts.ClusterName1
	os2NodeTemplate := fixtures.GetNodeWithUniqueComponents(5, 3)
	nodeConverter.MoveNodeVulnsToNewField(os2NodeTemplate)
	os2NodeTemplate.OperatingSystem = "os2"
	os2NodeTemplate.Scan.Components[3].Vulnerabilities[2].Cvss = 2.0
	os2NodeTemplate.ClusterId = fixtureconsts.Cluster2
	os2NodeTemplate.ClusterName = fixtureconsts.ClusterName2

	n1 := os1NodeTemplate.CloneVT()
	n1.Id = fixtureconsts.Node1
	n2 := os1NodeTemplate.CloneVT()
	n2.Id = fixtureconsts.Node2
	n3 := os2NodeTemplate.CloneVT()
	n3.Id = fixtureconsts.Node3
	n3.Labels = map[string]string{"searchLabel": "something"}
	nodes := []*storage.Node{n1, n2, n3}

	for i, n := range nodes {
		n.Name = fmt.Sprintf("Node-%d", i+1)
		n.Scan.OperatingSystem = n.OperatingSystem
		if n.OperatingSystem == os2NodeTemplate.OperatingSystem {
			for _, c := range n.GetScan().GetComponents() {
				for _, v := range c.GetVulnerabilities() {
					v.Severity = storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY
					v.SetFixedBy = &storage.NodeVulnerability_FixedBy{
						FixedBy: "",
					}
				}
			}
		}
	}
	return map[string]*storage.Node{n1.GetId(): n1, n2.GetId(): n2, n3.GetId(): n3}
}
