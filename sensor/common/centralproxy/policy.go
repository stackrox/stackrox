package centralproxy

import (
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
	// PermissionSets lists every access grant of the matched client. Client-level
	// decisions such as console classification read this list.
	PermissionSets []*centralv1.SelectedPermissionSet
	// AllowedPermissionSets lists the grants whose HTTP scope admits the authorized
	// request. Tokens are requested for these grants only, so grants scoped to other
	// routes cannot widen the permissions carried by the current request.
	AllowedPermissionSets []*centralv1.SelectedPermissionSet
}

func (g *ClientPolicyGrant) isConsole() bool {
	return len(g.PermissionSets) == 1 && isConsolePermissionSet(g.PermissionSets[0])
}

func isConsolePermissionSet(grant *centralv1.SelectedPermissionSet) bool {
	return grant.GetId() == accesscontrol.DefaultPermissionSetIDs[accesscontrol.ConsoleProxy] ||
		grant.GetName() == accesscontrol.ConsoleProxy
}
