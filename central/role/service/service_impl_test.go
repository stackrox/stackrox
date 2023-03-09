//go:build sql_integration

package service

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/globalindex"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/datastore"
	roleDatastore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	boltPkg "github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	dackboxConcurrency "github.com/stackrox/rox/pkg/dackbox/concurrency"
	"github.com/stackrox/rox/pkg/dackbox/utils/queue"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/labels"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.etcd.io/bbolt"
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

var (
	namespaceQueenQueen = &storage.NamespaceMetadata{
		Id:          "album.queen",
		Name:        "Queen",
		ClusterId:   "band.queen",
		ClusterName: "Queen",
		Labels: map[string]string{
			"released": "1973",
		},
	}

	namespaceQueenJazz = &storage.NamespaceMetadata{
		Id:          "album.jazz",
		Name:        "Jazz",
		ClusterId:   "band.queen",
		ClusterName: "Queen",
		Labels: map[string]string{
			"released": "1978",
		},
	}

	namespaceQueenInnuendo = &storage.NamespaceMetadata{
		Id:          "album.innuendo",
		Name:        "Innuendo",
		ClusterId:   "band.queen",
		ClusterName: "Queen",
		Labels: map[string]string{
			"released": "1991",
		},
	}

	namespacePinkFloydTheWall = &storage.NamespaceMetadata{
		Id:          "album.thewall",
		Name:        "The Wall",
		ClusterId:   "band.pinkfloyd",
		ClusterName: "Pink Floyd",
		Labels: map[string]string{
			"released": "1979",
		},
	}

	namespaceDeepPurpleMachineHead = &storage.NamespaceMetadata{
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
	namespaceQueenQueen,
	namespaceQueenJazz,
	namespaceQueenInnuendo,
	// Pink Floyd
	namespacePinkFloydTheWall,
	// Deep Purple
	namespaceDeepPurpleMachineHead,
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
			resHigh, err := effectiveAccessScopeForSimpleAccessScope(tc.rules, clusters, namespaces, v1.ComputeEffectiveAccessScopeRequest_HIGH)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedHigh, resHigh)
		})
		t.Run(tc.desc+"detail: STANDARD", func(t *testing.T) {
			resStandard, err := effectiveAccessScopeForSimpleAccessScope(tc.rules, clusters, namespaces, v1.ComputeEffectiveAccessScopeRequest_STANDARD)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedStandard, resStandard)
		})
		t.Run(tc.desc+"detail: MINIMAL", func(t *testing.T) {
			resMinimal, err := effectiveAccessScopeForSimpleAccessScope(tc.rules, clusters, namespaces, v1.ComputeEffectiveAccessScopeRequest_MINIMAL)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedMinimal, resMinimal)
		})
		t.Run(tc.desc+"unknown detail maps to STANDARD", func(t *testing.T) {
			resUnknown, err := effectiveAccessScopeForSimpleAccessScope(tc.rules, clusters, namespaces, 42)
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

	boltEngine  *bbolt.DB
	rocksEngine *rocksdb.RocksDB
	bleveIndex  bleve.Index

	service *serviceImpl

	storedClusterIDs   []string
	storedNamespaceIDs []string
}

func (s *serviceImplTestSuite) SetupSuite() {
	var err error
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		s.postgres = pgtest.ForT(s.T())
		s.Require().NotNil(s.postgres)
		roleStore, err := roleDatastore.GetTestPostgresDataStore(s.T(), s.postgres.Pool)
		s.Require().NoError(err)
		clusterStore, err := clusterDataStore.GetTestPostgresDataStore(s.T(), s.postgres.Pool)
		s.Require().NoError(err)
		namespaceStore, err := namespaceDataStore.GetTestPostgresDataStore(s.T(), s.postgres.Pool)
		s.Require().NoError(err)

		s.service = &serviceImpl{
			roleDataStore:      roleStore,
			clusterDataStore:   clusterStore,
			namespaceDataStore: namespaceStore,
		}
	} else {
		s.boltEngine, err = boltPkg.NewTemp("roleServiceTestBolt")
		s.Require().NoError(err)
		s.rocksEngine, err = rocksdb.NewTemp("roleServiceTest")
		s.Require().NoError(err)
		s.bleveIndex, err = globalindex.MemOnlyIndex()
		s.Require().NoError(err)

		keyFence := dackboxConcurrency.NewKeyFence()
		indexQ := queue.NewWaitableQueue()
		dacky, err := dackbox.NewRocksDBDackBox(s.rocksEngine, indexQ, []byte("graph"), []byte("dirty"), []byte("valid"))
		s.Require().NoError(err)

		roleStore, err := roleDatastore.GetTestRocksBleveDataStore(s.T(), s.rocksEngine)
		s.Require().NoError(err)
		clusterStore, err := clusterDataStore.GetTestRocksBleveDataStore(s.T(), s.rocksEngine, s.bleveIndex, dacky, keyFence, s.boltEngine)
		s.Require().NoError(err)
		namespaceStore, err := namespaceDataStore.GetTestRocksBleveDataStore(s.T(), s.rocksEngine, s.bleveIndex, dacky, keyFence)
		s.Require().NoError(err)

		s.service = &serviceImpl{
			roleDataStore:      roleStore,
			clusterDataStore:   clusterStore,
			namespaceDataStore: namespaceStore,
		}
	}
}

func (s *serviceImplTestSuite) TearDownSuite() {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		s.postgres.Teardown(s.T())
	} else {
		s.Require().NoError(s.boltEngine.Close())
		s.Require().NoError(rocksdb.CloseAndRemove(s.rocksEngine))
		s.Require().NoError(s.bleveIndex.Close())
	}
}

func (s *serviceImplTestSuite) SetupTest() {
	s.storedClusterIDs = make([]string, 0)
	s.storedNamespaceIDs = make([]string, 0)
}

func (s *serviceImplTestSuite) TearDownTest() {
	writeCtx := sac.WithAllAccess(context.Background())
	doneSignal := concurrency.NewSignal()
	for _, clusterID := range s.storedClusterIDs {
		s.NoError(s.service.clusterDataStore.RemoveCluster(writeCtx, clusterID, &doneSignal))
	}
	<-doneSignal.Done()
	s.storedClusterIDs = s.storedClusterIDs[:0]
	for _, namespaceID := range s.storedNamespaceIDs {
		s.NoError(s.service.namespaceDataStore.RemoveNamespace(writeCtx, namespaceID))
	}
}

const (
	namespaceUUIDNamespace = "namespace"

	clusterPermission             = "Cluster"
	deploymentPermission          = "Deployment"
	deploymentExtensionPermission = "DeploymentExtension"
	integrationPermission         = "Integration"
	namespacePermission           = "Namespace"
	networkGraphPermission        = "NetworkGraph"
	nodePermission                = "Node"
	rolePermission                = "Role"
)

func getTestResourceScopeSingleNamespace(_ *testing.T, clusterID string, namespace string) *sac.TestResourceScope {
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
	writeCtx := sac.WithAllAccess(context.Background())
	clusterNameToIDMap := make(map[string]string, 0)
	for _, cluster := range clusters {
		clusterToAdd := cluster.Clone()
		clusterToAdd.Id = ""
		id, err := s.service.clusterDataStore.AddCluster(writeCtx, clusterToAdd)
		s.Require().NoError(err)
		clusterNameToIDMap[cluster.GetName()] = id
		s.storedClusterIDs = append(s.storedClusterIDs, id)
	}
	for _, namespace := range namespaces {
		ns := namespace.Clone()
		ns.Id = getNamespaceID(ns.GetName())
		s.Require().NoError(s.service.namespaceDataStore.AddNamespace(writeCtx, ns))
		s.storedNamespaceIDs = append(s.storedNamespaceIDs, ns.GetId())
	}
	queenClusterID := clusterNameToIDMap[clusterQueen.GetName()]
	testResourceScope1 := getTestResourceScopeSingleNamespace(
		s.T(),
		queenClusterID,
		namespaceQueenInnuendo.GetName())
	pinkFloydClusterID := clusterNameToIDMap[clusterPinkFloyd.GetName()]
	testResourceScope2 := getTestResourceScopeSingleNamespace(
		s.T(),
		pinkFloydClusterID,
		namespacePinkFloydTheWall.GetName())
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

	queenClusterResponse := &v1.ScopeElementForPermission{
		Id:   queenClusterID,
		Name: clusterQueen.GetName(),
	}

	pinkFloydClusterResponse := &v1.ScopeElementForPermission{
		Id:   pinkFloydClusterID,
		Name: clusterPinkFloyd.GetName(),
	}

	testCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.TestScopeCheckerCoreFromFullScopeMap(s.T(), testScopeMap))

	testCases := []struct {
		name              string
		testedPermissions []string
		expectedClusters  []*v1.ScopeElementForPermission
	}{
		{
			name:              "Global permission (Not Granted) gets no cluster data.",
			testedPermissions: []string{rolePermission},
			expectedClusters:  []*v1.ScopeElementForPermission{},
		},
		{
			name:              "Global permission (Granted) gets no cluster data.",
			testedPermissions: []string{integrationPermission},
			expectedClusters:  []*v1.ScopeElementForPermission{},
		},
		{
			name:              "Not granted Cluster scoped permission gets no cluster data.",
			testedPermissions: []string{clusterPermission},
			expectedClusters:  []*v1.ScopeElementForPermission{},
		},
		{
			name:              "Granted Cluster scoped permission gets only cluster data for clusters in permission scope.",
			testedPermissions: []string{nodePermission},
			expectedClusters:  []*v1.ScopeElementForPermission{queenClusterResponse},
		},
		{
			name:              "Not granted Namespace scoped permission gets no cluster data.",
			testedPermissions: []string{namespacePermission},
			expectedClusters:  []*v1.ScopeElementForPermission{},
		},
		{
			name:              "Granted Namespace scoped permission gets only cluster data for clusters in permission scope.",
			testedPermissions: []string{deploymentPermission},
			expectedClusters:  []*v1.ScopeElementForPermission{queenClusterResponse},
		},
		{
			name:              "Multiple not granted Namespace scoped permissions get no cluster data.",
			testedPermissions: []string{namespacePermission, deploymentExtensionPermission},
			expectedClusters:  []*v1.ScopeElementForPermission{},
		},
		{
			name:              "Multiple Namespace scoped permissions get only cluster data for clusters in granted permission scopes.",
			testedPermissions: []string{namespacePermission, deploymentPermission},
			expectedClusters:  []*v1.ScopeElementForPermission{queenClusterResponse},
		},
		{
			name:              "empty permission list get cluster data for all cluster data in scope of granted cluster and namespace permissions.",
			testedPermissions: []string{},
			expectedClusters:  []*v1.ScopeElementForPermission{queenClusterResponse, pinkFloydClusterResponse},
		},
	}

	for _, c := range testCases {
		s.Run(c.name, func() {
			clusterResponse, err := s.service.GetClustersForPermissions(testCtx, &v1.GetClustersForPermissionsRequest{
				Pagination:  nil,
				Permissions: c.testedPermissions,
			})
			s.NoError(err)
			s.ElementsMatch(clusterResponse.GetClusters(), c.expectedClusters)
		})
	}
}

func (s *serviceImplTestSuite) TestGetNamespacesForClusterAndPermissions() {
	writeCtx := sac.WithAllAccess(context.Background())
	clusterNameToIDMap := make(map[string]string, 0)
	for _, cluster := range clusters {
		clusterToAdd := cluster.Clone()
		clusterToAdd.Id = ""
		id, err := s.service.clusterDataStore.AddCluster(writeCtx, clusterToAdd)
		s.Require().NoError(err)
		clusterNameToIDMap[cluster.GetName()] = id
		s.storedClusterIDs = append(s.storedClusterIDs, id)
	}
	for _, namespace := range namespaces {
		ns := namespace.Clone()
		ns.Id = getNamespaceID(ns.GetName())
		ns.ClusterId = clusterNameToIDMap[ns.GetClusterName()]
		s.Require().NoError(s.service.namespaceDataStore.AddNamespace(writeCtx, ns))
		s.storedNamespaceIDs = append(s.storedNamespaceIDs, ns.GetId())
	}
	queenClusterID := clusterNameToIDMap[clusterQueen.GetName()]
	testResourceScope1 := getTestResourceScopeSingleNamespace(
		s.T(),
		queenClusterID,
		namespaceQueenInnuendo.GetName())
	pinkFloydClusterID := clusterNameToIDMap[clusterPinkFloyd.GetName()]
	testResourceScope2 := getTestResourceScopeSingleNamespace(
		s.T(),
		pinkFloydClusterID,
		namespacePinkFloydTheWall.GetName())
	testResourceScope3 := getTestResourceScopeSingleNamespace(
		s.T(),
		queenClusterID,
		namespaceQueenQueen.GetName())
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

	queenQueenNamespaceResponse := &v1.ScopeElementForPermission{
		Id:   getNamespaceID(namespaceQueenQueen.GetName()),
		Name: namespaceQueenQueen.GetName(),
	}

	// queenJazzNamespaceResponse := &v1.ScopeElementForPermission{
	// 	Id:   getNamespaceID(namespaceQueenJazz.GetName()),
	// 	Name: namespaceQueenJazz.GetName(),
	// }

	queenInnuendoNamespaceResponse := &v1.ScopeElementForPermission{
		Id:   getNamespaceID(namespaceQueenInnuendo.GetName()),
		Name: namespaceQueenInnuendo.GetName(),
	}

	pinkFloydTheWallNamespaceResponse := &v1.ScopeElementForPermission{
		Id:   getNamespaceID(namespacePinkFloydTheWall.GetName()),
		Name: namespacePinkFloydTheWall.GetName(),
	}

	testCases := []struct {
		name               string
		testedClusterID    string
		testedPermissions  []string
		expectedNamespaces []*v1.ScopeElementForPermission
	}{
		{
			name:               "Global permission (Not Granted) gets no namespace data.",
			testedClusterID:    clusterNameToIDMap[clusterQueen.GetName()],
			testedPermissions:  []string{rolePermission},
			expectedNamespaces: []*v1.ScopeElementForPermission{},
		},
		{
			name:               "Global permission (Granted) gets no namespace data.",
			testedClusterID:    clusterNameToIDMap[clusterQueen.GetName()],
			testedPermissions:  []string{integrationPermission},
			expectedNamespaces: []*v1.ScopeElementForPermission{},
		},
		{
			name:               "Not granted Cluster scoped permission gets no namespace data.",
			testedClusterID:    clusterNameToIDMap[clusterQueen.GetName()],
			testedPermissions:  []string{clusterPermission},
			expectedNamespaces: []*v1.ScopeElementForPermission{},
		},
		{
			name:               "Granted Cluster scoped permission gets no namespace data.",
			testedClusterID:    clusterNameToIDMap[clusterQueen.GetName()],
			testedPermissions:  []string{nodePermission},
			expectedNamespaces: []*v1.ScopeElementForPermission{},
		},
		{
			name:               "Not granted Namespace scoped permission gets no namespace data.",
			testedClusterID:    clusterNameToIDMap[clusterQueen.GetName()],
			testedPermissions:  []string{namespacePermission},
			expectedNamespaces: []*v1.ScopeElementForPermission{},
		},
		{
			name:               "Granted Namespace scoped permission gets only namespace data for namespaces in cluster and permission scope.",
			testedClusterID:    clusterNameToIDMap[clusterQueen.GetName()],
			testedPermissions:  []string{deploymentPermission},
			expectedNamespaces: []*v1.ScopeElementForPermission{queenInnuendoNamespaceResponse},
		},
		{
			name:               "Granted Namespace scoped permission gets only namespace data for namespaces in cluster and permission scope (other permission).",
			testedClusterID:    clusterNameToIDMap[clusterPinkFloyd.GetName()],
			testedPermissions:  []string{networkGraphPermission},
			expectedNamespaces: []*v1.ScopeElementForPermission{pinkFloydTheWallNamespaceResponse},
		},
		{
			name:               "Multiple not granted Namespace scoped permissions get no namespace data.",
			testedClusterID:    clusterNameToIDMap[clusterQueen.GetName()],
			testedPermissions:  []string{namespacePermission, deploymentExtensionPermission},
			expectedNamespaces: []*v1.ScopeElementForPermission{},
		},
		{
			name:               "Multiple Namespace scoped permissions get only namespace data for namespaces in granted permission scopes.",
			testedClusterID:    clusterNameToIDMap[clusterQueen.GetName()],
			testedPermissions:  []string{namespacePermission, deploymentPermission},
			expectedNamespaces: []*v1.ScopeElementForPermission{queenInnuendoNamespaceResponse},
		},
		{
			name:               "empty permission list get namespace data for all namespaces in scope of target cluster and granted namespace permissions.",
			testedClusterID:    clusterNameToIDMap[clusterQueen.GetName()],
			testedPermissions:  []string{},
			expectedNamespaces: []*v1.ScopeElementForPermission{queenQueenNamespaceResponse, queenInnuendoNamespaceResponse},
		},
	}

	scc := sac.TestScopeCheckerCoreFromFullScopeMap(s.T(), testScopeMap)
	log.Info(scc.EffectiveAccessScope(permissions.View(resources.Deployment)))
	testCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.TestScopeCheckerCoreFromFullScopeMap(s.T(), testScopeMap))

	for _, c := range testCases {
		s.Run(c.name, func() {
			request := &v1.GetNamespaceForClusterAndPermissionsRequest{
				Pagination:  nil,
				ClusterId:   c.testedClusterID,
				Permissions: c.testedPermissions,
			}
			namespaceResponse, err := s.service.GetNamespacesForClusterAndPermissions(testCtx, request)
			s.NoError(err)
			s.ElementsMatch(namespaceResponse.GetNamespaces(), c.expectedNamespaces)
		})
	}
}
