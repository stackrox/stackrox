package phonehome

import (
	"net/http"

	"github.com/stackrox/rox/pkg/glob"
	"google.golang.org/grpc/metadata"
)

// Headers wraps http.Header with a metadata.MD-like interface.
type Headers http.Header

// NewHeaders creates Headers from gRPC metadata, canonicalizing the lowercase
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

// GetMatching returns all values matching value pattern for the given key.
// Returns nil if the key is absent or no values match the pattern.
// For the special case where the key exists with no values and the pattern
// matches empty string, returns a non-nil empty slice.
func (h Headers) GetMatching(key string, value glob.Pattern) []string {
	if h == nil {
		return nil
	}
	values, exists := http.Header(h)[http.CanonicalHeaderKey(key)]
	if !exists {
		return nil
	}
	if len(values) == 0 && value.Match("") {
		return make([]string, 0)
	}
	var result []string
	for _, v := range values {
		if value == NoHeaderOrAnyValue || value.Match(v) {
			result = append(result, v)
		}
	}
	return result
}

// Set implements the setter interface.
func (h Headers) Set(key string, values ...string) {
	for i, value := range values {
		if i == 0 {
			http.Header(h).Set(key, value)
		} else {
			http.Header(h).Add(key, value)
		}
	}
}
