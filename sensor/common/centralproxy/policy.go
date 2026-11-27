package centralproxy

import (
	"slices"

	centralv1 "github.com/stackrox/rox/generated/internalapi/central/v1"
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

// hasConsolePermissionSet reports whether any configured grant of the client
// selects the console permission set. The console feature flag gates these clients.
func (g *ClientPolicyGrant) hasConsolePermissionSet() bool {
	return slices.ContainsFunc(g.PermissionSets, isConsolePermissionSet)
}

// legacyEligible reports whether the request can use the token forms of older
// Centrals: authentication-only grants, which request no permissions, and
// console grants, which request the original Deployment/Image read permissions.
// Only the pre-existing bearer authentication is eligible; newer client kinds
// require the permission-set token capability.
func (g *ClientPolicyGrant) legacyEligible() bool {
	if g.AuthType != clientAuthKubernetesBearer {
		return false
	}
	for _, set := range g.AllowedPermissionSets {
		if !isConsolePermissionSet(set) {
			return false
		}
	}
	return true
}

func isConsolePermissionSet(grant *centralv1.SelectedPermissionSet) bool {
	return isConsoleSelector(grant.GetId(), grant.GetName())
}

func isConsoleSelector(id, name string) bool {
	return name == accesscontrol.ConsoleProxy || id == accesscontrol.DefaultPermissionSetIDs[accesscontrol.ConsoleProxy]
}
