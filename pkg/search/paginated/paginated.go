package paginated

import (
	"context"

	"github.com/gogo/protobuf/proto"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// WithDefaultSortOption is a higher order function that makes sure results are sorted.
func WithDefaultSortOption(searcher search.Searcher, defaultSortOption *v1.SortOption) search.Searcher {
	return search.WrapSearchFunc(func(ctx context.Context, q *v1.Query) ([]search.Result, error) {
		// Add pagination sort order if needed.
		if q.Pagination == nil {
			q.Pagination = new(v1.Pagination)
		}
		if q.Pagination.SortOption == nil {
			q.Pagination.SortOption = defaultSortOption
		}
		return searcher.Search(ctx, q)
	})
}

// Paginated is a higher order function for applying pagination.
func Paginated(searcher search.Searcher) search.Searcher {
	return search.WrapSearchFunc(func(ctx context.Context, q *v1.Query) ([]search.Result, error) {
		// If pagination not set, just skip.
		if q.GetPagination() == nil {
			return searcher.Search(ctx, q)
		}

		// Local copy to avoid changing input.
		local := proto.Clone(q).(*v1.Query)

		// Record used settings.
		offset := int(local.GetPagination().GetOffset())
		local.Pagination.Offset = 0
		limit := int(local.GetPagination().GetLimit())
		local.Pagination.Limit = 0

		// Run an paginate results.
		results, err := searcher.Search(ctx, local)
		return paginate(offset, limit, results, err)
	})
}

func paginate(offset, limit int, results []search.Result, err error) ([]search.Result, error) {
	if err != nil {
		return results, err
	}
	if len(results) == 0 {
		return nil, nil
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
