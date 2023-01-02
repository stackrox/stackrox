package phonehome

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/grpc/authn"
)

var errNoBody = errors.New("empty body")
var errBadType = errors.New("unexpected body type")

// RequestParams holds intercepted call parameters.
type RequestParams struct {
	UserAgent string
	UserID    authn.Identity
	Method    string
	Path      string
	Code      int
	GRPCReq   any
	HTTPReq   *http.Request
}

// ServiceMethod describes a service method with its gRPC and HTTP variants.
type ServiceMethod struct {
	GRPCMethod string
	HTTPMethod string
	HTTPPath   string
}

// PathMatches returns true if Path equals to pattern or matches '*'-terminating
// wildcard. E.g. path '/v1/object/id' will match pattern '/v1/object/*'.
func (rp *RequestParams) PathMatches(pattern string) bool {
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(rp.Path, pattern[0:len(pattern)-1])
	}
	return rp.Path == pattern
}

// HasPathIn returns true if Path matches an element in patterns.
func (rp *RequestParams) HasPathIn(patterns []string) bool {
	for _, p := range patterns {
		if rp.PathMatches(p) {
			return true
		}
	}
	return false
}

// Is checks wether the request targets the service method: either gRPC or HTTP.
func (rp *RequestParams) Is(s *ServiceMethod) bool {
	return rp.Method == s.GRPCMethod || (rp.Method == s.HTTPMethod && rp.PathMatches(s.HTTPPath))
}

// GetRequestBody returns the request body.
func GetRequestBody[T any](rp *RequestParams) (*T, error) {
	if rp.GRPCReq != nil {
		if b, ok := rp.GRPCReq.(*T); ok {
			return b, nil
		} else {
			return nil, errBadType
		}
	}
	if rp.HTTPReq == nil {
		return nil, nil
	}
	if rp.HTTPReq.GetBody == nil {
		return nil, errNoBody
	}

	br, err := rp.HTTPReq.GetBody()
	if err != nil {
		return nil, err
	}

	var bb []byte
	if bb, err = ioutil.ReadAll(br); err != nil {
		return nil, err
	}
	var body *T
	if err = json.Unmarshal(bb, &body); err != nil {
		return nil, errors.Wrap(errBadType, err.Error())
	}
	return body, nil
}
