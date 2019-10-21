package datastore

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	deploymentMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	nsIndexMocks "github.com/stackrox/rox/central/namespace/index/mocks"
	nsMocks "github.com/stackrox/rox/central/namespace/store/mocks"
	"github.com/stackrox/rox/central/ranking"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

func TestClusterDataStore(t *testing.T) {
	suite.Run(t, new(NamespaceDataStoreTestSuite))
}

type NamespaceDataStoreTestSuite struct {
	suite.Suite

	hasNoneCtx  context.Context
	hasReadCtx  context.Context
	hasWriteCtx context.Context

	ns                  *nsMocks.MockStore
	indexer             *nsIndexMocks.MockIndexer
	nsDataStore         DataStore
	deploymentDataStore *deploymentMocks.MockDataStore
	mockCtrl            *gomock.Controller
}

func (suite *NamespaceDataStoreTestSuite) SetupTest() {
	suite.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	suite.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Namespace)))
	suite.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Namespace)))

	suite.mockCtrl = gomock.NewController(suite.T())
	suite.ns = nsMocks.NewMockStore(suite.mockCtrl)
	suite.indexer = nsIndexMocks.NewMockIndexer(suite.mockCtrl)

	suite.deploymentDataStore = deploymentMocks.NewMockDataStore(suite.mockCtrl)

	suite.ns.EXPECT().GetNamespaces().Return(([]*storage.NamespaceMetadata)(nil), nil)
	suite.indexer.EXPECT().AddNamespaceMetadatas(nil).Return(nil)

	var err error
	suite.nsDataStore, err = New(suite.ns,
		suite.indexer,
		suite.deploymentDataStore,
	)
	suite.NoError(err)
}

func (suite *NamespaceDataStoreTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *NamespaceDataStoreTestSuite) TestNamespacePriority() {
	ranker := ranking.DeploymentRanker()
	ranker.Add("dep1", 1.0)
	ranker.Add("dep2", 2.0)

	ranker.Add("dep3", 3.0)
	ranker.Add("dep4", 4.0)

	ranker.Add("dep5", 10.0)

	cases := []struct {
		ns               *storage.NamespaceMetadata
		deployments      []search.Result
		expectedPriority int64
	}{
		{
			ns: &storage.NamespaceMetadata{
				Id: "test1",
			},
			deployments: []search.Result{
				{
					ID: "dep1",
				},
				{
					ID: "dep2",
				},
			},
			expectedPriority: 3,
		},
		{
			ns: &storage.NamespaceMetadata{
				Id: "test2",
			},
			deployments: []search.Result{
				{
					ID: "dep3",
				},
				{
					ID: "dep4",
				},
			},
			expectedPriority: 2,
		},
		{
			ns: &storage.NamespaceMetadata{
				Id: "test3",
			},
			deployments: []search.Result{
				{
					ID: "dep5",
				},
			},
			expectedPriority: 1,
		},
	}

	deploymentReadCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Deployment),
		))

	var expectedNamespaces []*storage.NamespaceMetadata
	for _, c := range cases {
		expectedNamespaces = append(expectedNamespaces, c.ns)
	}

	suite.ns.EXPECT().GetNamespaces().Return(expectedNamespaces, nil)
	for _, c := range cases {
		suite.deploymentDataStore.EXPECT().Search(deploymentReadCtx,
			search.NewQueryBuilder().AddExactMatches(search.NamespaceID, c.ns.GetId()).ProtoQuery()).
			Return(c.deployments, nil)
	}

	actualNamespaces, err := suite.nsDataStore.GetNamespaces(suite.hasReadCtx)
	suite.Nil(err)

	for i, c := range cases {
		suite.Equal(c.expectedPriority, actualNamespaces[i].GetPriority())
	}
}
