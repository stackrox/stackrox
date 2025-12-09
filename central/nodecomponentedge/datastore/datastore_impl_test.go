package datastore

import (
	"context"
	"testing"

	storeMocks "github.com/stackrox/rox/central/nodecomponentedge/store/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type searchEdgesTestCase struct {
	desc                string
	query               *v1.Query
	storeResults        []search.Result
	storeError          error
	expectedResultCount int
	expectedError       error
	validateResults     func(t *testing.T, results []*v1.SearchResult)
}

func TestSearchEdges(t *testing.T) {
	ctx := context.Background()

	testCases := []searchEdgesTestCase{
		{
			desc:  "empty query returns all edges as search results",
			query: search.EmptyQuery(),
			storeResults: []search.Result{
				{
					ID:    "edge1",
					Score: 1.0,
				},
				{
					ID:    "edge2",
					Score: 2.0,
				},
			},
			expectedResultCount: 2,
			validateResults: func(t *testing.T, results []*v1.SearchResult) {
				assert.Equal(t, "edge1", results[0].GetId())
				assert.Equal(t, "edge1", results[0].GetName()) // Name should be populated from ID
				assert.Equal(t, v1.SearchCategory_NODE_COMPONENT_EDGE, results[0].GetCategory())
				assert.Equal(t, 1.0, results[0].GetScore())
				assert.Empty(t, results[0].GetLocation()) // NodeComponentEdge has no location

				assert.Equal(t, "edge2", results[1].GetId())
				assert.Equal(t, "edge2", results[1].GetName())
				assert.Equal(t, v1.SearchCategory_NODE_COMPONENT_EDGE, results[1].GetCategory())
				assert.Equal(t, 2.0, results[1].GetScore())
			},
		},
		{
			desc: "query with filters returns filtered search results",
			query: search.NewQueryBuilder().
				AddExactMatches(search.NodeID, "node1").
				ProtoQuery(),
			storeResults: []search.Result{
				{
					ID:    "edge1",
					Score: 5.5,
					Matches: map[string][]string{
						"Node ID": {"node1"},
					},
				},
			},
			expectedResultCount: 1,
			validateResults: func(t *testing.T, results []*v1.SearchResult) {
				assert.Equal(t, "edge1", results[0].GetId())
				assert.Equal(t, "edge1", results[0].GetName())
				assert.Equal(t, 5.5, results[0].GetScore())
				assert.Len(t, results[0].GetFieldToMatches(), 1)
				assert.Contains(t, results[0].GetFieldToMatches(), "Node ID")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockStore := storeMocks.NewMockStore(ctrl)

			if tc.storeError != nil {
				mockStore.EXPECT().
					Search(gomock.Any(), tc.query).
					Return(nil, tc.storeError)
			} else {
				mockStore.EXPECT().
					Search(gomock.Any(), tc.query).
					Return(tc.storeResults, nil)
			}

			ds := &datastoreImpl{storage: mockStore}
			results, err := ds.SearchEdges(ctx, tc.query)

			// Verify error expectations
			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tc.expectedError, err)
			} else {
				require.NoError(t, err)
			}

			// Verify result count
			assert.Len(t, results, tc.expectedResultCount)

			// Run custom validation if provided
			if tc.validateResults != nil {
				tc.validateResults(t, results)
			}
		})
	}
}
