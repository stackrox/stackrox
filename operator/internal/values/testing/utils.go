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
