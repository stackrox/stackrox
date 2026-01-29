//go:build sql_integration

package service

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
)

const (
	clusterPermission             = "Cluster"
	compliancePermission          = "Compliance"
	deploymentPermission          = "Deployment"
	deploymentExtensionPermission = "DeploymentExtension"
	integrationPermission         = "Integration"
	namespacePermission           = "Namespace"
	networkGraphPermission        = "NetworkGraph"
	nodePermission                = "Node"
	rolePermission                = "Role"
)

func TestServiceImplWithDB_Other(t *testing.T) {
	suite.Run(t, new(serviceImplOtherTestSuite))
}

type serviceImplOtherTestSuite struct {
	suite.Suite

	tester *serviceImplTester
}

func (s *serviceImplOtherTestSuite) SetupSuite() {
	s.tester = &serviceImplTester{}
	s.tester.Setup(s.T())
}

func (s *serviceImplOtherTestSuite) SetupTest() {
	s.Require().NotNil(s.tester)
	s.tester.SetupTest(s.T())
}

func (s *serviceImplOtherTestSuite) TearDownTest() {
	s.Require().NotNil(s.tester)
	s.tester.TearDownTest(s.T())
}

func (s *serviceImplOtherTestSuite) TestGetClustersForPermissions() {
	deepPurpleClusterID := s.tester.clusterNameToIDMap[clusterDeepPurple.GetName()]
	queenClusterID := s.tester.clusterNameToIDMap[clusterQueen.GetName()]
	testResourceScope1 := getTestResourceScopeSingleNamespace(
		queenClusterID,
		namespaceInnuendoInClusterQueen.GetName())
	pinkFloydClusterID := s.tester.clusterNameToIDMap[clusterPinkFloyd.GetName()]
	testResourceScope2 := getTestResourceScopeSingleNamespace(
		pinkFloydClusterID,
		namespaceTheWallInClusterPinkFloyd.GetName())

	testScopeMap := sac.TestScopeMap{
		storage.Access_READ_ACCESS: map[permissions.Resource]*sac.TestResourceScope{
			resources.Integration.GetResource(): {
				Included: true,
			},
			resources.Node.GetResource():         testResourceScope1,
			resources.Deployment.GetResource():   testResourceScope1,
			resources.NetworkGraph.GetResource(): testResourceScope2,
		},
	}

	extendedAccessTestScopeMap := sac.TestScopeMap{
		storage.Access_READ_ACCESS: map[permissions.Resource]*sac.TestResourceScope{
			resources.Compliance.GetResource(): {
				Included: true,
			},
			resources.Integration.GetResource(): {
				Included: true,
			},
			resources.Node.GetResource():         testResourceScope1,
			resources.Deployment.GetResource():   testResourceScope1,
			resources.NetworkGraph.GetResource(): testResourceScope2,
		},
	}

	deepPurpleClusterResponse := &v1.ScopeObject{
		Id:   deepPurpleClusterID,
		Name: clusterDeepPurple.GetName(),
	}

	queenClusterResponse := &v1.ScopeObject{
		Id:   queenClusterID,
		Name: clusterQueen.GetName(),
	}

	pinkFloydClusterResponse := &v1.ScopeObject{
		Id:   pinkFloydClusterID,
		Name: clusterPinkFloyd.GetName(),
	}

	testCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.TestScopeCheckerCoreFromFullScopeMap(s.T(), testScopeMap))

	extendedAccessTestCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.TestScopeCheckerCoreFromFullScopeMap(s.T(), extendedAccessTestScopeMap))

	testCases := []struct {
		name              string
		context           context.Context
		testedPermissions []string
		expectedClusters  []*v1.ScopeObject
	}{
		{
			name:              "Global permission (Not Granted) gets no cluster data.",
			context:           testCtx,
			testedPermissions: []string{rolePermission},
			expectedClusters:  []*v1.ScopeObject{},
		},
		{
			name:              "Global permission (Granted) gets no cluster data.",
			context:           testCtx,
			testedPermissions: []string{integrationPermission},
			expectedClusters:  []*v1.ScopeObject{},
		},
		{
			name:              "Not granted Cluster scoped permission gets no cluster data.",
			context:           testCtx,
			testedPermissions: []string{clusterPermission},
			expectedClusters:  []*v1.ScopeObject{},
		},
		{
			name:              "Granted Cluster scoped permission gets only cluster data for clusters in permission scope.",
			context:           testCtx,
			testedPermissions: []string{nodePermission},
			expectedClusters:  []*v1.ScopeObject{queenClusterResponse},
		},
		{
			name:              "Not granted Namespace scoped permission gets no cluster data.",
			context:           testCtx,
			testedPermissions: []string{namespacePermission},
			expectedClusters:  []*v1.ScopeObject{},
		},
		{
			name:              "Granted Namespace scoped permission gets only cluster data for clusters in permission scope.",
			context:           testCtx,
			testedPermissions: []string{deploymentPermission},
			expectedClusters:  []*v1.ScopeObject{queenClusterResponse},
		},
		{
			name:              "Multiple not granted Namespace scoped permissions get no cluster data.",
			context:           testCtx,
			testedPermissions: []string{namespacePermission, deploymentExtensionPermission},
			expectedClusters:  []*v1.ScopeObject{},
		},
		{
			name:              "Multiple Namespace scoped permissions get only cluster data for clusters in granted permission scopes.",
			context:           testCtx,
			testedPermissions: []string{namespacePermission, deploymentPermission},
			expectedClusters:  []*v1.ScopeObject{queenClusterResponse},
		},
		{
			name:              "empty permission list get cluster data for all cluster data in scope of granted cluster and namespace permissions.",
			context:           testCtx,
			testedPermissions: []string{},
			expectedClusters:  []*v1.ScopeObject{queenClusterResponse, pinkFloydClusterResponse},
		},
		{
			name:              "Extended Access - Global permission (Not Granted) gets no cluster data.",
			context:           extendedAccessTestCtx,
			testedPermissions: []string{rolePermission},
			expectedClusters:  []*v1.ScopeObject{},
		},
		{
			name:              "Extended Access - Global permission (Granted) gets no cluster data.",
			context:           extendedAccessTestCtx,
			testedPermissions: []string{integrationPermission},
			expectedClusters:  []*v1.ScopeObject{},
		},
		{
			name:              "Extended Access - Not granted Cluster scoped permission gets no cluster data.",
			context:           extendedAccessTestCtx,
			testedPermissions: []string{clusterPermission},
			expectedClusters:  []*v1.ScopeObject{},
		},
		{
			name:              "Extended Access - Granted Cluster scoped permission at resource level gets all cluster data",
			context:           extendedAccessTestCtx,
			testedPermissions: []string{compliancePermission},
			expectedClusters: []*v1.ScopeObject{
				deepPurpleClusterResponse,
				queenClusterResponse,
				pinkFloydClusterResponse,
			},
		},
		{
			name:              "Extended Access - Granted Cluster scoped permission gets only cluster data for clusters in permission scope.",
			context:           extendedAccessTestCtx,
			testedPermissions: []string{nodePermission},
			expectedClusters:  []*v1.ScopeObject{queenClusterResponse},
		},
		{
			name:              "Extended Access - Not granted Namespace scoped permission gets no cluster data.",
			context:           extendedAccessTestCtx,
			testedPermissions: []string{namespacePermission},
			expectedClusters:  []*v1.ScopeObject{},
		},
		{
			name:              "Extended Access - Granted Namespace scoped permission gets only cluster data for clusters in permission scope.",
			context:           extendedAccessTestCtx,
			testedPermissions: []string{deploymentPermission},
			expectedClusters:  []*v1.ScopeObject{queenClusterResponse},
		},
		{
			name:              "Extended Access - Multiple not granted Namespace scoped permissions get no cluster data.",
			context:           extendedAccessTestCtx,
			testedPermissions: []string{namespacePermission, deploymentExtensionPermission},
			expectedClusters:  []*v1.ScopeObject{},
		},
		{
			name:              "Extended Access - Multiple Namespace scoped permissions get only cluster data for clusters in granted permission scopes.",
			context:           extendedAccessTestCtx,
			testedPermissions: []string{namespacePermission, deploymentPermission},
			expectedClusters:  []*v1.ScopeObject{queenClusterResponse},
		},
		{
			name:              "Extended Access - empty permission list get cluster data for all cluster data in scope of granted cluster and namespace permissions.",
			context:           extendedAccessTestCtx,
			testedPermissions: []string{},
			expectedClusters: []*v1.ScopeObject{
				deepPurpleClusterResponse,
				queenClusterResponse,
				pinkFloydClusterResponse,
			},
		},
	}

	for _, c := range testCases {
		s.Run(c.name, func() {
			clusterResponse, err := s.tester.service.GetClustersForPermissions(c.context, &v1.GetClustersForPermissionsRequest{
				Pagination:  nil,
				Permissions: c.testedPermissions,
			})
			s.NoError(err)
			protoassert.ElementsMatch(s.T(), clusterResponse.GetClusters(), c.expectedClusters)
		})
	}
}

func (s *serviceImplOtherTestSuite) TestGetClustersForPermissionsPagination() {
	queenClusterID := s.tester.clusterNameToIDMap[clusterQueen.GetName()]
	testResourceScope1 := getTestResourceScopeSingleNamespace(
		queenClusterID,
		namespaceInnuendoInClusterQueen.GetName())
	pinkFloydClusterID := s.tester.clusterNameToIDMap[clusterPinkFloyd.GetName()]
	testResourceScope2 := getTestResourceScopeSingleNamespace(
		pinkFloydClusterID,
		namespaceTheWallInClusterPinkFloyd.GetName())
	testScopeMap := sac.TestScopeMap{
		storage.Access_READ_ACCESS: map[permissions.Resource]*sac.TestResourceScope{
			resources.Integration.GetResource(): {
				Included: true,
			},
			resources.Node.GetResource():         testResourceScope1,
			resources.Deployment.GetResource():   testResourceScope1,
			resources.NetworkGraph.GetResource(): testResourceScope2,
		},
	}

	queenClusterResponse := &v1.ScopeObject{
		Id:   queenClusterID,
		Name: clusterQueen.GetName(),
	}

	pinkFloydClusterResponse := &v1.ScopeObject{
		Id:   pinkFloydClusterID,
		Name: clusterPinkFloyd.GetName(),
	}

	testCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.TestScopeCheckerCoreFromFullScopeMap(s.T(), testScopeMap))

	testCases := []struct {
		name             string
		pagination       *v1.Pagination
		expectedClusters []*v1.ScopeObject
	}{
		{
			name: "No offset and a limit restricts to a list of appropriate size",
			pagination: &v1.Pagination{
				Limit:  1,
				Offset: 0,
				SortOption: &v1.SortOption{
					Field:    "Cluster",
					Reversed: false,
				},
			},
			expectedClusters: []*v1.ScopeObject{pinkFloydClusterResponse},
		},
		{
			name: "Offset and no limit restricts to a list of appropriate size starting with expected value",
			pagination: &v1.Pagination{
				Limit:  0,
				Offset: 1,
				SortOption: &v1.SortOption{
					Field:    "Cluster",
					Reversed: false,
				},
			},
			expectedClusters: []*v1.ScopeObject{queenClusterResponse},
		},
		{
			name: "Sort options without offset nor limit return the expected results",
			pagination: &v1.Pagination{
				Limit:  0,
				Offset: 0,
				SortOption: &v1.SortOption{
					Field:    "Cluster",
					Reversed: false,
				},
			},
			expectedClusters: []*v1.ScopeObject{pinkFloydClusterResponse, queenClusterResponse},
		},
		{
			name: "Reversed sort without offset nor limit return the expected results",
			pagination: &v1.Pagination{
				Limit:  0,
				Offset: 0,
				SortOption: &v1.SortOption{
					Field:    "Cluster",
					Reversed: true,
				},
			},
			expectedClusters: []*v1.ScopeObject{queenClusterResponse, pinkFloydClusterResponse},
		},
	}

	for _, c := range testCases {
		s.Run(c.name, func() {
			clusterResponse, err := s.tester.service.GetClustersForPermissions(testCtx, &v1.GetClustersForPermissionsRequest{
				Pagination:  c.pagination,
				Permissions: []string{},
			})
			s.NoError(err)
			protoassert.SlicesEqual(s.T(), clusterResponse.GetClusters(), c.expectedClusters)
		})
	}
}

func (s *serviceImplOtherTestSuite) TestGetNamespacesForClusterAndPermissions() {
	queenClusterID := s.tester.clusterNameToIDMap[clusterQueen.GetName()]
	testResourceScope1 := getTestResourceScopeSingleNamespace(
		queenClusterID,
		namespaceInnuendoInClusterQueen.GetName())
	pinkFloydClusterID := s.tester.clusterNameToIDMap[clusterPinkFloyd.GetName()]
	testResourceScope2 := getTestResourceScopeSingleNamespace(
		pinkFloydClusterID,
		namespaceTheWallInClusterPinkFloyd.GetName())
	testResourceScope3 := getTestResourceScopeSingleNamespace(
		queenClusterID,
		namespaceQueenInClusterQueen.GetName())
	testScopeMap := sac.TestScopeMap{
		storage.Access_READ_ACCESS: map[permissions.Resource]*sac.TestResourceScope{
			resources.Integration.GetResource(): {
				Included: true,
			},
			resources.Node.GetResource():         testResourceScope1,
			resources.Deployment.GetResource():   testResourceScope1,
			resources.NetworkGraph.GetResource(): testResourceScope2,
			resources.Image.GetResource():        testResourceScope3,
		},
	}

	queenQueenNamespaceResponse := &v1.ScopeObject{
		Id:   getNamespaceID(namespaceQueenInClusterQueen.GetName()),
		Name: namespaceQueenInClusterQueen.GetName(),
	}

	queenInnuendoNamespaceResponse := &v1.ScopeObject{
		Id:   getNamespaceID(namespaceInnuendoInClusterQueen.GetName()),
		Name: namespaceInnuendoInClusterQueen.GetName(),
	}

	pinkFloydTheWallNamespaceResponse := &v1.ScopeObject{
		Id:   getNamespaceID(namespaceTheWallInClusterPinkFloyd.GetName()),
		Name: namespaceTheWallInClusterPinkFloyd.GetName(),
	}

	testCases := []struct {
		name               string
		testedClusterID    string
		testedPermissions  []string
		expectedNamespaces []*v1.ScopeObject
	}{
		{
			name:               "Global permission (Not Granted) gets no namespace data.",
			testedClusterID:    s.tester.clusterNameToIDMap[clusterQueen.GetName()],
			testedPermissions:  []string{rolePermission},
			expectedNamespaces: []*v1.ScopeObject{},
		},
		{
			name:               "Global permission (Granted) gets no namespace data.",
			testedClusterID:    s.tester.clusterNameToIDMap[clusterQueen.GetName()],
			testedPermissions:  []string{integrationPermission},
			expectedNamespaces: []*v1.ScopeObject{},
		},
		{
			name:               "Not granted Cluster scoped permission gets no namespace data.",
			testedClusterID:    s.tester.clusterNameToIDMap[clusterQueen.GetName()],
			testedPermissions:  []string{clusterPermission},
			expectedNamespaces: []*v1.ScopeObject{},
		},
		{
			name:               "Granted Cluster scoped permission gets no namespace data.",
			testedClusterID:    s.tester.clusterNameToIDMap[clusterQueen.GetName()],
			testedPermissions:  []string{nodePermission},
			expectedNamespaces: []*v1.ScopeObject{},
		},
		{
			name:               "Not granted Namespace scoped permission gets no namespace data.",
			testedClusterID:    s.tester.clusterNameToIDMap[clusterQueen.GetName()],
			testedPermissions:  []string{namespacePermission},
			expectedNamespaces: []*v1.ScopeObject{},
		},
		{
			name:               "Granted Namespace scoped permission gets only namespace data for namespaces in cluster and permission scope.",
			testedClusterID:    s.tester.clusterNameToIDMap[clusterQueen.GetName()],
			testedPermissions:  []string{deploymentPermission},
			expectedNamespaces: []*v1.ScopeObject{queenInnuendoNamespaceResponse},
		},
		{
			name:               "Granted Namespace scoped permission gets only namespace data for namespaces in cluster and permission scope (other permission).",
			testedClusterID:    s.tester.clusterNameToIDMap[clusterPinkFloyd.GetName()],
			testedPermissions:  []string{networkGraphPermission},
			expectedNamespaces: []*v1.ScopeObject{pinkFloydTheWallNamespaceResponse},
		},
		{
			name:               "Multiple not granted Namespace scoped permissions get no namespace data.",
			testedClusterID:    s.tester.clusterNameToIDMap[clusterQueen.GetName()],
			testedPermissions:  []string{namespacePermission, deploymentExtensionPermission},
			expectedNamespaces: []*v1.ScopeObject{},
		},
		{
			name:               "Multiple Namespace scoped permissions get only namespace data for namespaces in granted permission scopes.",
			testedClusterID:    s.tester.clusterNameToIDMap[clusterQueen.GetName()],
			testedPermissions:  []string{namespacePermission, deploymentPermission},
			expectedNamespaces: []*v1.ScopeObject{queenInnuendoNamespaceResponse},
		},
		{
			name:               "empty permission list get namespace data for all namespaces in scope of target cluster and granted namespace permissions.",
			testedClusterID:    s.tester.clusterNameToIDMap[clusterQueen.GetName()],
			testedPermissions:  []string{},
			expectedNamespaces: []*v1.ScopeObject{queenQueenNamespaceResponse, queenInnuendoNamespaceResponse},
		},
	}

	testCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.TestScopeCheckerCoreFromFullScopeMap(s.T(), testScopeMap))

	for _, c := range testCases {
		s.Run(c.name, func() {
			request := &v1.GetNamespaceForClusterAndPermissionsRequest{
				ClusterId:   c.testedClusterID,
				Permissions: c.testedPermissions,
			}
			namespaceResponse, err := s.tester.service.GetNamespacesForClusterAndPermissions(testCtx, request)
			s.NoError(err)
			protoassert.ElementsMatch(s.T(), namespaceResponse.GetNamespaces(), c.expectedNamespaces)
		})
	}
}

func getTestResourceScopeSingleNamespace(clusterID string, namespace string) *sac.TestResourceScope {
	return &sac.TestResourceScope{
		Clusters: map[string]*sac.TestClusterScope{
			clusterID: {
				Namespaces: []string{namespace},
				Included:   false,
			},
		},
		Included: false,
	}
}
