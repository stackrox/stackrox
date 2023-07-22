package datastore

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	searcherMocks "github.com/stackrox/rox/central/deployment/datastore/internal/search/mocks"
	indexerMocks "github.com/stackrox/rox/central/deployment/index/mocks"
	storeMocks "github.com/stackrox/rox/central/deployment/store/mocks"
	"github.com/stackrox/rox/central/ranking"
	riskMocks "github.com/stackrox/rox/central/risk/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestDeploymentDatastoreSuite(t *testing.T) {
	suite.Run(t, new(DeploymentDataStoreTestSuite))
}

type DeploymentDataStoreTestSuite struct {
	suite.Suite

	storage   *storeMocks.MockStore
	indexer   *indexerMocks.MockIndexer
	searcher  *searcherMocks.MockSearcher
	riskStore *riskMocks.MockDataStore
	filter    filter.Filter

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
	suite.filter = filter.NewFilter(5, 5, []int{5, 4, 3, 2, 1})
}

func (suite *DeploymentDataStoreTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *DeploymentDataStoreTestSuite) TestInitializeRanker() {
	clusterRanker := ranking.NewRanker()
	nsRanker := ranking.NewRanker()
	deploymentRanker := ranking.NewRanker()

	ds := newDatastoreImpl(suite.storage, suite.searcher, nil, nil, nil, suite.riskStore, nil, suite.filter, clusterRanker, nsRanker, deploymentRanker)

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
	suite.storage.EXPECT().Get(gomock.Any(), deployments[0].Id).Return(deployments[0], true, nil)
	suite.storage.EXPECT().Get(gomock.Any(), deployments[1].Id).Return(deployments[1], true, nil)
	suite.storage.EXPECT().Get(gomock.Any(), deployments[2].Id).Return(deployments[2], true, nil)
	suite.storage.EXPECT().Get(gomock.Any(), deployments[3].Id).Return(nil, false, nil)
	suite.storage.EXPECT().Get(gomock.Any(), deployments[4].Id).Return(nil, false, errors.New("fake error"))

	ds.initializeRanker()

	suite.Equal(int64(1), clusterRanker.GetRankForID("c1"))
	suite.Equal(int64(2), clusterRanker.GetRankForID("c2"))

	suite.Equal(int64(1), nsRanker.GetRankForID("ns1"))
	suite.Equal(int64(2), nsRanker.GetRankForID("ns2"))

	suite.Equal(int64(1), deploymentRanker.GetRankForID("2"))
	suite.Equal(int64(2), deploymentRanker.GetRankForID("1"))
	suite.Equal(int64(3), deploymentRanker.GetRankForID("3"))
}

func (suite *DeploymentDataStoreTestSuite) TestMergeCronJobs() {
	ds := newDatastoreImpl(suite.storage, suite.searcher, nil, nil, nil, suite.riskStore, nil, suite.filter, nil, nil, nil)
	ctx := sac.WithAllAccess(context.Background())

	// Not a cronjob so no merging
	dep := &storage.Deployment{
		Id:   "id",
		Type: kubernetes.Deployment,
	}
	expectedDep := dep.Clone()
	suite.NoError(ds.mergeCronJobs(ctx, dep))
	suite.Equal(expectedDep, dep)

	dep.Containers = []*storage.Container{
		{
			Image: &storage.ContainerImage{
				Id: "abc",
			},
		},
		{
			Image: &storage.ContainerImage{
				Id: "def",
			},
		},
	}
	dep.Type = kubernetes.CronJob
	expectedDep = dep.Clone()
	// All container have images with digests
	suite.NoError(ds.mergeCronJobs(ctx, dep))
	suite.Equal(expectedDep, dep)

	// All containers don't have images with digests, but old deployment does not exist
	dep.Containers[1].Image.Id = ""
	expectedDep = dep.Clone()
	suite.storage.EXPECT().Get(ctx, "id").Return(nil, false, nil)
	suite.NoError(ds.mergeCronJobs(ctx, dep))
	suite.Equal(expectedDep, dep)

	// Different numbers of containers for the CronJob so early exit with no changes
	returnedDep := dep.Clone()
	returnedDep.Containers = returnedDep.Containers[:1]

	suite.storage.EXPECT().Get(ctx, "id").Return(returnedDep, true, nil)
	suite.NoError(ds.mergeCronJobs(ctx, dep))
	suite.Equal(expectedDep, dep)

	// Filled in for missing last container, but names do not match
	returnedDep.Containers = append(returnedDep.Containers, dep.Containers[1].Clone())
	returnedDep.Containers[1].Image.Id = "xyz"
	returnedDep.Containers[1].Image.Name = &storage.ImageName{
		FullName: "fullname",
	}
	suite.storage.EXPECT().Get(ctx, "id").Return(returnedDep, true, nil)
	suite.NoError(ds.mergeCronJobs(ctx, dep))
	suite.Equal(expectedDep, dep)

	// Fill in missing last container value since names match
	dep.Containers[1].Image.Name = returnedDep.Containers[1].Image.Name
	expectedDep.Containers[1].Image.Name = returnedDep.Containers[1].Image.Name
	expectedDep.Containers[1].Image.Id = "xyz"
	suite.storage.EXPECT().Get(ctx, "id").Return(returnedDep, true, nil)
	suite.NoError(ds.mergeCronJobs(ctx, dep))
	suite.Equal(expectedDep, dep)
}
