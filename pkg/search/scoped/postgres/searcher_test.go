package postgres

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres/mapping"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stretchr/testify/assert"
)

func TestScoping(t *testing.T) {
	if mapping.GetTableFromCategory(v1.SearchCategory_CLUSTERS) == nil {
		mapping.RegisterCategoryToTable(v1.SearchCategory_CLUSTERS, schema.ClustersSchema)
	}
	if mapping.GetTableFromCategory(v1.SearchCategory_NAMESPACES) == nil {
		mapping.RegisterCategoryToTable(v1.SearchCategory_NAMESPACES, schema.NamespacesSchema)
	}
	query := search.NewQueryBuilder().AddExactMatches(search.DeploymentName, "dep").ProtoQuery()
	scopes := []scoped.Scope{
		{
			ID:    "c1",
			Level: v1.SearchCategory_CLUSTERS,
		},
	}
	expected := search.ConjunctionQuery(
		query,
		search.NewQueryBuilder().AddExactMatches(search.ClusterID, "c1").ProtoQuery(),
	)
	actual, err := scopeQuery(query, scopes)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)

	scopes = []scoped.Scope{
		{
			ID:    "c1",
			Level: v1.SearchCategory_CLUSTERS,
		},
		{
			ID:    "n1",
			Level: v1.SearchCategory_NAMESPACES,
		},
	}
	expected = search.ConjunctionQuery(
		query,
		search.NewQueryBuilder().AddExactMatches(search.ClusterID, "c1").ProtoQuery(),
		search.NewQueryBuilder().AddExactMatches(search.NamespaceID, "n1").ProtoQuery(),
	)
	actual, err = scopeQuery(query, scopes)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}
