package centralproxy

import (
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
)

const (
	clientAuthMTLS             = "mtls"
	clientAuthKubernetesBearer = "kubernetesBearer" //nolint:gosec // Authentication type, not a credential.
)

// ClientPolicyGrant identifies an authenticated client and its single permission set.
type ClientPolicyGrant struct {
	AuthType          string
	Subject           string
	PermissionSetID   string
	PermissionSetName string
}

func (g *ClientPolicyGrant) isConsole() bool {
	return g.PermissionSetID == accesscontrol.DefaultPermissionSetIDs[accesscontrol.ConsoleProxy] ||
		g.PermissionSetName == accesscontrol.ConsoleProxy
}
