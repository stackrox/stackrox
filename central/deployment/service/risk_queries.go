package service

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/ranking"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

func filterDeploymentQuery(q *v1.Query) *v1.Query {
	// Filter the query.
	newQuery, _ := search.FilterQuery(proto.Clone(q).(*v1.Query), func(bq *v1.BaseQuery) bool {
		matchFieldQuery, ok := bq.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
		if !ok {
			return false
		}
		return matchFieldQuery.MatchFieldQuery.GetField() != search.Priority.String()
	})
	if newQuery != nil {
		newQuery.Pagination = nil
	}
	return newQuery
}

func filterDeploymentPagination(q *v1.Query) *v1.QueryPagination {
	// Filter the pagination.
	if len(q.GetPagination().GetSortOptions()) == 1 &&
		q.GetPagination().GetSortOptions()[0].Field != search.Priority.String() {
		return proto.Clone(q.Pagination).(*v1.QueryPagination)
	}
	return nil
}

func filterRiskPagination(q *v1.Query) *v1.QueryPagination {
	if q == nil {
		return nil
	}

	// If the one sort option in the query is priority, add risk based pagination, otherwise skip pagination.
	if len(q.GetPagination().GetSortOptions()) == 1 &&
		q.GetPagination().GetSortOptions()[0].GetField() == search.Priority.String() {
		newPagination := proto.Clone(q.Pagination).(*v1.QueryPagination)
		newPagination.GetSortOptions()[0].Field = search.AggregateRiskScore.String()
		newPagination.GetSortOptions()[0].Reversed = !q.GetPagination().GetSortOptions()[0].Reversed
		return newPagination
	}
	return nil
}

func filterRiskQuery(q *v1.Query, ranker *ranking.Ranker) *v1.Query {
	if q == nil {
		return nil
	}

	newQuery, _ := search.FilterQuery(proto.Clone(q).(*v1.Query), func(bq *v1.BaseQuery) bool {
		matchFieldQuery, ok := bq.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
		if !ok {
			return false
		}
		return matchFieldQuery.MatchFieldQuery.GetField() == search.Priority.String()
	})
	if newQuery == nil {
		return nil
	}
	newQuery.Pagination = nil

	var err error
	search.ApplyFnToAllBaseQueries(newQuery, func(bq *v1.BaseQuery) {
		matchFieldQuery, ok := bq.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
		if !ok {
			return
		}
		// Any Priority query, we want to change to be a risk query.
		// Parse the numeric query so we can swap it to a risk score query.
		var numericValue blevesearch.NumericQueryValue
		numericValue, err = blevesearch.ParseNumericQueryValue(matchFieldQuery.MatchFieldQuery.GetValue())
		if err != nil {
			return
		}

		// Go from priority space to risk score space by inverting comparison.
		numericValue.Comparator = priorityComparatorToRiskScoreComparator(numericValue.Comparator)
		numericValue.Value = float64(ranker.GetScoreForRank(int64(numericValue.Value)))

		// Set the query to the new value.
		matchFieldQuery.MatchFieldQuery.Field = search.AggregateRiskScore.String()
		matchFieldQuery.MatchFieldQuery.Value = blevesearch.PrintNumericQueryValue(numericValue)
	})
	if err != nil {
		log.Error(err)
		return nil
	}

	// If we end up with a query, add the deployment type specification for it.
	return search.ConjunctionQuery(
		search.NewQueryBuilder().
			AddStrings(search.RiskEntityType, storage.RiskEntityType_DEPLOYMENT.String()).
			ProtoQuery(),
		newQuery,
	)
}

func priorityComparatorToRiskScoreComparator(comparator storage.Comparator) storage.Comparator {
	switch comparator {
	case storage.Comparator_LESS_THAN_OR_EQUALS:
		return storage.Comparator_GREATER_THAN_OR_EQUALS
	case storage.Comparator_LESS_THAN:
		return storage.Comparator_GREATER_THAN
	case storage.Comparator_GREATER_THAN_OR_EQUALS:
		return storage.Comparator_LESS_THAN_OR_EQUALS
	case storage.Comparator_GREATER_THAN:
		return storage.Comparator_LESS_THAN
	default: // storage.Comparator_EQUALS:
		return storage.Comparator_EQUALS
	}
}
