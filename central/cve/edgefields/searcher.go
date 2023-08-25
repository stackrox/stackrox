package edgefields

import (
	"context"
	"strconv"

	"github.com/gogo/protobuf/proto"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/scoped"
)

var (
	log = logging.LoggerForModule()
)

// TransformFixableFields transform fixable search fields for cluster vulnerabilities.
func TransformFixableFields(searcher search.Searcher) search.Searcher {
	return search.FuncSearcher{
		SearchFunc: func(ctx context.Context, q *v1.Query) ([]search.Result, error) {
			// Local copy to avoid changing input.
			local := q.Clone()
			pagination := local.GetPagination()
			local.Pagination = nil

			handleFixableQuery(local)

			local.Pagination = pagination
			return searcher.Search(ctx, local)
		},
		CountFunc: func(ctx context.Context, q *v1.Query) (int, error) {
			// Local copy to avoid changing input.
			local := q.Clone()
			pagination := local.GetPagination()
			local.Pagination = nil

			handleFixableQuery(local)

			local.Pagination = pagination
			return searcher.Count(ctx, local)
		},
	}
}

func handleFixableQuery(q *v1.Query) {
	if q.GetQuery() == nil {
		return
	}

	search.ApplyFnToAllBaseQueries(q, func(bq *v1.BaseQuery) {
		matchFieldQuery, ok := bq.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
		if !ok {
			return
		}

		if matchFieldQuery.MatchFieldQuery.GetField() == search.FixedBy.String() {
			matchFieldQuery.MatchFieldQuery.Field = search.ClusterCVEFixedBy.String()
		}

		if matchFieldQuery.MatchFieldQuery.GetField() == search.Fixable.String() {
			matchFieldQuery.MatchFieldQuery.Field = search.ClusterCVEFixable.String()
		}
	})
}

// HandleCVEEdgeSearchQuery handles the query cve edge query
func HandleCVEEdgeSearchQuery(searcher search.Searcher) search.Searcher {
	return search.FuncSearcher{
		SearchFunc: func(ctx context.Context, q *v1.Query) ([]search.Result, error) {
			// Local copy to avoid changing input.
			local := q.Clone()
			pagination := local.GetPagination()
			local.Pagination = nil

			getCVEEdgeQuery(local)

			local.Pagination = pagination
			return searcher.Search(ctx, local)
		},
		CountFunc: func(ctx context.Context, q *v1.Query) (int, error) {
			// Local copy to avoid changing input.
			local := q.Clone()
			pagination := local.GetPagination()
			local.Pagination = nil

			getCVEEdgeQuery(local)

			local.Pagination = pagination
			return searcher.Count(ctx, local)
		},
	}
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
			*q = *search.DisjunctionQuery(
				search.NewQueryBuilder().AddRegexes(search.FixedBy, matchFieldQuery.MatchFieldQuery.GetValue()).ProtoQuery(),
				search.NewQueryBuilder().AddRegexes(search.ClusterCVEFixedBy, matchFieldQuery.MatchFieldQuery.GetValue()).ProtoQuery())

		}

		if matchFieldQuery.MatchFieldQuery.GetField() == search.Fixable.String() {
			val, err := strconv.ParseBool(matchFieldQuery.MatchFieldQuery.GetValue())
			if err != nil {
				return
			}
			*q = *search.DisjunctionQuery(
				search.NewQueryBuilder().AddBools(search.Fixable, val).ProtoQuery(),
				search.NewQueryBuilder().AddBools(search.ClusterCVEFixable, val).ProtoQuery())
		}
	default:
		log.Errorf("Unhandled query type: %T; query was %s", q, proto.MarshalTextString(q))
	}
}

// HandleSnoozeSearchQuery ensures that when vulns are being searched by `Snoozed`,
// the vulns deferred by new workflow are also included.
func HandleSnoozeSearchQuery(searcher search.Searcher) search.Searcher {
	return search.FuncSearcher{
		SearchFunc: func(ctx context.Context, q *v1.Query) ([]search.Result, error) {
			// Local copy to avoid changing input.
			local := q.Clone()
			pagination := local.GetPagination()
			local.Pagination = nil

			local = handleSnoozedCVEQuery(ctx, local)

			local.Pagination = pagination
			return searcher.Search(ctx, local)
		},
		CountFunc: func(ctx context.Context, q *v1.Query) (int, error) {
			// Local copy to avoid changing input.
			local := q.Clone()
			pagination := local.GetPagination()
			local.Pagination = nil

			local = handleSnoozedCVEQuery(ctx, local)

			local.Pagination = pagination
			return searcher.Count(ctx, local)
		},
	}
}

func handleSnoozedCVEQuery(ctx context.Context, q *v1.Query) *v1.Query {
	var searchBySuppressed, searchByVulnState bool
	search.ApplyFnToAllBaseQueries(q, func(bq *v1.BaseQuery) {
		mfQ, ok := bq.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
		if ok && mfQ.MatchFieldQuery.GetField() == search.CVESuppressed.String() && mfQ.MatchFieldQuery.GetValue() == "true" {
			searchBySuppressed = true
		}
		if ok && mfQ.MatchFieldQuery.GetField() == search.VulnerabilityState.String() {
			searchByVulnState = true
		}
	})

	if !searchBySuppressed || searchByVulnState {
		return q
	}

	_, found := scoped.GetScopeAtLevel(ctx, v1.SearchCategory_IMAGES)
	if !found {
		return q
	}
	return search.ConjunctionQuery(
		q,
		search.NewQueryBuilder().AddExactMatches(
			search.VulnerabilityState,
			storage.VulnerabilityState_DEFERRED.String(),
			storage.VulnerabilityState_FALSE_POSITIVE.String(),
		).ProtoQuery())
}
