package service

import (
	"context"
	"time"

	v1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"google.golang.org/protobuf/types/known/durationpb"
)

const (
	// defaultMaxLifetime is the maximum token lifetime enforced by the
	// default policy.
	defaultMaxLifetime = 1 * time.Hour
)

// defaultAllowedPermissions lists the resource permissions that the default
// policy allows sensors to request. These must stay in sync with the needs
// of the OCP console plugin.
var defaultAllowedPermissions = map[string]v1.Access{
	"Deployment": v1.Access_READ_ACCESS,
	"Image":      v1.Access_READ_ACCESS,
}

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

// defaultTokenPolicy returns the hardcoded policy for internal tokens issued
// to sensors. It caps lifetime to 1 hour and allows read access to Deployment
// and Image resources.
// This policy should be kept in sync with the needs of the ocp console plugin.
func defaultTokenPolicy() *tokenPolicy {
	return newTokenPolicy(defaultMaxLifetime, defaultAllowedPermissions)
}

// validatePermissions checks that every requested permission is present in the
// allowlist with an access level no greater than the allowed level.
func (p *tokenPolicy) validatePermissions(requested map[string]v1.Access) error {
	for resource, requestedAccess := range requested {
		allowedAccess, ok := p.allowedPermissions[resource]
		if !ok {
			return errox.InvalidArgs.Newf(
				"permission %q for resource %q is not allowed", requestedAccess, resource)
		}
		if requestedAccess > allowedAccess {
			return errox.InvalidArgs.Newf(
				"requested permission %q for resource %q exceeds the allowed maximum permission %q",
				requestedAccess, resource, allowedAccess)
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

// enforce validates the request against all policy checks and returns the
// request with a capped lifetime if necessary.
func (p *tokenPolicy) enforce(
	ctx context.Context,
	req *v1.GenerateTokenForPermissionsAndScopeRequest,
) (*v1.GenerateTokenForPermissionsAndScopeRequest, error) {
	sensorClusterID, err := p.validateSensorIdentity(ctx)
	if err != nil {
		return nil, err
	}
	if err := p.validatePermissions(req.GetPermissions()); err != nil {
		return nil, err
	}
	if err := p.validateClusterScope(req.GetClusterScopes(), sensorClusterID); err != nil {
		return nil, err
	}
	return p.enforceLifetimeLimit(req)
}
