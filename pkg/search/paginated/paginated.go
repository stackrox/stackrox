package paginated

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/search"
)

// WithDefaultSortOption is a higher order function that makes sure results are sorted.
func WithDefaultSortOption(searcher search.Searcher, defaultSortOption *v1.QuerySortOption) search.Searcher {
	return search.FuncSearcher{
		SearchFunc: func(ctx context.Context, q *v1.Query) ([]search.Result, error) {
			// Add pagination sort order if needed.
			local := FillDefaultSortOption(q, defaultSortOption.Clone())
			return searcher.Search(ctx, local)
		},
		CountFunc: func(ctx context.Context, q *v1.Query) (int, error) {
			return searcher.Count(ctx, q)
		},
	}
}

// Paginated is a higher order function for applying pagination.
func Paginated(searcher search.Searcher) search.Searcher {
	return search.FuncSearcher{
		SearchFunc: func(ctx context.Context, q *v1.Query) ([]search.Result, error) {
			// If pagination not set, just skip.
			if q.GetPagination() == nil {
				return searcher.Search(ctx, q)
			}

			// Local copy to avoid changing input.
			local := q.Clone()

			// Record used settings.
			offset := int(local.GetPagination().GetOffset())
			local.Pagination.Offset = 0
			limit := int(local.GetPagination().GetLimit())
			local.Pagination.Limit = 0

			// Run an paginate results.
			results, err := searcher.Search(ctx, local)
			return paginate(offset, limit, results, err)
		},
		CountFunc: func(ctx context.Context, q *v1.Query) (int, error) {
			return searcher.Count(ctx, q)
		},
	}
}

func paginate(offset, limit int, results []search.Result, err error) ([]search.Result, error) {
	if err != nil {
		return results, err
	}
	if len(results) == 0 {
		return nil, nil
	}

	if offset < 0 {
		offset = 0
	}
	if limit < 0 {
		limit = 0
	}

	remnants := len(results) - offset
	if remnants <= 0 {
		return nil, nil
	}

	var end int
	if limit == 0 || remnants < limit {
		end = offset + remnants
	} else {
		end = offset + limit
	}

	return results[offset:end], nil
}

// FillPagination fills in the pagination information for a query.
func FillPagination(query *v1.Query, pagination *v1.Pagination, maxLimit int32) {
	queryPagination := &v1.QueryPagination{}

	// Fill in limit, and check boundaries.
	if pagination.GetLimit() == 0 || pagination.GetLimit() > maxLimit {
		queryPagination.Limit = maxLimit
	} else {
		queryPagination.Limit = pagination.GetLimit()
	}
	// Fill in offset.
	queryPagination.Offset = pagination.GetOffset()

	// Fill in sort options.
	for _, so := range pagination.GetSortOptions() {
		queryPagination.SortOptions = append(queryPagination.SortOptions, toQuerySortOption(so))
	}

	// Prefer the new field over the old one.
	if len(pagination.GetSortOptions()) > 0 {
		query.Pagination = queryPagination
		return
	}

	if pagination.GetSortOption() != nil {
		queryPagination.SortOptions = append(queryPagination.SortOptions, toQuerySortOption(pagination.GetSortOption()))
	}
	query.Pagination = queryPagination
}

// FillDefaultSortOption returns a copy of the query with the default sort option added if none is present.
func FillDefaultSortOption(q *v1.Query, defaultSortOption *v1.QuerySortOption) *v1.Query {
	if q == nil {
		q = search.EmptyQuery()
	}
	// Add pagination sort order if needed.
	local := q.Clone()
	if local.GetPagination() == nil {
		local.Pagination = new(v1.QueryPagination)
	}
	if len(local.GetPagination().GetSortOptions()) == 0 {
		local.Pagination.SortOptions = append(local.Pagination.SortOptions, defaultSortOption)
	}
	return local
}

func toQuerySortOption(sortOption *v1.SortOption) *v1.QuerySortOption {
	ret := &v1.QuerySortOption{
		Field:    sortOption.GetField(),
		Reversed: sortOption.GetReversed(),
	}
	if sortOption.GetAggregateBy() != nil {
		ret.AggregateBy = sortOption.GetAggregateBy()
	}
	return ret
}

// FillPaginationV2 fills in the pagination information for a query.
func FillPaginationV2(query *v1.Query, pagination *v2.Pagination, maxLimit int32) {
	queryPagination := &v1.QueryPagination{}

	// Fill in limit, and check boundaries.
	if pagination.GetLimit() == 0 || pagination.GetLimit() > maxLimit {
		queryPagination.Limit = maxLimit
	} else {
		queryPagination.Limit = pagination.GetLimit()
	}
	// Fill in offset.
	queryPagination.Offset = pagination.GetOffset()

	// Fill in sort options.
	for _, so := range pagination.GetSortOptions() {
		queryPagination.SortOptions = append(queryPagination.SortOptions, toQuerySortOptionV2(so))
	}

	// Prefer the new field over the old one.
	if len(pagination.GetSortOptions()) > 0 {
		query.Pagination = queryPagination
		return
	}

	if pagination.GetSortOption() != nil {
		queryPagination.SortOptions = append(queryPagination.SortOptions, toQuerySortOptionV2(pagination.GetSortOption()))
	}
	query.Pagination = queryPagination
}

func toQuerySortOptionV2(sortOption *v2.SortOption) *v1.QuerySortOption {
	ret := &v1.QuerySortOption{
		Field:    sortOption.GetField(),
		Reversed: sortOption.GetReversed(),
	}
	if sortOption.GetAggregateBy() != nil {
		ret.AggregateBy = convertV2AggregateByToV1(sortOption.GetAggregateBy())
	}
	return ret
}

func convertV2AggregateByToV1(aggregateBy *v2.AggregateBy) *v1.AggregateBy {
	return &v1.AggregateBy{
		AggrFunc: v1.Aggregation(aggregateBy.GetAggrFunc()),
		Distinct: aggregateBy.GetDistinct(),
	}
}
