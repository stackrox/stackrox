package common

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/search"
)

// WithRowsFromImageV2Only decorates the query to exclude V1 image CVE and component rows when FlattenImageData is enabled.
func WithRowsFromImageV2Only(q *v1.Query) *v1.Query {
	if !features.FlattenImageData.Enabled() {
		return q
	}
	// V1 rows have NULL imagev2id, so this query will exclude them.
	v2Filter := search.NewQueryBuilder().
		AddStrings(search.ImageID, search.WildcardString).
		ProtoQuery()
	pagination := q.GetPagination()
	selects := q.GetSelects()
	groupBy := q.GetGroupBy()
	query := search.ConjunctionQuery(v2Filter, q)
	query.Pagination = pagination
	query.Selects = selects
	query.GroupBy = groupBy
	return query
}
