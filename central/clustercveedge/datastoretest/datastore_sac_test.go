//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/clustercveedge/datastore"
	graphDBTestUtils "github.com/stackrox/rox/central/graphdb/testutils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

const (
	cluster1ToCVE1EdgeID = "cluster1ToCVE1EdgeID"
	cluster1ToCVE2EdgeID = "cluster1ToCVE2EdgeID"
	cluster2ToCVE2EdgeID = "cluster2ToCVE2EdgeID"
	cluster2ToCVE3EdgeID = "cluster2ToCVE3EdgeID"
)

func TestClusterCVEEdgeDatastoreSAC(t *testing.T) {
	suite.Run(t, new(clusterCVEEdgeDatastoreSACSuite))
}

type clusterCVEEdgeDatastoreSACSuite struct {
	suite.Suite

	datastore          datastore.DataStore
	testGraphDatastore graphDBTestUtils.TestGraphDataStore
}

func (s *clusterCVEEdgeDatastoreSACSuite) SetupSuite() {
	var err error
	s.testGraphDatastore, err = graphDBTestUtils.NewTestGraphDataStore(s.T())
	s.Require().NoError(err)
	s.datastore, err = datastore.GetTestPostgresDataStore(s.T(), s.testGraphDatastore.GetPostgresPool())
	s.Require().NoError(err)
}

func (s *clusterCVEEdgeDatastoreSACSuite) TearDownSuite() {
	s.testGraphDatastore.Cleanup(s.T())
}

func (s *clusterCVEEdgeDatastoreSACSuite) cleanImageToVulnerabilitiesGraph() {
	s.Require().NoError(s.testGraphDatastore.CleanClusterToVulnerabilitiesGraph())
}

func getCveID(vulnerability *storage.EmbeddedVulnerability) string {
	return vulnerability.GetCve()
}

func getEdgeID(vulnerability *storage.EmbeddedVulnerability, clusterID string) string {
	return clusterID + "#" + vulnerability.GetCve()
}

var (
	embeddedCVE1 = fixtures.GetEmbeddedClusterCVE1234x0001()
	embeddedCVE2 = fixtures.GetEmbeddedClusterCVE4567x0002()
	embeddedCVE3 = fixtures.GetEmbeddedClusterCVE2345x0003()
)

type testCase struct {
	name         string
	ctx          context.Context
	visibleEdges map[string]bool
}

func getClusterCVEEdges(cluster1, cluster2 string) map[string]string {
	return map[string]string{
		cluster1ToCVE1EdgeID: getEdgeID(embeddedCVE1, cluster1),
		cluster1ToCVE2EdgeID: getEdgeID(embeddedCVE2, cluster1),
		cluster2ToCVE2EdgeID: getEdgeID(embeddedCVE2, cluster2),
		cluster2ToCVE3EdgeID: getEdgeID(embeddedCVE3, cluster2),
	}
}

func getClusterCVEEdgeReadTestCases(_ *testing.T, validCluster1 string, validCluster2 string) []testCase {
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
			visibleEdges: map[string]bool{
				cluster1ToCVE1EdgeID: true,
				cluster1ToCVE2EdgeID: true,
				cluster2ToCVE2EdgeID: true,
				cluster2ToCVE3EdgeID: true,
			},
		},
		{
			name: "Full read-only access has read access to all data",
			ctx: sac.WithGlobalAccessScopeChecker(
				context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resources.Cluster),
				),
			),
			visibleEdges: map[string]bool{
				cluster1ToCVE1EdgeID: true,
				cluster1ToCVE2EdgeID: true,
				cluster2ToCVE2EdgeID: true,
				cluster2ToCVE3EdgeID: true,
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
			visibleEdges: map[string]bool{
				cluster1ToCVE1EdgeID: true,
				cluster1ToCVE2EdgeID: true,
				cluster2ToCVE2EdgeID: false,
				cluster2ToCVE3EdgeID: false,
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
			visibleEdges: map[string]bool{
				cluster1ToCVE1EdgeID: true,
				cluster1ToCVE2EdgeID: true,
				cluster2ToCVE2EdgeID: false,
				cluster2ToCVE3EdgeID: false,
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
			visibleEdges: map[string]bool{
				cluster1ToCVE1EdgeID: false,
				cluster1ToCVE2EdgeID: false,
				cluster2ToCVE2EdgeID: true,
				cluster2ToCVE3EdgeID: true,
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
			visibleEdges: map[string]bool{
				cluster1ToCVE1EdgeID: false,
				cluster1ToCVE2EdgeID: false,
				cluster2ToCVE2EdgeID: true,
				cluster2ToCVE3EdgeID: true,
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
			visibleEdges: map[string]bool{
				cluster1ToCVE1EdgeID: false,
				cluster1ToCVE2EdgeID: false,
				cluster2ToCVE2EdgeID: false,
				cluster2ToCVE3EdgeID: false,
			},
		},
	}
}

func (s *clusterCVEEdgeDatastoreSACSuite) TestExists() {
	err := s.testGraphDatastore.PushClusterToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph()
	s.Require().NoError(err)
	validClusters := s.testGraphDatastore.GetStoredClusterIDs()
	s.Require().True(len(validClusters) >= 2)

	testCases := getClusterCVEEdgeReadTestCases(s.T(), validClusters[0], validClusters[1])
	for _, c := range testCases {
		s.Run(c.name, func() {
			ctx := c.ctx
			targetEdgeID := getClusterCVEEdges(validClusters[0], validClusters[1])[cluster1ToCVE1EdgeID]
			exists, err := s.datastore.Exists(ctx, targetEdgeID)
			s.NoError(err)
			s.Equal(c.visibleEdges[cluster1ToCVE1EdgeID], exists)
		})
	}
}

func (s *clusterCVEEdgeDatastoreSACSuite) TestGet() {
	err := s.testGraphDatastore.PushClusterToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph()
	s.Require().NoError(err)
	validClusters := s.testGraphDatastore.GetStoredClusterIDs()
	s.Require().True(len(validClusters) >= 2)

	targetEdgeCveID := getCveID(embeddedCVE1)
	targetEdgeID := getClusterCVEEdges(validClusters[0], validClusters[1])[cluster1ToCVE1EdgeID]
	testCases := getClusterCVEEdgeReadTestCases(s.T(), validClusters[0], validClusters[1])
	for _, c := range testCases {
		s.Run(c.name, func() {
			ctx := c.ctx
			edge, exists, err := s.datastore.Get(ctx, targetEdgeID)
			s.NoError(err)
			if c.visibleEdges[cluster1ToCVE1EdgeID] {
				s.True(exists)
				s.NotNil(edge)
				s.Equal(validClusters[0], edge.GetClusterId())
				s.Equal(targetEdgeCveID, edge.GetCveId())
			} else {
				s.False(exists)
				s.Nil(edge)
			}
		})
	}
}

func (s *clusterCVEEdgeDatastoreSACSuite) TestGetBatch() {
	err := s.testGraphDatastore.PushClusterToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph()
	s.Require().NoError(err)
	validClusters := s.testGraphDatastore.GetStoredClusterIDs()
	s.Require().True(len(validClusters) >= 2)

	testEdgeIDs := getClusterCVEEdges(validClusters[0], validClusters[1])
	testCases := getClusterCVEEdgeReadTestCases(s.T(), validClusters[0], validClusters[1])
	targetEdgeIDs := []string{
		testEdgeIDs[cluster1ToCVE1EdgeID],
		testEdgeIDs[cluster1ToCVE2EdgeID],
		testEdgeIDs[cluster2ToCVE2EdgeID],
		testEdgeIDs[cluster2ToCVE3EdgeID],
	}
	for _, c := range testCases {
		s.Run(c.name, func() {
			ctx := c.ctx
			edges, err := s.datastore.GetBatch(ctx, targetEdgeIDs)
			s.NoError(err)
			edgesByID := make(map[string]*storage.ClusterCVEEdge, 0)
			for ix := range edges {
				e := edges[ix]
				edgesByID[e.GetId()] = e
			}
			visibleEdgeCount := 0
			for id, visible := range c.visibleEdges {
				if visible {
					_, found := edgesByID[testEdgeIDs[id]]
					s.True(found)
					visibleEdgeCount++
				}
			}
			s.Equal(visibleEdgeCount, len(edges))
		})
	}
}

func (s *clusterCVEEdgeDatastoreSACSuite) TestCount() {
	err := s.testGraphDatastore.PushClusterToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph()
	s.Require().NoError(err)
	validClusters := s.testGraphDatastore.GetStoredClusterIDs()
	s.Require().True(len(validClusters) >= 2)

	testCases := getClusterCVEEdgeReadTestCases(s.T(), validClusters[0], validClusters[1])
	for _, c := range testCases {
		s.Run(c.name, func() {
			ctx := c.ctx
			expectedCount := 0
			for _, visible := range c.visibleEdges {
				if visible {
					expectedCount++
				}
			}
			count, err := s.datastore.Count(ctx, search.EmptyQuery())
			s.NoError(err)
			s.Equal(expectedCount, count)
		})
	}
}

func (s *clusterCVEEdgeDatastoreSACSuite) TestSearch() {
	err := s.testGraphDatastore.PushClusterToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph()
	s.Require().NoError(err)
	validClusters := s.testGraphDatastore.GetStoredClusterIDs()
	s.Require().True(len(validClusters) >= 2)

	testEdgeIDs := getClusterCVEEdges(validClusters[0], validClusters[1])
	testCases := getClusterCVEEdgeReadTestCases(s.T(), validClusters[0], validClusters[1])
	for _, c := range testCases {
		s.Run(c.name, func() {
			ctx := c.ctx
			expectedIDs := make([]string, 0, len(c.visibleEdges))
			for id, visible := range c.visibleEdges {
				if visible {
					expectedIDs = append(expectedIDs, testEdgeIDs[id])
				}
			}
			results, err := s.datastore.Search(ctx, search.EmptyQuery())
			s.NoError(err)
			resultIDs := make([]string, 0, len(results))
			for _, r := range results {
				resultIDs = append(resultIDs, r.ID)
			}
			s.ElementsMatch(resultIDs, expectedIDs)
		})
	}
}

func (s *clusterCVEEdgeDatastoreSACSuite) TestSearchEdges() {
	err := s.testGraphDatastore.PushClusterToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph()
	s.Require().NoError(err)
	validClusters := s.testGraphDatastore.GetStoredClusterIDs()
	s.Require().True(len(validClusters) >= 2)

	testEdgeIDs := getClusterCVEEdges(validClusters[0], validClusters[1])
	testCases := getClusterCVEEdgeReadTestCases(s.T(), validClusters[0], validClusters[1])
	for _, c := range testCases {
		s.Run(c.name, func() {
			ctx := c.ctx
			expectedIDs := make([]string, 0, len(c.visibleEdges))
			for id, visible := range c.visibleEdges {
				if visible {
					expectedIDs = append(expectedIDs, testEdgeIDs[id])
				}
			}
			results, err := s.datastore.SearchEdges(ctx, search.EmptyQuery())
			s.NoError(err)
			resultIDs := make([]string, 0, len(results))
			for _, r := range results {
				resultIDs = append(resultIDs, r.GetId())
			}
			s.ElementsMatch(resultIDs, expectedIDs)
		})
	}
}

func (s *clusterCVEEdgeDatastoreSACSuite) TestSearchRawEdges() {
	err := s.testGraphDatastore.PushClusterToVulnerabilitiesGraph()
	defer s.cleanImageToVulnerabilitiesGraph()
	s.Require().NoError(err)
	validClusters := s.testGraphDatastore.GetStoredClusterIDs()
	s.Require().True(len(validClusters) >= 2)

	testEdgeIDs := getClusterCVEEdges(validClusters[0], validClusters[1])
	testCases := getClusterCVEEdgeReadTestCases(s.T(), validClusters[0], validClusters[1])
	for _, c := range testCases {
		s.Run(c.name, func() {
			ctx := c.ctx
			expectedIDs := make([]string, 0, len(c.visibleEdges))
			for id, visible := range c.visibleEdges {
				if visible {
					expectedIDs = append(expectedIDs, testEdgeIDs[id])
				}
			}
			results, err := s.datastore.SearchRawEdges(ctx, search.EmptyQuery())
			s.NoError(err)
			resultIDs := make([]string, 0, len(results))
			for _, r := range results {
				resultIDs = append(resultIDs, r.GetId())
			}
			s.ElementsMatch(resultIDs, expectedIDs)
		})
	}
}
