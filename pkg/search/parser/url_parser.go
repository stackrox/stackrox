package parser

import (
	"math"
	"net/url"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/grpc-ecosystem/grpc-gateway/v2/utilities"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
)

// ParseURLQuery parses the URL raw query values into a v1.Query object
func ParseURLQuery(values url.Values) (*v1.Query, *v1.RawQuery, error) {
	var rawQuery v1.RawQuery
	if err := runtime.PopulateQueryParameters(&rawQuery, values, &utilities.DoubleArray{}); err != nil {
		return nil, nil, err
	}

	query, err := search.ParseQuery(rawQuery.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, nil, err
	}

	paginated.FillPagination(query, rawQuery.GetPagination(), math.MaxInt32)
	return query, &rawQuery, nil
}
