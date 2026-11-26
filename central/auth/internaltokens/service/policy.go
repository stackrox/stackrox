package service

import (
	"context"
	"strings"
	"time"

	v1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/uuid"
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

// defaultTokenPolicy caps Sensor token lifetimes to one hour. Requests without
// permission sets use the legacy console permission allowlist.
func defaultTokenPolicy() *tokenPolicy {
	// defaultAllowedPermissions lists the resource permissions that the default
	// policy allows sensors to request. These must stay in sync with the needs
	// of the known token consumers (including the OCP console plugin).
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
	clusterID := identity.Service().GetId()
	if clusterID == "" {
		return "", errox.NotAuthorized.New("sensor cluster ID is missing")
	}
	return clusterID, nil
}

// maxPermissionSetsPerRequest bounds how many permission sets a single token
// request may select.
const maxPermissionSetsPerRequest = 32

// validatePermissionSets checks that permission sets are not combined with
// legacy permissions, that their number stays bounded, and that every
// selector is well-formed.
func validatePermissionSets(req *v1.GenerateTokenForPermissionsAndScopeRequest) error {
	if len(req.GetPermissions()) != 0 {
		return errox.InvalidArgs.New("permission grant forms are mutually exclusive")
	}
	if len(req.GetPermissionSets()) > maxPermissionSetsPerRequest {
		return errox.InvalidArgs.New("too many permission sets")
	}
	for _, selected := range req.GetPermissionSets() {
		if err := validatePermissionSetSelector(selected); err != nil {
			return err
		}
	}
	return nil
}

// validatePermissionSetSelector checks that exactly one nonempty id or name
// selector is present.
func validatePermissionSetSelector(selected *v1.SelectedPermissionSet) error {
	switch selector := selected.GetSelector().(type) {
	case *v1.SelectedPermissionSet_Id:
		if _, err := uuid.FromString(selector.Id); err != nil {
			return errox.InvalidArgs.New("permission set ID must be a UUID")
		}
	case *v1.SelectedPermissionSet_Name:
		if selector.Name == "" || strings.TrimSpace(selector.Name) != selector.Name {
			return errox.InvalidArgs.New("permission set name must be nonempty without surrounding whitespace")
		}
	default:
		return errox.InvalidArgs.New("permission set selector is required")
	}
	return nil
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
	if len(req.GetPermissionSets()) > 0 {
		if err := validatePermissionSets(req); err != nil {
			return nil, err
		}
	} else {
		if err := p.validatePermissions(req.GetPermissions()); err != nil {
			return nil, err
		}
	}
	if err := p.validateClusterScope(req.GetClusterScopes(), sensorClusterID); err != nil {
		return nil, err
	}
	return p.enforceLifetimeLimit(req)
}
