package globaldatastore

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/node/datastore"
	"github.com/stackrox/rox/central/node/globalstore"
	"github.com/stackrox/rox/central/node/globalstore/mocks"
	"github.com/stackrox/rox/central/node/index"
	"github.com/stackrox/rox/central/node/store"
	mocks2 "github.com/stackrox/rox/central/node/store/mocks"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

func createNodesFromIDs(clusterID string, nodeIDs []string) ([]*storage.Node, map[string]*storage.Node) {
	var nodes []*storage.Node
	for _, id := range nodeIDs {
		nodes = append(nodes, &storage.Node{
			Id:        id,
			Name:      "node-" + id,
			ClusterId: clusterID,
		})
	}

	nodeMap := make(map[string]*storage.Node)
	for _, node := range nodes {
		nodeMap[node.GetId()] = node
	}

	return nodes, nodeMap
}

func setupMockGlobalStore(mockCtrl *gomock.Controller, nodesInClusters map[string][]string) *mocks.MockGlobalStore {
	mockGlobalStore := mocks.NewMockGlobalStore(mockCtrl)
	allClusterNodeStores := make(map[string]store.Store)
	totalNodeCount := 0
	for clusterID, nodeIDs := range nodesInClusters {
		mockStore := mocks2.NewMockStore(mockCtrl)

		nodes, nodeMap := createNodesFromIDs(clusterID, nodeIDs)
		mockStore.EXPECT().ListNodes().AnyTimes().Return(nodes, nil)
		mockStore.EXPECT().CountNodes().AnyTimes().Return(len(nodes), nil)
		mockStore.EXPECT().GetNode(gomock.Any()).AnyTimes().DoAndReturn(func(nodeID string) (*storage.Node, error) {
			return nodeMap[nodeID], nil
		})

		allClusterNodeStores[clusterID] = mockStore

		totalNodeCount += len(nodeIDs)
	}

	mockGlobalStore.EXPECT().GetAllClusterNodeStores().AnyTimes().Return(allClusterNodeStores, nil)
	mockGlobalStore.EXPECT().GetClusterNodeStore(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(clusterID string, writeAccess bool) (datastore.DataStore, error) {
		store := allClusterNodeStores[clusterID]
		if store == nil {
			if writeAccess {
				store = mocks2.NewMockStore(mockCtrl)
			} else {
				return nil, errors.New("not found")
			}
		}
		return store, nil
	})
	mockGlobalStore.EXPECT().CountAllNodes().AnyTimes().Return(totalNodeCount, nil)
	return mockGlobalStore
}

func TestGlobalDatastore(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	mockCtrl        *gomock.Controller
	mockGlobalStore *mocks.MockGlobalStore

	globalDataStore GlobalDataStore

	ctx context.Context
}

func (s *testSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockGlobalStore = setupMockGlobalStore(s.mockCtrl, map[string][]string{
		"cluster-1-no-access":   {"1-1", "1-2"},
		"cluster-2-read-access": {"2-1", "2-2", "2-3"},
		"cluster-3-full-access": {"3-1", "3-2", "3-3", "3-4"},
	})

	tmpIndex, err := globalindex.TempInitializeIndices("")
	s.Require().NoError(err)
	s.globalDataStore, err = New(s.mockGlobalStore, index.New(tmpIndex))
	s.Require().NoError(err)

	scc := sac.OneStepSCC{
		sac.AccessModeScopeKey(storage.Access_READ_ACCESS): sac.OneStepSCC{
			sac.ResourceScopeKey(resources.Node.GetResource()): sac.AllowFixedScopes(sac.ClusterScopeKeys("cluster-2-read-access", "cluster-3-full-access")),
		},
		sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS): sac.OneStepSCC{
			sac.ResourceScopeKey(resources.Node.GetResource()): sac.AllowFixedScopes(sac.ClusterScopeKeys("cluster-3-full-access")),
		},
	}

	s.ctx = sac.WithGlobalAccessScopeChecker(context.Background(), scc)
}

func (s *testSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *testSuite) TestCount() {
	count, err := s.globalDataStore.CountAllNodes(s.ctx)
	s.NoError(err)

	s.Equal(7, count)
}

func (s *testSuite) TestSearch() {
	results, err := s.globalDataStore.Search(s.ctx, search.EmptyQuery())
	s.NoError(err)

	ids := search.ResultsToIDs(results)

	expected := []string{"2-1", "2-2", "2-3", "3-1", "3-2", "3-3", "3-4"}
	s.ElementsMatch(ids, expected)
}

func (s *testSuite) TestGetAllClusterNodeStores_Read() {
	stores, err := s.globalDataStore.GetAllClusterNodeStores(s.ctx, false)
	s.NoError(err)

	var ids []string
	for clusterID := range stores {
		ids = append(ids, clusterID)
	}

	expected := []string{"cluster-2-read-access", "cluster-3-full-access"}

	s.ElementsMatch(ids, expected)
}

func (s *testSuite) TestGetAllClusterNodeStores_Write() {
	stores, err := s.globalDataStore.GetAllClusterNodeStores(s.ctx, true)
	s.NoError(err)

	var ids []string
	for clusterID := range stores {
		ids = append(ids, clusterID)
	}

	expected := []string{"cluster-3-full-access"}

	s.ElementsMatch(ids, expected)
}

func (s *testSuite) TestGetClusterNodeStore_Read_OK() {
	store, err := s.globalDataStore.GetClusterNodeStore(s.ctx, "cluster-2-read-access", false)
	s.NoError(err)
	s.NotNil(store)
}

func (s *testSuite) TestGetClusterNodeStore_Read_NonExisting() {
	_, err := s.globalDataStore.GetClusterNodeStore(s.ctx, "cluster-0-non-existing", false)
	s.Error(err)
}

func (s *testSuite) TestGetClusterNodeStore_Read_PermissionDenied() {
	_, err := s.globalDataStore.GetClusterNodeStore(s.ctx, "cluster-1-no-access", false)

	s.Error(err)
}

func (s *testSuite) TestGetClusterNodeStore_Write_OK() {
	store, err := s.globalDataStore.GetClusterNodeStore(s.ctx, "cluster-3-full-access", true)
	s.NoError(err)
	s.NotNil(store)
}

func (s *testSuite) TestGetClusterNodeStore_Write_NonExisting() {
	_, err := s.globalDataStore.GetClusterNodeStore(s.ctx, "cluster-0-non-existing", true)
	s.Error(err)
}

func (s *testSuite) TestGetClusterNodeStore_Write_NonExisting_WithGlobalAccess() {
	ctx := sac.WithAllAccess(context.Background())

	store, err := s.globalDataStore.GetClusterNodeStore(ctx, "cluster-0-non-existing", true)
	s.NoError(err)
	s.NotNil(store)
}

func (s *testSuite) TestGetClusterNodeStore_Write_PermissionDenied() {
	_, err := s.globalDataStore.GetClusterNodeStore(s.ctx, "cluster-2-read-access", true)

	s.Error(err)
}

func TestLiveGlobalDatastore(t *testing.T) {
	suite.Run(t, new(liveTestSuite))
}

type liveTestSuite struct {
	suite.Suite

	globalDataStore GlobalDataStore
	ctx             context.Context
}

func (s *liveTestSuite) SetupTest() {
	bleveIndex, err := globalindex.MemOnlyIndex()
	s.NoError(err)

	store := globalstore.NewGlobalStore(testutils.DBForT(s.T()))
	indexer := index.New(bleveIndex)

	s.globalDataStore, err = New(store, indexer)
	s.NoError(err)
	s.ctx = sac.WithAllAccess(context.Background())
}

func (s *liveTestSuite) addCluster(id string, numNodes int) {
	ns, err := s.globalDataStore.GetClusterNodeStore(s.ctx, id, true)
	s.NoError(err)

	for i := 0; i < numNodes; i++ {
		err = ns.UpsertNode(&storage.Node{
			Id:        fmt.Sprintf("%s-node%d", id, i),
			ClusterId: id,
		})
		s.NoError(err)
	}
}

func (s *liveTestSuite) TestRemoveClusterNodeStores() {
	s.addCluster("cluster1", 3)

	count, err := s.globalDataStore.CountAllNodes(s.ctx)
	s.NoError(err)
	s.Equal(3, count)

	results, err := s.globalDataStore.Search(s.ctx, search.EmptyQuery())
	s.NoError(err)
	s.Len(results, 3)

	s.NoError(s.globalDataStore.RemoveClusterNodeStores(s.ctx, "cluster1"))
	count, err = s.globalDataStore.CountAllNodes(s.ctx)
	s.NoError(err)
	s.Equal(0, count)

	results, err = s.globalDataStore.Search(s.ctx, search.EmptyQuery())
	s.NoError(err)
	s.Len(results, 0)

	s.addCluster("cluster1", 3)
	s.addCluster("cluster2", 3)
	s.addCluster("cluster3", 3)

	// Test multi delete
	err = s.globalDataStore.RemoveClusterNodeStores(s.ctx, "cluster1", "cluster2")
	s.NoError(err)

	count, err = s.globalDataStore.CountAllNodes(s.ctx)
	s.NoError(err)
	s.Equal(3, count)

	results, err = s.globalDataStore.Search(s.ctx, search.EmptyQuery())
	s.NoError(err)
	s.Len(results, 3)
}
