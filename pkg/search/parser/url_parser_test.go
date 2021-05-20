package parser

import (
	"net/url"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
)

func TestParseURLQuery(t *testing.T) {
	vals := url.Values{
		"query":                          []string{"Namespace:ABC"},
		"pagination.offset":              []string{"5"},
		"pagination.limit":               []string{"50"},
		"pagination.sortOption.field":    []string{"Deployment"},
		"pagination.sortOption.reversed": []string{"true"},
	}

	expectedQuery := &v1.Query{
		Query: &v1.Query_BaseQuery{
			BaseQuery: &v1.BaseQuery{
				Query: &v1.BaseQuery_MatchFieldQuery{
					MatchFieldQuery: &v1.MatchFieldQuery{Field: search.Namespace.String(), Value: "ABC"},
				},
			},
		},
		Pagination: &v1.QueryPagination{
			Offset: 5,
			Limit:  50,
			SortOptions: []*v1.QuerySortOption{
				{
					Field:    search.DeploymentName.String(),
					Reversed: true,
				},
			},
		},
	}

	actual, _, err := ParseURLQuery(vals)
	assert.NoError(t, err)
	assert.Equal(t, expectedQuery, actual)
}

func TestParseURLQueryWithExtraValues(t *testing.T) {
	vals := url.Values{
		"query":                          []string{"Namespace:ABC"},
		"pagination.offset":              []string{"5"},
		"pagination.limit":               []string{"50"},
		"pagination.sortOption.field":    []string{"Deployment"},
		"pagination.sortOption.reversed": []string{"true"},
		"blah":                           []string{"blah"},
	}

	expectedQuery := &v1.Query{
		Query: &v1.Query_BaseQuery{
			BaseQuery: &v1.BaseQuery{
				Query: &v1.BaseQuery_MatchFieldQuery{
					MatchFieldQuery: &v1.MatchFieldQuery{Field: search.Namespace.String(), Value: "ABC"},
				},
			},
		},
		Pagination: &v1.QueryPagination{
			Offset: 5,
			Limit:  50,
			SortOptions: []*v1.QuerySortOption{
				{
					Field:    search.DeploymentName.String(),
					Reversed: true,
				},
			},
		},
	}

	actual, _, err := ParseURLQuery(vals)
	assert.NoError(t, err)
	assert.Equal(t, expectedQuery, actual)
}

func TestParseURLQueryConjunctionQuery(t *testing.T) {
	vals := url.Values{
		"query":                          []string{"Namespace:ABC+Cluster:ABC"},
		"pagination.offset":              []string{"5"},
		"pagination.limit":               []string{"50"},
		"pagination.sortOption.field":    []string{"Deployment"},
		"pagination.sortOption.reversed": []string{"true"},
	}

	expectedQuery := &v1.Query{
		Query: &v1.Query_Conjunction{Conjunction: &v1.ConjunctionQuery{
			Queries: []*v1.Query{
				{
					Query: &v1.Query_BaseQuery{
						BaseQuery: &v1.BaseQuery{
							Query: &v1.BaseQuery_MatchFieldQuery{
								MatchFieldQuery: &v1.MatchFieldQuery{Field: search.Cluster.String(), Value: "ABC"},
							},
						},
					},
				},
				{
					Query: &v1.Query_BaseQuery{
						BaseQuery: &v1.BaseQuery{
							Query: &v1.BaseQuery_MatchFieldQuery{
								MatchFieldQuery: &v1.MatchFieldQuery{Field: search.Namespace.String(), Value: "ABC"},
							},
						},
					},
				},
			},
		}},
		Pagination: &v1.QueryPagination{
			Offset: 5,
			Limit:  50,
			SortOptions: []*v1.QuerySortOption{
				{
					Field:    search.DeploymentName.String(),
					Reversed: true,
				},
			},
		},
	}

	actual, _, err := ParseURLQuery(vals)
	assert.NoError(t, err)
	assert.EqualValues(t, expectedQuery, actual)
}
