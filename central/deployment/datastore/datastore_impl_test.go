package datastore

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/analystnotes"
	"github.com/stackrox/rox/central/deployment/datastore/internal/processtagsstore"
	searcherMocks "github.com/stackrox/rox/central/deployment/datastore/internal/search/mocks"
	indexerMocks "github.com/stackrox/rox/central/deployment/index/mocks"
	storeMocks "github.com/stackrox/rox/central/deployment/store/mocks"
	"github.com/stackrox/rox/central/ranking"
	riskMocks "github.com/stackrox/rox/central/risk/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
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
	suite.filter = filter.NewFilter(5, []int{5, 4, 3, 2, 1})
}

func (suite *DeploymentDataStoreTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func getCommentKey(deploymentID string) *analystnotes.ProcessNoteKey {
	return &analystnotes.ProcessNoteKey{DeploymentID: deploymentID, ExecFilePath: "/bin/sh", ContainerName: "container"}
}

func (suite *DeploymentDataStoreTestSuite) TestTags() {
	testDB := testutils.DBForSuite(suite)
	defer testutils.TearDownDB(testDB)

	processTagsStorage := processtagsstore.New(testDB)
	datastore := newDatastoreImpl(suite.storage, processTagsStorage, suite.indexer, nil, nil, nil, nil,
		suite.riskStore, nil, suite.filter, ranking.NewRanker(),
		ranking.NewRanker(), ranking.NewRanker())

	suite.storage.EXPECT().Get(suite.ctx, "blah").Return(nil, false, nil)
	suite.NoError(datastore.AddTagsToProcessKey(suite.ctx, getCommentKey("blah"), []string{"new", "tag", "in-both"}))

	suite.storage.EXPECT().Get(suite.ctx, "exists").Return(&storage.Deployment{Id: "exists", ProcessTags: []string{"existing"}}, true, nil)
	suite.storage.EXPECT().Upsert(suite.ctx, &storage.Deployment{Id: "exists", ProcessTags: []string{"existing", "in-both", "new", "tag"}, Priority: 1}).Return(nil)
	suite.NoError(datastore.AddTagsToProcessKey(suite.ctx, getCommentKey("exists"), []string{"new", "tag", "in-both"}))
	mutatedExistsKey := getCommentKey("exists")
	mutatedExistsKey.ExecFilePath = "MUTATED"
	suite.storage.EXPECT().Get(suite.ctx, "exists").Return(&storage.Deployment{Id: "exists", ProcessTags: []string{"existing", "in-both", "new", "tag"}}, true, nil)
	suite.NoError(datastore.AddTagsToProcessKey(suite.ctx, mutatedExistsKey, []string{"in-both"}))

	tags, err := datastore.GetTagsForProcessKey(suite.ctx, getCommentKey("exists"))
	suite.Require().NoError(err)
	suite.Equal([]string{"in-both", "new", "tag"}, tags)

	suite.storage.EXPECT().Get(suite.ctx, "exists").Return(&storage.Deployment{Id: "exists", ProcessTags: []string{"existing", "new", "tag", "in-both"}}, true, nil)
	suite.storage.EXPECT().Upsert(suite.ctx, &storage.Deployment{Id: "exists", ProcessTags: []string{"existing", "in-both"}, Priority: 1}).Return(nil)
	suite.NoError(datastore.RemoveTagsFromProcessKey(suite.ctx, getCommentKey("exists"), []string{"new", "tag", "in-both"}))
	tags, err = datastore.GetTagsForProcessKey(suite.ctx, getCommentKey("exists"))
	suite.Require().NoError(err)
	suite.Empty(tags)
}

func (suite *DeploymentDataStoreTestSuite) TestInitializeRanker() {
	clusterRanker := ranking.NewRanker()
	nsRanker := ranking.NewRanker()
	deploymentRanker := ranking.NewRanker()

	ds := newDatastoreImpl(suite.storage, nil, suite.indexer, suite.searcher, nil, nil, nil,
		suite.riskStore, nil, suite.filter, clusterRanker,
		nsRanker, deploymentRanker)

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
