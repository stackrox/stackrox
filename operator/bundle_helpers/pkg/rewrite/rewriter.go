package rewrite

// RewriteStrings recursively traverses data structures and replaces all
// string values matching 'old' with 'new'
// Returns true if any replacements were made
func RewriteStrings(data any, old, new string) bool {
	modified := false

	switch v := data.(type) {
	case string:
		// Can't modify strings in place, caller must handle
		return false

	case map[string]any:
		for key, value := range v {
			if str, ok := value.(string); ok && str == old {
				v[key] = new
				modified = true
			} else if RewriteStrings(value, old, new) {
				modified = true
			}
		}

	case []any:
		for i, value := range v {
			if str, ok := value.(string); ok && str == old {
				v[i] = new
				modified = true
			} else if RewriteStrings(value, old, new) {
				modified = true
			}
		}
	}

	return modified
}
