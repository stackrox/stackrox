package filter

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stretchr/testify/assert"
)

func TestCompileEvaluationFilter_AllFiltersAreNonDefault(t *testing.T) {
	t.Setenv(features.EvaluationFilter.EnvVar(), "true")

	for _, proto := range []*storage.EvaluationFilter{
		nil,
		{},
	} {
		filters := CompileEvaluationFilter(proto)
		for i, f := range filters {
			assert.Truef(t, f.IsNonDefault(), "filter at index %d should be non-default", i)
		}
	}
}

func TestCompileEvaluationFilter_DisabledFeatureReturnsNil(t *testing.T) {
	t.Setenv(features.EvaluationFilter.EnvVar(), "false")
	assert.Nil(t, CompileEvaluationFilter(&storage.EvaluationFilter{}))
}
