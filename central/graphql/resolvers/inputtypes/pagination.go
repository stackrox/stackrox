package inputtypes

import (
	v1 "github.com/stackrox/stackrox/generated/api/v1"
)

// Pagination struct contains limit, offset and sort options
type Pagination struct {
	Offset     *int32
	Limit      *int32
	SortOption *SortOption
}

// AsV1Pagination returns a proto Pagination struct
func (r *Pagination) AsV1Pagination() *v1.Pagination {
	if r == nil {
		return nil
	}
	return &v1.Pagination{
		Offset: func() int32 {
			if r.Offset == nil || *r.Offset < 0 {
				return 0
			}
			return *r.Offset
		}(),
		Limit: func() int32 {
			if r.Limit == nil || *r.Limit < 0 {
				return 0
			}
			return *r.Limit
		}(),
		SortOption: r.SortOption.AsV1SortOption(),
	}
}
