package main

// rewriteStrings recursively traverses data structures and replaces all
// string values matching 'old' with 'new'
// Returns true if any replacements were made
func rewriteStrings(data interface{}, old, new string) bool {
	modified := false

	switch v := data.(type) {
	case string:
		// Can't modify strings in place, caller must handle
		return false

	case map[string]interface{}:
		for key, value := range v {
			if str, ok := value.(string); ok && str == old {
				v[key] = new
				modified = true
			} else if rewriteStrings(value, old, new) {
				modified = true
			}
		}

	case []interface{}:
		for i, value := range v {
			if str, ok := value.(string); ok && str == old {
				v[i] = new
				modified = true
			} else if rewriteStrings(value, old, new) {
				modified = true
			}
		}
	}

	return modified
}
