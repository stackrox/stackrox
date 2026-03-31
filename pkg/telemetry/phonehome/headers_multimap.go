package phonehome

import (
	"net/http"

	"github.com/stackrox/rox/pkg/glob"
	"google.golang.org/grpc/metadata"
)

// Headers wraps http.Header with a metadata.MD-like interface.
type Headers http.Header

// NewHeaders creates Headers from gRPC metadata, canonicalizing the lowercase
// keys used by metadata.MD into the format expected by http.Header.
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

// GetAll returns filtered map of the headers and their values.
func (h Headers) GetAll(canonicalKey glob.Pattern, value glob.Pattern) (map[string][]string, error) {
	if err := canonicalKey.Compile(); err != nil {
		return nil, err
	}
	if err := value.Compile(); err != nil {
		return nil, err
	}
	result := make(map[string][]string)
	for k, values := range h {
		if canonicalKey.Match(k) {
			for _, v := range values {
				if value == NoHeaderOrAnyValue || value.Match(v) {
					result[k] = append(result[k], v)
				}
			}
		}
	}
	return result, nil
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
