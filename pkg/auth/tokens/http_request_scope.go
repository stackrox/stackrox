package tokens

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
)

// HTTPRequestScopesFromProto converts and validates optional request scopes.
func HTTPRequestScopesFromProto(requested []*v1.HTTPRequestScope) (HTTPRequestScopes, error) {
	if len(requested) == 0 {
		return nil, nil
	}
	if len(requested) > 32 {
		return nil, fmt.Errorf("too many HTTP request scopes: %d", len(requested))
	}
	scopes := make(HTTPRequestScopes, len(requested))
	for _, scope := range requested {
		if scope == nil {
			return nil, errors.New("nil HTTP request scope")
		}
		if _, found := scopes[scope.GetPath()]; found {
			return nil, fmt.Errorf("duplicate HTTP request path %q", scope.GetPath())
		}
		scopes[scope.GetPath()] = append([]string{}, scope.GetMethods()...)
	}
	if err := ValidateHTTPRequestScopes(scopes); err != nil {
		return nil, err
	}
	return scopes, nil
}

// ValidateHTTPRequestScopes checks optional path-to-method restrictions.
// Callers must validate scopes before signing them into a token.
func ValidateHTTPRequestScopes(scopes HTTPRequestScopes) error {
	if len(scopes) > 32 {
		return fmt.Errorf("too many HTTP request scopes: %d", len(scopes))
	}
	methodCount := 0
	for path, methods := range scopes {
		if !validHTTPRequestPath(path) {
			return fmt.Errorf("invalid HTTP request path %q", path)
		}
		seen := make(map[string]struct{}, len(methods))
		for _, method := range methods {
			methodCount++
			if methodCount > 32 {
				return errors.New("too many HTTP request methods")
			}
			if !ValidHTTPRequestScope(HTTPRequestScope{Method: method, Path: path}) {
				return fmt.Errorf("invalid HTTP request method %q for path %q", method, path)
			}
			if _, found := seen[method]; found {
				return fmt.Errorf("duplicate HTTP request method %q for path %q", method, path)
			}
			seen[method] = struct{}{}
		}
	}
	return nil
}

// AllowsHTTPRequest checks the signed HTTP scope against trusted request information.
// The most specific matching path determines the permitted methods. An
// empty method list denies that path, even if a broader prefix allows it.
// An ambiguous URL fails closed.
func AllowsHTTPRequest(scopes HTTPRequestScopes, ri requestinfo.RequestInfo) bool {
	if len(scopes) == 0 || ri.HTTPRequest == nil || ri.HTTPRequest.URL == nil {
		return false
	}
	u := ri.HTTPRequest.URL
	if !canonicalHTTPPath(u) {
		return false
	}
	var bestPath string
	for scopedPath := range scopes {
		if !validHTTPRequestPath(scopedPath) {
			continue
		}
		if scopedPath != u.Path && !(strings.HasSuffix(scopedPath, "/") && strings.HasPrefix(u.Path, scopedPath)) {
			continue
		}
		if len(scopedPath) > len(bestPath) {
			bestPath = scopedPath
		}
	}
	for _, method := range scopes[bestPath] {
		if ValidHTTPRequestScope(HTTPRequestScope{Method: method, Path: bestPath}) && method == ri.HTTPRequest.Method {
			return true
		}
	}
	return false
}

// ValidHTTPRequestScope accepts only canonical, uppercase-method scope entries.
func ValidHTTPRequestScope(scope HTTPRequestScope) bool {
	return scope.Method != "" && strings.Trim(scope.Method, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") == "" &&
		validHTTPRequestPath(scope.Path)
}

func validHTTPRequestPath(p string) bool {
	return canonicalHTTPPath(&url.URL{Path: p})
}

func canonicalHTTPPath(u *url.URL) bool {
	p := u.Path
	return strings.HasPrefix(p, "/") && !strings.HasPrefix(p, "//") &&
		!strings.Contains(p, "\\") && (p == "/" || path.Clean(p) == strings.TrimSuffix(p, "/")) &&
		u.EscapedPath() == p
}
