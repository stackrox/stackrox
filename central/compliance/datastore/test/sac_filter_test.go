package test

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/compliance"
	"github.com/stackrox/rox/central/compliance/datastore"
	"github.com/stackrox/rox/central/compliance/datastore/types"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

func TestSacFilter(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(sacFilterTestSuite))
}

type sacFilterTestSuite struct {
	suite.Suite

	filter datastore.SacFilter
}

func (s *sacFilterTestSuite) SetupTest() {
	s.filter = datastore.NewSacFilter()
}

func (s *sacFilterTestSuite) TestRunNotFiltered() {
	clusterID := "c1"
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Cluster, resources.Deployment, resources.Node)))

	resultToFilter := &storage.ComplianceRunResults{
		Domain: &storage.ComplianceDomain{
			Cluster: &storage.ComplianceDomain_Cluster{
				Id: clusterID,
			},
			Deployments: map[string]*storage.ComplianceDomain_Deployment{
				"dep1": {
					Id: "dep1",
				},
				"dep2": {
					Id: "dep2",
				},
				"dep3": {
					Id: "dep3",
				},
			},
			Nodes: map[string]*storage.ComplianceDomain_Node{
				"node1": {
					Id: "node1",
				},
				"node2": {
					Id: "node2",
				},
				"node3": {
					Id: "node3",
				},
			},
		},
		ClusterResults: &storage.ComplianceRunResults_EntityResults{
			ControlResults: make(map[string]*storage.ComplianceResultValue),
		},
		DeploymentResults: map[string]*storage.ComplianceRunResults_EntityResults{
			"dep1": {},
			"dep2": {},
			"dep3": {},
		},
		NodeResults: map[string]*storage.ComplianceRunResults_EntityResults{
			"node1": {},
			"node2": {},
			"node3": {},
		},
	}
	filtered, err := s.filter.FilterRunResults(ctx, resultToFilter)

	s.NoError(err)
	s.Equal(resultToFilter, filtered)
}

func (s *sacFilterTestSuite) TestFilterCluster() {
	clusterID := "c1"
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Deployment, resources.Node)))

	resultToFilter := &storage.ComplianceRunResults{
		Domain: &storage.ComplianceDomain{
			Cluster: &storage.ComplianceDomain_Cluster{
				Id: clusterID,
			},
			Deployments: map[string]*storage.ComplianceDomain_Deployment{
				"dep1": {
					Id: "dep1",
				},
				"dep2": {
					Id: "dep2",
				},
				"dep3": {
					Id: "dep3",
				},
			},
			Nodes: map[string]*storage.ComplianceDomain_Node{
				"node1": {
					Id: "node1",
				},
				"node2": {
					Id: "node2",
				},
				"node3": {
					Id: "node3",
				},
			},
		},
		ClusterResults: &storage.ComplianceRunResults_EntityResults{
			ControlResults: make(map[string]*storage.ComplianceResultValue),
		},
		DeploymentResults: map[string]*storage.ComplianceRunResults_EntityResults{
			"dep1": {},
			"dep2": {},
			"dep3": {},
		},
		NodeResults: map[string]*storage.ComplianceRunResults_EntityResults{
			"node1": {},
			"node2": {},
			"node3": {},
		},
	}
	filtered, err := s.filter.FilterRunResults(ctx, resultToFilter)

	expectedResults := &storage.ComplianceRunResults{
		Domain: &storage.ComplianceDomain{
			Deployments: map[string]*storage.ComplianceDomain_Deployment{
				"dep1": {
					Id: "dep1",
				},
				"dep2": {
					Id: "dep2",
				},
				"dep3": {
					Id: "dep3",
				},
			},
			Nodes: map[string]*storage.ComplianceDomain_Node{
				"node1": {
					Id: "node1",
				},
				"node2": {
					Id: "node2",
				},
				"node3": {
					Id: "node3",
				},
			},
		},
		DeploymentResults: map[string]*storage.ComplianceRunResults_EntityResults{
			"dep1": {},
			"dep2": {},
			"dep3": {},
		},
		NodeResults: map[string]*storage.ComplianceRunResults_EntityResults{
			"node1": {},
			"node2": {},
			"node3": {},
		},
	}
	s.NoError(err)
	s.Equal(expectedResults, filtered)
}

func (s *sacFilterTestSuite) TestFiltersAllDeployments() {
	clusterID := "c1"
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Cluster, resources.Node)))

	resultToFilter := &storage.ComplianceRunResults{
		Domain: &storage.ComplianceDomain{
			Cluster: &storage.ComplianceDomain_Cluster{
				Id: clusterID,
			},
			Deployments: map[string]*storage.ComplianceDomain_Deployment{
				"dep1": {
					Id: "dep1",
				},
				"dep2": {
					Id: "dep2",
				},
				"dep3": {
					Id: "dep3",
				},
			},
			Nodes: map[string]*storage.ComplianceDomain_Node{
				"node1": {
					Id: "node1",
				},
				"node2": {
					Id: "node2",
				},
				"node3": {
					Id: "node3",
				},
			},
		},
		ClusterResults: &storage.ComplianceRunResults_EntityResults{
			ControlResults: make(map[string]*storage.ComplianceResultValue),
		},
		DeploymentResults: map[string]*storage.ComplianceRunResults_EntityResults{
			"dep1": {},
			"dep2": {},
			"dep3": {},
		},
		NodeResults: map[string]*storage.ComplianceRunResults_EntityResults{
			"node1": {},
			"node2": {},
			"node3": {},
		},
	}
	filtered, err := s.filter.FilterRunResults(ctx, resultToFilter)

	expectedResults := &storage.ComplianceRunResults{
		Domain: &storage.ComplianceDomain{
			Cluster: &storage.ComplianceDomain_Cluster{
				Id: clusterID,
			},
			Nodes: map[string]*storage.ComplianceDomain_Node{
				"node1": {
					Id: "node1",
				},
				"node2": {
					Id: "node2",
				},
				"node3": {
					Id: "node3",
				},
			},
			Deployments: map[string]*storage.ComplianceDomain_Deployment{},
		},
		ClusterResults: &storage.ComplianceRunResults_EntityResults{
			ControlResults: make(map[string]*storage.ComplianceResultValue),
		},
		NodeResults: map[string]*storage.ComplianceRunResults_EntityResults{
			"node1": {},
			"node2": {},
			"node3": {},
		},
	}
	s.NoError(err)
	s.Equal(expectedResults, filtered)
}

func (s *sacFilterTestSuite) TestFiltersSomeDeployments() {
	clusterID := "c1"
	namespace1 := "n1"
	namespace2 := "n2"
	ctx := sac.WithGlobalAccessScopeChecker(
		context.Background(),
		sac.TestScopeCheckerCoreFromFullScopeMap(s.T(),
			sac.TestScopeMap{
				storage.Access_READ_ACCESS: {
					resources.Cluster.GetResource(): &sac.TestResourceScope{Included: true},
					resources.Node.GetResource():    &sac.TestResourceScope{Included: true},
					resources.Deployment.GetResource(): &sac.TestResourceScope{
						Clusters: map[string]*sac.TestClusterScope{
							clusterID: {Namespaces: []string{namespace2}},
						},
					},
				},
			},
		))

	resultToFilter := &storage.ComplianceRunResults{
		Domain: &storage.ComplianceDomain{
			Cluster: &storage.ComplianceDomain_Cluster{
				Id: clusterID,
			},
			Deployments: map[string]*storage.ComplianceDomain_Deployment{
				"dep1": {
					Id:        "dep1",
					ClusterId: clusterID,
					Namespace: namespace2,
				},
				"dep2": {
					Id:        "dep2",
					ClusterId: clusterID,
					Namespace: namespace1,
				},
				"dep3": {
					Id:        "dep3",
					ClusterId: clusterID,
					Namespace: namespace2,
				},
			},
			Nodes: map[string]*storage.ComplianceDomain_Node{
				"node1": {
					Id: "node1",
				},
				"node2": {
					Id: "node2",
				},
				"node3": {
					Id: "node3",
				},
			},
		},
		ClusterResults: &storage.ComplianceRunResults_EntityResults{
			ControlResults: make(map[string]*storage.ComplianceResultValue),
		},
		DeploymentResults: map[string]*storage.ComplianceRunResults_EntityResults{
			"dep1": {},
			"dep2": {},
			"dep3": {},
		},
		NodeResults: map[string]*storage.ComplianceRunResults_EntityResults{
			"node1": {},
			"node2": {},
			"node3": {},
		},
	}
	filtered, err := s.filter.FilterRunResults(ctx, resultToFilter)

	expectedResults := &storage.ComplianceRunResults{
		Domain: &storage.ComplianceDomain{
			Cluster: &storage.ComplianceDomain_Cluster{
				Id: clusterID,
			},
			Deployments: map[string]*storage.ComplianceDomain_Deployment{
				"dep1": {
					Id:        "dep1",
					ClusterId: clusterID,
					Namespace: namespace2,
				},
				"dep3": {
					Id:        "dep3",
					ClusterId: clusterID,
					Namespace: namespace2,
				},
			},
			Nodes: map[string]*storage.ComplianceDomain_Node{
				"node1": {
					Id: "node1",
				},
				"node2": {
					Id: "node2",
				},
				"node3": {
					Id: "node3",
				},
			},
		},
		ClusterResults: &storage.ComplianceRunResults_EntityResults{
			ControlResults: make(map[string]*storage.ComplianceResultValue),
		},
		DeploymentResults: map[string]*storage.ComplianceRunResults_EntityResults{
			"dep1": {},
			"dep3": {},
		},
		NodeResults: map[string]*storage.ComplianceRunResults_EntityResults{
			"node1": {},
			"node2": {},
			"node3": {},
		},
	}
	s.NoError(err)
	s.Equal(expectedResults, filtered)
}

func (s *sacFilterTestSuite) TestFilterNodes() {
	clusterID := "c1"
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Cluster, resources.Deployment)))

	resultToFilter := &storage.ComplianceRunResults{
		Domain: &storage.ComplianceDomain{
			Cluster: &storage.ComplianceDomain_Cluster{
				Id: clusterID,
			},
			Deployments: map[string]*storage.ComplianceDomain_Deployment{
				"dep1": {
					Id: "dep1",
				},
				"dep2": {
					Id: "dep2",
				},
				"dep3": {
					Id: "dep3",
				},
			},
			Nodes: map[string]*storage.ComplianceDomain_Node{
				"node1": {
					Id: "node1",
				},
				"node2": {
					Id: "node2",
				},
				"node3": {
					Id: "node3",
				},
			},
		},
		ClusterResults: &storage.ComplianceRunResults_EntityResults{
			ControlResults: make(map[string]*storage.ComplianceResultValue),
		},
		DeploymentResults: map[string]*storage.ComplianceRunResults_EntityResults{
			"dep1": {},
			"dep2": {},
			"dep3": {},
		},
		NodeResults: map[string]*storage.ComplianceRunResults_EntityResults{
			"node1": {},
			"node2": {},
			"node3": {},
		},
	}
	filtered, err := s.filter.FilterRunResults(ctx, resultToFilter)

	expectedResults := &storage.ComplianceRunResults{
		Domain: &storage.ComplianceDomain{
			Cluster: &storage.ComplianceDomain_Cluster{
				Id: clusterID,
			},
			Deployments: map[string]*storage.ComplianceDomain_Deployment{
				"dep1": {
					Id: "dep1",
				},
				"dep2": {
					Id: "dep2",
				},
				"dep3": {
					Id: "dep3",
				},
			},
		},
		ClusterResults: &storage.ComplianceRunResults_EntityResults{
			ControlResults: make(map[string]*storage.ComplianceResultValue),
		},
		DeploymentResults: map[string]*storage.ComplianceRunResults_EntityResults{
			"dep1": {},
			"dep2": {},
			"dep3": {},
		},
	}
	s.NoError(err)
	s.Equal(expectedResults, filtered)
}

func (s *sacFilterTestSuite) TestFiltersClustersBatch() {
	cluster1 := "c1"
	cluster2 := "c2"
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Cluster, resources.Compliance, resources.Deployment, resources.Node),
			sac.ClusterScopeKeys(cluster2)))

	csPair1 := compliance.ClusterStandardPair{
		ClusterID:  cluster1,
		StandardID: "sid1",
	}
	csPair2 := compliance.ClusterStandardPair{
		ClusterID:  cluster2,
		StandardID: "sid2",
	}
	resultToFilter := map[compliance.ClusterStandardPair]types.ResultsWithStatus{
		csPair1: {
			LastSuccessfulResults: &storage.ComplianceRunResults{
				Domain: &storage.ComplianceDomain{
					Cluster: &storage.ComplianceDomain_Cluster{
						Id: cluster1,
					},
				},
			},
		},
		csPair2: {
			LastSuccessfulResults: &storage.ComplianceRunResults{
				Domain: &storage.ComplianceDomain{
					Cluster: &storage.ComplianceDomain_Cluster{
						Id: cluster2,
					},
				},
			},
		},
	}

	results, err := s.filter.FilterBatchResults(ctx, resultToFilter)
	s.NoError(err)

	expectedResults := map[compliance.ClusterStandardPair]types.ResultsWithStatus{
		csPair2: {
			LastSuccessfulResults: &storage.ComplianceRunResults{
				Domain: &storage.ComplianceDomain{
					Cluster: &storage.ComplianceDomain_Cluster{
						Id: cluster2,
					},
				},
			},
		},
	}
	s.Equal(expectedResults, results)
}
