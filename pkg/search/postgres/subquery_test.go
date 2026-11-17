package postgres

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubQueryParameters(t *testing.T) {
	ctx := sac.WithAllAccess(context.Background())
	inner := search.NewQueryBuilder().AddExactMatches(search.DeploymentName, "inner $$ ' value").
		AddSelectFields(search.NewQuerySelect(search.ImageID)).ProtoQuery()
	original := inner.CloneVT()
	outer := search.ConjunctionQuery(
		search.NewQueryBuilder().AddExactMatches(search.DeploymentName, "outer").
			AddStrings(search.DeploymentCount, ">1").ProtoQuery(),
		search.MatchFieldSubQuery(search.DeploymentName, v1.SearchCategory_DEPLOYMENTS, inner),
	)
	compiled, err := standardizeQueryAndPopulatePath(ctx, outer, schema.DeploymentsSchema, COUNT)
	require.NoError(t, err)
	require.NotNil(t, compiled)
	protoassert.Equal(t, original, inner, "building a subquery must not mutate its input")
	assert.Equal(t, []any{"outer", "inner $$ ' value", "1"}, compiled.Data)
	assert.NotContains(t, compiled.AsSQL(), "inner $$ ' value")
	assert.Contains(t, compiled.AsSQL(), "deployments.Name = $1")
	assert.Contains(t, compiled.AsSQL(), "deployments.Name = $2")
	assert.Contains(t, compiled.AsSQL(), "count(deployments.Id) > $3")
}

func TestSubQueryPredicates(t *testing.T) {
	unsupported := search.NewQueryBuilder().AddExactMatches(search.FieldLabel("unsupported"), "value").ProtoQuery()
	supported := search.NewQueryBuilder().AddExactMatches(search.DeploymentName, "app").ProtoQuery()
	nested := supported.CloneVT()
	nested.Selects = []*v1.QuerySelect{search.NewQuerySelect(search.DeploymentName).Proto()}
	contexts := map[string]context.Context{
		"all access": sac.WithAllAccess(context.Background()),
		"namespace access": sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS), sac.ResourceScopeKeys(resources.Deployment),
			sac.ClusterScopeKeys("00000000-0000-4000-8000-000000000001"), sac.NamespaceScopeKeys("payments"))),
	}
	for name, test := range map[string]struct {
		query     *v1.Query
		matchNone bool
	}{
		"empty query":         {query: search.EmptyQuery()},
		"empty conjunction":   {query: search.ConjunctionQuery(search.EmptyQuery(), search.EmptyQuery())},
		"empty disjunction":   {query: search.DisjunctionQuery(search.EmptyQuery(), search.EmptyQuery())},
		"supported predicate": {query: supported},
		"supported boolean": {query: search.NewBooleanQuery(nil,
			&v1.DisjunctionQuery{Queries: []*v1.Query{supported}})},
		"supported nested membership": {query: search.MatchFieldSubQuery(search.DeploymentName, v1.SearchCategory_DEPLOYMENTS, nested)},
		"unsupported nested membership field": {
			query: search.MatchFieldSubQuery(search.FieldLabel("unsupported"), v1.SearchCategory_DEPLOYMENTS, nested), matchNone: true,
		},
		"unsupported base query": {query: unsupported, matchNone: true},
		"unsupported conjunction": {
			query: search.ConjunctionQuery(unsupported, search.EmptyQuery()), matchNone: true,
		},
		"unsupported disjunction": {
			query: search.DisjunctionQuery(unsupported, search.EmptyQuery()), matchNone: true,
		},
		"mixed conjunction": {query: search.ConjunctionQuery(supported, unsupported), matchNone: true},
		"mixed disjunction": {query: search.DisjunctionQuery(supported, unsupported), matchNone: true},
		"unsupported boolean must": {query: search.NewBooleanQuery(
			&v1.ConjunctionQuery{Queries: []*v1.Query{unsupported}}, nil), matchNone: true},
		"unsupported boolean must not": {query: search.NewBooleanQuery(nil,
			&v1.DisjunctionQuery{Queries: []*v1.Query{unsupported}}), matchNone: true},
		"mixed boolean must": {query: search.NewBooleanQuery(
			&v1.ConjunctionQuery{Queries: []*v1.Query{supported, unsupported}}, nil), matchNone: true},
		"mixed boolean must not": {query: search.NewBooleanQuery(
			&v1.ConjunctionQuery{Queries: []*v1.Query{supported}},
			&v1.DisjunctionQuery{Queries: []*v1.Query{unsupported}}), matchNone: true},
		"nested boolean": {query: search.ConjunctionQuery(supported, search.NewBooleanQuery(nil,
			&v1.DisjunctionQuery{Queries: []*v1.Query{search.DisjunctionQuery(supported, unsupported)}})), matchNone: true},
		"mixed linked fields": {query: search.NewQueryBuilder().AddLinkedFields(
			[]search.FieldLabel{search.DeploymentName, search.FieldLabel("unsupported")}, []string{"app", "value"}).ProtoQuery(), matchNone: true},
	} {
		t.Run(name, func(t *testing.T) {
			for name, ctx := range contexts {
				t.Run(name, func(t *testing.T) {
					inner := test.query.CloneVT()
					inner.Selects = []*v1.QuerySelect{search.NewQuerySelect(search.DeploymentName).Proto()}
					q := search.MatchFieldSubQuery(search.DeploymentName, v1.SearchCategory_DEPLOYMENTS, inner)
					compiled, err := standardizeQueryAndPopulatePath(ctx, q, schema.DeploymentsSchema, COUNT)
					require.NoError(t, err)
					require.NotNil(t, compiled)
					if test.matchNone {
						assert.Contains(t, compiled.Where, "false")
						assert.NotContains(t, compiled.Where, "EXISTS")
					} else {
						assert.Contains(t, compiled.Where, "EXISTS (SELECT 1 FROM (")
					}
				})
			}
		})
	}
}

func TestInvalidSubQuery(t *testing.T) {
	ctx := sac.WithAllAccess(context.Background())
	valid := search.NewQueryBuilder().AddSelectFields(search.NewQuerySelect(search.ImageID)).ProtoQuery()
	unsupportedAggregate := search.NewQueryBuilder().AddSelectFields(search.NewQuerySelect(search.DeploymentName).AggrFunc(aggregatefunc.Count)).ProtoQuery()
	unsupportedAggregate.Selects[0].Filter = &v1.QuerySelectFilter{
		Query: search.NewQueryBuilder().AddExactMatches(search.FieldLabel("unsupported"), "value").ProtoQuery(),
	}
	filteredSelection := valid.CloneVT()
	filteredSelection.Selects[0].Filter = &v1.QuerySelectFilter{
		Query: search.NewQueryBuilder().AddExactMatches(search.DeploymentName, "app").ProtoQuery(),
	}
	aggregateOrdering := valid.CloneVT()
	aggregateOrdering.Pagination = &v1.QueryPagination{SortOptions: []*v1.QuerySortOption{{
		Field: search.DeploymentName.String(), AggregateBy: &v1.AggregateBy{AggrFunc: v1.Aggregation_COUNT},
	}}}
	for name, test := range map[string]struct {
		field    search.FieldLabel
		category v1.SearchCategory
		query    *v1.Query
	}{
		"no category":   {field: search.DeploymentName, query: valid},
		"nil query":     {field: search.DeploymentName, category: v1.SearchCategory_DEPLOYMENTS},
		"no selects":    {field: search.DeploymentName, category: v1.SearchCategory_DEPLOYMENTS, query: search.EmptyQuery()},
		"unknown field": {field: search.FieldLabel("unsupported"), category: v1.SearchCategory_DEPLOYMENTS, query: valid},
		"derived field": {field: search.DeploymentCount, category: v1.SearchCategory_DEPLOYMENTS, query: valid},
		"array field":   {field: search.AddCapabilities, category: v1.SearchCategory_DEPLOYMENTS, query: valid},
		"multiple selects": {field: search.DeploymentName, category: v1.SearchCategory_DEPLOYMENTS,
			query: search.NewQueryBuilder().AddSelectFields(search.NewQuerySelect(search.ImageID), search.NewQuerySelect(search.ImageName)).ProtoQuery()},
		"array select": {field: search.DeploymentName, category: v1.SearchCategory_DEPLOYMENTS,
			query: search.NewQueryBuilder().AddSelectFields(search.NewQuerySelect(search.AddCapabilities)).ProtoQuery()},
		"derived max selection": {field: search.DeploymentName, category: v1.SearchCategory_DEPLOYMENTS,
			query: search.NewQueryBuilder().AddSelectFields(search.NewQuerySelect(search.CVSSMax)).ProtoQuery()},
		"derived priority selection": {field: search.DeploymentName, category: v1.SearchCategory_DEPLOYMENTS,
			query: search.NewQueryBuilder().AddSelectFields(search.NewQuerySelect(search.DeploymentPriority)).ProtoQuery()},
		"filtered aggregate": {field: search.DeploymentName, category: v1.SearchCategory_DEPLOYMENTS, query: unsupportedAggregate},
		"select filter":      {field: search.DeploymentName, category: v1.SearchCategory_DEPLOYMENTS, query: filteredSelection},
		"count selection": {field: search.DeploymentName, category: v1.SearchCategory_DEPLOYMENTS,
			query: search.NewQueryBuilder().AddSelectFields(search.NewQuerySelect(search.DeploymentName).AggrFunc(aggregatefunc.Count)).ProtoQuery()},
		"max selection": {field: search.DeploymentName, category: v1.SearchCategory_DEPLOYMENTS,
			query: search.NewQueryBuilder().AddSelectFields(search.NewQuerySelect(search.DeploymentName).AggrFunc(aggregatefunc.Max)).ProtoQuery()},
		"grouped selection": {field: search.DeploymentName, category: v1.SearchCategory_DEPLOYMENTS,
			query: search.NewQueryBuilder().AddSelectFields(search.NewQuerySelect(search.DeploymentName)).AddGroupBy(search.DeploymentName).ProtoQuery()},
		"grouping another field": {field: search.DeploymentName, category: v1.SearchCategory_DEPLOYMENTS,
			query: search.NewQueryBuilder().AddSelectFields(search.NewQuerySelect(search.DeploymentName)).AddGroupBy(search.Cluster).ProtoQuery()},
		"implicit filter grouping": {field: search.DeploymentName, category: v1.SearchCategory_DEPLOYMENTS,
			query: search.NewQueryBuilder().AddSelectFields(search.NewQuerySelect(search.DeploymentName)).AddStrings(search.DeploymentCount, ">1").ProtoQuery()},
		"aggregate filter without grouping": {field: search.DeploymentName, category: v1.SearchCategory_DEPLOYMENTS,
			query: search.NewQueryBuilder().AddSelectFields(search.NewQuerySelect(search.DeploymentName)).AddStrings(search.CVSSMax, ">1").ProtoQuery()},
		"implicit sort grouping": {field: search.DeploymentName, category: v1.SearchCategory_DEPLOYMENTS,
			query: search.NewQueryBuilder().AddSelectFields(search.NewQuerySelect(search.DeploymentName)).
				WithPagination(search.NewPagination().AddSortOption(search.NewSortOption(search.DeploymentCount))).ProtoQuery()},
		"aggregate ordering": {field: search.DeploymentName, category: v1.SearchCategory_DEPLOYMENTS, query: aggregateOrdering},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := standardizeQueryAndPopulatePath(ctx, search.MatchFieldSubQuery(test.field, test.category, test.query), schema.DeploymentsSchema, GET)
			assert.ErrorIs(t, err, errox.InvalidArgs)
		})
	}
}
