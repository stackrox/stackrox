package allowedpaths

import (
	"strings"

	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	mutex        sync.RWMutex
	allowedPaths set.Set[string]
)

// Set stores the allowed proxy paths received from Central.
// An empty or nil slice means no path filtering is applied (backward compat).
func Set(paths []string) {
	mutex.Lock()
	defer mutex.Unlock()
	if len(paths) == 0 {
		allowedPaths = nil
		return
	}
	allowedPaths = set.NewSet(paths...)
}

// IsAllowed returns true if the given path matches any of the allowed paths.
// Entries ending with "/" are treated as prefixes (any path starting with that
// prefix is allowed). Entries without a trailing "/" require an exact match.
// If no paths have been configured, all paths are allowed for backward
// compatibility.
//
// The caller must pass a pure path (no query string); IsAllowed does not strip
// query parameters.
func IsAllowed(path string) bool {
	mutex.RLock()
	defer mutex.RUnlock()
	if allowedPaths == nil {
		return true
	}
	for allowed := range allowedPaths {
		if strings.HasSuffix(allowed, "/") {
			if strings.HasPrefix(path, allowed) {
				return true
			}
		} else if path == allowed {
			return true
		}
	}
	return false
}

// Reset clears the stored paths. Intended for testing.
func Reset() {
	mutex.Lock()
	defer mutex.Unlock()
	allowedPaths = nil
}
