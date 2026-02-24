package values

import (
	"fmt"

	"helm.sh/helm/v3/pkg/chartutil"
)

// GetString reads a string value at the given dot-separated path.
// Returns error if path doesn't exist or value is not a string.
func GetString(vals chartutil.Values, path string) (string, error) {
	val, err := vals.PathValue(path)
	if err != nil {
		return "", fmt.Errorf("path %s not found: %w", path, err)
	}

	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("value at %s is not a string (got %T)", path, val)
	}

	return str, nil
}
