package search

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	indexMock "github.com/stackrox/rox/central/processindicator/index/mocks"
	storeMock "github.com/stackrox/rox/central/processindicator/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/role/resources"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

func TestIndicatorSearch(t *testing.T) {
	suite.Run(t, new(IndicatorSearchTestSuite))
}

type IndicatorSearchTestSuite struct {
	hasNoneCtx  context.Context
	hasReadCtx  context.Context
	hasWriteCtx context.Context

	suite.Suite

	searcher Searcher

	indexer *indexMock.MockIndexer
	storage *storeMock.MockStore

	mockCtrl *gomock.Controller
}

func (suite *IndicatorSearchTestSuite) SetupSuite() {
	suite.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	suite.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Indicator)))
	suite.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Indicator)))

	suite.mockCtrl = gomock.NewController(suite.T())

	suite.indexer = indexMock.NewMockIndexer(suite.mockCtrl)
	suite.storage = storeMock.NewMockStore(suite.mockCtrl)

	suite.searcher = New(suite.storage, suite.indexer)
}

func (suite *IndicatorSearchTestSuite) TearDownSuite() {
	suite.mockCtrl.Finish()
}

func (suite *IndicatorSearchTestSuite) TestEnforcesSearch() {
	suite.indexer.EXPECT().Search(gomock.Any()).Return([]search.Result{{ID: "hgdskdf"}}, nil)

	processIndicators, err := suite.searcher.Search(suite.hasNoneCtx, search.EmptyQuery())
	suite.NoError(err, "expected no error, should return nil without access")
	suite.Nil(processIndicators, "expected return value to be nil")
}

func (suite *IndicatorSearchTestSuite) TestAllowsSearch() {
	suite.indexer.EXPECT().Search(gomock.Any()).Return([]search.Result{{ID: "hgdskdf"}}, nil)

	processIndicators, err := suite.searcher.Search(suite.hasReadCtx, search.EmptyQuery())
	suite.NoError(err, "expected no error trying to read with permissions")
	suite.NotEmpty(processIndicators)

	suite.indexer.EXPECT().Search(gomock.Any()).Return([]search.Result{{ID: "hgdskdf"}}, nil)

	processIndicators, err = suite.searcher.Search(suite.hasWriteCtx, search.EmptyQuery())
	suite.NoError(err, "expected no error trying to read with permissions")
	suite.NotEmpty(processIndicators)
}

func (suite *IndicatorSearchTestSuite) TestEnforcesSearchRaw() {
	suite.indexer.EXPECT().Search(gomock.Any()).Return([]search.Result{{ID: "hgdskdf"}}, nil)
	suite.storage.EXPECT().GetMany(gomock.Any(), gomock.Any()).Return([]*storage.ProcessIndicator{}, []int{}, nil)

	processIndicators, err := suite.searcher.SearchRawProcessIndicators(suite.hasNoneCtx, search.EmptyQuery())
	suite.NoError(err, "expected no error, should return nil without access")
	suite.Empty(processIndicators, "expected return value to be nil")
}

func (suite *IndicatorSearchTestSuite) TestAllowsSearchRaw() {
	suite.indexer.EXPECT().Search(gomock.Any()).Return([]search.Result{{ID: ""}}, nil)
	suite.storage.EXPECT().GetMany(gomock.Any(), gomock.Any()).Return([]*storage.ProcessIndicator{{}}, []int{}, nil)

	processIndicators, err := suite.searcher.SearchRawProcessIndicators(suite.hasReadCtx, search.EmptyQuery())
	suite.NoError(err, "expected no error trying to read with permissions")
	suite.NotEmpty(processIndicators)

	suite.indexer.EXPECT().Search(gomock.Any()).Return([]search.Result{{ID: "hgdskdf"}}, nil)
	suite.storage.EXPECT().GetMany(gomock.Any(), gomock.Any()).Return([]*storage.ProcessIndicator{{}}, []int{}, nil)

	processIndicators, err = suite.searcher.SearchRawProcessIndicators(suite.hasWriteCtx, search.EmptyQuery())
	suite.NoError(err, "expected no error trying to read with permissions")
	suite.NotEmpty(processIndicators)
}
