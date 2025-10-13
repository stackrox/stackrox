package postgres

import (
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres/mapping"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/utils"
)

func scopeQuery(q *v1.Query, scopes []scoped.Scope) (*v1.Query, error) {
	pagination := q.GetPagination()
	q.Pagination = nil
	conjuncts := []*v1.Query{q}
	for _, scope := range scopes {
		schema := mapping.GetTableFromCategory(scope.Level)
		if schema == nil {
			utils.Should(errors.Errorf("no schema registered for search category %s", scope.Level))
			return q, nil
		}
		idField := schema.ID()
		if len(scope.IDs) > 0 {
			conjuncts = append(conjuncts, search.NewQueryBuilder().
				AddExactMatches(search.FieldLabel(idField.Search.FieldName), scope.IDs...).ProtoQuery())
		}
	}
	ret := search.ConjunctionQuery(conjuncts...)
	ret.Pagination = pagination
	return ret, nil
}
