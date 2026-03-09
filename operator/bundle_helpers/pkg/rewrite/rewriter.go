package rewrite

import "helm.sh/helm/v3/pkg/chartutil"

// Strings recursively traverses data structures and replaces all
// string values matching 'old' with 'new'.
// Returns true if any replacements were made.
func Strings(data any, old, new string) bool {
	modified := false

	switch v := data.(type) {

	case map[string]any:
		for key, value := range v {
			if str, ok := value.(string); ok && str == old {
				v[key] = new
				modified = true
			} else if Strings(value, old, new) {
				modified = true
			}
		}

	case chartutil.Values:
		// chartutil.Values is a named type over map[string]any; convert
		// and re-dispatch so the map[string]any branch handles it uniformly.
		modified = Strings(map[string]any(v), old, new)

	case []any:
		for i, value := range v {
			if str, ok := value.(string); ok && str == old {
				v[i] = new
				modified = true
			} else if Strings(value, old, new) {
				modified = true
			}
		}
	}

	return modified
}
