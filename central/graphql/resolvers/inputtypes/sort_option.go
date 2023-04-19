package inputtypes

import (
	"strings"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
)

// SortOption is the sort option input type.
type SortOption struct {
	Field       *string
	Aggregation *string
	Reversed    *bool
}

// AsV1SortOption converts the sort option to proto.
func (s *SortOption) AsV1SortOption() *v1.SortOption {
	if s == nil {
		return nil
	}
	return &v1.SortOption{
		Field: func() string {
			if s.Field == nil {
				return ""
			}
			return *s.Field
		}(),
		Aggregation: func() v1.Aggregation {
			if s.Aggregation == nil {
				return v1.Aggregation_UNSET
			}
			aggrFunc := aggregatefunc.GetAggrFunc(strings.ToLower(*s.Aggregation))
			return aggrFunc.Proto()
		}(),
		Reversed: func() bool {
			if s.Reversed == nil {
				return false
			}
			return *s.Reversed
		}(),
	}
}
