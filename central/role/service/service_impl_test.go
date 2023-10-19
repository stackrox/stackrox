//go:build sql_integration

package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/datastore"
	roleDatastore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/central/role/sachelper"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn/basic"
	"github.com/stackrox/rox/pkg/labels"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

////////////////////////////////////////////////////////////////////////////////
// Cluster and namespace configuration                                        //
//                                                                            //
// Queen       { genre: rock }                                                //
//   Queen        { released: 1973 }                                          //
//   Jazz         { released: 1978 }                                          //
//   Innuendo     { released: 1991 }                                          //
//                                                                            //
// Pink Floyd  { genre: psychedelic_rock }                                    //
//   The Wall     { released: 1979 }                                          //
//                                                                            //
// Deep Purple { genre: hard_rock }                                           //
//   Machine Head { released: 1972 }                                          //
//                                                                            //

var (
	clusterQueen = &storage.Cluster{
		Id:   "band.queen",
		Name: "Queen",
		Labels: map[string]string{
			"genre": "rock",
		},
	}

	clusterPinkFloyd = &storage.Cluster{
		Id:   "band.pinkfloyd",
		Name: "Pink Floyd",
		Labels: map[string]string{
			"genre": "psychedelic_rock",
		},
	}

	clusterDeepPurple = &storage.Cluster{
		Id:   "band.deeppurple",
		Name: "Deep Purple",
		Labels: map[string]string{
			"genre": "hard_rock",
		},
	}
)

var clusters = []*storage.Cluster{
	clusterQueen,
	clusterPinkFloyd,
	clusterDeepPurple,
}

var clustersForSAC = []effectiveaccessscope.ClusterForSAC{
	effectiveaccessscope.StorageClusterToClusterForSAC(clusterQueen),
	effectiveaccessscope.StorageClusterToClusterForSAC(clusterPinkFloyd),
	effectiveaccessscope.StorageClusterToClusterForSAC(clusterDeepPurple),
}

var (
	namespaceQueenInClusterQueen = &storage.NamespaceMetadata{
		Id:          "album.queen",
		Name:        "Queen",
		ClusterId:   "band.queen",
		ClusterName: "Queen",
		Labels: map[string]string{
			"released": "1973",
		},
	}

	namespaceJazzInClusterQueen = &storage.NamespaceMetadata{
		Id:          "album.jazz",
		Name:        "Jazz",
		ClusterId:   "band.queen",
		ClusterName: "Queen",
		Labels: map[string]string{
			"released": "1978",
		},
	}

	namespaceInnuendoInClusterQueen = &storage.NamespaceMetadata{
		Id:          "album.innuendo",
		Name:        "Innuendo",
		ClusterId:   "band.queen",
		ClusterName: "Queen",
		Labels: map[string]string{
			"released": "1991",
		},
	}

	namespaceTheWallInClusterPinkFloyd = &storage.NamespaceMetadata{
		Id:          "album.thewall",
		Name:        "The Wall",
		ClusterId:   "band.pinkfloyd",
		ClusterName: "Pink Floyd",
		Labels: map[string]string{
			"released": "1979",
		},
	}

	namespaceMachineHeadInClusterDeepPurple = &storage.NamespaceMetadata{
		Id:          "album.machinehead",
		Name:        "Machine Head",
		ClusterId:   "band.deeppurple",
		ClusterName: "Deep Purple",
		Labels: map[string]string{
			"released": "1972",
		},
	}
)

var namespaces = []*storage.NamespaceMetadata{
	// Queen
	namespaceQueenInClusterQueen,
	namespaceJazzInClusterQueen,
	namespaceInnuendoInClusterQueen,
	// Pink Floyd
	namespaceTheWallInClusterPinkFloyd,
	// Deep Purple
	namespaceMachineHeadInClusterDeepPurple,
}

var namespacesForSAC = []effectiveaccessscope.NamespaceForSAC{
	// Queen
	effectiveaccessscope.StorageNamespaceToNamespaceForSAC(namespaceQueenInClusterQueen),
	effectiveaccessscope.StorageNamespaceToNamespaceForSAC(namespaceJazzInClusterQueen),
	effectiveaccessscope.StorageNamespaceToNamespaceForSAC(namespaceInnuendoInClusterQueen),
	// Pink Floyd
	effectiveaccessscope.StorageNamespaceToNamespaceForSAC(namespaceTheWallInClusterPinkFloyd),
	// Deep Purple
	effectiveaccessscope.StorageNamespaceToNamespaceForSAC(namespaceMachineHeadInClusterDeepPurple),
}

////////////////////////////////////////////////////////////////////////////////
// Access scope rules and expected effective access scopes                    //
//                                                                            //
// Valid rules:                                                               //
//   `namespace: "Queen::Jazz" OR cluster.labels: genre in (psychedelic_rock)`//
//     => { "Queen::Jazz", "Pink Floyd::*" }                                  //
//                                                                            //
// Invalid rules:                                                             //
//   `namespace: "::Jazz"` => { }                                             //
//                                                                            //

var validRules = &storage.SimpleAccessScope_Rules{
	IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
		{
			ClusterName:   "Queen",
			NamespaceName: "Jazz",
		},
	},
	ClusterLabelSelectors: labels.LabelSelectors("genre", storage.SetBasedLabelSelector_IN, []string{"psychedelic_rock"}),
}

var validExpectedHigh = &storage.EffectiveAccessScope{
	Clusters: []*storage.EffectiveAccessScope_Cluster{
		{
			Id:    "band.deeppurple",
			Name:  "Deep Purple",
			State: storage.EffectiveAccessScope_EXCLUDED,
			Labels: map[string]string{
				"genre": "hard_rock",
			},
			Namespaces: []*storage.EffectiveAccessScope_Namespace{
				{
					Id:    "album.machinehead",
					Name:  "Machine Head",
					State: storage.EffectiveAccessScope_EXCLUDED,
					Labels: map[string]string{
						"released": "1972",
					},
				},
			},
		},
		{
			Id:    "band.pinkfloyd",
			Name:  "Pink Floyd",
			State: storage.EffectiveAccessScope_INCLUDED,
			Labels: map[string]string{
				"genre": "psychedelic_rock",
			},
			Namespaces: []*storage.EffectiveAccessScope_Namespace{
				{
					Id:    "album.thewall",
					Name:  "The Wall",
					State: storage.EffectiveAccessScope_INCLUDED,
					Labels: map[string]string{
						"released": "1979",
					},
				},
			},
		},
		{
			Id:    "band.queen",
			Name:  "Queen",
			State: storage.EffectiveAccessScope_PARTIAL,
			Labels: map[string]string{
				"genre": "rock",
			},
			Namespaces: []*storage.EffectiveAccessScope_Namespace{
				{
					Id:    "album.innuendo",
					Name:  "Innuendo",
					State: storage.EffectiveAccessScope_EXCLUDED,
					Labels: map[string]string{
						"released": "1991",
					},
				},
				{
					Id:    "album.jazz",
					Name:  "Jazz",
					State: storage.EffectiveAccessScope_INCLUDED,
					Labels: map[string]string{
						"released": "1978",
					},
				},
				{
					Id:    "album.queen",
					Name:  "Queen",
					State: storage.EffectiveAccessScope_EXCLUDED,
					Labels: map[string]string{
						"released": "1973",
					},
				},
			},
		},
	},
}

var validExpectedStandard = &storage.EffectiveAccessScope{
	Clusters: []*storage.EffectiveAccessScope_Cluster{
		{
			Id:    "band.deeppurple",
			Name:  "Deep Purple",
			State: storage.EffectiveAccessScope_EXCLUDED,
			Namespaces: []*storage.EffectiveAccessScope_Namespace{
				{
					Id:    "album.machinehead",
					Name:  "Machine Head",
					State: storage.EffectiveAccessScope_EXCLUDED,
				},
			},
		},
		{
			Id:    "band.pinkfloyd",
			Name:  "Pink Floyd",
			State: storage.EffectiveAccessScope_INCLUDED,
			Namespaces: []*storage.EffectiveAccessScope_Namespace{
				{
					Id:    "album.thewall",
					Name:  "The Wall",
					State: storage.EffectiveAccessScope_INCLUDED,
				},
			},
		},
		{
			Id:    "band.queen",
			Name:  "Queen",
			State: storage.EffectiveAccessScope_PARTIAL,
			Namespaces: []*storage.EffectiveAccessScope_Namespace{
				{
					Id:    "album.innuendo",
					Name:  "Innuendo",
					State: storage.EffectiveAccessScope_EXCLUDED,
				},
				{
					Id:    "album.jazz",
					Name:  "Jazz",
					State: storage.EffectiveAccessScope_INCLUDED,
				},
				{
					Id:    "album.queen",
					Name:  "Queen",
					State: storage.EffectiveAccessScope_EXCLUDED,
				},
			},
		},
	},
}

var validExpectedMinimal = &storage.EffectiveAccessScope{
	Clusters: []*storage.EffectiveAccessScope_Cluster{
		{
			Id:    "band.pinkfloyd",
			State: storage.EffectiveAccessScope_INCLUDED,
		},
		{
			Id:    "band.queen",
			State: storage.EffectiveAccessScope_PARTIAL,
			Namespaces: []*storage.EffectiveAccessScope_Namespace{
				{
					Id:    "album.jazz",
					State: storage.EffectiveAccessScope_INCLUDED,
				},
			},
		},
	},
}

var invalidRules = &storage.SimpleAccessScope_Rules{
	IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
		{
			NamespaceName: "Jazz",
		},
	},
}

var invalidExpectedHigh = &storage.EffectiveAccessScope{
	Clusters: []*storage.EffectiveAccessScope_Cluster{
		{
			Id:    "band.deeppurple",
			Name:  "Deep Purple",
			State: storage.EffectiveAccessScope_EXCLUDED,
			Labels: map[string]string{
				"genre": "hard_rock",
			},
			Namespaces: []*storage.EffectiveAccessScope_Namespace{
				{
					Id:    "album.machinehead",
					Name:  "Machine Head",
					State: storage.EffectiveAccessScope_EXCLUDED,
					Labels: map[string]string{
						"released": "1972",
					},
				},
			},
		},
		{
			Id:    "band.pinkfloyd",
			Name:  "Pink Floyd",
			State: storage.EffectiveAccessScope_EXCLUDED,
			Labels: map[string]string{
				"genre": "psychedelic_rock",
			},
			Namespaces: []*storage.EffectiveAccessScope_Namespace{
				{
					Id:    "album.thewall",
					Name:  "The Wall",
					State: storage.EffectiveAccessScope_EXCLUDED,
					Labels: map[string]string{
						"released": "1979",
					},
				},
			},
		},
		{
			Id:    "band.queen",
			Name:  "Queen",
			State: storage.EffectiveAccessScope_EXCLUDED,
			Labels: map[string]string{
				"genre": "rock",
			},
			Namespaces: []*storage.EffectiveAccessScope_Namespace{
				{
					Id:    "album.innuendo",
					Name:  "Innuendo",
					State: storage.EffectiveAccessScope_EXCLUDED,
					Labels: map[string]string{
						"released": "1991",
					},
				},
				{
					Id:    "album.jazz",
					Name:  "Jazz",
					State: storage.EffectiveAccessScope_EXCLUDED,
					Labels: map[string]string{
						"released": "1978",
					},
				},
				{
					Id:    "album.queen",
					Name:  "Queen",
					State: storage.EffectiveAccessScope_EXCLUDED,
					Labels: map[string]string{
						"released": "1973",
					},
				},
			},
		},
	},
}

var invalidExpectedStandard = &storage.EffectiveAccessScope{
	Clusters: []*storage.EffectiveAccessScope_Cluster{
		{
			Id:    "band.deeppurple",
			Name:  "Deep Purple",
			State: storage.EffectiveAccessScope_EXCLUDED,
			Namespaces: []*storage.EffectiveAccessScope_Namespace{
				{
					Id:    "album.machinehead",
					Name:  "Machine Head",
					State: storage.EffectiveAccessScope_EXCLUDED,
				},
			},
		},
		{
			Id:    "band.pinkfloyd",
			Name:  "Pink Floyd",
			State: storage.EffectiveAccessScope_EXCLUDED,
			Namespaces: []*storage.EffectiveAccessScope_Namespace{
				{
					Id:    "album.thewall",
					Name:  "The Wall",
					State: storage.EffectiveAccessScope_EXCLUDED,
				},
			},
		},
		{
			Id:    "band.queen",
			Name:  "Queen",
			State: storage.EffectiveAccessScope_EXCLUDED,
			Namespaces: []*storage.EffectiveAccessScope_Namespace{
				{
					Id:    "album.innuendo",
					Name:  "Innuendo",
					State: storage.EffectiveAccessScope_EXCLUDED,
				},
				{
					Id:    "album.jazz",
					Name:  "Jazz",
					State: storage.EffectiveAccessScope_EXCLUDED,
				},
				{
					Id:    "album.queen",
					Name:  "Queen",
					State: storage.EffectiveAccessScope_EXCLUDED,
				},
			},
		},
	},
}

var invalidExpectedMinimal = &storage.EffectiveAccessScope{}

////////////////////////////////////////////////////////////////////////////////
// Tests                                                                      //
//                                                                            //

func TestEffectiveAccessScopeForSimpleAccessScope(t *testing.T) {
	type testCase struct {
		desc             string
		rules            *storage.SimpleAccessScope_Rules
		expectedHigh     *storage.EffectiveAccessScope
		expectedStandard *storage.EffectiveAccessScope
		expectedMinimal  *storage.EffectiveAccessScope
	}

	testCases := []testCase{
		{
			"valid access scope rules",
			validRules,
			validExpectedHigh,
			validExpectedStandard,
			validExpectedMinimal,
		},
		{
			"invalid access scope rules",
			invalidRules,
			invalidExpectedHigh,
			invalidExpectedStandard,
			invalidExpectedMinimal,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc+"detail: HIGH", func(t *testing.T) {
			resHigh, err := effectiveAccessScopeForSimpleAccessScope(tc.rules, clustersForSAC, namespacesForSAC, v1.ComputeEffectiveAccessScopeRequest_HIGH)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedHigh, resHigh)
		})
		t.Run(tc.desc+"detail: STANDARD", func(t *testing.T) {
			resStandard, err := effectiveAccessScopeForSimpleAccessScope(tc.rules, clustersForSAC, namespacesForSAC, v1.ComputeEffectiveAccessScopeRequest_STANDARD)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedStandard, resStandard)
		})
		t.Run(tc.desc+"detail: MINIMAL", func(t *testing.T) {
			resMinimal, err := effectiveAccessScopeForSimpleAccessScope(tc.rules, clustersForSAC, namespacesForSAC, v1.ComputeEffectiveAccessScopeRequest_MINIMAL)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedMinimal, resMinimal)
		})
		t.Run(tc.desc+"unknown detail maps to STANDARD", func(t *testing.T) {
			resUnknown, err := effectiveAccessScopeForSimpleAccessScope(tc.rules, clustersForSAC, namespacesForSAC, 42)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedStandard, resUnknown)
		})
	}
}

func TestServiceImplWithDB(t *testing.T) {
	suite.Run(t, new(serviceImplTestSuite))
}

type serviceImplTestSuite struct {
	suite.Suite

	postgres *pgtest.TestPostgres
	service  *serviceImpl

	storedClusterIDs   []string
	storedNamespaceIDs []string
	clusterNameToIDMap map[string]string

	storedPermissionSetIDs []string
	storedAccessScopeIDs   []string
	storedRoleNames        []string
}

func (s *serviceImplTestSuite) SetupSuite() {
	var err error
	s.postgres = pgtest.ForT(s.T())
	s.Require().NotNil(s.postgres)
	roleStore, err := roleDatastore.GetTestPostgresDataStore(s.T(), s.postgres.DB)
	s.Require().NoError(err)
	clusterStore, err := clusterDataStore.GetTestPostgresDataStore(s.T(), s.postgres.DB)
	s.Require().NoError(err)
	namespaceStore, err := namespaceDataStore.GetTestPostgresDataStore(s.T(), s.postgres.DB)
	s.Require().NoError(err)

	s.service = &serviceImpl{
		roleDataStore:      roleStore,
		clusterDataStore:   clusterStore,
		namespaceDataStore: namespaceStore,
		clusterSACHelper:   sachelper.NewClusterSacHelper(clusterStore),
		namespaceSACHelper: sachelper.NewClusterNamespaceSacHelper(clusterStore, namespaceStore),
	}
}

func (s *serviceImplTestSuite) TearDownSuite() {
	s.postgres.Teardown(s.T())
}

func (s *serviceImplTestSuite) SetupTest() {
	s.storedAccessScopeIDs = make([]string, 0)
	s.storedPermissionSetIDs = make([]string, 0)
	s.storedRoleNames = make([]string, 0)

	s.storedClusterIDs = make([]string, 0)
	s.storedNamespaceIDs = make([]string, 0)
	s.clusterNameToIDMap = make(map[string]string, 0)

	writeCtx := sac.WithAllAccess(context.Background())

	for _, cluster := range clusters {
		clusterToAdd := cluster.Clone()
		clusterToAdd.Id = ""
		clusterToAdd.MainImage = "quay.io/rhacs-eng/main:latest"
		id, err := s.service.clusterDataStore.AddCluster(writeCtx, clusterToAdd)
		s.Require().NoError(err)
		s.clusterNameToIDMap[clusterToAdd.GetName()] = id
		s.storedClusterIDs = append(s.storedClusterIDs, id)
	}

	for _, namespace := range namespaces {
		ns := namespace.Clone()
		ns.Id = getNamespaceID(ns.GetName())
		ns.ClusterId = s.clusterNameToIDMap[ns.GetClusterName()]
		s.Require().NoError(s.service.namespaceDataStore.AddNamespace(writeCtx, ns))
		s.storedNamespaceIDs = append(s.storedNamespaceIDs, ns.GetId())
	}
}

func (s *serviceImplTestSuite) TearDownTest() {
	writeCtx := sac.WithAllAccess(context.Background())
	for _, clusterID := range s.storedClusterIDs {
		doneSignal := concurrency.NewSignal()
		s.Require().NoError(s.service.clusterDataStore.RemoveCluster(writeCtx, clusterID, &doneSignal))
		require.Eventually(s.T(),
			func() bool { return doneSignal.IsDone() },
			5*time.Second,
			10*time.Millisecond,
		)
	}
	s.storedClusterIDs = s.storedClusterIDs[:0]
	for _, namespaceID := range s.storedNamespaceIDs {
		s.Require().NoError(s.service.namespaceDataStore.RemoveNamespace(writeCtx, namespaceID))
	}
	for _, roleName := range s.storedRoleNames {
		s.deleteRole(roleName)
	}
	for _, permissionSetID := range s.storedPermissionSetIDs {
		s.deletePermissionSet(permissionSetID)
	}
	for _, accessScopeID := range s.storedAccessScopeIDs {
		s.deleteAccessScope(accessScopeID)
	}
}

const (
	namespaceUUIDNamespace = "namespace"

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

func getNamespaceID(namespaceName string) string {
	return uuid.NewV5FromNonUUIDs(namespaceUUIDNamespace, namespaceName).String()
}

func (s *serviceImplTestSuite) TestGetClustersForPermissions() {
	deepPurpleClusterID := s.clusterNameToIDMap[clusterDeepPurple.GetName()]
	queenClusterID := s.clusterNameToIDMap[clusterQueen.GetName()]
	testResourceScope1 := getTestResourceScopeSingleNamespace(
		queenClusterID,
		namespaceInnuendoInClusterQueen.GetName())
	pinkFloydClusterID := s.clusterNameToIDMap[clusterPinkFloyd.GetName()]
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
			clusterResponse, err := s.service.GetClustersForPermissions(c.context, &v1.GetClustersForPermissionsRequest{
				Pagination:  nil,
				Permissions: c.testedPermissions,
			})
			s.NoError(err)
			s.ElementsMatch(clusterResponse.GetClusters(), c.expectedClusters)
		})
	}
}

func (s *serviceImplTestSuite) TestGetClustersForPermissionsPagination() {
	queenClusterID := s.clusterNameToIDMap[clusterQueen.GetName()]
	testResourceScope1 := getTestResourceScopeSingleNamespace(
		queenClusterID,
		namespaceInnuendoInClusterQueen.GetName())
	pinkFloydClusterID := s.clusterNameToIDMap[clusterPinkFloyd.GetName()]
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
			clusterResponse, err := s.service.GetClustersForPermissions(testCtx, &v1.GetClustersForPermissionsRequest{
				Pagination:  c.pagination,
				Permissions: []string{},
			})
			s.NoError(err)
			s.Equal(clusterResponse.GetClusters(), c.expectedClusters)
		})
	}
}

func (s *serviceImplTestSuite) TestGetNamespacesForClusterAndPermissions() {
	queenClusterID := s.clusterNameToIDMap[clusterQueen.GetName()]
	testResourceScope1 := getTestResourceScopeSingleNamespace(
		queenClusterID,
		namespaceInnuendoInClusterQueen.GetName())
	pinkFloydClusterID := s.clusterNameToIDMap[clusterPinkFloyd.GetName()]
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
			testedClusterID:    s.clusterNameToIDMap[clusterQueen.GetName()],
			testedPermissions:  []string{rolePermission},
			expectedNamespaces: []*v1.ScopeObject{},
		},
		{
			name:               "Global permission (Granted) gets no namespace data.",
			testedClusterID:    s.clusterNameToIDMap[clusterQueen.GetName()],
			testedPermissions:  []string{integrationPermission},
			expectedNamespaces: []*v1.ScopeObject{},
		},
		{
			name:               "Not granted Cluster scoped permission gets no namespace data.",
			testedClusterID:    s.clusterNameToIDMap[clusterQueen.GetName()],
			testedPermissions:  []string{clusterPermission},
			expectedNamespaces: []*v1.ScopeObject{},
		},
		{
			name:               "Granted Cluster scoped permission gets no namespace data.",
			testedClusterID:    s.clusterNameToIDMap[clusterQueen.GetName()],
			testedPermissions:  []string{nodePermission},
			expectedNamespaces: []*v1.ScopeObject{},
		},
		{
			name:               "Not granted Namespace scoped permission gets no namespace data.",
			testedClusterID:    s.clusterNameToIDMap[clusterQueen.GetName()],
			testedPermissions:  []string{namespacePermission},
			expectedNamespaces: []*v1.ScopeObject{},
		},
		{
			name:               "Granted Namespace scoped permission gets only namespace data for namespaces in cluster and permission scope.",
			testedClusterID:    s.clusterNameToIDMap[clusterQueen.GetName()],
			testedPermissions:  []string{deploymentPermission},
			expectedNamespaces: []*v1.ScopeObject{queenInnuendoNamespaceResponse},
		},
		{
			name:               "Granted Namespace scoped permission gets only namespace data for namespaces in cluster and permission scope (other permission).",
			testedClusterID:    s.clusterNameToIDMap[clusterPinkFloyd.GetName()],
			testedPermissions:  []string{networkGraphPermission},
			expectedNamespaces: []*v1.ScopeObject{pinkFloydTheWallNamespaceResponse},
		},
		{
			name:               "Multiple not granted Namespace scoped permissions get no namespace data.",
			testedClusterID:    s.clusterNameToIDMap[clusterQueen.GetName()],
			testedPermissions:  []string{namespacePermission, deploymentExtensionPermission},
			expectedNamespaces: []*v1.ScopeObject{},
		},
		{
			name:               "Multiple Namespace scoped permissions get only namespace data for namespaces in granted permission scopes.",
			testedClusterID:    s.clusterNameToIDMap[clusterQueen.GetName()],
			testedPermissions:  []string{namespacePermission, deploymentPermission},
			expectedNamespaces: []*v1.ScopeObject{queenInnuendoNamespaceResponse},
		},
		{
			name:               "empty permission list get namespace data for all namespaces in scope of target cluster and granted namespace permissions.",
			testedClusterID:    s.clusterNameToIDMap[clusterQueen.GetName()],
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
			namespaceResponse, err := s.service.GetNamespacesForClusterAndPermissions(testCtx, request)
			s.NoError(err)
			s.ElementsMatch(namespaceResponse.GetNamespaces(), c.expectedNamespaces)
		})
	}
}

func getValidRole(name string) *storage.Role {
	permissionSetID := accesscontrol.DefaultPermissionSetIDs[accesscontrol.Admin]
	scopeID := accesscontrol.DefaultAccessScopeIDs[accesscontrol.UnrestrictedAccessScope]
	return &storage.Role{
		Name:            name,
		Description:     fmt.Sprintf("Test role for %s", name),
		PermissionSetId: permissionSetID,
		AccessScopeId:   scopeID,
		Traits:          nil,
	}
}

func (s *serviceImplTestSuite) TestCreateRoleValidAccessScopeID() {
	ctx := sac.WithAllAccess(context.Background())
	roleName := "TestCreateRoleValidAccessScopeID"

	ps := s.createPermissionSet(roleName)
	scope := s.createAccessScope(roleName)

	role := getValidRole(roleName)
	role.PermissionSetId = ps.GetId()
	role.AccessScopeId = scope.GetId()
	createRoleRequest := &v1.CreateRoleRequest{
		Name: roleName,
		Role: role,
	}
	_, err := s.service.CreateRole(ctx, createRoleRequest)
	s.NoError(err)
	s.storedRoleNames = append(s.storedRoleNames, role.GetName())
}

func (s *serviceImplTestSuite) TestCreateRoleEmptyAccessScopeID() {
	ctx := sac.WithAllAccess(context.Background())
	roleName := "TestCreateRoleEmptyAccessScopeID"

	ps := s.createPermissionSet(roleName)

	role := getValidRole(roleName)
	role.PermissionSetId = ps.GetId()
	role.AccessScopeId = ""
	createRoleRequest := &v1.CreateRoleRequest{
		Name: roleName,
		Role: role,
	}
	_, err := s.service.CreateRole(ctx, createRoleRequest)
	s.ErrorContains(err, "role access_scope_id field must be set")
}

func (s *serviceImplTestSuite) TestUpdateExistingRoleValidAccessScopeID() {
	ctx := sac.WithAllAccess(context.Background())
	role := s.createRole("TestUpdateExistingRoleValidAccessScopeID")
	newScope := s.createAccessScope("new scope")
	role.AccessScopeId = newScope.GetId()
	_, err := s.service.UpdateRole(ctx, role)
	s.NoError(err)
}

func (s *serviceImplTestSuite) TestUpdateExistingRoleEmptyAccessScopeID() {
	ctx := sac.WithAllAccess(context.Background())
	roleName := "TestUpdateExistingRoleEmptyAccessScopeID"
	role := s.createRole(roleName)
	role.AccessScopeId = ""
	_, err := s.service.UpdateRole(ctx, role)
	s.ErrorContains(err, "role access_scope_id field must be set")
}

func (s *serviceImplTestSuite) TestUpdateMissingRoleValidAccessScopeID() {
	ctx := sac.WithAllAccess(context.Background())
	roleName := "TestUpdateMissingRoleValidAccessScopeID"
	ps := s.createPermissionSet(roleName)
	scope := s.createAccessScope(roleName)
	role := getValidRole(roleName)
	role.PermissionSetId = ps.GetId()
	role.AccessScopeId = scope.GetId()
	_, err := s.service.UpdateRole(ctx, role)
	s.ErrorIs(err, errox.NotFound)
}

func (s *serviceImplTestSuite) TestUpdateMissingRoleEmptyAccessScopeID() {
	ctx := sac.WithAllAccess(context.Background())
	roleName := "TestUpdateMissingRoleEmptyAccessScopeID"
	ps := s.createPermissionSet(roleName)
	role := getValidRole(roleName)
	role.PermissionSetId = ps.GetId()
	role.AccessScopeId = ""
	_, err := s.service.UpdateRole(ctx, role)
	s.ErrorContains(err, "role access_scope_id field must be set")
}

func (s *serviceImplTestSuite) createAccessScope(name string) *storage.SimpleAccessScope {
	ctx := sac.WithAllAccess(context.Background())
	scope := &storage.SimpleAccessScope{
		Name:        name,
		Description: fmt.Sprintf("Test access scope for %s", name),
		Rules: &storage.SimpleAccessScope_Rules{
			IncludedClusters: []string{"test"},
		},
		Traits: nil,
	}
	postedScope, postErr := s.service.PostSimpleAccessScope(ctx, scope)
	s.Require().NoError(postErr)
	s.storedAccessScopeIDs = append(s.storedAccessScopeIDs, postedScope.GetId())
	return postedScope
}

func (s *serviceImplTestSuite) deleteAccessScope(id string) {
	ctx := sac.WithAllAccess(context.Background())
	request := &v1.ResourceByID{
		Id: id,
	}
	_, deleteErr := s.service.DeleteSimpleAccessScope(ctx, request)
	s.Require().NoError(deleteErr)
}

func (s *serviceImplTestSuite) createPermissionSet(name string) *storage.PermissionSet {
	ctx := sac.WithAllAccess(context.Background())
	permissionSet := &storage.PermissionSet{
		Name:             name,
		Description:      fmt.Sprintf("Test permission set for %s", name),
		ResourceToAccess: nil,
		Traits:           nil,
	}
	ps, postErr := s.service.PostPermissionSet(ctx, permissionSet)
	s.Require().NoError(postErr)
	s.storedPermissionSetIDs = append(s.storedPermissionSetIDs, ps.GetId())
	return ps
}

func (s *serviceImplTestSuite) deletePermissionSet(id string) {
	ctx := sac.WithAllAccess(context.Background())
	request := &v1.ResourceByID{
		Id: id,
	}
	_, deleteErr := s.service.DeletePermissionSet(ctx, request)
	s.Require().NoError(deleteErr)
}

func (s *serviceImplTestSuite) createRole(roleName string) *storage.Role {
	ctx := sac.WithAllAccess(context.Background())

	ps := s.createPermissionSet(roleName)
	scope := s.createAccessScope(roleName)

	createRoleRequest := &v1.CreateRoleRequest{
		Name: roleName,
		Role: getValidRole(roleName),
	}
	createRoleRequest.Role.PermissionSetId = ps.GetId()
	createRoleRequest.Role.AccessScopeId = scope.GetId()

	_, createErr := s.service.CreateRole(ctx, createRoleRequest)
	s.Require().NoError(createErr)
	s.storedRoleNames = append(s.storedRoleNames, roleName)

	readRoleRequest := &v1.ResourceByID{
		Id: roleName,
	}
	role, readErr := s.service.GetRole(ctx, readRoleRequest)
	s.Require().NoError(readErr)
	return role
}

func (s *serviceImplTestSuite) deleteRole(name string) {
	ctx := sac.WithAllAccess(context.Background())
	request := &v1.ResourceByID{
		Id: name,
	}
	_, deleteErr := s.service.DeleteRole(ctx, request)
	s.Require().NoError(deleteErr)
}

func TestGetMyPermissions(t *testing.T) {
	suite.Run(t, new(roleServiceGetMyPermissionsTestSuite))
}

const (
	getMyPermissionsServiceName = "/v1.RoleService/GetMyPermissions"
)

type roleServiceGetMyPermissionsTestSuite struct {
	suite.Suite

	svc *serviceImpl

	withAdminRoleCtx context.Context
	withNoneRoleCtx  context.Context
	withNoAccessCtx  context.Context
	withNoRoleCtx    context.Context
	anonymousCtx     context.Context
}

func (s *roleServiceGetMyPermissionsTestSuite) SetupTest() {
	s.svc = &serviceImpl{}

	authProvider, err := authproviders.NewProvider(
		authproviders.WithEnabled(true),
		authproviders.WithID(uuid.NewDummy().String()),
		authproviders.WithName("Test Auth Provider"),
	)
	s.Require().NoError(err)
	s.withAdminRoleCtx = basic.ContextWithAdminIdentity(s.T(), authProvider)
	s.withNoneRoleCtx = basic.ContextWithNoneIdentity(s.T(), authProvider)
	s.withNoAccessCtx = basic.ContextWithNoAccessIdentity(s.T(), authProvider)
	s.withNoRoleCtx = basic.ContextWithNoRoleIdentity(s.T(), authProvider)
	s.anonymousCtx = context.Background()
}

type testCase struct {
	name string
	ctx  context.Context

	expectedPermissionCount int
	expectedAuthorizerError error
	expectedServiceError    error
}

func (s *roleServiceGetMyPermissionsTestSuite) getTestCases() []testCase {
	return []testCase{
		{
			name: accesscontrol.Admin,
			ctx:  s.withAdminRoleCtx,

			expectedPermissionCount: len(resources.ListAll()),
			expectedServiceError:    nil,
			expectedAuthorizerError: nil,
		},
		{
			name: accesscontrol.None,
			ctx:  s.withNoneRoleCtx,

			expectedPermissionCount: 0,
			expectedServiceError:    nil,
			expectedAuthorizerError: errox.NoCredentials,
		},
		{
			name: "No Access",
			ctx:  s.withNoAccessCtx,

			expectedPermissionCount: len(resources.ListAll()),
			expectedServiceError:    nil,
			expectedAuthorizerError: nil,
		},
		{
			name: "No Role",
			ctx:  s.withNoRoleCtx,

			expectedPermissionCount: 0,
			expectedServiceError:    nil,
			expectedAuthorizerError: errox.NoCredentials,
		},
		{
			name: "Anonymous",
			ctx:  s.anonymousCtx,

			expectedPermissionCount: 0,
			expectedServiceError:    errox.NoCredentials,
			expectedAuthorizerError: errox.NoCredentials,
		},
	}
}

func (s *roleServiceGetMyPermissionsTestSuite) TestAuthorizer() {
	for _, c := range s.getTestCases() {
		s.Run(c.name, func() {
			ctx, err := s.svc.AuthFuncOverride(c.ctx, getMyPermissionsServiceName)
			s.ErrorIs(err, c.expectedAuthorizerError)
			s.Equal(c.ctx, ctx)
		})
	}
}

func (s *roleServiceGetMyPermissionsTestSuite) TestGetMyPermissions() {
	emptyRequest := &v1.Empty{}
	for _, c := range s.getTestCases() {
		s.Run(c.name, func() {
			rsp, err := s.svc.GetMyPermissions(c.ctx, emptyRequest)
			s.ErrorIs(err, c.expectedServiceError)
			if c.expectedServiceError == nil {
				s.NotNil(rsp)
				if rsp != nil {
					s.Len(rsp.GetResourceToAccess(), c.expectedPermissionCount)
				}
			} else {
				s.Nil(rsp)
			}
		})
	}
}
