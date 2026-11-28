package tokens

import (
	"fmt"
	"net/url"
	"path"
	"slices"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
)

// ValidateHTTPRequestScopes checks optional path-to-method restrictions.
// Callers must validate scopes before signing them into a token.
func ValidateHTTPRequestScopes(scopes HTTPRequestScopes) error {
	if scopes != nil && len(scopes) == 0 {
		return errors.New("HTTP request scopes must be nil or contain a path")
	}
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
	bestPath := MostSpecificHTTPPath([]HTTPRequestScopes{scopes}, ri)
	if bestPath == "" {
		return false
	}
	return slices.Contains(scopes[bestPath], ri.HTTPRequest.Method)
}

// MostSpecificHTTPPath finds the longest matching path across all scope groups.
// Selecting the path before the method prevents a broader prefix from granting
// access when a more specific path denies the requested method.
func MostSpecificHTTPPath(groups []HTTPRequestScopes, ri requestinfo.RequestInfo) string {
	if ri.HTTPRequest == nil || ri.HTTPRequest.URL == nil || !canonicalHTTPPath(ri.HTTPRequest.URL) {
		return ""
	}
	u := ri.HTTPRequest.URL
	var bestPath string
	for _, scopes := range groups {
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
	}
	return bestPath
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
