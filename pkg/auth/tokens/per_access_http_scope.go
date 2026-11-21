package tokens

import (
	"slices"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
)

// ValidatePerAccessHTTPScopes validates tokens with at least one HTTP-scoped role.
// Other roles may have nil scopes, imposing no additional HTTP restriction.
// Per-access scopes further restrict any token-wide HTTP scopes. Empty method
// lists deny the matching path, using specificity independently of token-wide scopes.
func ValidatePerAccessHTTPScopes(claims *RoxClaims) (bool, error) {
	if claims == nil {
		return false, nil
	}
	perAccess := false
	for _, role := range claims.InternalRoles {
		if role != nil && role.HTTPRequestScopes != nil {
			perAccess = true
		}
	}
	if !perAccess {
		return false, nil
	}
	if len(claims.InternalRoles) > 32 {
		return false, errors.New("too many HTTP-scoped roles")
	}
	paths, methods := 0, 0
	for _, role := range claims.InternalRoles {
		if role == nil {
			return false, errors.New("per-access roles must be non-nil")
		}
		if err := ValidateHTTPRequestScopes(role.HTTPRequestScopes); err != nil {
			return false, err
		}
		for _, allowedMethods := range role.HTTPRequestScopes {
			paths++
			methods += len(allowedMethods)
		}
	}
	if paths > 32 || methods > 32 {
		return false, errors.New("too many per-access HTTP paths or methods")
	}
	return true, nil
}

// ScopedRolesForHTTPRequest selects roles with nil HTTP scopes and scoped roles
// permitted for the most specific matching path and method. A direct gRPC
// request selects only roles with nil HTTP scopes.
func ScopedRolesForHTTPRequest(roles []*InternalRole, ri requestinfo.RequestInfo) []*InternalRole {
	groups := make([]HTTPRequestScopes, 0, len(roles))
	for _, role := range roles {
		if role != nil {
			groups = append(groups, role.HTTPRequestScopes)
		}
	}
	path := MostSpecificHTTPPath(groups, ri)
	var selected []*InternalRole
	for _, role := range roles {
		if role != nil && (role.HTTPRequestScopes == nil ||
			(path != "" && slices.Contains(role.HTTPRequestScopes[path], ri.HTTPRequest.Method))) {
			selected = append(selected, role)
		}
	}
	return selected
}
