package filter

import "github.com/stackrox/rox/generated/storage"

// NewTestFilter creates an EvaluationFilter from a callback for use in tests.
func NewTestFilter(fn func(*storage.Deployment, []*storage.Image) (*storage.Deployment, []*storage.Image)) EvaluationFilter {
	return EvaluationFilter{
		isNonDefault: func() bool { return true },
		apply:        fn,
	}
}
