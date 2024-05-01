package requestinfo

import (
	"net/http"
)

// HeadersMultiMap is an interface for http.Header and for metadata.MD types.
// The former will require a wrapper to implement this interface.
type HeadersMultiMap interface {
	Get(key string) []string
}

// AsHeadersMultiMap adds the HeadersMultiMap implementation to http.Header.
type AsHeadersMultiMap http.Header

// Get implements HeadersMultiMap interface.
func (h AsHeadersMultiMap) Get(key string) []string {
	return http.Header(h).Values(key)
}
