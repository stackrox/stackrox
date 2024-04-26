package requestinfo

import (
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc/metadata"
)

// HeadersMultiMap is an interface for http.Header and for metadata.MD types.
// The former will require a wrapper to implement this interface.
type HeadersMultiMap interface {
	Get(key string) []string
}

// AsHeadersMultiMap adds the HeaderGetter implementation to http.Header.
type AsHeadersMultiMap http.Header

// Get implements HeaderGetter interface.
func (h AsHeadersMultiMap) Get(key string) []string {
	return http.Header(h).Values(key)
}

// GetFirst returns the first value of the header by key, or empty string.
func GetFirst(header HeadersMultiMap, key string) string {
	if header == nil {
		return ""
	}
	if values := header.Get(key); len(values) > 0 {
		return values[0]
	}
	return ""
}

// withHeaderMatcher wraps a header map and implements HeaderGetter interface
// that queries the map ignoring gRPC prefixes of the header keys, stored in
// the map: given key 'Accept' it will query for 'grpcgateway-Accept' instead.
type withHeaderMatcher metadata.MD

// Get implements the HeaderGetter interface. It uses the key prefix according
// to the header type.
func (md withHeaderMatcher) Get(key string) []string {
	if matchedKey, matched := runtime.DefaultHeaderMatcher(key); matched {
		key = matchedKey
	}
	return metadata.MD(md).Get(key)
}
