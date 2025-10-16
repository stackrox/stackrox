package parser

import (
	"net/url"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/protoassert"
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

	expectedQuery := v1.Query_builder{
		BaseQuery: v1.BaseQuery_builder{
			MatchFieldQuery: v1.MatchFieldQuery_builder{Field: search.Namespace.String(), Value: "ABC"}.Build(),
		}.Build(),
		Pagination: v1.QueryPagination_builder{
			Offset: 5,
			Limit:  50,
			SortOptions: []*v1.QuerySortOption{
				v1.QuerySortOption_builder{
					Field:    search.DeploymentName.String(),
					Reversed: true,
				}.Build(),
			},
		}.Build(),
	}.Build()

	actual, _, err := ParseURLQuery(vals)
	assert.NoError(t, err)
	protoassert.Equal(t, expectedQuery, actual)
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

	expectedQuery := v1.Query_builder{
		BaseQuery: v1.BaseQuery_builder{
			MatchFieldQuery: v1.MatchFieldQuery_builder{Field: search.Namespace.String(), Value: "ABC"}.Build(),
		}.Build(),
		Pagination: v1.QueryPagination_builder{
			Offset: 5,
			Limit:  50,
			SortOptions: []*v1.QuerySortOption{
				v1.QuerySortOption_builder{
					Field:    search.DeploymentName.String(),
					Reversed: true,
				}.Build(),
			},
		}.Build(),
	}.Build()

	actual, _, err := ParseURLQuery(vals)
	assert.NoError(t, err)
	protoassert.Equal(t, expectedQuery, actual)
}

func TestParseURLQueryConjunctionQuery(t *testing.T) {
	vals := url.Values{
		"query":                          []string{"Namespace:ABC+Cluster:ABC"},
		"pagination.offset":              []string{"5"},
		"pagination.limit":               []string{"50"},
		"pagination.sortOption.field":    []string{"Deployment"},
		"pagination.sortOption.reversed": []string{"true"},
	}

	expectedQuery := v1.Query_builder{
		Conjunction: v1.ConjunctionQuery_builder{
			Queries: []*v1.Query{
				v1.Query_builder{
					BaseQuery: v1.BaseQuery_builder{
						MatchFieldQuery: v1.MatchFieldQuery_builder{Field: search.Cluster.String(), Value: "ABC"}.Build(),
					}.Build(),
				}.Build(),
				v1.Query_builder{
					BaseQuery: v1.BaseQuery_builder{
						MatchFieldQuery: v1.MatchFieldQuery_builder{Field: search.Namespace.String(), Value: "ABC"}.Build(),
					}.Build(),
				}.Build(),
			},
		}.Build(),
		Pagination: v1.QueryPagination_builder{
			Offset: 5,
			Limit:  50,
			SortOptions: []*v1.QuerySortOption{
				v1.QuerySortOption_builder{
					Field:    search.DeploymentName.String(),
					Reversed: true,
				}.Build(),
			},
		}.Build(),
	}.Build()

	actual, _, err := ParseURLQuery(vals)
	assert.NoError(t, err)
	protoassert.Equal(t, expectedQuery, actual)
}
