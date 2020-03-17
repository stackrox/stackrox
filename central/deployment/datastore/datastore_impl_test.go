package datastore

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	searcherMocks "github.com/stackrox/rox/central/deployment/datastore/internal/search/mocks"
	indexerMocks "github.com/stackrox/rox/central/deployment/index/mocks"
	storeMocks "github.com/stackrox/rox/central/deployment/store/mocks"
	"github.com/stackrox/rox/central/globaldb"
	indicatorMocks "github.com/stackrox/rox/central/processindicator/datastore/mocks"
	"github.com/stackrox/rox/central/ranking"
	riskMocks "github.com/stackrox/rox/central/risk/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

func TestDeploymentDatastoreSuite(t *testing.T) {
	suite.Run(t, new(DeploymentDataStoreTestSuite))
}

type DeploymentDataStoreTestSuite struct {
	suite.Suite

	storage      *storeMocks.MockStore
	indexer      *indexerMocks.MockIndexer
	searcher     *searcherMocks.MockSearcher
	riskStore    *riskMocks.MockDataStore
	processStore *indicatorMocks.MockDataStore
	filter       filter.Filter

	ctx context.Context

	mockCtrl *gomock.Controller
}

func (suite *DeploymentDataStoreTestSuite) SetupTest() {
	suite.ctx = sac.WithAllAccess(context.Background())

	mockCtrl := gomock.NewController(suite.T())
	suite.mockCtrl = mockCtrl
	suite.storage = storeMocks.NewMockStore(mockCtrl)
	suite.indexer = indexerMocks.NewMockIndexer(mockCtrl)
	suite.searcher = searcherMocks.NewMockSearcher(mockCtrl)
	suite.riskStore = riskMocks.NewMockDataStore(mockCtrl)
	suite.processStore = indicatorMocks.NewMockDataStore(mockCtrl)
	suite.filter = filter.NewFilter(5, []int{5, 4, 3, 2, 1})
}

func (suite *DeploymentDataStoreTestSuite) TestIndexerAcknowledgement() {
	suite.storage.EXPECT().GetKeysToIndex().Return(nil, nil)
	suite.indexer.EXPECT().NeedsInitialIndexing().Return(false, nil)

	datastore, err := newDatastoreImpl(suite.storage, suite.indexer, nil, nil, suite.processStore, nil, nil,
		suite.riskStore, nil, suite.filter, ranking.NewRanker(),
		ranking.NewRanker(), ranking.NewRanker(), concurrency.NewKeyedMutex(globaldb.DefaultDataStorePoolSize))
	suite.NoError(err)

	deployment := fixtures.GetDeployment()
	suite.storage.EXPECT().AckKeysIndexed(deployment.GetId()).Return(nil)
	suite.storage.EXPECT().UpsertDeployment(deployment).Return(nil)
	suite.indexer.EXPECT().AddDeployment(deployment).Return(nil)
	suite.processStore.EXPECT().RemoveProcessIndicatorsOfStaleContainers(gomock.Any(), gomock.Any()).Return(nil)
	suite.NoError(datastore.UpsertDeployment(suite.ctx, deployment))

	suite.storage.EXPECT().AckKeysIndexed(deployment.GetId()).Return(nil)
	suite.storage.EXPECT().UpsertDeployment(deployment).Return(nil)
	suite.processStore.EXPECT().RemoveProcessIndicatorsOfStaleContainers(gomock.Any(), gomock.Any()).Return(nil)
	suite.NoError(datastore.UpsertDeploymentIntoStoreOnly(suite.ctx, deployment))

	suite.storage.EXPECT().AckKeysIndexed(deployment.GetId()).Return(nil)
	suite.storage.EXPECT().RemoveDeployment(deployment.GetId()).Return(nil)
}

func (suite *DeploymentDataStoreTestSuite) TestReconciliationFullReindex() {
	suite.indexer.EXPECT().NeedsInitialIndexing().Return(true, nil)

	dep1 := fixtures.GetDeployment()
	dep1.Id = "A"
	dep2 := fixtures.GetDeployment()
	dep2.Id = "B"

	suite.storage.EXPECT().GetDeploymentIDs().Return([]string{"A", "B", "C"}, nil)
	suite.storage.EXPECT().GetDeploymentsWithIDs([]string{"A", "B", "C"}).Return([]*storage.Deployment{dep1, dep2}, nil, nil)
	suite.indexer.EXPECT().AddDeployments([]*storage.Deployment{dep1, dep2}).Return(nil)

	suite.storage.EXPECT().GetKeysToIndex().Return([]string{"D", "E"}, nil)
	suite.storage.EXPECT().AckKeysIndexed([]string{"D", "E"}).Return(nil)

	suite.indexer.EXPECT().MarkInitialIndexingComplete().Return(nil)

	_, err := newDatastoreImpl(suite.storage, suite.indexer, nil, nil, suite.processStore, nil, nil,
		suite.riskStore, nil, suite.filter, ranking.NewRanker(),
		ranking.NewRanker(), ranking.NewRanker(), concurrency.NewKeyedMutex(globaldb.DefaultDataStorePoolSize))
	suite.NoError(err)
}

func (suite *DeploymentDataStoreTestSuite) TestReconciliationPartialReindex() {
	suite.storage.EXPECT().GetKeysToIndex().Return([]string{"A", "B", "C"}, nil)
	suite.indexer.EXPECT().NeedsInitialIndexing().Return(false, nil)

	dep1 := fixtures.GetDeployment()
	dep1.Id = "A"
	dep2 := fixtures.GetDeployment()
	dep2.Id = "B"
	dep3 := fixtures.GetDeployment()
	dep3.Id = "C"

	deploymentList := []*storage.Deployment{dep1, dep2, dep3}

	suite.storage.EXPECT().GetDeploymentsWithIDs([]string{"A", "B", "C"}).Return(deploymentList, nil, nil)
	suite.indexer.EXPECT().AddDeployments(deploymentList).Return(nil)
	suite.storage.EXPECT().AckKeysIndexed([]string{"A", "B", "C"}).Return(nil)

	_, err := newDatastoreImpl(suite.storage, suite.indexer, nil, nil, suite.processStore, nil, nil,
		suite.riskStore, nil, suite.filter, ranking.NewRanker(),
		ranking.NewRanker(), ranking.NewRanker(), concurrency.NewKeyedMutex(globaldb.DefaultDataStorePoolSize))
	suite.NoError(err)

	// Make deploymentlist just A,B so C should be deleted
	deploymentList = deploymentList[:1]
	suite.storage.EXPECT().GetKeysToIndex().Return([]string{"A", "B", "C"}, nil)
	suite.indexer.EXPECT().NeedsInitialIndexing().Return(false, nil)

	suite.storage.EXPECT().GetDeploymentsWithIDs([]string{"A", "B", "C"}).Return(deploymentList, []int{2}, nil)
	suite.indexer.EXPECT().AddDeployments(deploymentList).Return(nil)
	suite.indexer.EXPECT().DeleteDeployments([]string{"C"}).Return(nil)
	suite.storage.EXPECT().AckKeysIndexed([]string{"A", "B", "C"}).Return(nil)

	_, err = newDatastoreImpl(suite.storage, suite.indexer, nil, nil, suite.processStore, nil, nil,
		suite.riskStore, nil, suite.filter, ranking.NewRanker(),
		ranking.NewRanker(), ranking.NewRanker(), concurrency.NewKeyedMutex(globaldb.DefaultDataStorePoolSize))
	suite.NoError(err)
}

func (suite *DeploymentDataStoreTestSuite) TestInitializeRanker() {
	clusterRanker := ranking.NewRanker()
	nsRanker := ranking.NewRanker()
	deploymentRanker := ranking.NewRanker()

	ds, err := newDatastoreImpl(suite.storage, suite.indexer, suite.searcher, nil, suite.processStore, nil, nil,
		suite.riskStore, nil, suite.filter, clusterRanker,
		nsRanker, deploymentRanker, concurrency.NewKeyedMutex(globaldb.DefaultDataStorePoolSize))
	suite.NoError(err)

	deployments := []*storage.Deployment{
		{
			Id:          "1",
			RiskScore:   float32(1.0),
			NamespaceId: "ns1",
			ClusterId:   "c1",
		},
		{
			Id:          "2",
			RiskScore:   float32(2.0),
			NamespaceId: "ns1",
			ClusterId:   "c1",
		},
		{
			Id:          "3",
			NamespaceId: "ns2",
			ClusterId:   "c2",
		},
		{
			Id: "4",
		},
		{
			Id: "5",
		},
	}

	suite.searcher.EXPECT().Search(gomock.Any(), search.EmptyQuery()).Return([]search.Result{{ID: "1"}, {ID: "2"}, {ID: "3"}, {ID: "4"}, {ID: "5"}}, nil)
	suite.storage.EXPECT().GetDeployment(deployments[0].Id).Return(deployments[0], true, nil)
	suite.storage.EXPECT().GetDeployment(deployments[1].Id).Return(deployments[1], true, nil)
	suite.storage.EXPECT().GetDeployment(deployments[2].Id).Return(deployments[2], true, nil)
	suite.storage.EXPECT().GetDeployment(deployments[3].Id).Return(nil, false, nil)
	suite.storage.EXPECT().GetDeployment(deployments[4].Id).Return(nil, false, errors.New("fake error"))

	ds.initializeRanker()

	suite.Equal(int64(1), clusterRanker.GetRankForID("c1"))
	suite.Equal(int64(2), clusterRanker.GetRankForID("c2"))

	suite.Equal(int64(1), nsRanker.GetRankForID("ns1"))
	suite.Equal(int64(2), nsRanker.GetRankForID("ns2"))

	suite.Equal(int64(1), deploymentRanker.GetRankForID("2"))
	suite.Equal(int64(2), deploymentRanker.GetRankForID("1"))
	suite.Equal(int64(3), deploymentRanker.GetRankForID("3"))
}
