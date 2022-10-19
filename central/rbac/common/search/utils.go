package search

import (
	"strconv"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// ProcessQuery before running it against database
func ProcessQuery(q *v1.Query) {
	if q.GetQuery() == nil {
		return
	}
	switch typedQ := q.GetQuery().(type) {
	case *v1.Query_Disjunction:
		for _, subQ := range typedQ.Disjunction.GetQueries() {
			ProcessQuery(subQ)
		}
	case *v1.Query_Conjunction:
		for _, subQ := range typedQ.Conjunction.GetQueries() {
			ProcessQuery(subQ)
		}
	case *v1.Query_BaseQuery:
		matchFieldQuery, ok := typedQ.BaseQuery.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
		if !ok {
			return
		}

		if matchFieldQuery.MatchFieldQuery.GetField() == search.ClusterRole.String() {
			val, err := strconv.ParseBool(matchFieldQuery.MatchFieldQuery.GetValue())
			if err != nil {
				return
			}
			*q = *search.NewQueryBuilder().AddBools(search.ClusterRole, val).ProtoQuery()
		}
	default:
		return
	}
}
