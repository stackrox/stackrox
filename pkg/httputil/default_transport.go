package httputil

import (
	"net/http"
)

var (
	// should this ever panic, look at the implementation of `http.DefaultTransport` and try to recreate it as
	// faithfully as possible as an `*http.Transport`.
	defaultTransport = http.DefaultTransport.(*http.Transport)
)

// DefaultTransport returns a default HTTP transport as an *http.Transport object. The returned object may be
// modified freely without causing side effects.
func DefaultTransport() *http.Transport {
	return defaultTransport.Clone()
}
