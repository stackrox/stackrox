package testing

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chartutil"
)

// AssertEqualPathValue helps asserting path values which requires a path to exist, otherwise it fails.
func AssertEqualPathValue(t *testing.T, values chartutil.Values, expected interface{}, path string, msgAndArgs ...interface{}) {
	v := readPath(t, values, path)
	assert.Equal(t, expected, v, msgAndArgs)
}

// AssertPathValueMatches helps asserting path values which requires a path to exist, otherwise it fails.
func AssertPathValueMatches(t *testing.T, values chartutil.Values, regex *regexp.Regexp, path string, msgAndArgs ...interface{}) {
	v := readPath(t, values, path)
	assert.Regexp(t, regex, v, msgAndArgs)
}

// AssertNotNilPathValue helps asserting path values which requires a path to exist, otherwise it fails.
func AssertNotNilPathValue(t *testing.T, values chartutil.Values, path string, msgAndArgs ...interface{}) {
	v := readPath(t, values, path)
	assert.NotNil(t, v, msgAndArgs)
}

func readPath(t *testing.T, values chartutil.Values, path string) interface{} {
	v, err := values.PathValue(path)
	require.NoError(t, err)
	return v
}

// InfraScheduling is the expected scheduling for infrastructure nodes used in tests.
var InfraScheduling = SchedulingExpectation{
	NodeSelector: map[string]any{"node-role.kubernetes.io/infra": ""},
	Tolerations: []any{
		map[string]any{"effect": "NoSchedule", "key": "node-role.kubernetes.io/infra", "operator": "Equal", "value": "reserved"},
		map[string]any{"effect": "NoExecute", "key": "node-role.kubernetes.io/infra", "operator": "Equal", "value": "reserved"},
	},
}

// GlobalScheduling is a generic global scheduling expectation used in tests.
var GlobalScheduling = SchedulingExpectation{
	NodeSelector: map[string]any{"global-label": "global-value"},
	Tolerations:  []any{map[string]any{"key": "global-taint", "operator": "Exists"}},
}

// ComponentPath defines the Helm value paths for a component's scheduling fields.
type ComponentPath struct {
	Name             string
	NodeSelectorPath string
	TolerationsPath  string
}

// SchedulingExpectation defines expected scheduling values for a component.
type SchedulingExpectation struct {
	NodeSelector map[string]any
	Tolerations  []any
}

// SchedulingExpectations maps component names to their expected scheduling values.
type SchedulingExpectations map[string]SchedulingExpectation

// NewSchedulingExpectations creates expectations where all components have the same scheduling values.
func NewSchedulingExpectations(paths []ComponentPath, expectation SchedulingExpectation) SchedulingExpectations {
	expectations := make(SchedulingExpectations, len(paths))
	for _, path := range paths {
		expectations[path.Name] = expectation
	}
	return expectations
}

// WithOverride returns a copy of the expectations with the specified component's values overridden.
func (e SchedulingExpectations) WithOverride(name string, expectation SchedulingExpectation) SchedulingExpectations {
	result := make(SchedulingExpectations, len(e))
	for k, v := range e {
		result[k] = v
	}
	result[name] = expectation
	return result
}
