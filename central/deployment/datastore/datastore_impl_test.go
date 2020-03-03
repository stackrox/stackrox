package datastore

import (
	"context"
	"errors"
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

func (suite *DeploymentDataStoreTestSuite) TestInitializeRanker() {
	risks := []*storage.Risk{
		{
			Id: "1",
			Subject: &storage.RiskSubject{
				Id:        "1",
				Type:      storage.RiskSubjectType_DEPLOYMENT,
				Namespace: "1",
				ClusterId: "1",
			},
		},
		{
			Id: "2",
			Subject: &storage.RiskSubject{
				Id:        "2",
				Type:      storage.RiskSubjectType_DEPLOYMENT,
				Namespace: "2",
				ClusterId: "2",
			},
		},
		{
			Id: "3",
			Subject: &storage.RiskSubject{
				Id:        "3",
				Type:      storage.RiskSubjectType_DEPLOYMENT,
				Namespace: "3",
				ClusterId: "3",
			},
		},
	}

	deployments := []*storage.Deployment{
		{
			Id: "1",
		},
		{
			Id: "2",
		},
		{
			Id: "3",
		},
	}

	ds, err := newDatastoreImpl(suite.storage, suite.indexer, nil, nil, suite.processStore, nil, nil,
		suite.riskStore, nil, suite.filter)
	suite.NoError(err)

	suite.riskStore.EXPECT().SearchRawRisks(gomock.Any(), gomock.Any()).Return(risks, nil)
	suite.storage.EXPECT().GetDeployment(deployments[0].Id).Return(deployments[0], true, nil)
	suite.storage.EXPECT().GetDeployment(deployments[1].Id).Return(nil, false, nil)
	suite.riskStore.EXPECT().RemoveRisk(gomock.Any(), deployments[1].Id, storage.RiskSubjectType_DEPLOYMENT)
	suite.storage.EXPECT().GetDeployment(deployments[2].Id).Return(nil, false, errors.New("fake error"))
	err = ds.initializeRanker()
	suite.NoError(err)
}
