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
		"metadata": map[string]any{
			"labels": map[string]any{
				"app": "test",
			},
		},
	}

	result, err := GetMap(vals, "metadata.labels")
	require.NoError(t, err)
	assert.Equal(t, "test", result["app"])
}

func TestGetMap_WrongType(t *testing.T) {
	vals := chartutil.Values{
		"metadata": map[string]any{
			"name": "test",
		},
	}

	_, err := GetMap(vals, "metadata.name")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a table")
}

func TestGetMap_MissingPath(t *testing.T) {
	vals := chartutil.Values{}

	_, err := GetMap(vals, "metadata.labels")
	assert.Error(t, err)
}
