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
