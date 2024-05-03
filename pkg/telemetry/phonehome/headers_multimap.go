package phonehome

import (
	"net/http"
)

// headers wraps http.Header with a metadata.MD-like interface.
type headers http.Header

// Get implements the getter interface.
func (h headers) Get(key string) []string {
	return http.Header(h).Values(key)
}
