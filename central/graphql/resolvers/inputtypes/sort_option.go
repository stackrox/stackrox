package inputtypes

import (
	"strings"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
)

// SortOption is the sort option input type.
type SortOption struct {
	Field       *string
	AggregateBy *AggregateBy
	Reversed    *bool
}

// AggregateBy is the input type to specifies the aggregation to be applied to sort field. e.g. count, min, max.
type AggregateBy struct {
	AggregateFunc *string
	Distinct      *bool
}

// AsV1SortOption converts the sort option to proto.
func (s *SortOption) AsV1SortOption() *v1.SortOption {
	if s == nil {
		return nil
	}
	so := &v1.SortOption{}
	so.SetField(func() string {
		if s.Field == nil {
			return ""
		}
		return *s.Field
	}())
	so.SetAggregateBy(s.AggregateBy.AsV1AggregateBy())
	so.SetReversed(func() bool {
		if s.Reversed == nil {
			return false
		}
		return *s.Reversed
	}())
	return so
}

// AsV1AggregateBy converts the aggregation to proto.
func (a *AggregateBy) AsV1AggregateBy() *v1.AggregateBy {
	if a == nil {
		return nil
	}
	aggrFunc := aggregatefunc.GetAggrFunc(strings.ToLower(*a.AggregateFunc))
	if aggrFunc == aggregatefunc.Unset {
		return nil
	}
	ab := &v1.AggregateBy{}
	ab.SetAggrFunc(aggrFunc.Proto())
	ab.SetDistinct(func() bool {
		if a.Distinct == nil {
			return false
		}
		return *a.Distinct
	}())
	return ab
}
