package filter

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
)

// CompileEvaluationFilter converts a proto EvaluationFilter into a slice of
// EvaluationFilters. Returns nil if the feature is disabled or the proto is
// nil. Every filter in the returned slice is guaranteed to be non-default
// (i.e. IsNonDefault() == true), so callers can treat a non-empty slice as
// an indication that filtering is active.
func CompileEvaluationFilter(proto *storage.EvaluationFilter) []EvaluationFilter {
	if !features.EvaluationFilter.Enabled() {
		return nil
	}
	var filters []EvaluationFilter
	if f := newContainerTypeFilter(proto.GetSkipContainerTypes()); f != nil {
		filters = append(filters, *f)
	}
	return filters
}

func newContainerTypeFilter(skipTypes []storage.ContainerType) *EvaluationFilter {
	if len(skipTypes) == 0 {
		return nil
	}
	skip := make(map[storage.ContainerType]struct{}, len(skipTypes))
	for _, t := range skipTypes {
		skip[t] = struct{}{}
	}
	return &EvaluationFilter{
		isNonDefault: func() bool { return true },
		apply: func(dep *storage.Deployment, imgs []*storage.Image) (*storage.Deployment, []*storage.Image) {
			hasSkipped := false
			for _, c := range dep.GetContainers() {
				if _, ok := skip[c.GetType()]; ok {
					hasSkipped = true
					break
				}
			}
			if !hasSkipped {
				return dep, imgs
			}
			filtered := dep.CloneVT()
			filtered.Containers = nil
			var filteredImgs []*storage.Image
			for i, c := range dep.GetContainers() {
				if _, ok := skip[c.GetType()]; ok {
					continue
				}
				filtered.Containers = append(filtered.Containers, c)
				if i < len(imgs) {
					filteredImgs = append(filteredImgs, imgs[i])
				}
			}
			return filtered, filteredImgs
		},
	}
}
