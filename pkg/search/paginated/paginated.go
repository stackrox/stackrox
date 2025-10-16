package paginated

import (
	"math"

	v1 "github.com/stackrox/rox/generated/api/v1"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/search"
)

// PageResults takes search results and performs paging in go.  This is needed for searches that require
// sorting by a priority field that is held within a ranker and not in the database.
func PageResults(results []search.Result, q *v1.Query) ([]search.Result, error) {
	// If pagination not set, just skip.
	if q.GetPagination() == nil {
		return results, nil
	}

	// Record used settings.
	offset := int(q.GetPagination().GetOffset())
	limit := int(q.GetPagination().GetLimit())

	return paginate(offset, limit, results, nil)
}

func paginate[T any](offset, limit int, results []T, err error) ([]T, error) {
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
		queryPagination.SetLimit(maxLimit)
	} else {
		queryPagination.SetLimit(pagination.GetLimit())
	}
	// Fill in offset.
	queryPagination.SetOffset(pagination.GetOffset())

	// Fill in sort options.
	for _, so := range pagination.GetSortOptions() {
		queryPagination.SetSortOptions(append(queryPagination.GetSortOptions(), toQuerySortOption(so)))
	}

	// Prefer the new field over the old one.
	if len(pagination.GetSortOptions()) > 0 {
		query.SetPagination(queryPagination)
		return
	}

	if pagination.GetSortOption() != nil {
		queryPagination.SetSortOptions(append(queryPagination.GetSortOptions(), toQuerySortOption(pagination.GetSortOption())))
	}
	query.SetPagination(queryPagination)
}

// FillDefaultSortOption returns a copy of the query with the default sort option added if none is present.
func FillDefaultSortOption(q *v1.Query, defaultSortOption *v1.QuerySortOption) *v1.Query {
	if q == nil {
		q = search.EmptyQuery()
	}
	// Add pagination sort order if needed.
	local := q.CloneVT()
	if local.GetPagination() == nil {
		local.SetPagination(new(v1.QueryPagination))
	}
	if len(local.GetPagination().GetSortOptions()) == 0 {
		local.GetPagination().SetSortOptions(append(local.GetPagination().GetSortOptions(), defaultSortOption))
	}
	return local
}

func toQuerySortOption(sortOption *v1.SortOption) *v1.QuerySortOption {
	ret := &v1.QuerySortOption{}
	ret.SetField(sortOption.GetField())
	ret.SetReversed(sortOption.GetReversed())
	if sortOption.GetAggregateBy() != nil {
		ret.SetAggregateBy(sortOption.GetAggregateBy())
	}
	return ret
}

// FillPaginationV2 fills in the pagination information for a query.
func FillPaginationV2(query *v1.Query, pagination *v2.Pagination, maxLimit int32) {
	queryPagination := &v1.QueryPagination{}

	// Fill in limit, and check boundaries.
	if pagination.GetLimit() == 0 || pagination.GetLimit() > maxLimit {
		queryPagination.SetLimit(maxLimit)
	} else {
		queryPagination.SetLimit(pagination.GetLimit())
	}
	// Fill in offset.
	queryPagination.SetOffset(pagination.GetOffset())

	// Fill in sort options.
	for _, so := range pagination.GetSortOptions() {
		queryPagination.SetSortOptions(append(queryPagination.GetSortOptions(), toQuerySortOptionV2(so)))
	}

	// Prefer the new field over the old one.
	if len(pagination.GetSortOptions()) > 0 {
		query.SetPagination(queryPagination)
		return
	}

	if pagination.GetSortOption() != nil {
		queryPagination.SetSortOptions(append(queryPagination.GetSortOptions(), toQuerySortOptionV2(pagination.GetSortOption())))
	}
	query.SetPagination(queryPagination)
}

func toQuerySortOptionV2(sortOption *v2.SortOption) *v1.QuerySortOption {
	ret := &v1.QuerySortOption{}
	ret.SetField(sortOption.GetField())
	ret.SetReversed(sortOption.GetReversed())
	if sortOption.GetAggregateBy() != nil {
		ret.SetAggregateBy(convertV2AggregateByToV1(sortOption.GetAggregateBy()))
	}
	return ret
}

func convertV2AggregateByToV1(aggregateBy *v2.AggregateBy) *v1.AggregateBy {
	ab := &v1.AggregateBy{}
	ab.SetAggrFunc(v1.Aggregation(aggregateBy.GetAggrFunc()))
	ab.SetDistinct(aggregateBy.GetDistinct())
	return ab
}

func PaginateSlice[T any](offset, limit int, slice []T) []T {
	// if we pass nil, then there can be no error
	result, _ := paginate(offset, limit, slice, nil)
	return result
}

// GetLimit returns pagination limit or a value if it's unlimited
func GetLimit(paginationLimit int32, whenUnlimited int32) int32 {
	if paginationLimit <= 0 || paginationLimit == math.MaxInt32 {
		return whenUnlimited
	}
	return paginationLimit
}
