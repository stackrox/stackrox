package service

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/ranking"
	riskDatastoreMocks "github.com/stackrox/rox/central/risk/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	testutils2 "github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, &serviceImpl{})
}

func TestLabelsMap(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		deployments    []*storage.Deployment
		expectedMap    map[string]*v1.DeploymentLabelsResponse_LabelValues
		expectedValues []string
	}{
		{
			name: "one deployment",
			deployments: []*storage.Deployment{
				{
					Id: uuid.NewV4().String(),
					Labels: map[string]string{
						"key": "value",
					},
				},
			},
			expectedMap: map[string]*v1.DeploymentLabelsResponse_LabelValues{
				"key": {
					Values: []string{"value"},
				},
			},
			expectedValues: []string{
				"value",
			},
		},
		{
			name: "multiple deployments",
			deployments: []*storage.Deployment{
				{
					Id: uuid.NewV4().String(),
					Labels: map[string]string{
						"key":   "value",
						"hello": "world",
						"foo":   "bar",
					},
				},
				{
					Id: uuid.NewV4().String(),
					Labels: map[string]string{
						"key": "hole",
						"app": "data",
						"foo": "bar",
					},
				},
				{
					Id: uuid.NewV4().String(),
					Labels: map[string]string{
						"hello": "bob",
						"foo":   "boo",
					},
				},
			},
			expectedMap: map[string]*v1.DeploymentLabelsResponse_LabelValues{
				"key": {
					Values: []string{"hole", "value"},
				},
				"hello": {
					Values: []string{"bob", "world"},
				},
				"foo": {
					Values: []string{"bar", "boo"},
				},
				"app": {
					Values: []string{"data"},
				},
			},
			expectedValues: []string{
				"bar", "bob", "boo", "data", "hole", "value", "world",
			},
		},
	}

	ctx := sac.WithAllAccess(context.Background())
	mockCtrl := gomock.NewController(t)
	mockRiskDatastore := riskDatastoreMocks.NewMockDataStore(mockCtrl)
	mockRiskDatastore.EXPECT().SearchRawRisks(gomock.Any(), gomock.Any()).AnyTimes()
	mockRiskDatastore.EXPECT().GetRisk(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			badgerDB := testutils2.BadgerDBForT(t)
			defer utils.IgnoreError(badgerDB.Close)

			bleveIndex, err := globalindex.MemOnlyIndex()
			require.NoError(t, err)

			deploymentsDS, err := datastore.NewBadger(badgerDB, bleveIndex, nil, nil, nil, nil, mockRiskDatastore, nil)
			require.NoError(t, err)

			for _, deployment := range c.deployments {
				assert.NoError(t, deploymentsDS.UpsertDeployment(ctx, deployment))
			}

			results, err := deploymentsDS.Search(ctx, queryForLabels())
			assert.NoError(t, err)
			actualMap, actualValues := labelsMapFromSearchResults(results)

			assert.Equal(t, c.expectedMap, actualMap)
			assert.ElementsMatch(t, c.expectedValues, actualValues)
		})
	}
}

func TestConvertQuery(t *testing.T) {
	t.Parallel()

	// Set up a ranker that to use when generating a risk query.
	ranker := ranking.NewRanker()
	ranker.Add("dep1", 1.0)
	ranker.Add("dep2", 2.0)
	ranker.Add("dep3", 3.0)
	ranker.Add("dep4", 4.0)

	// Test cases.
	cases := []struct {
		name                    string
		input                   *v1.Query
		expectedDeploymentQuery *v1.Query
		expectedRiskQuery       *v1.Query
		expectedFilterOnRisk    bool
	}{
		{
			name: "Only deployment query",
			input: func() *v1.Query {
				query := search.NewQueryBuilder().
					AddStrings(search.DeploymentName, "deployment").
					ProtoQuery()
				query.Pagination = &v1.QueryPagination{
					Limit:  10,
					Offset: 0,
					SortOptions: []*v1.QuerySortOption{
						{
							Field:    search.DeploymentType.String(),
							Reversed: true,
						},
					},
				}
				return query
			}(),
			expectedDeploymentQuery: func() *v1.Query {
				query := search.NewQueryBuilder().
					AddStrings(search.DeploymentName, "deployment").
					ProtoQuery()
				query.Pagination = &v1.QueryPagination{
					Limit:  10,
					Offset: 0,
					SortOptions: []*v1.QuerySortOption{
						{
							Field:    search.DeploymentType.String(),
							Reversed: true,
						},
					},
				}
				return query
			}(),
			expectedRiskQuery:    nil,
			expectedFilterOnRisk: false,
		},
		{
			name: "Only risk query",
			input: func() *v1.Query {
				query := search.NewQueryBuilder().
					AddStrings(search.Priority, "<2").
					ProtoQuery()
				query.Pagination = &v1.QueryPagination{
					Limit:  10,
					Offset: 0,
					SortOptions: []*v1.QuerySortOption{
						{
							Field:    search.Priority.String(),
							Reversed: true,
						},
					},
				}
				return query
			}(),
			expectedDeploymentQuery: nil,
			expectedRiskQuery: func() *v1.Query {
				query := search.ConjunctionQuery(
					search.NewQueryBuilder().
						AddStrings(search.RiskSubjectType, storage.RiskSubjectType_DEPLOYMENT.String()).
						ProtoQuery(),
					search.NewQueryBuilder().
						AddStrings(search.RiskScore, ">3.000000").
						ProtoQuery(),
				)
				query.Pagination = &v1.QueryPagination{
					Limit:  10,
					Offset: 0,
					SortOptions: []*v1.QuerySortOption{
						{
							Field:    search.RiskScore.String(),
							Reversed: false,
						},
					},
				}
				return query
			}(),
			expectedFilterOnRisk: true,
		},
		{
			name: "Mixed with deployment sort",
			input: func() *v1.Query {
				query := search.NewQueryBuilder().
					AddStrings(search.DeploymentName, "deployment").
					AddStrings(search.Priority, "<=2.0").
					ProtoQuery()
				query.Pagination = &v1.QueryPagination{
					Limit:  10,
					Offset: 0,
					SortOptions: []*v1.QuerySortOption{
						{
							Field:    search.DeploymentType.String(),
							Reversed: true,
						},
					},
				}
				return query
			}(),
			expectedDeploymentQuery: func() *v1.Query {
				query := search.NewQueryBuilder().
					AddStrings(search.DeploymentName, "deployment").
					ProtoQuery()
				query.Pagination = &v1.QueryPagination{
					Limit:  10,
					Offset: 0,
					SortOptions: []*v1.QuerySortOption{
						{
							Field:    search.DeploymentType.String(),
							Reversed: true,
						},
					},
				}
				return query
			}(),
			expectedRiskQuery: func() *v1.Query {
				query := search.ConjunctionQuery(
					search.NewQueryBuilder().
						AddStrings(search.RiskSubjectType, storage.RiskSubjectType_DEPLOYMENT.String()).
						ProtoQuery(),
					search.NewQueryBuilder().
						AddStrings(search.RiskScore, ">=3.000000").
						ProtoQuery(),
				)
				return query
			}(),
			expectedFilterOnRisk: true,
		},
		{
			name: "Mixed with risk sort",
			input: func() *v1.Query {
				query := search.NewQueryBuilder().
					AddStrings(search.DeploymentName, "deployment").
					AddStrings(search.Priority, ">=2.0").
					ProtoQuery()
				query.Pagination = &v1.QueryPagination{
					Limit:  10,
					Offset: 0,
					SortOptions: []*v1.QuerySortOption{
						{
							Field:    search.Priority.String(),
							Reversed: false,
						},
					},
				}
				return query
			}(),
			expectedDeploymentQuery: func() *v1.Query {
				query := search.NewQueryBuilder().
					AddStrings(search.DeploymentName, "deployment").
					ProtoQuery()
				return query
			}(),
			expectedRiskQuery: func() *v1.Query {
				query := search.ConjunctionQuery(
					search.NewQueryBuilder().
						AddStrings(search.RiskSubjectType, storage.RiskSubjectType_DEPLOYMENT.String()).
						ProtoQuery(),
					search.NewQueryBuilder().
						AddStrings(search.RiskScore, "<=3.000000").
						ProtoQuery(),
				)
				query.Pagination = &v1.QueryPagination{
					Limit:  10,
					Offset: 0,
					SortOptions: []*v1.QuerySortOption{
						{
							Field:    search.RiskScore.String(),
							Reversed: true,
						},
					},
				}
				return query
			}(),
			expectedFilterOnRisk: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			deploymentQuery, riskQuery, filterOnRisk := splitQueries(c.input, ranker)

			assert.Equal(t, c.expectedDeploymentQuery, deploymentQuery)
			assert.Equal(t, c.expectedRiskQuery, riskQuery)
			assert.Equal(t, c.expectedFilterOnRisk, filterOnRisk)
		})
	}
}
