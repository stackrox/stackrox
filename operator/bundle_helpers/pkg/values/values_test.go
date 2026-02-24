package values

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chartutil"
)

func TestGetString_Success(t *testing.T) {
	vals := chartutil.Values{
		"metadata": map[string]any{
			"name": "test-operator",
		},
	}

	result, err := GetString(vals, "metadata.name")
	require.NoError(t, err)
	assert.Equal(t, "test-operator", result)
}

func TestGetString_NestedPath(t *testing.T) {
	vals := chartutil.Values{
		"metadata": map[string]any{
			"annotations": map[string]any{
				"createdAt": "2024-01-01T00:00:00Z",
			},
		},
	}

	result, err := GetString(vals, "metadata.annotations.createdAt")
	require.NoError(t, err)
	assert.Equal(t, "2024-01-01T00:00:00Z", result)
}

func TestGetString_MissingPath(t *testing.T) {
	vals := chartutil.Values{
		"metadata": map[string]any{},
	}

	_, err := GetString(vals, "metadata.name")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "metadata.name")
}

func TestGetString_WrongType(t *testing.T) {
	vals := chartutil.Values{
		"metadata": map[string]any{
			"name": 123,
		},
	}

	_, err := GetString(vals, "metadata.name")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a string")
}

func TestGetMap_Success(t *testing.T) {
	vals := chartutil.Values{
		"metadata": chartutil.Values{
			"labels": chartutil.Values{
				"app": "test",
			},
		},
	}

	result, err := GetMap(vals, "metadata.labels")
	require.NoError(t, err)
	assert.Equal(t, chartutil.Values{"app": "test"}, result)
}

func TestGetMap_WrongType(t *testing.T) {
	vals := chartutil.Values{
		"metadata": chartutil.Values{
			"name": "test",
		},
	}

	_, err := GetMap(vals, "metadata.name")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a map")
}

func TestGetMap_MissingPath(t *testing.T) {
	vals := chartutil.Values{}

	_, err := GetMap(vals, "metadata.labels")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "metadata.labels")
}

func TestGetArray_Success(t *testing.T) {
	vals := chartutil.Values{
		"spec": map[string]any{
			"items": []any{"a", "b", "c"},
		},
	}

	result, err := GetArray(vals, "spec.items")
	require.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, "a", result[0])
}

func TestGetArray_WrongType(t *testing.T) {
	vals := chartutil.Values{
		"spec": map[string]any{
			"version": "1.0",
		},
	}

	_, err := GetArray(vals, "spec.version")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not an array")
}

func TestGetArray_MissingPath(t *testing.T) {
	vals := chartutil.Values{}

	_, err := GetArray(vals, "spec.items")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "spec.items")
}

func TestGetValue_String(t *testing.T) {
	vals := chartutil.Values{
		"metadata": map[string]any{
			"name": "test",
		},
	}

	result, err := GetValue(vals, "metadata.name")
	require.NoError(t, err)
	assert.Equal(t, "test", result)
}

func TestGetValue_MissingPath(t *testing.T) {
	vals := chartutil.Values{}

	_, err := GetValue(vals, "metadata.name")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "metadata.name")
}

func TestPathExists_True(t *testing.T) {
	vals := chartutil.Values{
		"metadata": chartutil.Values{
			"name": "test",
		},
	}

	assert.True(t, PathExists(vals, "metadata.name"))
	assert.True(t, PathExists(vals, "metadata"))
}

func TestPathExists_False(t *testing.T) {
	vals := chartutil.Values{
		"metadata": map[string]any{},
	}

	assert.False(t, PathExists(vals, "metadata.name"))
	assert.False(t, PathExists(vals, "spec"))
}
