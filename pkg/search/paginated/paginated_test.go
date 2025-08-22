package paginated

import (
	"fmt"
	"math"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/mocks"
	"github.com/stretchr/testify/assert"
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

// Test PageResults function
func TestPageResults(t *testing.T) {
	tests := []struct {
		name        string
		results     []search.Result
		query       *v1.Query
		expected    []search.Result
		expectError bool
	}{
		{
			name:     "no pagination - return all results",
			results:  fakeResults,
			query:    &v1.Query{},
			expected: fakeResults,
		},
		{
			name:     "nil pagination - return all results",
			results:  fakeResults,
			query:    &v1.Query{Pagination: nil},
			expected: fakeResults,
		},
		{
			name:    "offset only",
			results: fakeResults,
			query: &v1.Query{
				Pagination: &v1.QueryPagination{Offset: 2},
			},
			expected: fakeResults[2:],
		},
		{
			name:    "limit only",
			results: fakeResults,
			query: &v1.Query{
				Pagination: &v1.QueryPagination{Limit: 3},
			},
			expected: fakeResults[:3],
		},
		{
			name:    "offset and limit",
			results: fakeResults,
			query: &v1.Query{
				Pagination: &v1.QueryPagination{Offset: 1, Limit: 2},
			},
			expected: fakeResults[1:3],
		},
		{
			name:    "offset beyond results",
			results: fakeResults,
			query: &v1.Query{
				Pagination: &v1.QueryPagination{Offset: 10},
			},
			expected: nil,
		},
		{
			name:    "empty results",
			results: []search.Result{},
			query: &v1.Query{
				Pagination: &v1.QueryPagination{Offset: 0, Limit: 5},
			},
			expected: nil,
		},
		{
			name:    "negative offset",
			results: fakeResults,
			query: &v1.Query{
				Pagination: &v1.QueryPagination{Offset: -1, Limit: 2},
			},
			expected: fakeResults[:2], // Should treat negative offset as 0
		},
		{
			name:    "negative limit",
			results: fakeResults,
			query: &v1.Query{
				Pagination: &v1.QueryPagination{Offset: 1, Limit: -1},
			},
			expected: fakeResults[1:], // Should treat negative limit as unlimited
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := PageResults(tt.results, tt.query)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// Test FillPagination function
func TestFillPagination(t *testing.T) {
	tests := []struct {
		name            string
		query           *v1.Query
		pagination      *v1.Pagination
		maxLimit        int32
		expectedLimit   int32
		expectedOffset  int32
		expectedSortOps int
	}{
		{
			name:            "basic pagination",
			query:           &v1.Query{},
			pagination:      &v1.Pagination{Limit: 10, Offset: 5},
			maxLimit:        100,
			expectedLimit:   10,
			expectedOffset:  5,
			expectedSortOps: 0,
		},
		{
			name:            "limit exceeds max - should cap to max",
			query:           &v1.Query{},
			pagination:      &v1.Pagination{Limit: 200, Offset: 0},
			maxLimit:        100,
			expectedLimit:   100,
			expectedOffset:  0,
			expectedSortOps: 0,
		},
		{
			name:            "zero limit - should use max",
			query:           &v1.Query{},
			pagination:      &v1.Pagination{Limit: 0, Offset: 10},
			maxLimit:        50,
			expectedLimit:   50,
			expectedOffset:  10,
			expectedSortOps: 0,
		},
		{
			name:  "with sort options",
			query: &v1.Query{},
			pagination: &v1.Pagination{
				Limit:  25,
				Offset: 0,
				SortOptions: []*v1.SortOption{
					{Field: search.Cluster.String(), Reversed: true},
					{Field: search.Namespace.String(), Reversed: false},
				},
			},
			maxLimit:        100,
			expectedLimit:   25,
			expectedOffset:  0,
			expectedSortOps: 2,
		},
		{
			name:  "with legacy sort option",
			query: &v1.Query{},
			pagination: &v1.Pagination{
				Limit:      15,
				Offset:     5,
				SortOption: &v1.SortOption{Field: search.Priority.String(), Reversed: true},
			},
			maxLimit:        100,
			expectedLimit:   15,
			expectedOffset:  5,
			expectedSortOps: 1,
		},
		{
			name:  "sort options take precedence over legacy sort option",
			query: &v1.Query{},
			pagination: &v1.Pagination{
				Limit:  20,
				Offset: 0,
				SortOptions: []*v1.SortOption{
					{Field: search.Cluster.String()},
				},
				SortOption: &v1.SortOption{Field: search.Priority.String()},
			},
			maxLimit:        100,
			expectedLimit:   20,
			expectedOffset:  0,
			expectedSortOps: 1, // Should only have the SortOptions, not SortOption
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			FillPagination(tt.query, tt.pagination, tt.maxLimit)

			assert.NotNil(t, tt.query.Pagination)
			assert.Equal(t, tt.expectedLimit, tt.query.Pagination.GetLimit())
			assert.Equal(t, tt.expectedOffset, tt.query.Pagination.GetOffset())
			assert.Len(t, tt.query.Pagination.GetSortOptions(), tt.expectedSortOps)
		})
	}
}

// Test FillPaginationV2 function
func TestFillPaginationV2(t *testing.T) {
	tests := []struct {
		name            string
		query           *v1.Query
		pagination      *v2.Pagination
		maxLimit        int32
		expectedLimit   int32
		expectedOffset  int32
		expectedSortOps int
	}{
		{
			name:            "basic pagination v2",
			query:           &v1.Query{},
			pagination:      &v2.Pagination{Limit: 15, Offset: 3},
			maxLimit:        50,
			expectedLimit:   15,
			expectedOffset:  3,
			expectedSortOps: 0,
		},
		{
			name:  "with v2 sort options",
			query: &v1.Query{},
			pagination: &v2.Pagination{
				Limit:  30,
				Offset: 0,
				SortOptions: []*v2.SortOption{
					{Field: search.CVE.String(), Reversed: false},
				},
			},
			maxLimit:        100,
			expectedLimit:   30,
			expectedOffset:  0,
			expectedSortOps: 1,
		},
		{
			name:  "with v2 aggregate sort option",
			query: &v1.Query{},
			pagination: &v2.Pagination{
				Limit:  20,
				Offset: 5,
				SortOptions: []*v2.SortOption{
					{
						Field:    search.CVE.String(),
						Reversed: true,
						AggregateBy: &v2.AggregateBy{
							AggrFunc: v2.Aggregation_COUNT,
							Distinct: true,
						},
					},
				},
			},
			maxLimit:        100,
			expectedLimit:   20,
			expectedOffset:  5,
			expectedSortOps: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			FillPaginationV2(tt.query, tt.pagination, tt.maxLimit)

			assert.NotNil(t, tt.query.Pagination)
			assert.Equal(t, tt.expectedLimit, tt.query.Pagination.GetLimit())
			assert.Equal(t, tt.expectedOffset, tt.query.Pagination.GetOffset())
			assert.Len(t, tt.query.Pagination.GetSortOptions(), tt.expectedSortOps)
		})
	}
}

// Test FillDefaultSortOption function
func TestFillDefaultSortOption(t *testing.T) {
	defaultSort := &v1.QuerySortOption{
		Field:    search.ViolationTime.String(),
		Reversed: true,
	}

	tests := []struct {
		name     string
		query    *v1.Query
		expected *v1.Query
	}{
		{
			name:  "nil query - should create with default sort",
			query: nil,
			expected: &v1.Query{
				Pagination: &v1.QueryPagination{
					SortOptions: []*v1.QuerySortOption{defaultSort},
				},
			},
		},
		{
			name:  "query without pagination - should add pagination with default sort",
			query: &v1.Query{},
			expected: &v1.Query{
				Pagination: &v1.QueryPagination{
					SortOptions: []*v1.QuerySortOption{defaultSort},
				},
			},
		},
		{
			name: "query with pagination but no sort options - should add default sort",
			query: &v1.Query{
				Pagination: &v1.QueryPagination{
					Limit:  10,
					Offset: 5,
				},
			},
			expected: &v1.Query{
				Pagination: &v1.QueryPagination{
					Limit:       10,
					Offset:      5,
					SortOptions: []*v1.QuerySortOption{defaultSort},
				},
			},
		},
		{
			name: "query with existing sort options - should not add default",
			query: &v1.Query{
				Pagination: &v1.QueryPagination{
					SortOptions: []*v1.QuerySortOption{
						{Field: search.Cluster.String()},
					},
				},
			},
			expected: &v1.Query{
				Pagination: &v1.QueryPagination{
					SortOptions: []*v1.QuerySortOption{
						{Field: search.Cluster.String()},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FillDefaultSortOption(tt.query, defaultSort)
			protoassert.Equal(t, tt.expected, result)

			// Verify original query wasn't modified (unless it was nil)
			if tt.query != nil {
				assert.NotSame(t, tt.query, result)
			}
		})
	}
}

// Test PaginateSlice function
func TestPaginateSlice(t *testing.T) {
	testSlice := []string{"a", "b", "c", "d", "e"}

	tests := []struct {
		name     string
		offset   int
		limit    int
		slice    []string
		expected []string
	}{
		{
			name:     "normal pagination",
			offset:   1,
			limit:    2,
			slice:    testSlice,
			expected: []string{"b", "c"},
		},
		{
			name:     "zero offset and limit",
			offset:   0,
			limit:    0,
			slice:    testSlice,
			expected: testSlice,
		},
		{
			name:     "offset beyond slice length",
			offset:   10,
			limit:    3,
			slice:    testSlice,
			expected: nil,
		},
		{
			name:     "negative offset",
			offset:   -1,
			limit:    2,
			slice:    testSlice,
			expected: []string{"a", "b"},
		},
		{
			name:     "negative limit",
			offset:   2,
			limit:    -1,
			slice:    testSlice,
			expected: []string{"c", "d", "e"},
		},
		{
			name:     "empty slice",
			offset:   0,
			limit:    5,
			slice:    []string{},
			expected: nil,
		},
		{
			name:     "limit exceeds remaining elements",
			offset:   3,
			limit:    5,
			slice:    testSlice,
			expected: []string{"d", "e"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PaginateSlice(tt.offset, tt.limit, tt.slice)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test GetViolationTimeSortOption function
func TestGetViolationTimeSortOption(t *testing.T) {
	sortOption := GetViolationTimeSortOption()

	assert.NotNil(t, sortOption)
	assert.Equal(t, search.ViolationTime.String(), sortOption.GetField())
	assert.True(t, sortOption.GetReversed())
}

// Test helper functions
func TestToQuerySortOption(t *testing.T) {
	// Test basic sort option conversion
	sortOption := &v1.SortOption{
		Field:    search.Cluster.String(),
		Reversed: true,
	}

	result := toQuerySortOption(sortOption)
	assert.Equal(t, sortOption.GetField(), result.GetField())
	assert.Equal(t, sortOption.GetReversed(), result.GetReversed())
	assert.Nil(t, result.GetAggregateBy())

	// Test with aggregate
	sortOptionWithAggregate := &v1.SortOption{
		Field:    search.CVE.String(),
		Reversed: false,
		AggregateBy: &v1.AggregateBy{
			AggrFunc: v1.Aggregation_COUNT,
			Distinct: true,
		},
	}

	resultWithAggregate := toQuerySortOption(sortOptionWithAggregate)
	assert.Equal(t, sortOptionWithAggregate.GetField(), resultWithAggregate.GetField())
	assert.Equal(t, sortOptionWithAggregate.GetReversed(), resultWithAggregate.GetReversed())
	assert.NotNil(t, resultWithAggregate.GetAggregateBy())
	assert.Equal(t, v1.Aggregation_COUNT, resultWithAggregate.GetAggregateBy().GetAggrFunc())
	assert.True(t, resultWithAggregate.GetAggregateBy().GetDistinct())
}

func TestToQuerySortOptionV2(t *testing.T) {
	// Test basic v2 sort option conversion
	sortOption := &v2.SortOption{
		Field:    search.Namespace.String(),
		Reversed: false,
	}

	result := toQuerySortOptionV2(sortOption)
	assert.Equal(t, sortOption.GetField(), result.GetField())
	assert.Equal(t, sortOption.GetReversed(), result.GetReversed())
	assert.Nil(t, result.GetAggregateBy())

	// Test with v2 aggregate
	sortOptionWithAggregate := &v2.SortOption{
		Field:    search.CVE.String(),
		Reversed: true,
		AggregateBy: &v2.AggregateBy{
			AggrFunc: v2.Aggregation_MAX,
			Distinct: false,
		},
	}

	resultWithAggregate := toQuerySortOptionV2(sortOptionWithAggregate)
	assert.Equal(t, sortOptionWithAggregate.GetField(), resultWithAggregate.GetField())
	assert.Equal(t, sortOptionWithAggregate.GetReversed(), resultWithAggregate.GetReversed())
	assert.NotNil(t, resultWithAggregate.GetAggregateBy())
	assert.Equal(t, v1.Aggregation_MAX, resultWithAggregate.GetAggregateBy().GetAggrFunc())
	assert.False(t, resultWithAggregate.GetAggregateBy().GetDistinct())
}

func TestConvertV2AggregateByToV1(t *testing.T) {
	v2Aggregate := &v2.AggregateBy{
		AggrFunc: v2.Aggregation_MIN,
		Distinct: true,
	}

	result := convertV2AggregateByToV1(v2Aggregate)
	assert.NotNil(t, result)
	assert.Equal(t, v1.Aggregation_MIN, result.GetAggrFunc())
	assert.True(t, result.GetDistinct())
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
