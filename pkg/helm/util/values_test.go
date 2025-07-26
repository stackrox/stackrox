package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chartutil"
)

func TestValuesForKVPair_Error_EmptyKey(t *testing.T) {

	_, err := ValuesForKVPair("", 37)
	assert.Error(t, err)
}

func TestValuesForKVPair_FlatKey(t *testing.T) {

	vals, err := ValuesForKVPair("foo", 42)
	require.NoError(t, err)

	expected := chartutil.Values{
		"foo": 42,
	}
	assert.Equal(t, expected, vals)
}

func TestValuesForKVPair_NestedKey(t *testing.T) {

	vals, err := ValuesForKVPair("foo.bar.baz", 1337)
	require.NoError(t, err)

	expected := chartutil.Values{
		"foo": map[string]interface{}{
			"bar": map[string]interface{}{
				"baz": 1337,
			},
		},
	}
	assert.Equal(t, expected, vals)
}
