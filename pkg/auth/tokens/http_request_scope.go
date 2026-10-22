package tokens

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
)

// HTTPRequestScopesFromProto converts and validates optional request scopes.
func HTTPRequestScopesFromProto(requested []*v1.HTTPRequestScope) ([]HTTPRequestScope, error) {
	if len(requested) == 0 {
		return nil, nil
	}
	if len(requested) > 32 {
		return nil, fmt.Errorf("too many HTTP request scopes: %d", len(requested))
	}
	scopes := make([]HTTPRequestScope, 0, len(requested))
	for _, scope := range requested {
		scopes = append(scopes, HTTPRequestScope{Method: scope.GetMethod(), Path: scope.GetPath()})
	}
	if err := ValidateHTTPRequestScopes(scopes); err != nil {
		return nil, err
	}
	return scopes, nil
}

// HTTPRequestScopesByPath groups validated method/path pairs for signing.
func HTTPRequestScopesByPath(scopes []HTTPRequestScope) HTTPRequestScopes {
	if len(scopes) == 0 {
		return nil
	}
	byPath := make(HTTPRequestScopes)
	for _, scope := range scopes {
		byPath[scope.Path] = append(byPath[scope.Path], scope.Method)
	}
	return byPath
}

// ValidateHTTPRequestScopes checks an optional list of token restrictions.
// Callers must validate scopes before signing them into a token.
func ValidateHTTPRequestScopes(scopes []HTTPRequestScope) error {
	if len(scopes) > 32 {
		return fmt.Errorf("too many HTTP request scopes: %d", len(scopes))
	}
	seen := make(map[HTTPRequestScope]struct{}, len(scopes))
	for _, scope := range scopes {
		if !ValidHTTPRequestScope(scope) {
			return fmt.Errorf("invalid HTTP request scope %q %q", scope.Method, scope.Path)
		}
		if _, found := seen[scope]; found {
			return fmt.Errorf("duplicate HTTP request scope %q %q", scope.Method, scope.Path)
		}
		seen[scope] = struct{}{}
	}
	return nil
}

// AllowsHTTPRequest checks the signed HTTP scope against trusted request information.
// The most specific matching path determines the permitted methods. An
// ambiguous URL fails closed.
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
		if !ValidHTTPRequestScope(HTTPRequestScope{Method: "GET", Path: scopedPath}) {
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
		canonicalHTTPPath(&url.URL{Path: scope.Path})
}

func canonicalHTTPPath(u *url.URL) bool {
	p := u.Path
	return strings.HasPrefix(p, "/") && !strings.HasPrefix(p, "//") &&
		!strings.Contains(p, "\\") && path.Clean(p) == strings.TrimSuffix(p, "/") &&
		u.EscapedPath() == p
}
