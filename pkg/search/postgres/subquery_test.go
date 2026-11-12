package postgres

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubqueryBindings(t *testing.T) {
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

func TestInvalidSubQuery(t *testing.T) {
	ctx := sac.WithAllAccess(context.Background())
	valid := search.NewQueryBuilder().AddSelectFields(search.NewQuerySelect(search.ImageID)).ProtoQuery()
	unsupportedAggregate := search.NewQueryBuilder().AddSelectFields(search.NewQuerySelect(search.DeploymentName).AggrFunc(aggregatefunc.Count)).ProtoQuery()
	unsupportedAggregate.Selects[0].Filter = &v1.QuerySelectFilter{
		Query: search.NewQueryBuilder().AddExactMatches(search.ResourceID, "unsupported").ProtoQuery(),
	}
	for name, test := range map[string]struct {
		field    search.FieldLabel
		category v1.SearchCategory
		query    *v1.Query
	}{
		"no category":   {field: search.DeploymentName, query: valid},
		"nil query":     {field: search.DeploymentName, category: v1.SearchCategory_DEPLOYMENTS},
		"no selects":    {field: search.DeploymentName, category: v1.SearchCategory_DEPLOYMENTS, query: search.EmptyQuery()},
		"unknown field": {field: search.ResourceID, category: v1.SearchCategory_DEPLOYMENTS, query: valid},
		"derived field": {field: search.DeploymentCount, category: v1.SearchCategory_DEPLOYMENTS, query: valid},
		"array field":   {field: search.AddCapabilities, category: v1.SearchCategory_DEPLOYMENTS, query: valid},
		"multiple selects": {field: search.DeploymentName, category: v1.SearchCategory_DEPLOYMENTS,
			query: search.NewQueryBuilder().AddSelectFields(search.NewQuerySelect(search.ImageID), search.NewQuerySelect(search.ImageName)).ProtoQuery()},
		"array select": {field: search.DeploymentName, category: v1.SearchCategory_DEPLOYMENTS,
			query: search.NewQueryBuilder().AddSelectFields(search.NewQuerySelect(search.AddCapabilities)).ProtoQuery()},
		"empty compiled projection": {field: search.DeploymentName, category: v1.SearchCategory_DEPLOYMENTS, query: unsupportedAggregate},
		"implicit JSON aggregate": {field: search.DeploymentName, category: v1.SearchCategory_DEPLOYMENTS,
			query: search.NewQueryBuilder().AddSelectFields(search.NewQuerySelect(search.DeploymentName)).AddGroupBy(search.Cluster).ProtoQuery()},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := standardizeQueryAndPopulatePath(ctx, search.MatchFieldSubQuery(test.field, test.category, test.query), schema.DeploymentsSchema, GET)
			assert.ErrorIs(t, err, errox.InvalidArgs)
		})
	}
}
