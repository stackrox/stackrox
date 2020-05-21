package cveedge

import (
	"context"
	"strconv"

	"github.com/gogo/protobuf/proto"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
)

var (
	log = logging.LoggerForModule()
)

// HandleCVEEdgeSearchQuery handles the query cve edge query
func HandleCVEEdgeSearchQuery(searcher search.Searcher) search.Searcher {
	return search.Func(func(ctx context.Context, q *v1.Query) ([]search.Result, error) {
		// Local copy to avoid changing input.
		local := q.Clone()
		pagination := local.GetPagination()
		local.Pagination = nil

		getCVEEdgeQuery(local)

		local.Pagination = pagination
		return searcher.Search(ctx, local)
	})
}

func getCVEEdgeQuery(q *v1.Query) {
	if q.GetQuery() == nil {
		return
	}

	switch typedQ := q.GetQuery().(type) {
	case *v1.Query_Disjunction:
		for _, subQ := range typedQ.Disjunction.GetQueries() {
			getCVEEdgeQuery(subQ)
		}
	case *v1.Query_Conjunction:
		for _, subQ := range typedQ.Conjunction.GetQueries() {
			getCVEEdgeQuery(subQ)
		}
	case *v1.Query_BaseQuery:
		matchFieldQuery, ok := typedQ.BaseQuery.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
		if !ok {
			return
		}

		if matchFieldQuery.MatchFieldQuery.GetField() == search.FixedBy.String() {
			*q = *search.NewDisjunctionQuery(
				search.NewQueryBuilder().AddRegexes(search.FixedBy, matchFieldQuery.MatchFieldQuery.GetValue()).ProtoQuery(),
				search.NewQueryBuilder().AddRegexes(search.ClusterCVEFixedBy, matchFieldQuery.MatchFieldQuery.GetValue()).ProtoQuery())

		}

		if matchFieldQuery.MatchFieldQuery.GetField() == search.Fixable.String() {
			val, err := strconv.ParseBool(matchFieldQuery.MatchFieldQuery.GetValue())
			if err != nil {
				return
			}
			*q = *search.NewDisjunctionQuery(
				search.NewQueryBuilder().AddBools(search.Fixable, val).ProtoQuery(),
				search.NewQueryBuilder().AddBools(search.ClusterCVEFixable, val).ProtoQuery())
		}
	default:
		log.Errorf("Unhandled query type: %T; query was %s", q, proto.MarshalTextString(q))
	}
}
