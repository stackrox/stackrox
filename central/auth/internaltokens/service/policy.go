package service

import (
	"context"
	"slices"
	"time"

	v1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/centralproxy"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"google.golang.org/protobuf/types/known/durationpb"
)

const (
	// defaultMaxLifetime is the maximum token lifetime enforced by the
	// default policy.
	defaultMaxLifetime = 1 * time.Hour
)

type tokenPolicy struct {
	maxLifetime        *durationpb.Duration
	allowedPermissions map[string]v1.Access
}

// newTokenPolicy creates a tokenPolicy with the given maximum lifetime and
// allowed permissions.
func newTokenPolicy(maxLifetime time.Duration, allowedPermissions map[string]v1.Access) *tokenPolicy {
	var maxLT *durationpb.Duration
	if maxLifetime > 0 {
		maxLT = durationpb.New(maxLifetime)
	}
	return &tokenPolicy{
		maxLifetime:        maxLT,
		allowedPermissions: allowedPermissions,
	}
}

// defaultTokenPolicy validates explicit permission requests from Sensors that do
// not select a Central-managed permission set. Permission-set requests use
// Central-managed permissions instead.
// This policy caps lifetime to 1 hour and should remain compatible with the
// OCP console plugin.
func defaultTokenPolicy() *tokenPolicy {
	// defaultAllowedPermissions lists the resource permissions allowed for
	// requests without a named permission set. These must stay in sync with the
	// needs of the known token consumers (including the OCP console plugin).
	defaultAllowedPermissions := map[string]v1.Access{
		"Deployment": v1.Access_READ_ACCESS,
		"Image":      v1.Access_READ_ACCESS,
	}
	return newTokenPolicy(defaultMaxLifetime, defaultAllowedPermissions)
}

// validatePermissions checks that every requested permission is present in the
// allowlist with an access level no greater than the allowed level.
func (p *tokenPolicy) validatePermissions(requested map[string]v1.Access) error {
	for resource, requestedAccess := range requested {
		allowedAccess, ok := p.allowedPermissions[resource]
		if !ok {
			return errox.InvalidArgs.Newf(
				"permission %q for resource %q is not allowed", requestedAccess.String(), resource)
		}
		if requestedAccess > allowedAccess {
			return errox.InvalidArgs.Newf(
				"requested permission %q for resource %q exceeds the allowed maximum permission %q",
				requestedAccess.String(), resource, allowedAccess.String())
		}
	}
	return nil
}

// validateClusterScope checks that every ClusterScope in the request references
// only the requesting sensor's own cluster.
func (p *tokenPolicy) validateClusterScope(scopes []*v1.ClusterScope, sensorClusterID string) error {
	for _, scope := range scopes {
		if scope.GetClusterId() != sensorClusterID {
			return errox.InvalidArgs.Newf(
				"cluster scope references cluster %q, but requesting sensor belongs to cluster %q",
				scope.GetClusterId(), sensorClusterID)
		}
	}
	return nil
}

// enforceLifetimeLimit validates that the requested lifetime is a positive duration
// and caps it to the configured maximum. If the lifetime is capped, a cloned
// request with the adjusted lifetime is returned.
func (p *tokenPolicy) enforceLifetimeLimit(
	req *v1.GenerateTokenForPermissionsAndScopeRequest,
) (*v1.GenerateTokenForPermissionsAndScopeRequest, error) {
	lifetime := req.GetLifetime()
	if lifetime == nil {
		return nil, errox.InvalidArgs.New("token lifetime is required")
	}
	if err := lifetime.CheckValid(); err != nil {
		return nil, errox.InvalidArgs.CausedByf("converting requested token lifetime: %v", err)
	}
	if lifetime.AsDuration() <= 0 {
		return nil, errox.InvalidArgs.CausedBy("token lifetime must be positive")
	}
	if p.maxLifetime != nil && lifetime.AsDuration() > p.maxLifetime.AsDuration() {
		req = req.CloneVT()
		req.Lifetime = p.maxLifetime
	}
	return req, nil
}

// validateSensorIdentity validates and extracts the sensor identity from the
// context. Returns the sensor's cluster ID.
func (p *tokenPolicy) validateSensorIdentity(ctx context.Context) (string, error) {
	identity := authn.IdentityFromContextOrNil(ctx)
	if identity == nil || identity.Service() == nil {
		return "", errox.NotAuthorized.New("missing service identity")
	}
	if identity.Service().GetType() != storage.ServiceType_SENSOR_SERVICE {
		return "", errox.NotAuthorized.CausedByf(
			"only sensor may access this API, unexpected service type %q",
			identity.Service().GetType())
	}
	return identity.Service().GetId(), nil
}

// enforce validates the request and returns its HTTP scopes alongside the
// request with a capped lifetime if necessary.
func (p *tokenPolicy) enforce(
	ctx context.Context,
	req *v1.GenerateTokenForPermissionsAndScopeRequest,
) (*v1.GenerateTokenForPermissionsAndScopeRequest, tokens.HTTPRequestScopes, error) {
	sensorClusterID, err := p.validateSensorIdentity(ctx)
	if err != nil {
		return nil, nil, err
	}
	if sensorClusterID == "" {
		return nil, nil, errox.NotAuthorized.New("sensor cluster ID is missing")
	}
	if len(req.GetScopedPermissionSets()) > 0 {
		if len(req.GetPermissionSetNames()) != 0 || len(req.GetPermissions()) != 0 || len(req.GetHttpRequestScopes()) != 0 {
			return nil, nil, errox.InvalidArgs.New("per-access grants cannot be combined with legacy token permissions or HTTP scopes")
		}
		if err := p.validateClusterScope(req.GetClusterScopes(), sensorClusterID); err != nil {
			return nil, nil, err
		}
		claims := tokens.RoxClaims{Clusters: clusterScopesFromRequest(req.GetClusterScopes())}
		seen := make(map[string]struct{})
		for _, grant := range req.GetScopedPermissionSets() {
			if grant == nil || grant.GetPermissionSetName() == "" {
				return nil, nil, errox.InvalidArgs.New("per-access grant requires a permission set name")
			}
			name := grant.GetPermissionSetName()
			if _, exists := seen[name]; exists {
				return nil, nil, errox.InvalidArgs.Newf("duplicate per-access permission set %q", name)
			}
			seen[name] = struct{}{}
			if name == centralproxy.MetricsPermissionSetName {
				if len(req.GetScopedPermissionSets()) != 1 || len(req.GetClusterScopes()) != 1 ||
					req.GetClusterScopes()[0].GetClusterId() != sensorClusterID ||
					!req.GetClusterScopes()[0].GetFullClusterAccess() ||
					len(req.GetClusterScopes()[0].GetNamespaces()) != 0 {
					return nil, nil, errox.InvalidArgs.New("metrics permission set requires exclusive full access to the Sensor's cluster")
				}
			} else if name != centralproxy.ConsoleProxyPermissionSetName && len(req.GetClusterScopes()) == 0 {
				return nil, nil, errox.InvalidArgs.New("generic Sensor proxy tokens require a cluster scope")
			}
			scopes, err := tokens.HTTPRequestScopesFromProto(grant.GetHttpRequestScopes())
			if err != nil {
				return nil, nil, errox.InvalidArgs.CausedBy(err)
			}
			if name == centralproxy.MetricsPermissionSetName &&
				(len(scopes) != 1 || !slices.Equal(scopes["/metrics"], []string{"GET"})) {
				return nil, nil, errox.InvalidArgs.New("metrics permission set tokens may only access GET /metrics")
			}
			claims.InternalRoles = append(claims.InternalRoles, &tokens.InternalRole{HTTPRequestScopes: scopes})
		}
		scoped, err := tokens.ValidatePerAccessHTTPScopes(&claims)
		if err != nil {
			return nil, nil, errox.InvalidArgs.CausedBy(err)
		}
		if !scoped {
			return nil, nil, errox.InvalidArgs.New("per-access grants require HTTP method scopes")
		}
		req, err = p.enforceLifetimeLimit(req)
		return req, nil, err
	}
	scopes, err := tokens.HTTPRequestScopesFromProto(req.GetHttpRequestScopes())
	if err != nil {
		return nil, nil, errox.InvalidArgs.CausedBy(err)
	}
	if len(req.GetPermissionSetNames()) > 0 {
		if slices.Contains(req.GetPermissionSetNames(), centralproxy.MetricsPermissionSetName) {
			if len(req.GetPermissionSetNames()) != 1 {
				return nil, nil, errox.InvalidArgs.New("metrics permission set cannot be combined with other permission sets")
			}
			if len(req.GetClusterScopes()) != 1 ||
				req.GetClusterScopes()[0].GetClusterId() != sensorClusterID ||
				!req.GetClusterScopes()[0].GetFullClusterAccess() ||
				len(req.GetClusterScopes()[0].GetNamespaces()) != 0 {
				return nil, nil, errox.InvalidArgs.New("metrics permission set requests require full access to the Sensor's cluster")
			}
			if len(scopes) != 1 || !slices.Equal(scopes["/metrics"], centralproxy.MetricsHTTPScopes["/metrics"]) {
				return nil, nil, errox.InvalidArgs.New("metrics permission set tokens may only access GET /metrics")
			}
		}
		if len(req.GetClusterScopes()) == 0 && (len(req.GetPermissionSetNames()) != 1 ||
			req.GetPermissionSetNames()[0] != centralproxy.ConsoleProxyPermissionSetName) {
			return nil, nil, errox.InvalidArgs.New("generic Sensor proxy tokens require a cluster scope")
		}
		if len(scopes) == 0 {
			return nil, nil, errox.InvalidArgs.New("Sensor proxy tokens require HTTP method scopes")
		}
		hasMethod := false
		for _, methods := range scopes {
			if len(methods) > 0 {
				hasMethod = true
				break
			}
		}
		if !hasMethod {
			return nil, nil, errox.InvalidArgs.New("Sensor proxy tokens require an HTTP method scope")
		}
	} else {
		if err := p.validatePermissions(req.GetPermissions()); err != nil {
			return nil, nil, err
		}
	}
	if err := p.validateClusterScope(req.GetClusterScopes(), sensorClusterID); err != nil {
		return nil, nil, err
	}
	req, err = p.enforceLifetimeLimit(req)
	if err != nil {
		return nil, nil, err
	}
	return req, scopes, nil
}
