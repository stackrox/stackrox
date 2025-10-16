package common

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// WithoutOrphanedNodeCVEsQuery adds a filter to the query to exclude Orphaned Node CVEs from search results
func WithoutOrphanedNodeCVEsQuery(q *v1.Query) *v1.Query {
	ret := search.ConjunctionQuery(q, search.NewQueryBuilder().AddBools(search.CVEOrphaned, false).ProtoQuery())
	ret.SetPagination(q.GetPagination())
	return ret
}
