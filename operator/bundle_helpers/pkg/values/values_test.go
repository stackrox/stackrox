package values

import (
	"testing"

	"github.com/stackrox/rox/operator/bundle_helpers/pkg/rewrite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chartutil"
)

func TestGetString_Success(t *testing.T) {
	vals := chartutil.Values{
		"name": "test-value",
	}

	result, err := GetString(vals, "name")
	require.NoError(t, err)
	assert.Equal(t, "test-value", result)
}

func TestGetString_NestedPath(t *testing.T) {
	vals := chartutil.Values{
		"metadata": chartutil.Values{
			"annotations": chartutil.Values{
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
	assert.Contains(t, err.Error(), "not a table")
}

func TestGetMap_MissingPath(t *testing.T) {
	vals := chartutil.Values{
		"metadata": chartutil.Values{},
	}

	_, err := GetMap(vals, "metadata.spec")
	assert.Error(t, err)
}

func TestGetArray_Success(t *testing.T) {
	vals := chartutil.Values{
		"items": []any{"a", "b", "c"},
	}

	result, err := GetArray(vals, "items")
	require.NoError(t, err)
	assert.Equal(t, []any{"a", "b", "c"}, result)
}

func TestGetArray_WrongType(t *testing.T) {
	vals := chartutil.Values{
		"items": "not-an-array",
	}

	_, err := GetArray(vals, "items")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not an array")
}

func TestGetArray_MissingPath(t *testing.T) {
	vals := chartutil.Values{}

	_, err := GetArray(vals, "items")
	assert.Error(t, err)
}

func TestGetValue_String(t *testing.T) {
	vals := chartutil.Values{
		"key": "value",
	}

	result, err := GetValue(vals, "key")
	require.NoError(t, err)
	assert.Equal(t, "value", result)
}

func TestGetValue_MissingPath(t *testing.T) {
	vals := chartutil.Values{}

	_, err := GetValue(vals, "missing")
	assert.Error(t, err)
}

func TestPathExists_True(t *testing.T) {
	vals := chartutil.Values{
		"metadata": map[string]any{
			"name": "test",
		},
	}

	assert.True(t, PathExists(vals, "metadata.name"))
}

func TestPathExists_False(t *testing.T) {
	vals := chartutil.Values{
		"metadata": map[string]any{},
	}

	assert.False(t, PathExists(vals, "metadata.name"))
}

func TestSetValue_NewPath(t *testing.T) {
	vals := chartutil.Values{}

	err := SetValue(vals, "metadata.name", "test")
	require.NoError(t, err)

	result, err := GetString(vals, "metadata.name")
	require.NoError(t, err)
	assert.Equal(t, "test", result)
}

func TestSetValue_OverwriteExisting(t *testing.T) {
	vals := chartutil.Values{
		"metadata": map[string]any{
			"name": "old",
		},
	}

	err := SetValue(vals, "metadata.name", "new")
	require.NoError(t, err)

	result, err := GetString(vals, "metadata.name")
	require.NoError(t, err)
	assert.Equal(t, "new", result)
}

func TestSetValue_CreateIntermediateMaps(t *testing.T) {
	vals := chartutil.Values{}

	err := SetValue(vals, "a.b.c.d", "value")
	require.NoError(t, err)

	result, err := GetString(vals, "a.b.c.d")
	require.NoError(t, err)
	assert.Equal(t, "value", result)
}

func TestSetValue_PreservesSiblings(t *testing.T) {
	vals := chartutil.Values{
		"metadata": map[string]any{
			"name": "test",
			"annotations": map[string]any{
				"existing": "value",
			},
		},
	}

	err := SetValue(vals, "metadata.annotations.new", "added")
	require.NoError(t, err)

	// Verify existing value is preserved
	existing, err := GetString(vals, "metadata.annotations.existing")
	require.NoError(t, err)
	assert.Equal(t, "value", existing)

	// Verify new value was added
	added, err := GetString(vals, "metadata.annotations.new")
	require.NoError(t, err)
	assert.Equal(t, "added", added)

	// Verify sibling at same level is preserved
	name, err := GetString(vals, "metadata.name")
	require.NoError(t, err)
	assert.Equal(t, "test", name)
}

func TestSetValue_DeepNesting(t *testing.T) {
	vals := chartutil.Values{
		"a": map[string]any{
			"b": map[string]any{
				"c": "original",
			},
		},
	}

	err := SetValue(vals, "a.b.d", "new")
	require.NoError(t, err)

	// Verify original value preserved
	original, err := GetString(vals, "a.b.c")
	require.NoError(t, err)
	assert.Equal(t, "original", original)

	// Verify new value added
	new, err := GetString(vals, "a.b.d")
	require.NoError(t, err)
	assert.Equal(t, "new", new)
}

func TestSetValue_ThenRewriteStrings(t *testing.T) {
	doc := chartutil.Values{
		"metadata": map[string]any{
			"name": "test",
			"annotations": map[string]any{
				"containerImage": "old-image:1.0",
			},
		},
	}

	// Step 1: SetValue to add a new field
	err := SetValue(doc, "metadata.annotations.createdAt", "2024-01-01")
	require.NoError(t, err)

	// Debug: check type of metadata after SetValue
	t.Logf("Type of doc['metadata']: %T", doc["metadata"])
	t.Logf("Type of annotations: %T", doc["metadata"].(map[string]any)["annotations"])

	// Step 2: Verify containerImage is still present
	placeholderImage, err := GetString(doc, "metadata.annotations.containerImage")
	require.NoError(t, err)
	assert.Equal(t, "old-image:1.0", placeholderImage)

	// Step 3: Use rewrite.Strings to replace the image
	modified := rewrite.Strings(doc, placeholderImage, "new-image:2.0")
	assert.True(t, modified, "rewrite.Strings should have modified the document")

	// Step 4: Verify the replacement worked
	newImage, err := GetString(doc, "metadata.annotations.containerImage")
	require.NoError(t, err)
	assert.Equal(t, "new-image:2.0", newImage)
}
