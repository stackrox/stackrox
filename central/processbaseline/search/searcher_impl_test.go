package search

import (
	"context"
	"errors"
	"testing"

	mockIndex "github.com/stackrox/rox/central/processbaseline/index/mocks"
	mockStore "github.com/stackrox/rox/central/processbaseline/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestProcessBaselineSearch(t *testing.T) {
	suite.Run(t, new(ProcessBaselineSearchTestSuite))
}

func getFakeSearchResults(num int) ([]search.Result, []*storage.ProcessBaseline) {
	var dbResults []*storage.ProcessBaseline
	var indexResults []search.Result
	for i := 0; i < num; i++ {
		baseline := fixtures.GetProcessBaselineWithID()
		dbResults = append(dbResults, baseline)
		fakeResult := search.Result{ID: baseline.Id}
		indexResults = append(indexResults, fakeResult)
	}

	return indexResults, dbResults
}

type ProcessBaselineSearchTestSuite struct {
	suite.Suite

	controller *gomock.Controller
	indexer    *mockIndex.MockIndexer
	store      *mockStore.MockStore

	searcher    Searcher
	allowAllCtx context.Context
}

func (suite *ProcessBaselineSearchTestSuite) SetupTest() {
	suite.allowAllCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.DeploymentExtension),
		))
	suite.controller = gomock.NewController(suite.T())
	suite.indexer = mockIndex.NewMockIndexer(suite.controller)
	suite.store = mockStore.NewMockStore(suite.controller)
	searcher, err := New(suite.store, suite.indexer)

	suite.NoError(err)
	suite.searcher = searcher
}

func (suite *ProcessBaselineSearchTestSuite) TearDownTest() {
	suite.controller.Finish()
}

func (suite *ProcessBaselineSearchTestSuite) TestErrors() {
	q := search.EmptyQuery()
	someError := errors.New("this is a test error")
	suite.indexer.EXPECT().Search(gomock.Any(), q).Return(nil, someError)
	results, err := suite.searcher.SearchRawProcessBaselines(suite.allowAllCtx, q)
	suite.Equal(someError, err)
	suite.Nil(results)

	indexResults, _ := getFakeSearchResults(1)
	suite.indexer.EXPECT().Search(gomock.Any(), q).Return(indexResults, nil)
	suite.store.EXPECT().GetMany(suite.allowAllCtx, search.ResultsToIDs(indexResults)).Return(nil, nil, someError)
	results, err = suite.searcher.SearchRawProcessBaselines(suite.allowAllCtx, q)
	suite.Error(err)
	suite.Nil(results)
}

func (suite *ProcessBaselineSearchTestSuite) TestSearchForAll() {
	q := search.EmptyQuery()
	var emptyList []search.Result
	suite.indexer.EXPECT().Search(gomock.Any(), q).Return(emptyList, nil)
	// It's an implementation detail whether this method is called, so allow but don't require it.
	suite.store.EXPECT().GetMany(suite.allowAllCtx, testutils.AssertionMatcher(assert.Empty)).MinTimes(0).MaxTimes(1)
	results, err := suite.searcher.SearchRawProcessBaselines(suite.allowAllCtx, q)
	suite.NoError(err)
	suite.Empty(results)

	indexResults, dbResults := getFakeSearchResults(3)
	suite.indexer.EXPECT().Search(gomock.Any(), q).Return(indexResults, nil)
	suite.store.EXPECT().GetMany(suite.allowAllCtx, search.ResultsToIDs(indexResults)).Return(dbResults, nil, nil)
	results, err = suite.searcher.SearchRawProcessBaselines(suite.allowAllCtx, q)
	suite.NoError(err)
	suite.Equal(dbResults, results)
}
