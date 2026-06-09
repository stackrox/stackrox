package booleanpolicy

import (
	"reflect"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator/pathutil"
	"github.com/stretchr/testify/assert"
)

func TestCompileEvaluationFilter(t *testing.T) {
	saved := make(map[reflect.Type]registeredPlugin, len(filterPluginRegistry))
	for k, v := range filterPluginRegistry {
		saved[k] = v
	}

	stub := func(f *storage.EvaluationFilter) ValueFilterFactory {
		if f.GetSkipImageLayers() == storage.SkipImageLayers_SKIP_NONE {
			return nil
		}
		return func(data interface{}) pathutil.ValueFilter {
			return func(_ pathutil.AugmentedValue, _ int) bool { return true }
		}
	}
	RegisterFilterPlugin[storage.SkipImageLayers](stub, map[MatcherKind]ContextExtractor{
		DeploymentKind: func(m interface{}) interface{} { return m },
	})
	t.Cleanup(func() { filterPluginRegistry = saved })

	t.Run("nil filter", func(t *testing.T) {
		assert.Empty(t, CompileEvaluationFilter(nil).ForKind(DeploymentKind))
	})
	t.Run("SKIP_NONE produces no factory", func(t *testing.T) {
		f := &storage.EvaluationFilter{SkipImageLayers: storage.SkipImageLayers_SKIP_NONE}
		assert.Empty(t, CompileEvaluationFilter(f).ForKind(DeploymentKind))
	})
	t.Run("SKIP_APP produces one factory", func(t *testing.T) {
		f := &storage.EvaluationFilter{SkipImageLayers: storage.SkipImageLayers_SKIP_APP}
		assert.Len(t, CompileEvaluationFilter(f).ForKind(DeploymentKind), 1)
	})
	t.Run("SKIP_BASE produces one factory", func(t *testing.T) {
		f := &storage.EvaluationFilter{SkipImageLayers: storage.SkipImageLayers_SKIP_BASE}
		assert.Len(t, CompileEvaluationFilter(f).ForKind(DeploymentKind), 1)
	})
	t.Run("non-matching kind produces no factory", func(t *testing.T) {
		f := &storage.EvaluationFilter{SkipImageLayers: storage.SkipImageLayers_SKIP_APP}
		assert.Empty(t, CompileEvaluationFilter(f).ForKind(ImageKind))
	})
}

func TestCombineValueFilters(t *testing.T) {
	pass := func(_ pathutil.AugmentedValue, _ int) bool { return true }
	fail := func(_ pathutil.AugmentedValue, _ int) bool { return false }

	alwaysPass := func(_ interface{}) pathutil.ValueFilter { return pass }
	alwaysFail := func(_ interface{}) pathutil.ValueFilter { return fail }
	alwaysNil  := func(_ interface{}) pathutil.ValueFilter { return nil }

	t.Run("nil factories returns nil", func(t *testing.T) {
		assert.Nil(t, combineValueFilters(nil, nil))
	})
	t.Run("empty factories returns nil", func(t *testing.T) {
		assert.Nil(t, combineValueFilters([]ValueFilterFactory{}, nil))
	})
	t.Run("single factory returning nil gives nil filter", func(t *testing.T) {
		assert.Nil(t, combineValueFilters([]ValueFilterFactory{alwaysNil}, nil))
	})
	t.Run("single factory returning pass gives pass filter", func(t *testing.T) {
		f := combineValueFilters([]ValueFilterFactory{alwaysPass}, nil)
		assert.NotNil(t, f)
		assert.True(t, f(nil, 0))
	})
	t.Run("two pass factories AND to pass", func(t *testing.T) {
		f := combineValueFilters([]ValueFilterFactory{alwaysPass, alwaysPass}, nil)
		assert.NotNil(t, f)
		assert.True(t, f(nil, 0))
	})
	t.Run("pass and fail factories AND to fail", func(t *testing.T) {
		f := combineValueFilters([]ValueFilterFactory{alwaysPass, alwaysFail}, nil)
		assert.NotNil(t, f)
		assert.False(t, f(nil, 0))
	})
	t.Run("nil factory among actives is skipped", func(t *testing.T) {
		f := combineValueFilters([]ValueFilterFactory{alwaysNil, alwaysPass}, nil)
		assert.NotNil(t, f)
		assert.True(t, f(nil, 0))
	})
}
