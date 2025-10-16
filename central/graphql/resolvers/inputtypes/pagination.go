package inputtypes

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
)

// Pagination struct contains limit, offset and sort options
type Pagination struct {
	Offset      *int32
	Limit       *int32
	SortOptions *[]*SortOption

	// Retained for backward compatibility.
	SortOption *SortOption
}

// AsV1Pagination returns a proto Pagination struct
func (r *Pagination) AsV1Pagination() *v1.Pagination {
	if r == nil {
		return nil
	}
	pagination := &v1.Pagination{}
	pagination.SetOffset(func() int32 {
		if r.Offset == nil || *r.Offset < 0 {
			return 0
		}
		return *r.Offset
	}())
	pagination.SetLimit(func() int32 {
		if r.Limit == nil || *r.Limit < 0 {
			return 0
		}
		return *r.Limit
	}())
	pagination.SetSortOption(r.SortOption.AsV1SortOption())
	pagination.SetSortOptions(func() []*v1.SortOption {
		if r.SortOptions == nil {
			return nil
		}
		ret := make([]*v1.SortOption, 0, len(*r.SortOptions))
		for _, so := range *r.SortOptions {
			if so == nil {
				continue
			}
			ret = append(ret, so.AsV1SortOption())
		}
		return ret
	}())
	return pagination
}
