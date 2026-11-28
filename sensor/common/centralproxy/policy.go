package centralproxy

import (
	"maps"
	"slices"

	apiV1 "github.com/stackrox/rox/generated/api/v1"
	centralv1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/centralproxy"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
)

const (
	clientAuthMTLS             = "mtls"
	clientAuthKubernetesBearer = "kubernetesBearer" //nolint:gosec // Authentication type, not a credential.
)

// ClientPolicyGrant identifies an authenticated client and its separate access grants.
type ClientPolicyGrant struct {
	AuthType string
	Subject  string
	// PermissionSets lists every access grant of the matched client that selects
	// a permission set. Client-level decisions read this list.
	PermissionSets []*centralv1.SelectedPermissionSet
	// AllowedPermissionSets lists the grants whose HTTP scope admits the authorized
	// request. Tokens are requested for these grants only, so grants scoped to other
	// routes cannot widen the permissions carried by the current request. The list
	// is empty when the request matched only authentication-only grants.
	AllowedPermissionSets []*centralv1.SelectedPermissionSet
}

// isConsoleClient reports whether the matched client is the console
// integration client: its permission-set grants select only the console
// permission set. The console feature flag gates these clients; adding any
// other permission set turns the client into a custom client.
func (g *ClientPolicyGrant) isConsoleClient() bool {
	console := false
	for _, set := range g.PermissionSets {
		if !isConsolePermissionSet(set) {
			return false
		}
		console = true
	}
	return console
}

// legacyEligible reports whether the request can use the fallback token forms
// of older Centrals: authentication-only grants, which request no permissions,
// and console grants whose scopes stay within the default console routes, which
// request the original Deployment/Image read permissions. Only the pre-existing
// bearer authentication is eligible; newer client kinds require newer token
// capabilities.
func (g *ClientPolicyGrant) legacyEligible() bool {
	if g.AuthType != clientAuthKubernetesBearer {
		return false
	}
	for _, set := range g.AllowedPermissionSets {
		if !consoleScopesWithinDefaultRoutes(set) {
			return false
		}
	}
	return true
}

func isConsolePermissionSet(grant *centralv1.SelectedPermissionSet) bool {
	return isConsoleSelector(grant.GetId(), grant.GetName())
}

func isConsoleSelector(id, name string) bool {
	return name == accesscontrol.K8sCentralProxyUser || id == accesscontrol.DefaultPermissionSetIDs[accesscontrol.K8sCentralProxyUser]
}

func clientPolicyScopesToProto(scopes tokens.HTTPRequestScopes) []*apiV1.HTTPRequestScope {
	if scopes == nil {
		return nil
	}
	result := make([]*apiV1.HTTPRequestScope, 0, len(scopes))
	for _, path := range slices.Sorted(maps.Keys(scopes)) {
		result = append(result, &apiV1.HTTPRequestScope{Path: path, Methods: slices.Clone(scopes[path])})
	}
	return result
}

// consoleScopesWithinDefaultRoutes reports whether the grant selects the console
// permission set with explicit scopes that the default console routes contain.
// Older Centrals cannot enforce HTTP claims, so fallback token forms are limited
// to grants whose routes are known to be console routes.
func consoleScopesWithinDefaultRoutes(set *centralv1.SelectedPermissionSet) bool {
	if !isConsolePermissionSet(set) || len(set.GetHttpRequestScopes()) == 0 {
		return false
	}
	for _, scope := range set.GetHttpRequestScopes() {
		methods, found := centralproxy.ConsoleHTTPScopes[scope.GetPath()]
		if !found {
			return false
		}
		for _, method := range scope.GetMethods() {
			if !slices.Contains(methods, method) {
				return false
			}
		}
	}
	return true
}

func safeConsoleFallback(grants []*centralv1.SelectedPermissionSet) bool {
	return len(grants) == 1 && consoleScopesWithinDefaultRoutes(grants[0])
}
