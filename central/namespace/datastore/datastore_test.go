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
	"github.com/stretchr/testify/suite"
)

func TestNamespaceDataStore(t *testing.T) {
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

	suite.ns.EXPECT().Walk(gomock.Any(), gomock.Any()).Return(([]*storage.NamespaceMetadata)(nil), nil)
	suite.indexer.EXPECT().AddNamespaceMetadatas(nil).Return(nil)

	var err error
	suite.nsDataStore, err = New(suite.ns,
		nil,
		suite.indexer,
		suite.deploymentDataStore,
		ranking.NewRanker(),
		nil,
	)
	suite.NoError(err)
}

func (suite *NamespaceDataStoreTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}
