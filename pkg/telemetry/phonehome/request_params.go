package phonehome

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/grpc/authn"
)

// ErrNoBody tells that the request has got no body.
var ErrNoBody = errors.New("empty body")
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

func getGRPCBody[T any](req any) (*T, error) {
	if req == nil {
		return nil, nil
	}
	body, ok := req.(*T)
	if !ok {
		return nil, errBadType
	}
	return body, nil
}

func getHTTPBody[T any](req *http.Request) (*T, error) {
	if req == nil || req.GetBody == nil {
		return nil, nil
	}

	br, err := req.GetBody()
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

// GetRequestBody sets the output body argument. Returns ErrNoBody error on nil
// result.
func GetRequestBody[T any](rp *RequestParams, body **T) error {
	var err error

	*body, err = getGRPCBody[T](rp.GRPCReq)
	if err != nil {
		return err
	}
	if *body == nil {
		*body, err = getHTTPBody[T](rp.HTTPReq)
	}
	if err != nil {
		return err
	}

	if *body == nil {
		return ErrNoBody
	}
	return nil
}
