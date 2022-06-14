package scoped

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/dackbox/keys/transformation"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/search/mocks"
	"github.com/stretchr/testify/suite"
)

func TestWithScoping(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(scopedSearcherTestSuite))
}

type scopedSearcherTestSuite struct {
	suite.Suite

	mockSearcher *mocks.MockSearcher

	mockCtrl *gomock.Controller
}

func (s *scopedSearcherTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockSearcher = mocks.NewMockSearcher(s.mockCtrl)
}

func (s *scopedSearcherTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *scopedSearcherTestSuite) TestScoping() {
	results := []search.Result{
		{
			ID: "1",
		},
		{
			ID: "2",
		},
		{
			ID: "3",
		},
	}

	expectedResults := []search.Result{
		{
			ID: "1",
		},
		{
			ID: "2",
		},
	}

	query := search.NewQueryBuilder().AddBools(search.Fixable, true).ProtoQuery()

	inputCtx := Context(context.Background(), Scope{
		ID:    "c1",
		Level: v1.SearchCategory_CLUSTERS,
	})

	s.mockSearcher.EXPECT().Search(
		inputCtx,
		search.ConjunctionQuery(
			query,
			search.NewQueryBuilder().AddDocIDs("1", "2").ProtoQuery(),
		),
	).Return(results[:2], nil)

	fakeClusterToCVETransform := func(_ context.Context, key []byte) [][]byte {
		if string(key) == "c1" {
			return [][]byte{[]byte("1"), []byte("2")}
		}
		s.Fail("unexpected cluster level scope ID")
		return nil
	}
	transformer := WithScoping(s.mockSearcher, testProvider{
		v1.SearchCategory_CLUSTERS: fakeClusterToCVETransform,
	})

	transformedResults, err := transformer.Search(inputCtx, query)
	s.NoError(err)
	s.Equal(expectedResults, transformedResults)
}

func (s *scopedSearcherTestSuite) TestNoScoping() {
	results := []search.Result{
		{
			ID: "1",
		},
		{
			ID: "2",
		},
		{
			ID: "3",
		},
	}

	expectedResults := []search.Result{
		{
			ID: "1",
		},
		{
			ID: "2",
		},
		{
			ID: "3",
		},
	}

	query := search.NewQueryBuilder().AddBools(search.Fixable, true).ProtoQuery()

	inputCtx := context.Background()

	s.mockSearcher.EXPECT().Search(inputCtx, query).Return(results, nil)

	fakeClusterToCVETransform := func(_ context.Context, key []byte) [][]byte {
		s.Fail("scope transformation should not have been invoked")
		return nil
	}
	transformer := WithScoping(s.mockSearcher, testProvider{
		v1.SearchCategory_CLUSTERS: fakeClusterToCVETransform,
	})

	transformedResults, err := transformer.Search(inputCtx, query)
	s.NoError(err)
	s.Equal(expectedResults, transformedResults)
}

type testProvider map[v1.SearchCategory]transformation.OneToMany

func (tp testProvider) Get(sc v1.SearchCategory) transformation.OneToMany {
	return tp[sc]
}
