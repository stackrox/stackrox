package test

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/compliance"
	"github.com/stackrox/rox/central/compliance/datastore"
	"github.com/stackrox/rox/central/compliance/datastore/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
)

func TestSacFilter(t *testing.T) {
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

	resultToFilter := storage.ComplianceRunResults_builder{
		Domain: storage.ComplianceDomain_builder{
			Cluster: storage.ComplianceDomain_Cluster_builder{
				Id: clusterID,
			}.Build(),
			Deployments: map[string]*storage.ComplianceDomain_Deployment{
				"dep1": storage.ComplianceDomain_Deployment_builder{
					Id: "dep1",
				}.Build(),
				"dep2": storage.ComplianceDomain_Deployment_builder{
					Id: "dep2",
				}.Build(),
				"dep3": storage.ComplianceDomain_Deployment_builder{
					Id: "dep3",
				}.Build(),
			},
			Nodes: map[string]*storage.ComplianceDomain_Node{
				"node1": storage.ComplianceDomain_Node_builder{
					Id: "node1",
				}.Build(),
				"node2": storage.ComplianceDomain_Node_builder{
					Id: "node2",
				}.Build(),
				"node3": storage.ComplianceDomain_Node_builder{
					Id: "node3",
				}.Build(),
			},
		}.Build(),
		ClusterResults: storage.ComplianceRunResults_EntityResults_builder{
			ControlResults: make(map[string]*storage.ComplianceResultValue),
		}.Build(),
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
	}.Build()
	filtered, err := s.filter.FilterRunResults(ctx, resultToFilter)

	s.NoError(err)
	protoassert.Equal(s.T(), resultToFilter, filtered)
}

func (s *sacFilterTestSuite) TestFilterCluster() {
	clusterID := "c1"
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Deployment, resources.Node)))

	resultToFilter := storage.ComplianceRunResults_builder{
		Domain: storage.ComplianceDomain_builder{
			Cluster: storage.ComplianceDomain_Cluster_builder{
				Id: clusterID,
			}.Build(),
			Deployments: map[string]*storage.ComplianceDomain_Deployment{
				"dep1": storage.ComplianceDomain_Deployment_builder{
					Id: "dep1",
				}.Build(),
				"dep2": storage.ComplianceDomain_Deployment_builder{
					Id: "dep2",
				}.Build(),
				"dep3": storage.ComplianceDomain_Deployment_builder{
					Id: "dep3",
				}.Build(),
			},
			Nodes: map[string]*storage.ComplianceDomain_Node{
				"node1": storage.ComplianceDomain_Node_builder{
					Id: "node1",
				}.Build(),
				"node2": storage.ComplianceDomain_Node_builder{
					Id: "node2",
				}.Build(),
				"node3": storage.ComplianceDomain_Node_builder{
					Id: "node3",
				}.Build(),
			},
		}.Build(),
		ClusterResults: storage.ComplianceRunResults_EntityResults_builder{
			ControlResults: make(map[string]*storage.ComplianceResultValue),
		}.Build(),
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
	}.Build()
	filtered, err := s.filter.FilterRunResults(ctx, resultToFilter)

	expectedResults := storage.ComplianceRunResults_builder{
		Domain: storage.ComplianceDomain_builder{
			Deployments: map[string]*storage.ComplianceDomain_Deployment{
				"dep1": storage.ComplianceDomain_Deployment_builder{
					Id: "dep1",
				}.Build(),
				"dep2": storage.ComplianceDomain_Deployment_builder{
					Id: "dep2",
				}.Build(),
				"dep3": storage.ComplianceDomain_Deployment_builder{
					Id: "dep3",
				}.Build(),
			},
			Nodes: map[string]*storage.ComplianceDomain_Node{
				"node1": storage.ComplianceDomain_Node_builder{
					Id: "node1",
				}.Build(),
				"node2": storage.ComplianceDomain_Node_builder{
					Id: "node2",
				}.Build(),
				"node3": storage.ComplianceDomain_Node_builder{
					Id: "node3",
				}.Build(),
			},
		}.Build(),
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
	}.Build()
	s.NoError(err)
	protoassert.Equal(s.T(), expectedResults, filtered)
}

func (s *sacFilterTestSuite) TestFiltersAllDeployments() {
	clusterID := "c1"
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Cluster, resources.Node)))

	resultToFilter := storage.ComplianceRunResults_builder{
		Domain: storage.ComplianceDomain_builder{
			Cluster: storage.ComplianceDomain_Cluster_builder{
				Id: clusterID,
			}.Build(),
			Deployments: map[string]*storage.ComplianceDomain_Deployment{
				"dep1": storage.ComplianceDomain_Deployment_builder{
					Id: "dep1",
				}.Build(),
				"dep2": storage.ComplianceDomain_Deployment_builder{
					Id: "dep2",
				}.Build(),
				"dep3": storage.ComplianceDomain_Deployment_builder{
					Id: "dep3",
				}.Build(),
			},
			Nodes: map[string]*storage.ComplianceDomain_Node{
				"node1": storage.ComplianceDomain_Node_builder{
					Id: "node1",
				}.Build(),
				"node2": storage.ComplianceDomain_Node_builder{
					Id: "node2",
				}.Build(),
				"node3": storage.ComplianceDomain_Node_builder{
					Id: "node3",
				}.Build(),
			},
		}.Build(),
		ClusterResults: storage.ComplianceRunResults_EntityResults_builder{
			ControlResults: make(map[string]*storage.ComplianceResultValue),
		}.Build(),
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
	}.Build()
	filtered, err := s.filter.FilterRunResults(ctx, resultToFilter)

	expectedResults := storage.ComplianceRunResults_builder{
		Domain: storage.ComplianceDomain_builder{
			Cluster: storage.ComplianceDomain_Cluster_builder{
				Id: clusterID,
			}.Build(),
			Nodes: map[string]*storage.ComplianceDomain_Node{
				"node1": storage.ComplianceDomain_Node_builder{
					Id: "node1",
				}.Build(),
				"node2": storage.ComplianceDomain_Node_builder{
					Id: "node2",
				}.Build(),
				"node3": storage.ComplianceDomain_Node_builder{
					Id: "node3",
				}.Build(),
			},
			Deployments: map[string]*storage.ComplianceDomain_Deployment{},
		}.Build(),
		ClusterResults: storage.ComplianceRunResults_EntityResults_builder{
			ControlResults: make(map[string]*storage.ComplianceResultValue),
		}.Build(),
		NodeResults: map[string]*storage.ComplianceRunResults_EntityResults{
			"node1": {},
			"node2": {},
			"node3": {},
		},
	}.Build()
	s.NoError(err)
	protoassert.Equal(s.T(), expectedResults, filtered)
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

	resultToFilter := storage.ComplianceRunResults_builder{
		Domain: storage.ComplianceDomain_builder{
			Cluster: storage.ComplianceDomain_Cluster_builder{
				Id: clusterID,
			}.Build(),
			Deployments: map[string]*storage.ComplianceDomain_Deployment{
				"dep1": storage.ComplianceDomain_Deployment_builder{
					Id:        "dep1",
					ClusterId: clusterID,
					Namespace: namespace2,
				}.Build(),
				"dep2": storage.ComplianceDomain_Deployment_builder{
					Id:        "dep2",
					ClusterId: clusterID,
					Namespace: namespace1,
				}.Build(),
				"dep3": storage.ComplianceDomain_Deployment_builder{
					Id:        "dep3",
					ClusterId: clusterID,
					Namespace: namespace2,
				}.Build(),
			},
			Nodes: map[string]*storage.ComplianceDomain_Node{
				"node1": storage.ComplianceDomain_Node_builder{
					Id: "node1",
				}.Build(),
				"node2": storage.ComplianceDomain_Node_builder{
					Id: "node2",
				}.Build(),
				"node3": storage.ComplianceDomain_Node_builder{
					Id: "node3",
				}.Build(),
			},
		}.Build(),
		ClusterResults: storage.ComplianceRunResults_EntityResults_builder{
			ControlResults: make(map[string]*storage.ComplianceResultValue),
		}.Build(),
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
	}.Build()
	filtered, err := s.filter.FilterRunResults(ctx, resultToFilter)

	expectedResults := storage.ComplianceRunResults_builder{
		Domain: storage.ComplianceDomain_builder{
			Cluster: storage.ComplianceDomain_Cluster_builder{
				Id: clusterID,
			}.Build(),
			Deployments: map[string]*storage.ComplianceDomain_Deployment{
				"dep1": storage.ComplianceDomain_Deployment_builder{
					Id:        "dep1",
					ClusterId: clusterID,
					Namespace: namespace2,
				}.Build(),
				"dep3": storage.ComplianceDomain_Deployment_builder{
					Id:        "dep3",
					ClusterId: clusterID,
					Namespace: namespace2,
				}.Build(),
			},
			Nodes: map[string]*storage.ComplianceDomain_Node{
				"node1": storage.ComplianceDomain_Node_builder{
					Id: "node1",
				}.Build(),
				"node2": storage.ComplianceDomain_Node_builder{
					Id: "node2",
				}.Build(),
				"node3": storage.ComplianceDomain_Node_builder{
					Id: "node3",
				}.Build(),
			},
		}.Build(),
		ClusterResults: storage.ComplianceRunResults_EntityResults_builder{
			ControlResults: make(map[string]*storage.ComplianceResultValue),
		}.Build(),
		DeploymentResults: map[string]*storage.ComplianceRunResults_EntityResults{
			"dep1": {},
			"dep3": {},
		},
		NodeResults: map[string]*storage.ComplianceRunResults_EntityResults{
			"node1": {},
			"node2": {},
			"node3": {},
		},
	}.Build()
	s.NoError(err)
	protoassert.Equal(s.T(), expectedResults, filtered)
}

func (s *sacFilterTestSuite) TestFilterNodes() {
	clusterID := "c1"
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Cluster, resources.Deployment)))

	resultToFilter := storage.ComplianceRunResults_builder{
		Domain: storage.ComplianceDomain_builder{
			Cluster: storage.ComplianceDomain_Cluster_builder{
				Id: clusterID,
			}.Build(),
			Deployments: map[string]*storage.ComplianceDomain_Deployment{
				"dep1": storage.ComplianceDomain_Deployment_builder{
					Id: "dep1",
				}.Build(),
				"dep2": storage.ComplianceDomain_Deployment_builder{
					Id: "dep2",
				}.Build(),
				"dep3": storage.ComplianceDomain_Deployment_builder{
					Id: "dep3",
				}.Build(),
			},
			Nodes: map[string]*storage.ComplianceDomain_Node{
				"node1": storage.ComplianceDomain_Node_builder{
					Id: "node1",
				}.Build(),
				"node2": storage.ComplianceDomain_Node_builder{
					Id: "node2",
				}.Build(),
				"node3": storage.ComplianceDomain_Node_builder{
					Id: "node3",
				}.Build(),
			},
		}.Build(),
		ClusterResults: storage.ComplianceRunResults_EntityResults_builder{
			ControlResults: make(map[string]*storage.ComplianceResultValue),
		}.Build(),
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
	}.Build()
	filtered, err := s.filter.FilterRunResults(ctx, resultToFilter)

	expectedResults := storage.ComplianceRunResults_builder{
		Domain: storage.ComplianceDomain_builder{
			Cluster: storage.ComplianceDomain_Cluster_builder{
				Id: clusterID,
			}.Build(),
			Deployments: map[string]*storage.ComplianceDomain_Deployment{
				"dep1": storage.ComplianceDomain_Deployment_builder{
					Id: "dep1",
				}.Build(),
				"dep2": storage.ComplianceDomain_Deployment_builder{
					Id: "dep2",
				}.Build(),
				"dep3": storage.ComplianceDomain_Deployment_builder{
					Id: "dep3",
				}.Build(),
			},
		}.Build(),
		ClusterResults: storage.ComplianceRunResults_EntityResults_builder{
			ControlResults: make(map[string]*storage.ComplianceResultValue),
		}.Build(),
		DeploymentResults: map[string]*storage.ComplianceRunResults_EntityResults{
			"dep1": {},
			"dep2": {},
			"dep3": {},
		},
	}.Build()
	s.NoError(err)
	protoassert.Equal(s.T(), expectedResults, filtered)
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
			LastSuccessfulResults: storage.ComplianceRunResults_builder{
				Domain: storage.ComplianceDomain_builder{
					Cluster: storage.ComplianceDomain_Cluster_builder{
						Id: cluster1,
					}.Build(),
				}.Build(),
			}.Build(),
		},
		csPair2: {
			LastSuccessfulResults: storage.ComplianceRunResults_builder{
				Domain: storage.ComplianceDomain_builder{
					Cluster: storage.ComplianceDomain_Cluster_builder{
						Id: cluster2,
					}.Build(),
				}.Build(),
			}.Build(),
		},
	}

	results, err := s.filter.FilterBatchResults(ctx, resultToFilter)
	s.NoError(err)

	expectedResults := map[compliance.ClusterStandardPair]types.ResultsWithStatus{
		csPair2: {
			LastSuccessfulResults: storage.ComplianceRunResults_builder{
				Domain: storage.ComplianceDomain_builder{
					Cluster: storage.ComplianceDomain_Cluster_builder{
						Id: cluster2,
					}.Build(),
				}.Build(),
			}.Build(),
		},
	}
	s.Equal(expectedResults, results)
}
