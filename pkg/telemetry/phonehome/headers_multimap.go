package phonehome

import (
	"net/http"
)

// Headers wraps http.Header with a metadata.MD-like interface.
type Headers http.Header

// Get implements the getter interface.
func (h Headers) Get(key string) []string {
	return http.Header(h).Values(key)
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
