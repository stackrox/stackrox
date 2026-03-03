package allowedpaths

import (
	"strings"

	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	pathsMutex  sync.RWMutex
	exactPaths  set.Set[string]
	prefixPaths set.Set[string]
)

// Set stores the allowed proxy paths received from Central.
// Entries ending with "/" are treated as prefixes; all others require an exact
// match. An empty or nil slice means no path filtering is applied (backward
// compat).
func Set(paths []string) {
	pathsMutex.Lock()
	defer pathsMutex.Unlock()
	if len(paths) == 0 {
		exactPaths = nil
		prefixPaths = nil
		return
	}
	exact := set.NewSet[string]()
	prefixes := set.NewSet[string]()
	for _, p := range paths {
		if strings.HasSuffix(p, "/") {
			prefixes.Add(p)
		} else {
			exact.Add(p)
		}
	}
	exactPaths = exact
	prefixPaths = prefixes
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
	pathsMutex.RLock()
	defer pathsMutex.RUnlock()
	if exactPaths == nil && prefixPaths == nil {
		return true
	}
	if exactPaths.Contains(path) {
		return true
	}
	for prefix := range prefixPaths {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// Reset clears the stored paths. Intended for testing.
func Reset() {
	pathsMutex.Lock()
	defer pathsMutex.Unlock()
	exactPaths = nil
	prefixPaths = nil
}
