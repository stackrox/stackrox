package common

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
)

// WithCountQuery returns a query to count the number of distinct values of the given field
func WithCountQuery(q *v1.Query, field search.FieldLabel) *v1.Query {
	cloned := q.Clone()
	cloned.Selects = []*v1.QuerySelect{
		search.NewQuerySelect(field).AggrFunc(aggregatefunc.Count).Distinct().Proto(),
	}
	return cloned
}
