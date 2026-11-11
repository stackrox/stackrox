package postgres

import (
	"context"
	"fmt"
	"strings"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
)

func namespaceScopeFixture(clusterCount, namespaceCount int) ([]string, map[string]*sac.TestClusterScope) {
	names := make([]string, namespaceCount)
	for i := range names {
		names[i] = fmt.Sprintf("namespace-%d", i)
	}
	ids := make([]string, clusterCount)
	clusters := make(map[string]*sac.TestClusterScope, clusterCount)
	for i := range ids {
		ids[i] = uuid.NewV4().String()
		clusters[ids[i]] = &sac.TestClusterScope{Included: namespaceCount == 0, Namespaces: names}
	}
	return ids, clusters
}

func namespaceScopeContext(t *testing.T, clusters map[string]*sac.TestClusterScope) context.Context {
	t.Helper()
	return sac.WithGlobalAccessScopeChecker(t.Context(), sac.TestScopeCheckerCoreFromFullScopeMap(t, sac.TestScopeMap{
		storage.Access_READ_ACCESS: {
			resources.Deployment.Resource: {Clusters: clusters},
		},
		storage.Access_READ_WRITE_ACCESS: {
			resources.Deployment.Resource: {Clusters: clusters},
		},
	}))
}

func TestSACFilterParameterCounts(t *testing.T) {
	t.Setenv(env.PostgresParameterThreshold.EnvVar(), "100")
	for name, tc := range map[string]struct {
		clusters   int
		namespaces int
		queryType  QueryType
		parameters int
	}{
		"many small namespace lists": {700, 99, SEARCH, 1400},
		"count":                      {2, 2, COUNT, 4},
		"get":                        {2, 2, GET, 4},
		"delete":                     {2, 2, DELETE, 4},
		"delete returning IDs":       {2, 2, DELETERETURNINGIDS, 4},
		"single namespace":           {2, 1, SEARCH, 4},
		"whole clusters":             {2, 0, SEARCH, 1},
	} {
		t.Run(name, func(t *testing.T) {
			_, clusters := namespaceScopeFixture(tc.clusters, tc.namespaces)
			parsed, err := standardizeQueryAndPopulatePath(namespaceScopeContext(t, clusters), search.EmptyQuery(),
				schema.DeploymentsSchema, tc.queryType)
			require.NoError(t, err)
			require.NotNil(t, parsed)
			require.Equal(t, tc.parameters, len(parsed.Data))
		})
	}
}

func TestScopeFieldArrayParameters(t *testing.T) {
	t.Setenv(env.PostgresParameterThreshold.EnvVar(), "100")
	for name, tc := range map[string]struct {
		field  search.FieldLabel
		schema *walker.Schema
		values []string
		array  bool
	}{
		"namespace":               {search.Namespace, schema.DeploymentsSchema, []string{"one", "two"}, true},
		"namespace metadata name": {search.Namespace, schema.NamespacesSchema, []string{"one", "two"}, true},
		"cluster IDs":             {search.ClusterID, schema.DeploymentsSchema, []string{uuid.NewV4().String(), uuid.NewV4().String()}, true},
		"other fields keep IN":    {search.DeploymentName, schema.DeploymentsSchema, []string{"one", "two"}, false},
	} {
		t.Run(name, func(t *testing.T) {
			query := search.NewQueryBuilder().AddExactMatches(tc.field, tc.values...).ProtoQuery()
			parsed, err := standardizeQueryAndPopulatePath(sac.WithAllAccess(t.Context()), query, tc.schema, COUNT)
			require.NoError(t, err)
			if tc.array {
				require.Len(t, parsed.Data, 1)
				require.Contains(t, parsed.Where, " = ANY($$)")
			} else {
				require.Len(t, parsed.Data, 2)
				require.Contains(t, parsed.Where, " IN ($$, $$)")
			}
		})
	}
}

func TestSACInclusionExclusionParameterCounts(t *testing.T) {
	t.Setenv(env.PostgresParameterThreshold.EnvVar(), "100")
	ids, clusters := namespaceScopeFixture(700, 99)
	parsed, err := standardizeQueryAndPopulatePath(sac.WithAllAccess(t.Context()), namespaceExclusionQuery(ids, clusters),
		schema.DeploymentsSchema, COUNT)
	require.NoError(t, err)
	require.Equal(t, 2100, len(parsed.Data))
	require.Equal(t, 1400, strings.Count(parsed.Where, " = ANY("))
}

func namespaceExclusionQuery(ids []string, clusters map[string]*sac.TestClusterScope) *v1.Query {
	filters := make([]*v1.Query, 0, len(ids))
	for _, id := range ids {
		filters = append(filters, search.NewBooleanQuery(
			&v1.ConjunctionQuery{Queries: []*v1.Query{
				search.NewQueryBuilder().AddExactMatches(search.ClusterID, id).ProtoQuery(),
				search.NewQueryBuilder().AddExactMatches(search.Namespace, clusters[id].Namespaces...).ProtoQuery(),
			}},
			&v1.DisjunctionQuery{Queries: []*v1.Query{
				search.NewQueryBuilder().AddExactMatches(search.Namespace, "namespace-1", "namespace-2").ProtoQuery(),
			}}))
	}
	return search.DisjunctionQuery(filters...)
}
