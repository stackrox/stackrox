package paginated

import (
	"context"
	"fmt"
	"math"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestPagination(t *testing.T) {
	suite.Run(t, new(paginationTestSuite))
}

type paginationTestSuite struct {
	suite.Suite

	mockSearcher *mocks.MockSearcher

	mockCtrl *gomock.Controller
}

func (s *paginationTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockSearcher = mocks.NewMockSearcher(s.mockCtrl)
}

func (s *paginationTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *paginationTestSuite) TestHandlesNoPagination() {
	s.mockSearcher.EXPECT().Search(gomock.Any(), gomock.Any()).Return(fakeResults, nil)

	results, err := Paginated(s.mockSearcher).Search(context.Background(), &v1.Query{})
	s.NoError(err, "expected no error, should return nil without access")
	s.Equal(fakeResults, results, "with no pagination the result should be the same as the search output")
}

func (s *paginationTestSuite) TestHandlesNoOffset() {
	s.mockSearcher.EXPECT().Search(gomock.Any(), gomock.Eq(&v1.Query{
		Pagination: &v1.QueryPagination{
			Limit: 0,
		},
	})).Return(fakeResults, nil)

	results, err := Paginated(s.mockSearcher).Search(context.Background(), &v1.Query{
		Pagination: &v1.QueryPagination{
			Limit: 1,
		},
	})
	s.NoError(err, "expected no error, should return nil without access")
	s.Equal(fakeResults[:1], results, "results should use limit")
}

func (s *paginationTestSuite) TestHandlesNoLimit() {
	s.mockSearcher.EXPECT().Search(gomock.Any(), gomock.Eq(&v1.Query{
		Pagination: &v1.QueryPagination{
			Offset: 0,
		},
	})).Return(fakeResults, nil)

	results, err := Paginated(s.mockSearcher).Search(context.Background(), &v1.Query{
		Pagination: &v1.QueryPagination{
			Offset: 1,
		},
	})
	s.NoError(err, "expected no error, should return nil without access")
	s.Equal(fakeResults[1:], results, "results should use offset")
}

func (s *paginationTestSuite) TestHandlesOffSetAndLimit() {
	s.mockSearcher.EXPECT().Search(gomock.Any(), gomock.Eq(&v1.Query{
		Pagination: &v1.QueryPagination{
			Offset: 0,
			Limit:  0,
		},
	})).Return(fakeResults, nil)

	results, err := Paginated(s.mockSearcher).Search(context.Background(), &v1.Query{
		Pagination: &v1.QueryPagination{
			Offset: 1,
			Limit:  3,
		},
	})
	s.NoError(err, "expected no error, should return nil without access")
	s.Equal(fakeResults[1:4], results, "results should use offset and limit")
}

func (s *paginationTestSuite) TestGetLimit() {
	const whenUnlimited = 10
	for given, expected := range map[int32]int32{
		0:                 whenUnlimited,
		math.MaxInt32:     whenUnlimited,
		math.MinInt32:     whenUnlimited,
		-1:                whenUnlimited,
		1:                 1,
		math.MaxInt32 - 1: math.MaxInt32 - 1,
		whenUnlimited:     whenUnlimited,
		whenUnlimited + 1: whenUnlimited + 1,
		whenUnlimited - 1: whenUnlimited - 1,
	} {
		s.T().Run(fmt.Sprintf("%d %d", given, expected), func(t *testing.T) {
			actual := GetLimit(given, whenUnlimited)
			s.Equal(expected, actual)
		})
	}
}

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
