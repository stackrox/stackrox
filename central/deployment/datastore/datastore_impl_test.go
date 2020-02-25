package datastore

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	indexerMocks "github.com/stackrox/rox/central/deployment/index/mocks"
	storeMocks "github.com/stackrox/rox/central/deployment/store/mocks"
	indicatorMocks "github.com/stackrox/rox/central/processindicator/datastore/mocks"
	riskMocks "github.com/stackrox/rox/central/risk/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

func TestDeploymentDatastoreSuite(t *testing.T) {
	suite.Run(t, new(DeploymentDataStoreTestSuite))
}

type DeploymentDataStoreTestSuite struct {
	suite.Suite

	storage      *storeMocks.MockStore
	indexer      *indexerMocks.MockIndexer
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

	suite.riskStore = riskMocks.NewMockDataStore(mockCtrl)
	suite.processStore = indicatorMocks.NewMockDataStore(mockCtrl)
	suite.filter = filter.NewFilter(5, []int{5, 4, 3, 2, 1})
}

func (suite *DeploymentDataStoreTestSuite) TestIndexerAcknowledgement() {
	suite.storage.EXPECT().GetKeysToIndex().Return(nil, nil)
	suite.indexer.EXPECT().NeedsInitialIndexing().Return(false, nil)

	datastore, err := newDatastoreImpl(suite.storage, suite.indexer, nil, nil, suite.processStore, nil, nil,
		suite.riskStore, nil, suite.filter)
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
		suite.riskStore, nil, suite.filter)
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
		suite.riskStore, nil, suite.filter)
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
		suite.riskStore, nil, suite.filter)
	suite.NoError(err)
}
