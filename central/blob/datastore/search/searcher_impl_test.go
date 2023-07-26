package search

import (
	"context"
	"errors"
	"fmt"
	"testing"

	mockIndex "github.com/stackrox/rox/central/blob/datastore/index/mocks"
	mockStore "github.com/stackrox/rox/central/blob/datastore/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestBlobSearch(t *testing.T) {
	suite.Run(t, new(BlobSearchTestSuite))
}

type BlobSearchTestSuite struct {
	suite.Suite

	controller *gomock.Controller
	indexer    *mockIndex.MockIndexer
	store      *mockStore.MockStore

	searcher    Searcher
	allowAllCtx context.Context
}

func (suite *BlobSearchTestSuite) SetupTest() {
	suite.allowAllCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Administration),
		))
	suite.controller = gomock.NewController(suite.T())
	suite.indexer = mockIndex.NewMockIndexer(suite.controller)
	suite.store = mockStore.NewMockStore(suite.controller)
	searcher := New(suite.store, suite.indexer)
	suite.searcher = searcher
}

func (suite *BlobSearchTestSuite) TearDownTest() {
	suite.controller.Finish()
}

func getMockBlobResults(num int) ([]search.Result, []*storage.Blob, []string) {
	var (
		dbResults    []*storage.Blob
		indexResults []search.Result
		ids          []string
	)
	for i := 0; i < num; i++ {
		blob := &storage.Blob{
			Name: fmt.Sprintf("/path/path%d", i),
		}
		dbResults = append(dbResults, blob)
		indexResults = append(indexResults, search.Result{ID: blob.GetName()})
		ids = append(ids, blob.GetName())
	}

	return indexResults, dbResults, ids
}

func (suite *BlobSearchTestSuite) TestErrors() {
	q := search.EmptyQuery()
	someError := errors.New("this is a test error")
	suite.indexer.EXPECT().Search(gomock.Any(), q).Times(3).Return(nil, someError)
	ids, err := suite.searcher.SearchIDs(suite.allowAllCtx, q)
	suite.Equal(someError, err)
	suite.Nil(ids)

	results, err := suite.searcher.Search(suite.allowAllCtx, q)
	suite.Equal(someError, err)
	suite.Nil(results)

	ct, err := suite.searcher.Search(suite.allowAllCtx, q)
	suite.Equal(someError, err)
	suite.Zero(ct)

	suite.store.EXPECT().GetMetadataByQuery(suite.allowAllCtx, q).Return(nil, someError)
	blobs, err := suite.searcher.SearchMetadata(suite.allowAllCtx, q)
	suite.Error(err)
	suite.Nil(blobs)
}

func (suite *BlobSearchTestSuite) TestSearchForAll() {
	q := search.EmptyQuery()
	var emptyList []search.Result
	suite.indexer.EXPECT().Search(gomock.Any(), q).Return(emptyList, nil)
	// It's an implementation detail whether this method is called, so allow but don't require it.
	blobIDs, err := suite.searcher.SearchIDs(suite.allowAllCtx, q)
	suite.NoError(err)
	suite.Empty(blobIDs)

	indexResults, blobs, ids := getMockBlobResults(3)
	suite.indexer.EXPECT().Search(gomock.Any(), q).Return(indexResults, nil)
	blobIDs, err = suite.searcher.SearchIDs(suite.allowAllCtx, q)
	suite.NoError(err)
	suite.Equal(ids, blobIDs)

	suite.store.EXPECT().GetMetadataByQuery(suite.allowAllCtx, testutils.AssertionMatcher(assert.Empty)).Times(1).Return(nil, nil)
	results, err := suite.searcher.SearchMetadata(suite.allowAllCtx, q)
	suite.NoError(err)
	suite.Empty(results)

	suite.store.EXPECT().GetMetadataByQuery(suite.allowAllCtx, testutils.AssertionMatcher(assert.Empty)).Times(1).Return(blobs, nil)
	results, err = suite.searcher.SearchMetadata(suite.allowAllCtx, q)
	suite.NoError(err)
	suite.Equal(blobs, results)
}
