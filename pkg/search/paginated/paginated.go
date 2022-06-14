package paginated

import (
	"context"

	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/search"
)

// WithDefaultSortOption is a higher order function that makes sure results are sorted.
func WithDefaultSortOption(searcher search.Searcher, defaultSortOption *v1.QuerySortOption) search.Searcher {
	return search.FuncSearcher{
		SearchFunc: func(ctx context.Context, q *v1.Query) ([]search.Result, error) {
			// Add pagination sort order if needed.
			local := FillDefaultSortOption(q, defaultSortOption)
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
	// Fill in sort options.
	if pagination.GetSortOption() != nil {
		queryPagination.SortOptions = []*v1.QuerySortOption{
			{
				Field:    pagination.GetSortOption().GetField(),
				Reversed: pagination.GetSortOption().GetReversed(),
			},
		}
	}
	// Fill in offset.
	queryPagination.Offset = pagination.GetOffset()

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
