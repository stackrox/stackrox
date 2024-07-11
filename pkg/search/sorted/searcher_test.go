package sorted

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	searchMocks "github.com/stackrox/rox/pkg/search/mocks"
	"github.com/stackrox/rox/pkg/search/sorted/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

var fakeResults = []search.Result{
	{
		ID: "r1",
	},
	{
		ID: "r2",
	},
	{
		ID: "r3",
	},
	{
		ID: "r4",
	},
	{
		ID: "r5",
	},
}

func TestSorted(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(sortedTestSuite))
}

type sortedTestSuite struct {
	suite.Suite

	mockSearcher *searchMocks.MockSearcher
	mockRanker   *mocks.MockRanker
	mockCtrl     *gomock.Controller
}

func (s *sortedTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockSearcher = searchMocks.NewMockSearcher(s.mockCtrl)
	s.mockRanker = mocks.NewMockRanker(s.mockCtrl)
}

func (s *sortedTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *sortedTestSuite) TestHandlesSorting() {
	s.mockSearcher.EXPECT().Search(gomock.Any(), gomock.Any()).Return(fakeResults, nil)

	s.mockRanker.EXPECT().GetRankForID(fakeResults[0].ID).AnyTimes().Return(int64(2))
	s.mockRanker.EXPECT().GetRankForID(fakeResults[1].ID).AnyTimes().Return(int64(1))
	s.mockRanker.EXPECT().GetRankForID(fakeResults[2].ID).AnyTimes().Return(int64(0))
	s.mockRanker.EXPECT().GetRankForID(fakeResults[3].ID).AnyTimes().Return(int64(3))
	s.mockRanker.EXPECT().GetRankForID(fakeResults[4].ID).AnyTimes().Return(int64(4))

	expectedSorted := []search.Result{
		fakeResults[2],
		fakeResults[1],
		fakeResults[0],
		fakeResults[3],
		fakeResults[4],
	}

	results, err := Searcher(s.mockSearcher, search.Priority, s.mockRanker).Search(context.Background(), &v1.Query{
		Pagination: &v1.QueryPagination{
			Limit:  0,
			Offset: 0,
			SortOptions: []*v1.QuerySortOption{
				{
					Field:    search.Priority.String(),
					Reversed: false,
				},
			},
		},
	})
	s.NoError(err, "expected no error, should return nil without access")
	s.Equal(expectedSorted, results, "with no pagination the result should be the same as the search output")
}

func (s *sortedTestSuite) TestSkipsNonMatching() {
	s.mockSearcher.EXPECT().Search(gomock.Any(), gomock.Any()).Return(fakeResults, nil)

	results, err := Searcher(s.mockSearcher, search.Priority, s.mockRanker).Search(context.Background(), &v1.Query{
		Pagination: &v1.QueryPagination{
			Limit:  0,
			Offset: 0,
			SortOptions: []*v1.QuerySortOption{
				{
					Field:    search.CVE.String(),
					Reversed: false,
				},
			},
		},
	})
	s.NoError(err, "expected no error, should return nil without access")
	s.Equal(fakeResults, results, "with no pagination the result should be the same as the search output")
}
