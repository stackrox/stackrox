package tokens

import (
	"fmt"
	"slices"

	"github.com/stackrox/rox/pkg/grpc/requestinfo"
)

// ValidatePerAccessHTTPScopes checks that every role in a per-access token is
// HTTP-scoped, and that no method/path pair belongs to more than one role.
// The existing token-wide HTTP scope is mutually exclusive with this format.
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
		if len(claims.Clusters) != 0 {
			return false, fmt.Errorf("shared clusters require per-access HTTP scopes")
		}
		return false, nil
	}
	if claims.HTTPRequestScopes != nil {
		return false, fmt.Errorf("token-wide and per-access HTTP scopes cannot be combined")
	}
	if len(claims.InternalRoles) > 32 {
		return false, fmt.Errorf("too many HTTP-scoped roles")
	}
	seen := make(map[string]map[string]struct{})
	paths, methods := 0, 0
	for _, role := range claims.InternalRoles {
		if role == nil || len(role.HTTPRequestScopes) == 0 || len(role.Clusters) != 0 {
			return false, fmt.Errorf("per-access roles require HTTP scopes and shared clusters")
		}
		if err := ValidateHTTPRequestScopes(role.HTTPRequestScopes); err != nil {
			return false, err
		}
		for path, allowedMethods := range role.HTTPRequestScopes {
			paths++
			if len(allowedMethods) == 0 {
				return false, fmt.Errorf("HTTP methods are required for path %q", path)
			}
			if seen[path] == nil {
				seen[path] = make(map[string]struct{})
			}
			for _, method := range allowedMethods {
				methods++
				if _, exists := seen[path][method]; exists {
					return false, fmt.Errorf("duplicate HTTP method %q for path %q across roles", method, path)
				}
				seen[path][method] = struct{}{}
			}
		}
	}
	if paths > 32 || methods > 32 {
		return false, fmt.Errorf("too many per-access HTTP paths or methods")
	}
	return true, nil
}

// ScopedRolesForHTTPRequest selects only the roles permitted for the most
// specific matching path and method. A direct gRPC request matches no roles.
func ScopedRolesForHTTPRequest(roles []*InternalRole, ri requestinfo.RequestInfo) []*InternalRole {
	groups := make([]HTTPRequestScopes, 0, len(roles))
	for _, role := range roles {
		if role != nil {
			groups = append(groups, role.HTTPRequestScopes)
		}
	}
	path := MostSpecificHTTPPath(groups, ri)
	if path == "" {
		return nil
	}
	var selected []*InternalRole
	for _, role := range roles {
		if role != nil && slices.Contains(role.HTTPRequestScopes[path], ri.HTTPRequest.Method) {
			selected = append(selected, role)
		}
	}
	return selected
}
