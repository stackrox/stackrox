package search

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/ranking"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

// These are utilities for handling the fact that we show a synthetic "Priority" field on the UI and want the user
// to be able to interact with that field as any other. To accomplish this, we swap any uses of the synthetic Priority
// field to instead use the field that the priority is based on: Risk Score. A higher Risk Score means a lower Priority
// number (Priority is a rank, so Priority 1 is the highest Priority), so we need to invert numeric queries, and swap
// the Priority rank number for the Risk Score value.
///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// If priority is used to sort, swap with Risk Score.
/////////////////////////////////////////////////////
func swapPrioritySort(searcher search.Searcher) search.Searcher {
	return &swapPrioritySortImpl{
		searcher: searcher,
	}
}

type swapPrioritySortImpl struct {
	searcher search.Searcher
}

func (ds *swapPrioritySortImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	if q.GetPagination().GetSortOption().GetField() != search.Priority.String() {
		return ds.searcher.Search(ctx, q)
	}

	newQuery := proto.Clone(q).(*v1.Query)
	newQuery.Pagination.SortOption.Field = search.RiskScore.String()
	newQuery.Pagination.SortOption.Reversed = !q.GetPagination().GetSortOption().GetReversed()
	return ds.searcher.Search(ctx, newQuery)
}

// If Priority is used to filter, swap with Risk Score.
///////////////////////////////////////////////////////
func swapPriorityQuery(searcher search.Searcher) search.Searcher {
	return &swapPriorityQueryImpl{
		searcher: searcher,
		ranker:   ranking.DeploymentRanker(),
	}
}

type swapPriorityQueryImpl struct {
	searcher search.Searcher
	ranker   *ranking.Ranker
}

func (ds *swapPriorityQueryImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	var err error
	newQuery := proto.Clone(q).(*v1.Query)
	search.ApplyFnToAllBaseQueries(newQuery, func(bq *v1.BaseQuery) {
		matchFieldQuery, ok := bq.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
		if !ok {
			return
		}

		// Any Priority query, we want to change to be a risk query.
		if matchFieldQuery.MatchFieldQuery.GetField() == search.Priority.String() {
			// Parse the numeric query so we can swap it to a risk score query.
			var numericValue blevesearch.NumericQueryValue
			numericValue, err = blevesearch.ParseNumericQueryValue(matchFieldQuery.MatchFieldQuery.GetValue())
			if err == nil {
				return
			}

			// Go from priority space to risk score space by inverting comparison.
			numericValue.Comparator = priorityComparatorToRiskScoreComparator(numericValue.Comparator)
			numericValue.Value = float64(ds.ranker.GetScoreForRank(int64(numericValue.Value)))

			// Set the query to the new value.
			matchFieldQuery.MatchFieldQuery.Field = search.RiskScore.String()
			matchFieldQuery.MatchFieldQuery.Value = blevesearch.PrintNumericQueryValue(numericValue)
		}
	})
	if err != nil {
		return nil, err
	}

	return ds.searcher.Search(ctx, newQuery)
}

func priorityComparatorToRiskScoreComparator(comparator storage.Comparator) storage.Comparator {
	switch comparator {
	case storage.Comparator_LESS_THAN_OR_EQUALS:
		return storage.Comparator_GREATER_THAN
	case storage.Comparator_LESS_THAN:
		return storage.Comparator_GREATER_THAN_OR_EQUALS
	case storage.Comparator_GREATER_THAN_OR_EQUALS:
		return storage.Comparator_LESS_THAN
	case storage.Comparator_GREATER_THAN:
		return storage.Comparator_LESS_THAN_OR_EQUALS
	default: // storage.Comparator_EQUALS:
		return storage.Comparator_EQUALS
	}
}
