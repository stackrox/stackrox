package service

import (
	"testing"

	"github.com/stackrox/rox/central/ranking"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
)

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
		name                         string
		input                        *v1.Query
		expectedDeploymentQuery      *v1.Query
		expectedDeploymentPagination *v1.QueryPagination
		expectedRiskQuery            *v1.Query
		expectedRiskPagination       *v1.QueryPagination
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
				return query
			}(),
			expectedDeploymentPagination: &v1.QueryPagination{
				Limit:  10,
				Offset: 0,
				SortOptions: []*v1.QuerySortOption{
					{
						Field:    search.DeploymentType.String(),
						Reversed: true,
					},
				},
			},
			expectedRiskQuery:      nil,
			expectedRiskPagination: nil,
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
			expectedDeploymentQuery:      nil,
			expectedDeploymentPagination: nil,
			expectedRiskQuery: func() *v1.Query {
				query := search.ConjunctionQuery(
					search.NewQueryBuilder().
						AddStrings(search.RiskSubjectType, storage.RiskSubjectType_DEPLOYMENT.String()).
						ProtoQuery(),
					search.NewQueryBuilder().
						AddStrings(search.RiskScore, ">3.000000").
						ProtoQuery(),
				)
				return query
			}(),
			expectedRiskPagination: &v1.QueryPagination{
				Limit:  10,
				Offset: 0,
				SortOptions: []*v1.QuerySortOption{
					{
						Field:    search.RiskScore.String(),
						Reversed: false,
					},
				},
			},
		},
		{
			name: "Deployment query and risk sort.",
			input: func() *v1.Query {
				query := search.NewQueryBuilder().
					AddStrings(search.DeploymentName, "deployment").
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
			expectedDeploymentQuery: func() *v1.Query {
				query := search.NewQueryBuilder().
					AddStrings(search.DeploymentName, "deployment").
					ProtoQuery()
				return query
			}(),
			expectedDeploymentPagination: nil,
			expectedRiskQuery:            nil,
			expectedRiskPagination: &v1.QueryPagination{
				Limit:  10,
				Offset: 0,
				SortOptions: []*v1.QuerySortOption{
					{
						Field:    search.RiskScore.String(),
						Reversed: false,
					},
				},
			},
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
				return query
			}(),
			expectedDeploymentPagination: &v1.QueryPagination{
				Limit:  10,
				Offset: 0,
				SortOptions: []*v1.QuerySortOption{
					{
						Field:    search.DeploymentType.String(),
						Reversed: true,
					},
				},
			},
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
			expectedRiskPagination: nil,
		},
		{
			name: "Risk query with deployment sort",
			input: func() *v1.Query {
				query := search.NewQueryBuilder().
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
			expectedDeploymentQuery: nil,
			expectedDeploymentPagination: &v1.QueryPagination{
				Limit:  10,
				Offset: 0,
				SortOptions: []*v1.QuerySortOption{
					{
						Field:    search.DeploymentType.String(),
						Reversed: true,
					},
				},
			},
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
			expectedRiskPagination: nil,
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
			expectedDeploymentPagination: nil,
			expectedRiskQuery: func() *v1.Query {
				query := search.ConjunctionQuery(
					search.NewQueryBuilder().
						AddStrings(search.RiskSubjectType, storage.RiskSubjectType_DEPLOYMENT.String()).
						ProtoQuery(),
					search.NewQueryBuilder().
						AddStrings(search.RiskScore, "<=3.000000").
						ProtoQuery(),
				)
				return query
			}(),
			expectedRiskPagination: &v1.QueryPagination{
				Limit:  10,
				Offset: 0,
				SortOptions: []*v1.QuerySortOption{
					{
						Field:    search.RiskScore.String(),
						Reversed: true,
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			deploymentQuery := filterDeploymentQuery(c.input)
			deploymentPagination := filterDeploymentPagination(c.input)

			// Create the risk query.
			riskQuery := filterRiskQuery(c.input, ranker)
			riskPagination := filterRiskPagination(c.input)

			assert.Equal(t, c.expectedDeploymentQuery, deploymentQuery)
			assert.Equal(t, c.expectedDeploymentPagination, deploymentPagination)
			assert.Equal(t, c.expectedRiskQuery, riskQuery)
			assert.Equal(t, c.expectedRiskPagination, riskPagination)
		})
	}
}
