package postgres

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/require"
)

func TestSelectEmptyPredicates(t *testing.T) {
	for name, q := range map[string]*v1.Query{
		"empty query": search.EmptyQuery(),
		"conjunction": search.ConjunctionQuery(search.EmptyQuery(), search.EmptyQuery()),
		"disjunction": search.DisjunctionQuery(search.EmptyQuery(), search.EmptyQuery()),
		"nested": search.ConjunctionQuery(search.EmptyQuery(),
			search.DisjunctionQuery(search.EmptyQuery(), search.EmptyQuery())),
		"boolean": search.NewBooleanQuery(&v1.ConjunctionQuery{Queries: []*v1.Query{search.EmptyQuery()}}, nil),
	} {
		t.Run(name, func(t *testing.T) {
			q.Selects = []*v1.QuerySelect{search.NewQuerySelect(search.DeploymentName).Proto()}
			compiled, err := standardizeSelectQueryAndPopulatePath(sac.WithAllAccess(context.Background()), q, schema.DeploymentsSchema, SELECT, nil)
			require.NoError(t, err)
			require.NotNil(t, compiled, "empty filters must not suppress SELECT results")
		})
	}
}
