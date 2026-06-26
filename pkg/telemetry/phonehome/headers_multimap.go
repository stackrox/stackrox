package phonehome

import (
	"net/http"

	"github.com/stackrox/rox/pkg/glob"
	"google.golang.org/grpc/metadata"
)

// Headers wraps http.Header with a metadata.MD-like interface.
type Headers http.Header

// NewHeaders creates a Headers value from gRPC metadata.MD by copying each metadata
// key and its values into an http.Header. Multiple values for the same key are
// preserved and the header map is preallocated to the number of metadata keys.
func NewHeaders(m metadata.MD) Headers {
	h := make(http.Header, len(m))
	for k, vs := range m {
		for _, v := range vs {
			h.Add(k, v)
		}
	}
	return Headers(h)
}

// Get implements the getter interface.
func (h Headers) Get(key string) []string {
	return http.Header(h).Values(key)
}

// getMatchingValues returns all values matching value pattern for the given
// key.
// Returns nil if the key is absent or no values match the pattern. For the
// special case where the key exists with no values and the pattern matches
// empty string, returns a non-nil empty slice.
func (h Headers) getMatchingValues(key string, value glob.Pattern) []string {
	var result []string
	if value == NoHeaderOrAnyValue {
		result = make([]string, 0)
	}
	if h == nil {
		return result
	}
	values, exists := http.Header(h)[http.CanonicalHeaderKey(key)]
	if !exists {
		return result
	}
	for _, v := range values {
		if value == NoHeaderOrAnyValue || value.Match(v) {
			result = append(result, v)
		}
	}
	return result
}

// GetMatching returns a filtered map of headers whose keys match canonicalKey
// and whose values match value. Returns nil if no keys match. When value is
// NoHeaderOrAnyValue, returns a non-nil (possibly empty) map even if no keys
// match. Note: a malformed canonicalKey glob silently matches nothing, which
// is indistinguishable from "no keys matched".
func (h Headers) GetMatching(canonicalKey glob.Pattern, value glob.Pattern) map[string][]string {
	var result map[string][]string
	if value == NoHeaderOrAnyValue {
		result = make(map[string][]string)
	}
	for key := range h {
		if !canonicalKey.Match(key) {
			continue
		}
		matching := h.getMatchingValues(key, value)
		if matching == nil {
			continue
		}
		if result == nil {
			result = make(map[string][]string)
		}
		if existing, ok := result[key]; ok {
			result[key] = append(existing, matching...)
		} else {
			result[key] = matching
		}
	}
	return result
}

// Set overrides the current value(s) or deletes the key if no values provided.
func (h Headers) Set(key string, values ...string) {
	http.Header(h).Del(key)
	for i, value := range values {
		if i == 0 {
			http.Header(h).Set(key, value)
		} else {
			http.Header(h).Add(key, value)
		}
	}
}
