package sac

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
)

var (
	testClusterResource = permissions.ResourceMetadata{
		Resource: "testClusterResource",
		Scope:    permissions.ClusterScope,
	}

	testNamespaceResource = permissions.ResourceMetadata{
		Resource: "testNamespaceResource",
		Scope:    permissions.NamespaceScope,
	}

	paginationEmpty = &v1.QueryPagination{
		Limit:       0,
		Offset:      0,
		SortOptions: nil,
	}

	paginationLimitOnly = &v1.QueryPagination{
		Limit:       10,
		Offset:      0,
		SortOptions: nil,
	}

	paginationOffsetOnly = &v1.QueryPagination{
		Limit:       0,
		Offset:      10,
		SortOptions: nil,
	}

	paginationOffsetAndLimit = &v1.QueryPagination{
		Limit:       10,
		Offset:      10,
		SortOptions: nil,
	}

	paginationSortOneColumn = &v1.QueryPagination{
		Limit:  0,
		Offset: 0,
		SortOptions: []*v1.QuerySortOption{
			{
				Field:          "SortFieldOne",
				Reversed:       false,
				SearchAfterOpt: nil,
			},
		},
	}

	paginationSortTwoColumns = &v1.QueryPagination{
		Limit:  0,
		Offset: 0,
		SortOptions: []*v1.QuerySortOption{
			{
				Field:          "SortFieldOne",
				Reversed:       false,
				SearchAfterOpt: nil,
			},
			{
				Field:          "SortFieldTwo",
				Reversed:       true,
				SearchAfterOpt: nil,
			},
		},
	}

	paginationWithOffsetLimitAndSort = &v1.QueryPagination{
		Limit:  10,
		Offset: 10,
		SortOptions: []*v1.QuerySortOption{
			{
				Field:          "SortFieldOne",
				Reversed:       false,
				SearchAfterOpt: nil,
			},
			{
				Field:          "SortFieldTwo",
				Reversed:       true,
				SearchAfterOpt: nil,
			},
		},
	}

	emptyQueryNoPagination = &v1.Query{
		Query:      nil,
		Pagination: nil,
	}

	emptyQueryEmptyPagination = &v1.Query{
		Query:      nil,
		Pagination: paginationEmpty,
	}

	emptyQueryPaginationLimitOnly = &v1.Query{
		Query:      nil,
		Pagination: paginationLimitOnly,
	}

	emptyQueryPaginationOffsetOnly = &v1.Query{
		Query:      nil,
		Pagination: paginationOffsetOnly,
	}

	emptyQueryPaginationOffsetAndLimit = &v1.Query{
		Query:      nil,
		Pagination: paginationOffsetAndLimit,
	}

	emptyQueryPaginationSortOneColumn = &v1.Query{
		Query:      nil,
		Pagination: paginationSortOneColumn,
	}

	emptyQueryPaginationSortTwoColumns = &v1.Query{
		Query:      nil,
		Pagination: paginationSortTwoColumns,
	}

	emptyQueryPaginationWithOffsetLimitAndSort = &v1.Query{
		Query:      nil,
		Pagination: paginationWithOffsetLimitAndSort,
	}

	clusterIDArrakis = "planet.arrakis"

	clusterIDEarth = "planet.earth"

	namespaceAtreides = "Atreides"
)

type searchHelperTestCase struct {
	description           string
	scope                 *effectiveaccessscope.ScopeTree
	query                 *v1.Query
	expectedEnrichedQuery *v1.Query
}

func TestSACQueryEnricherForClusterResource(t *testing.T) {
	helper := &pgSearchHelper{
		resourceMD:          testClusterResource,
		scopeCheckerFactory: nil,
	}
	testCases := []searchHelperTestCase{
		// Nil query test scope filter generation and addition
		{
			description:           "nil scope Tree and nil query result in MatchNone query enrichment",
			scope:                 nil,
			query:                 nil,
			expectedEnrichedQuery: search.ConjunctionQuery(nil, getMatchNoneQuery()),
		},
		{
			description:           "DenyAll scope tree and nil query result in MatchNone query enrichment",
			scope:                 effectiveaccessscope.TestTreeDenyAllEffectiveAccessScope(t),
			query:                 nil,
			expectedEnrichedQuery: search.ConjunctionQuery(nil, getMatchNoneQuery()),
		},
		{
			description:           "Unrestricted scope tree and nil query result in empty query enrichment",
			scope:                 effectiveaccessscope.TestTreeUnrestrictedEffectiveAccessScope(t),
			query:                 nil,
			expectedEnrichedQuery: search.ConjunctionQuery(nil, nil),
		},
		{
			description:           "Cluster-level scope tree and nil query result in cluster match query enrichment",
			scope:                 effectiveaccessscope.TestTreeOneClusterRootFullyIncluded(t),
			query:                 nil,
			expectedEnrichedQuery: search.ConjunctionQuery(nil, clusterNonVerboseMatch(t, clusterIDArrakis)),
		},
		{
			description: "Cluster-level multi-cluster scope tree and nil query result in multi-cluster match query enrichment",
			scope:       effectiveaccessscope.TestTreeTwoClustersFullyIncluded(t),
			query:       nil,
			expectedEnrichedQuery: search.ConjunctionQuery(nil,
				search.DisjunctionQuery(
					clusterNonVerboseMatch(t, clusterIDArrakis),
					clusterNonVerboseMatch(t, clusterIDEarth),
				)),
		},
		{
			description:           "Namespace-only-access scope tree and nil query result in cluster match query enrichment",
			scope:                 effectiveaccessscope.TestTreeOneClusterNamespacePairOnlyIncluded(t),
			query:                 nil,
			expectedEnrichedQuery: search.ConjunctionQuery(nil, clusterNonVerboseMatch(t, clusterIDArrakis)),
		},
		// Empty query test pagination data propagation with unrestricted access
		{
			description:           "Empty query without pagination and unrestricted access result in nil query enrichment",
			scope:                 effectiveaccessscope.TestTreeUnrestrictedEffectiveAccessScope(t),
			query:                 emptyQueryNoPagination,
			expectedEnrichedQuery: search.ConjunctionQuery(emptyQueryNoPagination, nil),
		},
		{
			description: "Empty pagination and unrestricted access propagate pagination",
			scope:       effectiveaccessscope.TestTreeUnrestrictedEffectiveAccessScope(t),
			query:       emptyQueryEmptyPagination,
			expectedEnrichedQuery: &v1.Query{
				Query: &v1.Query_Conjunction{
					Conjunction: &v1.ConjunctionQuery{
						Queries: []*v1.Query{
							emptyQueryEmptyPagination,
							nil,
						},
					},
				},
				Pagination: paginationEmpty,
			},
		},
		{
			description: "Unrestricted access lets limit pagination propagate",
			scope:       effectiveaccessscope.TestTreeUnrestrictedEffectiveAccessScope(t),
			query:       emptyQueryPaginationLimitOnly,
			expectedEnrichedQuery: &v1.Query{
				Query: &v1.Query_Conjunction{
					Conjunction: &v1.ConjunctionQuery{
						Queries: []*v1.Query{
							emptyQueryPaginationLimitOnly,
							nil,
						},
					},
				},
				Pagination: paginationLimitOnly,
			},
		},
		{
			description: "Unrestricted access lets offset pagination propagate",
			scope:       effectiveaccessscope.TestTreeUnrestrictedEffectiveAccessScope(t),
			query:       emptyQueryPaginationOffsetOnly,
			expectedEnrichedQuery: &v1.Query{
				Query: &v1.Query_Conjunction{
					Conjunction: &v1.ConjunctionQuery{
						Queries: []*v1.Query{
							emptyQueryPaginationOffsetOnly,
							nil,
						},
					},
				},
				Pagination: paginationOffsetOnly,
			},
		},
		{
			description: "Unrestricted access lets limit and offset pagination propagate",
			scope:       effectiveaccessscope.TestTreeUnrestrictedEffectiveAccessScope(t),
			query:       emptyQueryPaginationOffsetAndLimit,
			expectedEnrichedQuery: &v1.Query{
				Query: &v1.Query_Conjunction{
					Conjunction: &v1.ConjunctionQuery{
						Queries: []*v1.Query{
							emptyQueryPaginationOffsetAndLimit,
							nil,
						},
					},
				},
				Pagination: paginationOffsetAndLimit,
			},
		},
		{
			description: "Unrestricted access lets simple sort pagination propagate",
			scope:       effectiveaccessscope.TestTreeUnrestrictedEffectiveAccessScope(t),
			query:       emptyQueryPaginationSortOneColumn,
			expectedEnrichedQuery: &v1.Query{
				Query: &v1.Query_Conjunction{
					Conjunction: &v1.ConjunctionQuery{
						Queries: []*v1.Query{
							emptyQueryPaginationSortOneColumn,
							nil,
						},
					},
				},
				Pagination: paginationSortOneColumn,
			},
		},
		{
			description: "Unrestricted access lets multiple sort pagination propagate",
			scope:       effectiveaccessscope.TestTreeUnrestrictedEffectiveAccessScope(t),
			query:       emptyQueryPaginationSortTwoColumns,
			expectedEnrichedQuery: &v1.Query{
				Query: &v1.Query_Conjunction{
					Conjunction: &v1.ConjunctionQuery{
						Queries: []*v1.Query{
							emptyQueryPaginationSortTwoColumns,
							nil,
						},
					},
				},
				Pagination: paginationSortTwoColumns,
			},
		},
		{
			description: "Unrestricted access lets complex pagination propagate",
			scope:       effectiveaccessscope.TestTreeUnrestrictedEffectiveAccessScope(t),
			query:       emptyQueryPaginationWithOffsetLimitAndSort,
			expectedEnrichedQuery: &v1.Query{
				Query: &v1.Query_Conjunction{
					Conjunction: &v1.ConjunctionQuery{
						Queries: []*v1.Query{
							emptyQueryPaginationWithOffsetLimitAndSort,
							nil,
						},
					},
				},
				Pagination: paginationWithOffsetLimitAndSort,
			},
		},
		// Cluster-level scope and complex query with pagination
		{
			description: "Complex scope and query mix let pagination and filter propagate",
			scope:       effectiveaccessscope.TestTreeTwoClustersFullyIncluded(t),
			query: &v1.Query{
				Query: &v1.Query_BaseQuery{
					BaseQuery: &v1.BaseQuery{
						Query: &v1.BaseQuery_MatchFieldQuery{
							MatchFieldQuery: &v1.MatchFieldQuery{
								Field:     "SortFieldOne",
								Value:     "ABC",
								Highlight: false,
							},
						},
					},
				},
				Pagination: paginationWithOffsetLimitAndSort,
			},
			expectedEnrichedQuery: &v1.Query{
				Query: &v1.Query_Conjunction{
					Conjunction: &v1.ConjunctionQuery{
						Queries: []*v1.Query{
							{
								Query: &v1.Query_BaseQuery{
									BaseQuery: &v1.BaseQuery{
										Query: &v1.BaseQuery_MatchFieldQuery{
											MatchFieldQuery: &v1.MatchFieldQuery{
												Field:     "SortFieldOne",
												Value:     "ABC",
												Highlight: false,
											},
										},
									},
								},
								Pagination: paginationWithOffsetLimitAndSort,
							},
							search.DisjunctionQuery(
								clusterNonVerboseMatch(t, clusterIDArrakis),
								clusterNonVerboseMatch(t, clusterIDEarth),
							),
						},
					},
				},
				Pagination: paginationWithOffsetLimitAndSort,
			},
		},
	}
	for _, c := range testCases {
		t.Run(c.description, func(t *testing.T) {
			enrichedQuery, err := helper.enrichQueryWithSACFilter(c.scope, c.query)
			assert.NoError(t, err)
			assert.True(t, isSameQuery(c.expectedEnrichedQuery, enrichedQuery))
		})
	}
}

func TestSACQueryEnricherForNamespaceResource(t *testing.T) {
	helper := &pgSearchHelper{
		resourceMD:          testNamespaceResource,
		scopeCheckerFactory: nil,
	}
	testCases := []searchHelperTestCase{
		// Nil query test scope filter generation and addition
		{
			description:           "nil scope Tree and nil query result in MatchNone query enrichment",
			scope:                 nil,
			query:                 nil,
			expectedEnrichedQuery: search.ConjunctionQuery(nil, getMatchNoneQuery()),
		},
		{
			description:           "DenyAll scope tree and nil query result in MatchNone query enrichment",
			scope:                 effectiveaccessscope.TestTreeDenyAllEffectiveAccessScope(t),
			query:                 nil,
			expectedEnrichedQuery: search.ConjunctionQuery(nil, getMatchNoneQuery()),
		},
		{
			description:           "Unrestricted scope tree and nil query result in empty query enrichment",
			scope:                 effectiveaccessscope.TestTreeUnrestrictedEffectiveAccessScope(t),
			query:                 nil,
			expectedEnrichedQuery: search.ConjunctionQuery(nil, nil),
		},
		{
			description: "Cluster-level scope tree and nil query result in cluster match query enrichment",
			scope:       effectiveaccessscope.TestTreeOneClusterRootFullyIncluded(t),
			query:       nil,
			expectedEnrichedQuery: search.ConjunctionQuery(nil,
				clusterNonVerboseMatch(t, clusterIDArrakis),
			),
		},
		{
			description: "Cluster-level multi-cluster scope tree and nil query result in multi-cluster match query enrichment",
			scope:       effectiveaccessscope.TestTreeTwoClustersFullyIncluded(t),
			query:       nil,
			expectedEnrichedQuery: search.ConjunctionQuery(nil,
				search.DisjunctionQuery(
					clusterNonVerboseMatch(t, clusterIDArrakis),
					clusterNonVerboseMatch(t, clusterIDEarth),
				)),
		},
		{
			description: "Namespace-only-access scope tree and nil query result in MatchNone query enrichment",
			scope:       effectiveaccessscope.TestTreeOneClusterNamespacePairOnlyIncluded(t),
			query:       nil,
			expectedEnrichedQuery: search.ConjunctionQuery(
				nil,
				search.ConjunctionQuery(
					clusterNonVerboseMatch(t, clusterIDArrakis),
					namespaceNonVerboseMatch(t, namespaceAtreides),
				),
			),
		},
		// Empty query test pagination data propagation with unrestricted access
		{
			description:           "Empty query without pagination and unrestricted access result in nil query enrichment",
			scope:                 effectiveaccessscope.TestTreeUnrestrictedEffectiveAccessScope(t),
			query:                 emptyQueryNoPagination,
			expectedEnrichedQuery: search.ConjunctionQuery(emptyQueryNoPagination, nil),
		},
		{
			description: "Empty pagination and unrestricted access propagate pagination",
			scope:       effectiveaccessscope.TestTreeUnrestrictedEffectiveAccessScope(t),
			query:       emptyQueryEmptyPagination,
			expectedEnrichedQuery: &v1.Query{
				Query: &v1.Query_Conjunction{
					Conjunction: &v1.ConjunctionQuery{
						Queries: []*v1.Query{
							emptyQueryEmptyPagination,
							nil,
						},
					},
				},
				Pagination: paginationEmpty,
			},
		},
		{
			description: "Unrestricted access lets limit pagination propagate",
			scope:       effectiveaccessscope.TestTreeUnrestrictedEffectiveAccessScope(t),
			query:       emptyQueryPaginationLimitOnly,
			expectedEnrichedQuery: &v1.Query{
				Query: &v1.Query_Conjunction{
					Conjunction: &v1.ConjunctionQuery{
						Queries: []*v1.Query{
							emptyQueryPaginationLimitOnly,
							nil,
						},
					},
				},
				Pagination: paginationLimitOnly,
			},
		},
		{
			description: "Unrestricted access lets offset pagination propagate",
			scope:       effectiveaccessscope.TestTreeUnrestrictedEffectiveAccessScope(t),
			query:       emptyQueryPaginationOffsetOnly,
			expectedEnrichedQuery: &v1.Query{
				Query: &v1.Query_Conjunction{
					Conjunction: &v1.ConjunctionQuery{
						Queries: []*v1.Query{
							emptyQueryPaginationOffsetOnly,
							nil,
						},
					},
				},
				Pagination: paginationOffsetOnly,
			},
		},
		{
			description: "Unrestricted access lets limit and offset pagination propagate",
			scope:       effectiveaccessscope.TestTreeUnrestrictedEffectiveAccessScope(t),
			query:       emptyQueryPaginationOffsetAndLimit,
			expectedEnrichedQuery: &v1.Query{
				Query: &v1.Query_Conjunction{
					Conjunction: &v1.ConjunctionQuery{
						Queries: []*v1.Query{
							emptyQueryPaginationOffsetAndLimit,
							nil,
						},
					},
				},
				Pagination: paginationOffsetAndLimit,
			},
		},
		{
			description: "Unrestricted access lets simple sort pagination propagate",
			scope:       effectiveaccessscope.TestTreeUnrestrictedEffectiveAccessScope(t),
			query:       emptyQueryPaginationSortOneColumn,
			expectedEnrichedQuery: &v1.Query{
				Query: &v1.Query_Conjunction{
					Conjunction: &v1.ConjunctionQuery{
						Queries: []*v1.Query{
							emptyQueryPaginationSortOneColumn,
							nil,
						},
					},
				},
				Pagination: paginationSortOneColumn,
			},
		},
		{
			description: "Unrestricted access lets multiple sort pagination propagate",
			scope:       effectiveaccessscope.TestTreeUnrestrictedEffectiveAccessScope(t),
			query:       emptyQueryPaginationSortTwoColumns,
			expectedEnrichedQuery: &v1.Query{
				Query: &v1.Query_Conjunction{
					Conjunction: &v1.ConjunctionQuery{
						Queries: []*v1.Query{
							emptyQueryPaginationSortTwoColumns,
							nil,
						},
					},
				},
				Pagination: paginationSortTwoColumns,
			},
		},
		{
			description: "Unrestricted access lets complex pagination propagate",
			scope:       effectiveaccessscope.TestTreeUnrestrictedEffectiveAccessScope(t),
			query:       emptyQueryPaginationWithOffsetLimitAndSort,
			expectedEnrichedQuery: &v1.Query{
				Query: &v1.Query_Conjunction{
					Conjunction: &v1.ConjunctionQuery{
						Queries: []*v1.Query{
							emptyQueryPaginationWithOffsetLimitAndSort,
							nil,
						},
					},
				},
				Pagination: paginationWithOffsetLimitAndSort,
			},
		},
		// Cluster-level scope and complex query with pagination
		{
			description: "Complex cluster-level scope and query mix let pagination and filter propagate",
			scope:       effectiveaccessscope.TestTreeTwoClustersFullyIncluded(t),
			query: &v1.Query{
				Query: &v1.Query_BaseQuery{
					BaseQuery: &v1.BaseQuery{
						Query: &v1.BaseQuery_MatchFieldQuery{
							MatchFieldQuery: &v1.MatchFieldQuery{
								Field:     "SortFieldOne",
								Value:     "ABC",
								Highlight: false,
							},
						},
					},
				},
				Pagination: paginationWithOffsetLimitAndSort,
			},
			expectedEnrichedQuery: &v1.Query{
				Query: &v1.Query_Conjunction{
					Conjunction: &v1.ConjunctionQuery{
						Queries: []*v1.Query{
							{
								Query: &v1.Query_BaseQuery{
									BaseQuery: &v1.BaseQuery{
										Query: &v1.BaseQuery_MatchFieldQuery{
											MatchFieldQuery: &v1.MatchFieldQuery{
												Field:     "SortFieldOne",
												Value:     "ABC",
												Highlight: false,
											},
										},
									},
								},
								Pagination: paginationWithOffsetLimitAndSort,
							},
							search.DisjunctionQuery(
								clusterNonVerboseMatch(t, clusterIDArrakis),
								clusterNonVerboseMatch(t, clusterIDEarth),
							),
						},
					},
				},
				Pagination: paginationWithOffsetLimitAndSort,
			},
		},
		// Namespace-level scope and complex query with pagination
		{
			description: "Complex cluster-level scope and query mix let pagination and filter propagate",
			scope:       effectiveaccessscope.TestTreeTwoClusterNamespacePairsIncluded(t),
			query: &v1.Query{
				Query: &v1.Query_BaseQuery{
					BaseQuery: &v1.BaseQuery{
						Query: &v1.BaseQuery_MatchFieldQuery{
							MatchFieldQuery: &v1.MatchFieldQuery{
								Field:     "SortFieldOne",
								Value:     "ABC",
								Highlight: false,
							},
						},
					},
				},
				Pagination: paginationWithOffsetLimitAndSort,
			},
			expectedEnrichedQuery: &v1.Query{
				Query: &v1.Query_Conjunction{
					Conjunction: &v1.ConjunctionQuery{
						Queries: []*v1.Query{
							{
								Query: &v1.Query_BaseQuery{
									BaseQuery: &v1.BaseQuery{
										Query: &v1.BaseQuery_MatchFieldQuery{
											MatchFieldQuery: &v1.MatchFieldQuery{
												Field:     "SortFieldOne",
												Value:     "ABC",
												Highlight: false,
											},
										},
									},
								},
								Pagination: paginationWithOffsetLimitAndSort,
							},
							search.DisjunctionQuery(
								search.ConjunctionQuery(
									clusterNonVerboseMatch(t, planetEarth),
									namespaceNonVerboseMatch(t, nsSkunkWorks),
								),
								search.ConjunctionQuery(
									clusterNonVerboseMatch(t, planetArrakis),
									namespaceNonVerboseMatch(t, nsSpacingGuild),
								),
							),
						},
					},
				},
				Pagination: paginationWithOffsetLimitAndSort,
			},
		},
	}
	for _, c := range testCases {
		t.Run(c.description, func(t *testing.T) {
			enrichedQuery, err := helper.enrichQueryWithSACFilter(c.scope, c.query)
			assert.NoError(t, err)
			assert.True(t, isSameQuery(c.expectedEnrichedQuery, enrichedQuery))
		})
	}
}
