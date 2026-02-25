// Package values provides helper functions for navigating and extracting
// typed values from Helm chartutil.Values using dot-separated paths.
package values

import (
	"fmt"

	"github.com/pkg/errors"
	helmUtil "github.com/stackrox/rox/pkg/helm/util"
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

// GetMap reads a nested map at the given dot-separated path.
// Returns error if path doesn't exist or value is not a map.
func GetMap(vals chartutil.Values, path string) (chartutil.Values, error) {
	val, err := vals.PathValue(path)
	if err != nil {
		return nil, errors.Wrapf(err, "path %q not found", path)
	}

	// PathValue can return either chartutil.Values or map[string]any
	switch m := val.(type) {
	case chartutil.Values:
		return m, nil
	case map[string]any:
		return chartutil.Values(m), nil
	default:
		return nil, fmt.Errorf("value at %q is not a map (got %T)", path, val)
	}
}

// GetArray reads an array at the given dot-separated path.
// Returns error if path doesn't exist or value is not an array.
func GetArray(vals chartutil.Values, path string) ([]any, error) {
	val, err := vals.PathValue(path)
	if err != nil {
		return nil, errors.Wrapf(err, "path %q not found", path)
	}

	arr, ok := val.([]any)
	if !ok {
		return nil, fmt.Errorf("value at %q is not an array (got %T)", path, val)
	}

	return arr, nil
}

// GetValue reads any value at the given dot-separated path.
// Useful when the type is dynamic or caller will type-assert.
func GetValue(vals chartutil.Values, path string) (any, error) {
	val, err := vals.PathValue(path)
	if err != nil {
		return nil, errors.Wrapf(err, "path %q not found", path)
	}
	return val, nil
}

// PathExists reports whether a value exists at the given dot-separated path.
func PathExists(vals chartutil.Values, path string) bool {
	_, err := vals.PathValue(path)
	return err == nil
}

// SetValue sets a value at the given dot-separated path in vals.
// Creates intermediate maps as needed.
func SetValue(vals chartutil.Values, path string, value any) error {
	update, err := helmUtil.ValuesForKVPair(path, value)
	if err != nil {
		return fmt.Errorf("failed to build update for path %q: %w", path, err)
	}

	// CoalesceTables(dst, src) fills dst with missing values from src,
	// giving priority to dst. By passing update as dst, the new value
	// takes precedence while existing sibling keys from vals are preserved.
	merged := chartutil.CoalesceTables(update, vals)

	// CoalesceTables returns its dst; copy results back into the original map.
	for k, v := range merged {
		vals[k] = v
	}

	return nil
}
