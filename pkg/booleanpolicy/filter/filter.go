package filter

import "github.com/stackrox/rox/generated/storage"

// EvaluationFilter pre-filters deployment containers or image components
// before policy evaluation. Each filter targets a single concern and is
// composed via slices.
type EvaluationFilter struct {
	isNonDefault func() bool
	apply        func(deployment *storage.Deployment, images []*storage.Image) (*storage.Deployment, []*storage.Image)
}

// IsNonDefault reports whether this filter has any effect.
func (f EvaluationFilter) IsNonDefault() bool {
	return f.isNonDefault != nil && f.isNonDefault()
}

// Apply runs the filter callback on the given deployment and images.
func (f EvaluationFilter) Apply(dep *storage.Deployment, imgs []*storage.Image) (*storage.Deployment, []*storage.Image) {
	return f.apply(dep, imgs)
}
