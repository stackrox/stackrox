package centralproxy

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"maps"
	"net/http"
	"slices"

	apiV1 "github.com/stackrox/rox/generated/api/v1"
	centralv1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/centralproxy"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	authenticationv1 "k8s.io/api/authentication/v1"
)

const (
	clientAuthMTLS             = "mtls"
	clientAuthKubernetesBearer = "kubernetesBearer" //nolint:gosec // Authentication type, not a credential.
)

// requestPolicy chooses a grant and the credential verification appropriate to it.
// authenticate reviews the bearer token lazily, so certificate-only clients never
// need Kubernetes TokenReview and cannot fall back to a bearer credential.
type requestPolicy interface {
	authorizeRequest(context.Context, *http.Request, func() (*authenticationv1.UserInfo, error)) (*ClientPolicyGrant, error)
	configureTLS(*tls.Config, func([]*x509.Certificate))
}

// ClientPolicyGrant is the policy decision for a single authenticated request.
// The full access list is sent to Central. Routes must also be enforced locally
// before forwarding, independently of Central's token checks.
type ClientPolicyGrant struct {
	// AuthType and Subject identify the matched policy entry for auditing, not PoP.
	AuthType       string
	Subject        string
	PermissionSets []*centralv1.ScopedPermissionSet
}

func (grant *ClientPolicyGrant) isConsole() bool {
	if len(grant.PermissionSets) != 1 {
		return false
	}
	set := grant.PermissionSets[0]
	return set.GetId() == accesscontrol.DefaultPermissionSetIDs[accesscontrol.ConsoleProxy] ||
		set.GetName() == accesscontrol.ConsoleProxy
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

func consoleRequestScopes() []*apiV1.HTTPRequestScope {
	return clientPolicyScopesToProto(centralproxy.ConsoleHTTPScopes)
}
