package postgres

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSubQuerySQLExamples prints SQL and bind parameters for review with:
// go test ./pkg/search/postgres -run '^TestSubQuerySQLExamples$' -v -count=1
func TestSubQuerySQLExamples(t *testing.T) {
	inner := search.NewQueryBuilder().AddExactMatches(search.Namespace, "payments").
		AddSelectFields(search.NewQuerySelect(search.ClusterID)).ProtoQuery()
	clustersWithPaymentsDeployments := search.MatchFieldSubQuery(search.ClusterID, v1.SearchCategory_DEPLOYMENTS, inner)
	limited := inner.CloneVT()
	limited.Pagination = search.NewPagination().Limit(2).Offset(1).
		AddSortOption(search.NewSortOption(search.DeploymentName)).Proto()
	productionClusters := search.NewQueryBuilder().AddExactMatches(search.Cluster, "production").
		AddSelectFields(search.NewQuerySelect(search.ClusterID)).ProtoQuery()
	nested := search.ConjunctionQuery(inner, search.MatchFieldSubQuery(search.ClusterID, v1.SearchCategory_CLUSTERS, productionClusters))
	nested.Selects = inner.GetSelects()

	for name, test := range map[string]struct {
		query  *v1.Query
		params []any
	}{
		"clusters with deployments in payments": {
			query: clustersWithPaymentsDeployments, params: []any{"payments"},
		},
		"production clusters with deployments in payments": {
			query:  search.ConjunctionQuery(search.NewQueryBuilder().AddExactMatches(search.Cluster, "production").ProtoQuery(), clustersWithPaymentsDeployments),
			params: []any{"production", "payments"},
		},
		"clusters without deployments in payments": {
			query:  search.NewBooleanQuery(nil, &v1.DisjunctionQuery{Queries: []*v1.Query{clustersWithPaymentsDeployments}}),
			params: []any{"payments"},
		},
		"paginated inner query": {
			query:  search.MatchFieldSubQuery(search.ClusterID, v1.SearchCategory_DEPLOYMENTS, limited),
			params: []any{"payments"},
		},
		"nested subquery for deployments in production clusters": {
			query:  search.MatchFieldSubQuery(search.ClusterID, v1.SearchCategory_DEPLOYMENTS, nested),
			params: []any{"payments", "production"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			compiled, err := standardizeQueryAndPopulatePath(sac.WithAllAccess(context.Background()), test.query.CloneVT(), schema.ClustersSchema, COUNT)
			require.NoError(t, err)
			require.NotNil(t, compiled)
			assert.Equal(t, test.params, compiled.Data)
			t.Logf("SQL: %s\nParameters: %#v", compiled.AsSQL(), compiled.Data)
		})
	}
}
