// Package values provides helper functions for navigating and extracting
// typed values from Helm chartutil.Values using dot-separated paths.
package values

import (
	"fmt"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/chartutil"
)

// GetString reads a string value at the given dot-separated path.
// Returns error if path doesn't exist or value is not a string.
func GetString(vals chartutil.Values, path string) (string, error) {
	val, err := vals.PathValue(path)
	if err != nil {
		return "", errors.Wrapf(err, "path %q not found", path)
	}

	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("value at %q is not a string (got %T)", path, val)
	}

	return str, nil
}
