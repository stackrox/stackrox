package filter

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
)

// CompileEvaluationFilter converts a proto EvaluationFilter into a slice of
// EvaluationFilters. Returns nil if the feature is disabled, the proto is nil,
// or specifies only default values.
func CompileEvaluationFilter(_ *storage.EvaluationFilter) []EvaluationFilter {
	if !features.EvaluationFilter.Enabled() {
		return nil
	}
	return nil
}
