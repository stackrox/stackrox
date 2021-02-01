package compound

import (
	"testing"

	"github.com/golang/mock/gomock"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	searchMocks "github.com/stackrox/rox/pkg/search/mocks"
	"github.com/stretchr/testify/suite"
)

func TestSorting(t *testing.T) {
	suite.Run(t, new(SortingTestSuite))
}

type SortingTestSuite struct {
	suite.Suite

	mockOptions1 *searchMocks.MockOptionsMap
	mockOptions2 *searchMocks.MockOptionsMap
	mockCtrl     *gomock.Controller
}

func (suite *SortingTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.mockOptions1 = searchMocks.NewMockOptionsMap(suite.mockCtrl)
	suite.mockOptions2 = searchMocks.NewMockOptionsMap(suite.mockCtrl)
}

func (suite *SortingTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *SortingTestSuite) TestHandlesEmpty() {
	spec := &searchRequestSpec{}
	actual, err := addSorting(spec, &v1.QueryPagination{}, []SearcherSpec{})
	suite.NotNil(err)
	suite.Equal((*searchRequestSpec)(nil), actual)
}

// When we have a single base search request, we can attach the sort directly to it.
func (suite *SortingTestSuite) TestHandlesBase() {
	searcherSpecs := []SearcherSpec{
		{
			Searcher: (search.Searcher)(nil),
			Options:  suite.mockOptions1,
		},
	}

	searchSpec := &searchRequestSpec{
		base: &baseRequestSpec{
			Spec:  &searcherSpecs[0],
			Query: &v1.Query{},
		},
	}

	pagination := &v1.QueryPagination{
		SortOptions: []*v1.QuerySortOption{
			{
				Field: "Deployment",
			},
		},
	}

	suite.mockOptions1.EXPECT().Get("Deployment").Return(nil, true)

	expectedSearchSpec := &searchRequestSpec{
		base: &baseRequestSpec{
			Spec: &searcherSpecs[0],
			Query: &v1.Query{
				Pagination: pagination,
			},
		},
	}

	actual, err := addSorting(searchSpec, pagination, searcherSpecs)
	suite.Nil(err)
	suite.Equal(expectedSearchSpec, actual)
}

// When the top level is an 'and'/'conjunction' query, and it has a base search request that matches our sort fields,
// we can just attach the sort to the matching base request.
func (suite *SortingTestSuite) TestHandlesAnd() {
	searcherSpecs := []SearcherSpec{
		{
			Searcher: (search.Searcher)(nil),
			Options:  suite.mockOptions1,
		},
		{
			Searcher: (search.Searcher)(nil),
			Options:  suite.mockOptions2,
		},
	}

	searchSpec := &searchRequestSpec{
		and: []*searchRequestSpec{
			{
				base: &baseRequestSpec{
					Spec:  &searcherSpecs[0],
					Query: &v1.Query{},
				},
			},
			{
				base: &baseRequestSpec{
					Spec:  &searcherSpecs[1],
					Query: &v1.Query{},
				},
			},
		},
	}

	pagination := &v1.QueryPagination{
		SortOptions: []*v1.QuerySortOption{
			{
				Field: "Deployment",
			},
		},
	}

	suite.mockOptions1.EXPECT().Get("Deployment").Return(nil, false)
	suite.mockOptions2.EXPECT().Get("Deployment").Return(nil, true)

	expectedSearchSpec := &searchRequestSpec{
		and: []*searchRequestSpec{
			{
				base: &baseRequestSpec{
					Spec:  &searcherSpecs[0],
					Query: &v1.Query{},
				},
			},
			{
				base: &baseRequestSpec{
					Spec: &searcherSpecs[1],
					Query: &v1.Query{
						Pagination: pagination,
					},
				},
			},
		},
	}

	actual, err := addSorting(searchSpec, pagination, searcherSpecs)
	suite.Nil(err)
	suite.Equal(expectedSearchSpec, actual)
}

// If we have a top level conjunction of base requests, but non of the specification matches our search, we need to add
// a new item to the conjunction to do the sorting.
func (suite *SortingTestSuite) TestHandlesAndWithoutMatch() {
	searcherSpecs := []SearcherSpec{
		{
			Searcher: (search.Searcher)(nil),
			Options:  suite.mockOptions1,
		},
		{
			Searcher: (search.Searcher)(nil),
			Options:  suite.mockOptions2,
		},
	}

	searchSpec := &searchRequestSpec{
		and: []*searchRequestSpec{
			{
				base: &baseRequestSpec{
					Spec:  &searcherSpecs[0],
					Query: &v1.Query{},
				},
			},
			{
				base: &baseRequestSpec{
					Spec:  &searcherSpecs[0],
					Query: &v1.Query{},
				},
			},
		},
	}

	pagination := &v1.QueryPagination{
		SortOptions: []*v1.QuerySortOption{
			{
				Field: "Deployment",
			},
		},
	}

	suite.mockOptions1.EXPECT().Get("Deployment").Return(nil, false).AnyTimes()
	suite.mockOptions2.EXPECT().Get("Deployment").Return(nil, true).AnyTimes()

	q := search.EmptyQuery()
	q.Pagination = pagination
	expectedSearchSpec := &searchRequestSpec{
		leftJoinWithRightOrder: &joinRequestSpec{
			left: &searchRequestSpec{
				and: []*searchRequestSpec{
					{
						base: &baseRequestSpec{
							Spec:  &searcherSpecs[0],
							Query: &v1.Query{},
						},
					},
					{
						base: &baseRequestSpec{
							Spec:  &searcherSpecs[0],
							Query: &v1.Query{},
						},
					},
				},
			},
			right: &searchRequestSpec{
				base: &baseRequestSpec{
					Spec:  &searcherSpecs[1],
					Query: q,
				},
			},
		},
	}

	actual, err := addSorting(searchSpec, pagination, searcherSpecs)
	suite.Nil(err)
	suite.Equal(expectedSearchSpec, actual)
}
