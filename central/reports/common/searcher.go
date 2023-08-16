package common

import (
	"context"
	"fmt"
	"strings"

	"github.com/gogo/protobuf/proto"
	v1 "github.com/stackrox/rox/generated/api/v1"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

// TransformReportStateSearchValues transforms match field queries with Report State == SUCCESS or Report State == "SUCCESS" (Exact match)
// to a disjunction query Report State == "GENERATED" OR Report State == "DELIVERED" query.
func TransformReportStateSearchValues(searcher search.Searcher) search.Searcher {
	return search.FuncSearcher{
		SearchFunc: func(ctx context.Context, q *v1.Query) ([]search.Result, error) {
			// Local copy to avoid changing input.
			local := q.Clone()
			pagination := local.GetPagination()
			local.Pagination = nil

			replaceSearchBySuccess(local)

			local.Pagination = pagination
			return searcher.Search(ctx, local)
		},
		CountFunc: func(ctx context.Context, q *v1.Query) (int, error) {
			// Local copy to avoid changing input.
			local := q.Clone()
			pagination := local.GetPagination()
			local.Pagination = nil

			replaceSearchBySuccess(local)

			local.Pagination = pagination
			return searcher.Count(ctx, local)
		},
	}
}

// Replace match field queries of type Report State == SUCCESS and Report State == "SUCCESS" (Exact match)
// Note : This func does no replacement
//  1. if the query is a prefix query designed to match any non-complete prefix of 'SUCCESS'
//  2. if the query is a regex query designed to match 'SUCCESS' among other matches
//  3. if the query is a negation query designed to not match 'SUCCESS' or any of its prefixes
//
// To avoid these discrepancies we should stop supporting 'SUCCESS' as a value for "Report State" field in the long run.
func replaceSearchBySuccess(q *v1.Query) {
	if q.GetQuery() == nil {
		return
	}
	switch typedQ := q.GetQuery().(type) {
	case *v1.Query_Disjunction:
		for _, subQ := range typedQ.Disjunction.GetQueries() {
			replaceSearchBySuccess(subQ)
		}
	case *v1.Query_Conjunction:
		for _, subQ := range typedQ.Conjunction.GetQueries() {
			replaceSearchBySuccess(subQ)
		}
	case *v1.Query_BooleanQuery:
		for _, subQ := range typedQ.BooleanQuery.GetMust().GetQueries() {
			replaceSearchBySuccess(subQ)
		}
		for _, subQ := range typedQ.BooleanQuery.GetMustNot().GetQueries() {
			replaceSearchBySuccess(subQ)
		}
	case *v1.Query_BaseQuery:
		mfQ, ok := typedQ.BaseQuery.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
		if !ok {
			return
		}
		if strings.ToLower(mfQ.MatchFieldQuery.GetField()) == strings.ToLower(search.ReportState.String()) &&
			doesValueMatchSuccess(mfQ.MatchFieldQuery.GetValue()) {
			*q = *search.NewQueryBuilder().
				AddExactMatches(search.ReportState, storage.ReportStatus_GENERATED.String(), storage.ReportStatus_DELIVERED.String()).
				ProtoQuery()
		}
	default:
		utils.Should(fmt.Errorf("unhandled query type: %T; query was %s", q, proto.MarshalTextString(q)))
	}
}

func doesValueMatchSuccess(val string) bool {
	return strings.ToLower(val) == strings.ToLower(apiV2.ReportStatus_SUCCESS.String()) ||
		strings.ToLower(val) == strings.ToLower(search.ExactMatchString(apiV2.ReportStatus_SUCCESS.String()))
}
