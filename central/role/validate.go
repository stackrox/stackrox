package role

import (
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac/accessscope"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/uuid"
)

func generateIdentifier() string {
	generatedIDSuffix := uuid.NewV4().String()
	return generatedIDSuffix
}

func isValidIdentifier(id string) bool {
	_, parseErr := uuid.FromString(id)
	return parseErr == nil
}

// GeneratePermissionSetID returns a random valid permission set ID.
func GeneratePermissionSetID() string {
	return generateIdentifier()
}

// EnsureValidPermissionSetID converts id to the correct format if necessary.
func EnsureValidPermissionSetID(id string) string {
	if isValidIdentifier(id) {
		return id
	}
	return generateIdentifier()
}

// GenerateAccessScopeID returns a random valid access scope ID.
func GenerateAccessScopeID() string {
	return generateIdentifier()
}

// EnsureValidAccessScopeID converts id to the correct format if necessary.
func EnsureValidAccessScopeID(id string) string {
	if isValidIdentifier(id) {
		return id
	}
	return generateIdentifier()
}

// ValidateAccessScopeID returns an error if the scope ID prefix is not correct.
func ValidateAccessScopeID(scope *storage.SimpleAccessScope) error {
	_, parseErr := uuid.FromString(scope.GetId())
	return parseErr
}

// ValidateRole checks whether the supplied protobuf message is a valid role.
func ValidateRole(role *storage.Role) error {
	var multiErr error

	if role.GetName() == "" {
		err := errors.New("role name field must be set")
		multiErr = multierror.Append(multiErr, err)
	}
	if role.GetGlobalAccess() != storage.Access_NO_ACCESS {
		err := errors.New("role global_access field must be 'NO_ACCESS' or unset")
		multiErr = multierror.Append(multiErr, err)
	}

	if len(role.GetResourceToAccess()) != 0 {
		err := errors.New("role must not set resource_to_access field")
		multiErr = multierror.Append(multiErr, err)
	}
	if role.GetPermissionSetId() == "" {
		err := errors.New("role permission_set_id field must be set")
		multiErr = multierror.Append(multiErr, err)
	}
	if role.GetAccessScopeId() == "" {
		err := errors.New("role access_scope_id field must be set")
		multiErr = multierror.Append(multiErr, err)
	}
	return multiErr
}

// ValidatePermissionSet checks whether the supplied protobuf message is a
// valid permission set.
func ValidatePermissionSet(ps *storage.PermissionSet) error {
	var multiErr error

	_, parseErr := uuid.FromString(ps.GetId())
	if parseErr != nil {
		multiErr = multierror.Append(multiErr, errors.Wrap(parseErr, "id field must be a valid UUID"))
	}
	if ps.GetName() == "" {
		multiErr = multierror.Append(multiErr, errors.New("name field must be set"))
	}
	if _, ok := storage.AccessConstraint_name[int32(ps.GetAccessConstraint())]; !ok {
		multiErr = multierror.Append(multiErr, errors.Errorf("unknown access constraint %d", ps.GetAccessConstraint()))
	}
	if _, ok := storage.ScopeConstraint_name[int32(ps.GetScopeConstraint())]; !ok {
		multiErr = multierror.Append(multiErr, errors.Errorf("unknown scope constraint %d", ps.GetScopeConstraint()))
	}
	for resource, access := range ps.GetResourceToAccess() {
		metadata, ok := resources.MetadataForResource(permissions.Resource(resource))
		if !ok {
			multiErr = multierror.Append(multiErr, errors.Errorf(
				"resource %q does not exist", resource))
			continue
		}
		if ps.GetAccessConstraint() == storage.AccessConstraint_ACCESS_CONSTRAINT_READ &&
			access != storage.Access_NO_ACCESS && access != storage.Access_READ_ACCESS {
			multiErr = multierror.Append(multiErr, errors.Errorf(
				"read-constrained permission set cannot grant %s access to resource %q", access, resource))
		}
		if ps.GetScopeConstraint() == storage.ScopeConstraint_SCOPE_CONSTRAINT_CLUSTER_NAMESPACE && access != storage.Access_NO_ACCESS &&
			metadata.GetScope() == permissions.GlobalScope {
			multiErr = multierror.Append(multiErr, errors.Errorf(
				"cluster-namespace-constrained permission set cannot grant access to global resource %q", resource))
		}
	}

	return multiErr
}

// ValidateSimpleAccessScope checks whether the supplied protobuf message is a
// valid simple access scope.
func ValidateSimpleAccessScope(scope *storage.SimpleAccessScope) error {
	var multiErr error

	if err := ValidateAccessScopeID(scope); err != nil {
		multiErr = multierror.Append(multiErr, err)
	}

	err := accessscope.ValidateSimpleAccessScopeProto(scope)
	if err != nil {
		multiErr = multierror.Append(multiErr, err)
	}

	return multiErr
}
