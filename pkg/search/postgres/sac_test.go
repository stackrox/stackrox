package postgres

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
)

var (
	expectedMatchFieldQuery = &v1.Query{
		Query: &v1.Query_BaseQuery{
			BaseQuery: &v1.BaseQuery{
				Query: &v1.BaseQuery_MatchFieldQuery{
					MatchFieldQuery: &v1.MatchFieldQuery{
						Field: "Cluster ID",
						Value: "\"clusterID\"",
					},
				},
			},
		},
	}

	expectedMatchNoneQuery = &v1.Query{
		Query: &v1.Query_BaseQuery{
			BaseQuery: &v1.BaseQuery{
				Query: &v1.BaseQuery_MatchNoneQuery{
					MatchNoneQuery: &v1.MatchNoneQuery{},
				},
			},
		},
	}
)

func TestGetReadWriteSACQuery(t *testing.T) {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), createTestReadMultipleResourcesSomeWithNamespaceScope(t))
	got, err := GetReadWriteSACQuery(ctx, metadata("Cluster", permissions.ClusterScope))
	protoassert.Equal(t, expectedMatchFieldQuery, got)
	assert.NoError(t, err)

	got, err = GetReadWriteSACQuery(ctx, metadata("Namespace", permissions.NamespaceScope))
	protoassert.Equal(t, expectedMatchNoneQuery, got)
	assert.NoError(t, err)

	got, err = GetReadSACQuery(sac.WithNoAccess(context.Background()), metadata("Integration", permissions.GlobalScope))
	protoassert.Equal(t, expectedMatchNoneQuery, got)
	assert.NoError(t, err)

	got, err = GetReadWriteSACQuery(sac.WithNoAccess(context.Background()), metadata("Integration", permissions.GlobalScope))
	assert.Nil(t, got)
	assert.ErrorIs(t, err, sac.ErrResourceAccessDenied)
}

func metadata(name permissions.Resource, scope permissions.ResourceScope) permissions.ResourceMetadata {
	md := permissions.ResourceMetadata{
		Resource: name,
		Scope:    scope,
	}
	return md
}

func createTestReadMultipleResourcesSomeWithNamespaceScope(t *testing.T) sac.ScopeCheckerCore {
	resourceCluster := permissions.Resource("Cluster")
	resourceDeployment := permissions.Resource("Deployment")
	resourceNode := permissions.Resource("Node")

	clusterClusterID := "clusterID"
	nsNamespace2 := "namespace2"

	testScope := map[storage.Access]map[permissions.Resource]*sac.TestResourceScope{
		storage.Access_READ_WRITE_ACCESS: {
			resourceCluster: &sac.TestResourceScope{
				Included: false,
				Clusters: map[string]*sac.TestClusterScope{
					clusterClusterID: {
						Included: true,
					},
				},
			},
			resourceNode: &sac.TestResourceScope{Included: true},
			resourceDeployment: &sac.TestResourceScope{
				Included: false,
				Clusters: map[string]*sac.TestClusterScope{
					clusterClusterID: {
						Included:   false,
						Namespaces: []string{nsNamespace2},
					},
				},
			},
		},
	}
	return sac.TestScopeCheckerCoreFromFullScopeMap(t, testScope)
}

func TestEnrichQueryWithSACFilter(t *testing.T) {
	// Create test schemas for testing
	clusterSchema := &walker.Schema{
		Table:           "test_clusters",
		ScopingResource: metadata("Cluster", permissions.ClusterScope),
	}

	// Create test context with scoped permissions
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), createTestReadMultipleResourcesSomeWithNamespaceScope(t))

	t.Run("preserves Selects and Pagination for READ queries", func(t *testing.T) {
		// Create a query with Selects and Pagination
		inputQuery := &v1.Query{
			Query: &v1.Query_BaseQuery{
				BaseQuery: &v1.BaseQuery{
					Query: &v1.BaseQuery_MatchFieldQuery{
						MatchFieldQuery: &v1.MatchFieldQuery{
							Field: "Name",
							Value: "test",
						},
					},
				},
			},
			Selects: []*v1.QuerySelect{
				{
					Field: &v1.QueryField{
						Name: "Cluster ID",
					},
				},
				{
					Field: &v1.QueryField{
						Name: "Name",
					},
				},
			},
			Pagination: &v1.QueryPagination{
				Limit:  10,
				Offset: 5,
				SortOptions: []*v1.QuerySortOption{
					{
						Field:    "Name",
						Reversed: false,
					},
				},
			},
		}

		// Call enrichQueryWithSACFilter for default (READ) case
		enrichedQuery, err := enrichQueryWithSACFilter(ctx, inputQuery, clusterSchema, SEARCH)
		assert.NoError(t, err)
		assert.NotNil(t, enrichedQuery)

		// Verify Selects are preserved
		assert.Len(t, enrichedQuery.GetSelects(), 2, "Selects should be preserved")

		// Verify Pagination is preserved
		assert.Equal(t, int32(10), enrichedQuery.GetPagination().GetLimit())
		assert.Equal(t, int32(5), enrichedQuery.GetPagination().GetOffset())
		assert.Equal(t, "Name", enrichedQuery.GetPagination().GetSortOptions()[0].GetField())
	})

	t.Run("adds SAC filter while preserving Selects and Pagination", func(t *testing.T) {
		// Create a query with Selects, Pagination, and a base query
		inputQuery := &v1.Query{
			Query: &v1.Query_BaseQuery{
				BaseQuery: &v1.BaseQuery{
					Query: &v1.BaseQuery_MatchFieldQuery{
						MatchFieldQuery: &v1.MatchFieldQuery{
							Field: "Name",
							Value: "production",
						},
					},
				},
			},
			Selects: []*v1.QuerySelect{
				{
					Field: &v1.QueryField{
						Name: "Cluster ID",
					},
				},
				{
					Field: &v1.QueryField{
						Name: "Name",
					},
				},
				{
					Field: &v1.QueryField{
						Name: "Status",
					},
				},
			},
			Pagination: &v1.QueryPagination{
				Limit:  25,
				Offset: 10,
				SortOptions: []*v1.QuerySortOption{
					{
						Field:    "Name",
						Reversed: true,
					},
				},
			},
		}

		// Call enrichQueryWithSACFilter with restricted access context
		enrichedQuery, err := enrichQueryWithSACFilter(ctx, inputQuery, clusterSchema, SEARCH)
		assert.NoError(t, err)
		assert.NotNil(t, enrichedQuery)

		// Verify SAC filter was added - should create a Conjunction
		conjunction := enrichedQuery.GetConjunction()

		// Verify SAC filter is present
		// The SAC filter should be the first query in the conjunction
		// It's not necessarily the original query (which should be second)
		sacQuery := conjunction.GetQueries()[0]
		originalQueryInConj := conjunction.GetQueries()[1]

		// Verify the original query is the second one
		originalMfq := originalQueryInConj.GetBaseQuery().GetMatchFieldQuery()
		assert.NotNil(t, originalMfq)
		assert.Equal(t, "Name", originalMfq.GetField())
		assert.Equal(t, "production", originalMfq.GetValue())

		// SAC filter should have a base query (it's not nil and not a simple match field query)
		protoassert.Equal(t, sacQuery, expectedMatchNoneQuery, "sacQuery must be a match none query")

		// Verify Selects are preserved
		assert.Len(t, enrichedQuery.GetSelects(), 3, "All 3 Selects should be preserved")

		// Verify Pagination is preserved completely
		assert.Equal(t, int32(25), enrichedQuery.GetPagination().GetLimit())
		assert.Equal(t, int32(10), enrichedQuery.GetPagination().GetOffset())
	})
}
