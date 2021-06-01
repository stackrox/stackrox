package utils

import (
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/uuid"
)

const (
	// roleIDPrefix should be prepended to every human-hostile ID of a role for
	//  readability, e.g.,
	//     "io.stackrox.authz.role.94ac7bfe-f9b2-402e-b4f2-bfda480e1a13".
	roleIDPrefix = "io.stackrox.authz.role."

	// permissionSetIDPrefix should be prepended to every human-hostile ID of a
	// permission set for readability, e.g.,
	//     "io.stackrox.authz.permissionset.94ac7bfe-f9b2-402e-b4f2-bfda480e1a13".
	permissionSetIDPrefix = "io.stackrox.authz.permissionset."

	// accessScopeIDPrefix should be prepended to every human-hostile ID of an
	// access scope for readability, e.g.,
	//     "io.stackrox.authz.accessscope.94ac7bfe-f9b2-402e-b4f2-bfda480e1a13".
	accessScopeIDPrefix = "io.stackrox.authz.accessscope."
)

// GenerateRoleID returns a random valid role ID.
func GenerateRoleID() string {
	return roleIDPrefix + uuid.NewV4().String()
}

// EnsureValidRoleID converts id to the correct format if necessary.
func EnsureValidRoleID(id string) string {
	if strings.HasPrefix(id, roleIDPrefix) {
		return id
	}
	return roleIDPrefix + id
}

// GeneratePermissionSetID returns a random valid permission set ID.
func GeneratePermissionSetID() string {
	return permissionSetIDPrefix + uuid.NewV4().String()
}

// EnsureValidPermissionSetID converts id to the correct format if necessary.
func EnsureValidPermissionSetID(id string) string {
	if strings.HasPrefix(id, permissionSetIDPrefix) {
		return id
	}
	return permissionSetIDPrefix + id
}

// GenerateAccessScopeID returns a random valid access scope ID.
func GenerateAccessScopeID() string {
	return accessScopeIDPrefix + uuid.NewV4().String()
}

// EnsureValidAccessScopeID converts id to the correct format if necessary.
func EnsureValidAccessScopeID(id string) string {
	if strings.HasPrefix(id, accessScopeIDPrefix) {
		return id
	}
	return accessScopeIDPrefix + id
}

// ValidatePermissionSet checks whether the supplied protobuf message is a
// valid permission set.
func ValidatePermissionSet(ps *storage.PermissionSet) error {
	var multiErr error

	if !strings.HasPrefix(ps.GetId(), permissionSetIDPrefix) {
		multiErr = multierror.Append(multiErr, errors.Errorf("id field must be in '%s*' format", permissionSetIDPrefix))
	}
	if ps.GetName() == "" {
		multiErr = multierror.Append(multiErr, errors.New("name field must be set"))
	}
	for resource := range ps.GetResourceToAccess() {
		if _, ok := resources.MetadataForResource(permissions.Resource(resource)); !ok {
			multiErr = multierror.Append(multiErr, errors.Errorf(
				"resource %q does not exist", resource))
		}
	}

	return multiErr
}

// ValidateSimpleAccessScope checks whether the supplied protobuf message is a
// valid simple access scope.
func ValidateSimpleAccessScope(scope *storage.SimpleAccessScope) error {
	var multiErr error

	if !strings.HasPrefix(scope.GetId(), accessScopeIDPrefix) {
		multiErr = multierror.Append(multiErr, errors.Errorf("id field must be in '%s*' format", accessScopeIDPrefix))
	}
	if scope.GetName() == "" {
		multiErr = multierror.Append(multiErr, errors.New("name field must be set"))
	}

	err := ValidateSimpleAccessScopeRules(scope.GetRules())
	if err != nil {
		multiErr = multierror.Append(err)
	}

	return multiErr
}

// ValidateSimpleAccessScopeRules checks whether the supplied protobuf message
// represents valid simple access scope rules.
//
// TODO(ROX-7136): Add checks to verify that supplied keys and values are valid.
func ValidateSimpleAccessScopeRules(scopeRules *storage.SimpleAccessScope_Rules) error {
	var multiErr error

	for _, ns := range scopeRules.GetIncludedNamespaces() {
		if ns.GetClusterName() == "" || ns.GetNamespaceName() == "" {
			multiErr = multierror.Append(multiErr, errors.Errorf(
				"both cluster_name and namespace_name fields must be set in namespace rule <%s, %s>",
				ns.GetClusterName(), ns.GetNamespaceName()))
		}
	}
	for _, labelSelector := range scopeRules.GetClusterLabelSelectors() {
		if len(labelSelector.GetRequirements()) == 0 {
			multiErr = multierror.Append(multiErr, errors.New(
				"requirements field must be set in every cluster label selector"))
			break
		}
	}
	for _, labelSelector := range scopeRules.GetNamespaceLabelSelectors() {
		if len(labelSelector.GetRequirements()) == 0 {
			multiErr = multierror.Append(multiErr, errors.New(
				"requirements field must be set in every namespace label selector"))
			break
		}
	}

	return multiErr
}
