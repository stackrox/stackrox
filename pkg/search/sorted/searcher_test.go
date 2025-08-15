package sorted

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	searchMocks "github.com/stackrox/rox/pkg/search/mocks"
	"github.com/stackrox/rox/pkg/search/sorted/mocks"
	"github.com/stretchr/testify/assert"
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

func (s *sortedTestSuite) TestSearcherCount() {
	expectedCount := 42
	s.mockSearcher.EXPECT().Count(gomock.Any(), gomock.Any()).Return(expectedCount, nil)

	count, err := Searcher(s.mockSearcher, search.Priority, s.mockRanker).Count(context.Background(), &v1.Query{})
	s.NoError(err)
	s.Equal(expectedCount, count)
}

func (s *sortedTestSuite) TestHandlesSortingReversed() {
	s.mockSearcher.EXPECT().Search(gomock.Any(), gomock.Any()).Return(fakeResults, nil)

	s.mockRanker.EXPECT().GetRankForID(fakeResults[0].ID).AnyTimes().Return(int64(2))
	s.mockRanker.EXPECT().GetRankForID(fakeResults[1].ID).AnyTimes().Return(int64(1))
	s.mockRanker.EXPECT().GetRankForID(fakeResults[2].ID).AnyTimes().Return(int64(0))
	s.mockRanker.EXPECT().GetRankForID(fakeResults[3].ID).AnyTimes().Return(int64(3))
	s.mockRanker.EXPECT().GetRankForID(fakeResults[4].ID).AnyTimes().Return(int64(4))

	expectedSorted := []search.Result{
		fakeResults[4],
		fakeResults[3],
		fakeResults[0],
		fakeResults[1],
		fakeResults[2],
	}

	results, err := Searcher(s.mockSearcher, search.Priority, s.mockRanker).Search(context.Background(), &v1.Query{
		Pagination: &v1.QueryPagination{
			Limit:  0,
			Offset: 0,
			SortOptions: []*v1.QuerySortOption{
				{
					Field:    search.Priority.String(),
					Reversed: true,
				},
			},
		},
	})
	s.NoError(err)
	s.Equal(expectedSorted, results, "reversed sorting should return results in descending order")
}

// Test IsValidPriorityQuery function
func TestIsValidPriorityQuery(t *testing.T) {
	tests := []struct {
		name          string
		query         *v1.Query
		field         search.FieldLabel
		expectedValid bool
		expectedError bool
		errorMsg      string
	}{
		{
			name: "valid single priority sort",
			query: &v1.Query{
				Pagination: &v1.QueryPagination{
					SortOptions: []*v1.QuerySortOption{
						{Field: search.Priority.String()},
					},
				},
			},
			field:         search.Priority,
			expectedValid: true,
			expectedError: false,
		},
		{
			name: "valid single cluster priority sort",
			query: &v1.Query{
				Pagination: &v1.QueryPagination{
					SortOptions: []*v1.QuerySortOption{
						{Field: search.ClusterPriority.String()},
					},
				},
			},
			field:         search.ClusterPriority,
			expectedValid: true,
			expectedError: false,
		},
		{
			name: "invalid - priority with other sort options",
			query: &v1.Query{
				Pagination: &v1.QueryPagination{
					SortOptions: []*v1.QuerySortOption{
						{Field: search.Priority.String()},
						{Field: search.CVE.String()},
					},
				},
			},
			field:         search.Priority,
			expectedValid: false,
			expectedError: true,
			errorMsg:      "not supported with other sort options",
		},
		{
			name:          "no pagination",
			query:         &v1.Query{},
			field:         search.Priority,
			expectedValid: false,
			expectedError: false,
		},
		{
			name: "empty sort options",
			query: &v1.Query{
				Pagination: &v1.QueryPagination{
					SortOptions: []*v1.QuerySortOption{},
				},
			},
			field:         search.Priority,
			expectedValid: false,
			expectedError: false,
		},
		{
			name: "different field",
			query: &v1.Query{
				Pagination: &v1.QueryPagination{
					SortOptions: []*v1.QuerySortOption{
						{Field: search.CVE.String()},
					},
				},
			},
			field:         search.Priority,
			expectedValid: false,
			expectedError: false,
		},
		{
			name: "multiple sort options without priority",
			query: &v1.Query{
				Pagination: &v1.QueryPagination{
					SortOptions: []*v1.QuerySortOption{
						{Field: search.CVE.String()},
						{Field: search.Cluster.String()},
					},
				},
			},
			field:         search.Priority,
			expectedValid: false,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := IsValidPriorityQuery(tt.query, tt.field)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedValid, valid)
		})
	}
}

// Test RemovePrioritySortFromQuery function
func TestBuildPriorityQuery(t *testing.T) {
	tests := []struct {
		name             string
		query            *v1.Query
		field            search.FieldLabel
		expectedReversed bool
		expectedError    bool
		errorMsg         string
	}{
		{
			name: "valid priority query - not reversed",
			query: &v1.Query{
				Query: &v1.Query_BaseQuery{
					BaseQuery: &v1.BaseQuery{
						Query: &v1.BaseQuery_MatchFieldQuery{
							MatchFieldQuery: &v1.MatchFieldQuery{
								Field: "Cluster",
								Value: "test",
							},
						},
					},
				},
				Pagination: &v1.QueryPagination{
					SortOptions: []*v1.QuerySortOption{
						{Field: search.Priority.String(), Reversed: false},
					},
				},
			},
			field:            search.Priority,
			expectedReversed: false,
			expectedError:    false,
		},
		{
			name: "valid priority query - reversed",
			query: &v1.Query{
				Query: &v1.Query_BaseQuery{
					BaseQuery: &v1.BaseQuery{
						Query: &v1.BaseQuery_MatchFieldQuery{
							MatchFieldQuery: &v1.MatchFieldQuery{
								Field: "Cluster",
								Value: "test",
							},
						},
					},
				},
				Pagination: &v1.QueryPagination{
					SortOptions: []*v1.QuerySortOption{
						{Field: search.Priority.String(), Reversed: true},
					},
				},
			},
			field:            search.Priority,
			expectedReversed: true,
			expectedError:    false,
		},
		{
			name: "invalid - not a priority query",
			query: &v1.Query{
				Pagination: &v1.QueryPagination{
					SortOptions: []*v1.QuerySortOption{
						{Field: search.CVE.String()},
					},
				},
			},
			field:         search.Priority,
			expectedError: true,
			errorMsg:      "does not sort by",
		},
		{
			name: "invalid - priority with other sort options",
			query: &v1.Query{
				Pagination: &v1.QueryPagination{
					SortOptions: []*v1.QuerySortOption{
						{Field: search.Priority.String()},
						{Field: search.CVE.String()},
					},
				},
			},
			field:         search.Priority,
			expectedError: true,
			errorMsg:      "not supported with other sort options",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, reversed, err := RemovePrioritySortFromQuery(tt.query, tt.field)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, query)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, query)
				assert.Equal(t, tt.expectedReversed, reversed)
				// Verify pagination was removed
				assert.Nil(t, query.Pagination)
				// Verify original query content is preserved
				assert.Equal(t, tt.query.GetQuery(), query.GetQuery())
			}
		})
	}
}

// Test SortResults function
func TestSortResults(t *testing.T) {
	testResults := []search.Result{
		{ID: "r1"},
		{ID: "r2"},
		{ID: "r3"},
		{ID: "r4"},
		{ID: "r5"},
	}

	tests := []struct {
		name        string
		results     []search.Result
		reversed    bool
		rankSetup   func(*mocks.MockRanker)
		expectedIDs []string
	}{
		{
			name:     "sort ascending",
			results:  append([]search.Result{}, testResults...),
			reversed: false,
			rankSetup: func(mockRanker *mocks.MockRanker) {
				mockRanker.EXPECT().GetRankForID("r1").Return(int64(3)).AnyTimes()
				mockRanker.EXPECT().GetRankForID("r2").Return(int64(1)).AnyTimes()
				mockRanker.EXPECT().GetRankForID("r3").Return(int64(4)).AnyTimes()
				mockRanker.EXPECT().GetRankForID("r4").Return(int64(2)).AnyTimes()
				mockRanker.EXPECT().GetRankForID("r5").Return(int64(0)).AnyTimes()
			},
			expectedIDs: []string{"r5", "r2", "r4", "r1", "r3"},
		},
		{
			name:     "sort descending",
			results:  append([]search.Result{}, testResults...),
			reversed: true,
			rankSetup: func(mockRanker *mocks.MockRanker) {
				mockRanker.EXPECT().GetRankForID("r1").Return(int64(3)).AnyTimes()
				mockRanker.EXPECT().GetRankForID("r2").Return(int64(1)).AnyTimes()
				mockRanker.EXPECT().GetRankForID("r3").Return(int64(4)).AnyTimes()
				mockRanker.EXPECT().GetRankForID("r4").Return(int64(2)).AnyTimes()
				mockRanker.EXPECT().GetRankForID("r5").Return(int64(0)).AnyTimes()
			},
			expectedIDs: []string{"r3", "r1", "r4", "r2", "r5"},
		},
		{
			name:     "empty results",
			results:  []search.Result{},
			reversed: false,
			rankSetup: func(mockRanker *mocks.MockRanker) {
				// No expectations needed for empty results
			},
			expectedIDs: []string{},
		},
		{
			name:     "single result",
			results:  []search.Result{{ID: "r1"}},
			reversed: false,
			rankSetup: func(mockRanker *mocks.MockRanker) {
				mockRanker.EXPECT().GetRankForID("r1").Return(int64(1)).AnyTimes()
			},
			expectedIDs: []string{"r1"},
		},
		{
			name:     "equal ranks",
			results:  []search.Result{{ID: "r1"}, {ID: "r2"}},
			reversed: false,
			rankSetup: func(mockRanker *mocks.MockRanker) {
				mockRanker.EXPECT().GetRankForID("r1").Return(int64(1)).AnyTimes()
				mockRanker.EXPECT().GetRankForID("r2").Return(int64(1)).AnyTimes()
			},
			expectedIDs: []string{"r1", "r2"}, // Stable sort preserves original order
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh mock controller and ranker for each test
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockRanker := mocks.NewMockRanker(mockCtrl)

			tt.rankSetup(mockRanker)

			sorted := SortResults(tt.results, tt.reversed, mockRanker)

			actualIDs := make([]string, len(sorted))
			for i, result := range sorted {
				actualIDs[i] = result.ID
			}

			assert.Equal(t, tt.expectedIDs, actualIDs)
		})
	}
}

// Test resultsSorter methods
func TestResultsSorter(t *testing.T) {
	results := []search.Result{
		{ID: "r1"},
		{ID: "r2"},
		{ID: "r3"},
	}

	t.Run("Len", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockRanker := mocks.NewMockRanker(mockCtrl)

		sorter := &resultsSorter{
			results: results,
			ranker:  mockRanker,
		}
		assert.Equal(t, 3, sorter.Len())
	})

	t.Run("Less - not reversed", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockRanker := mocks.NewMockRanker(mockCtrl)

		mockRanker.EXPECT().GetRankForID("r1").Return(int64(2)).AnyTimes()
		mockRanker.EXPECT().GetRankForID("r2").Return(int64(1)).AnyTimes()
		mockRanker.EXPECT().GetRankForID("r3").Return(int64(3)).AnyTimes()

		sorter := &resultsSorter{
			results:  results,
			reversed: false,
			ranker:   mockRanker,
		}
		// r2 (rank 1) < r1 (rank 2)
		assert.True(t, sorter.Less(1, 0))
		// r1 (rank 2) < r3 (rank 3)
		assert.True(t, sorter.Less(0, 2))
		// r3 (rank 3) not < r1 (rank 2)
		assert.False(t, sorter.Less(2, 0))
	})

	t.Run("Less - reversed", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockRanker := mocks.NewMockRanker(mockCtrl)

		mockRanker.EXPECT().GetRankForID("r1").Return(int64(2)).AnyTimes()
		mockRanker.EXPECT().GetRankForID("r2").Return(int64(1)).AnyTimes()
		mockRanker.EXPECT().GetRankForID("r3").Return(int64(3)).AnyTimes()

		sorter := &resultsSorter{
			results:  results,
			reversed: true,
			ranker:   mockRanker,
		}
		// With reversed=true, indices are swapped
		// r1 (rank 2) < r2 (rank 1) becomes r2 (rank 1) < r1 (rank 2)
		assert.False(t, sorter.Less(1, 0))
		// r3 (rank 3) < r1 (rank 2) becomes r1 (rank 2) < r3 (rank 3)
		assert.True(t, sorter.Less(2, 0))
	})

	t.Run("Swap", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mockRanker := mocks.NewMockRanker(mockCtrl)

		testResults := append([]search.Result{}, results...)
		sorter := &resultsSorter{
			results: testResults,
			ranker:  mockRanker,
		}

		// Swap first and second elements
		sorter.Swap(0, 1)

		assert.Equal(t, "r2", testResults[0].ID)
		assert.Equal(t, "r1", testResults[1].ID)
		assert.Equal(t, "r3", testResults[2].ID)
	})
}
