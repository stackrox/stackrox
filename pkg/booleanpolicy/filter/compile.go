package filter

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
)

// CompileEvaluationFilter converts a proto EvaluationFilter into a slice of
// EvaluationFilters. Returns nil if all features are disabled or the proto is
// nil. Every filter in the returned slice is guaranteed to be non-default
// (i.e. IsNonDefault() == true), so callers can treat a non-empty slice as
// an indication that filtering is active.
func CompileEvaluationFilter(proto *storage.EvaluationFilter) []EvaluationFilter {
	var filters []EvaluationFilter
	if features.EvaluationFilter.Enabled() {
		if f := newContainerTypeFilter(proto.GetSkipContainerTypes()); f != nil {
			filters = append(filters, *f)
		}
	}
	if features.ImageLayerFilter.Enabled() {
		if f := newImageLayerFilter(proto.GetSkipImageLayers()); f != nil {
			filters = append(filters, *f)
		}
	}
	return filters
}
