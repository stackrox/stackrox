package administrationevents

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithWorkloadQuery(t *testing.T) {
	query := search.NewQueryBuilder().AddExactMatches(search.EventDomain, "Image Scanning").
		WithPagination(search.NewPagination().Limit(2).Offset(1)).ProtoQuery()
	workload := search.NewQueryBuilder().AddExactMatches(search.Cluster, "prod").
		WithPagination(search.NewPagination().Limit(1)).ProtoQuery()
	originalQuery, originalWorkload := query.CloneVT(), workload.CloneVT()
	filtered, err := WithWorkloadQuery(query, workload)
	require.NoError(t, err)
	protoassert.Equal(t, originalQuery, query)
	protoassert.Equal(t, originalWorkload, workload)
	protoassert.Equal(t, query.GetPagination(), filtered.GetPagination())
	var subqueries []*v1.MatchFieldSubQuery
	search.ApplyFnToAllBaseQueries(filtered, func(base *v1.BaseQuery) {
		if sub := base.GetMatchFieldSubQuery(); sub != nil {
			subqueries = append(subqueries, sub)
		}
	})
	require.Len(t, subqueries, 1)
	assert.Equal(t, v1.SearchCategory_DEPLOYMENTS, subqueries[0].GetCategory())
	assert.Equal(t, search.ResourceID.String(), subqueries[0].GetField())
	assert.Nil(t, subqueries[0].GetQuery().GetPagination(), "event pagination must not limit candidate images")
	protoassert.Equal(t, search.NewQuerySelect(search.ImageID).Proto(), subqueries[0].GetQuery().GetSelects()[0])

	unfiltered, err := WithWorkloadQuery(query, nil)
	require.NoError(t, err)
	protoassert.Equal(t, query, unfiltered)
}

func TestViewQueryValidation(t *testing.T) {
	view := NewEventView(nil)
	for name, query := range map[string]*v1.Query{
		"select":   search.NewQueryBuilder().AddSelectFields(search.NewQuerySelect(search.ResourceID)).ProtoQuery(),
		"group by": search.NewQueryBuilder().AddGroupBy(search.ResourceType).ProtoQuery(),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := view.Count(context.Background(), query)
			assert.Error(t, err)
			_, err = view.Get(context.Background(), query)
			assert.Error(t, err)
			_, err = WithWorkloadQuery(search.EmptyQuery(), query)
			assert.Error(t, err)
			_, err = WithWorkloadQuery(query, search.EmptyQuery())
			assert.Error(t, err)
		})
	}
}
