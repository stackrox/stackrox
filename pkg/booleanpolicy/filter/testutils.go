package filter

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
)

// NewTestFilter creates an EvaluationFilter from a callback for use in tests.
func NewTestFilter(_ testing.TB, fn func(*storage.Deployment, []*storage.Image) (*storage.Deployment, []*storage.Image)) EvaluationFilter {
	return EvaluationFilter{
		isNonDefault: func() bool { return true },
		apply:        fn,
	}
}
