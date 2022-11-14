package role

import (
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/uuid"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	// permissionSetIDPrefix should be prepended to every human-hostile ID of a
	// permission set for readability, e.g.,
	//     "io.stackrox.authz.permissionset.94ac7bfe-f9b2-402e-b4f2-bfda480e1a13".
	permissionSetIDPrefix = "io.stackrox.authz.permissionset."

	// accessScopeIDPrefix should be prepended to every human-hostile ID of an
	// access scope for readability, e.g.,
	//     "io.stackrox.authz.accessscope.94ac7bfe-f9b2-402e-b4f2-bfda480e1a13".
	accessScopeIDPrefix = "io.stackrox.authz.accessscope."
)

func generateIdentifier(prefix string) string {
	generatedIDSuffix := uuid.NewV4().String()
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return generatedIDSuffix
	}
	return prefix + generatedIDSuffix
}

func isValidIdentifier(prefix string, id string) bool {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		_, parseErr := uuid.FromString(id)
		return parseErr == nil
	}
	return strings.HasPrefix(id, prefix)
}

// GeneratePermissionSetID returns a random valid permission set ID.
func GeneratePermissionSetID() string {
	return generateIdentifier(permissionSetIDPrefix)
}

// EnsureValidPermissionSetID converts id to the correct format if necessary.
func EnsureValidPermissionSetID(id string) string {
	if isValidIdentifier(permissionSetIDPrefix, id) {
		return id
	}
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return generateIdentifier(permissionSetIDPrefix)
	}
	return permissionSetIDPrefix + id
}

// GenerateAccessScopeID returns a random valid access scope ID.
func GenerateAccessScopeID() string {
	return generateIdentifier(accessScopeIDPrefix)
}

// EnsureValidAccessScopeID converts id to the correct format if necessary.
func EnsureValidAccessScopeID(id string) string {
	if isValidIdentifier(accessScopeIDPrefix, id) {
		return id
	}
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return generateIdentifier(accessScopeIDPrefix)
	}
	return accessScopeIDPrefix + id
}

// ValidateAccessScopeID returns an error if the scope ID prefix is not correct.
func ValidateAccessScopeID(scope *storage.SimpleAccessScope) error {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		_, parseErr := uuid.FromString(scope.GetId())
		return parseErr
	}
	if !strings.HasPrefix(scope.GetId(), accessScopeIDPrefix) {
		return errors.Errorf("id field must be in '%s*' format", accessScopeIDPrefix)
	}
	return nil
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

	if env.PostgresDatastoreEnabled.BooleanSetting() {
		_, parseErr := uuid.FromString(ps.GetId())
		if parseErr != nil {
			multiErr = multierror.Append(multiErr, errors.Wrap(parseErr, "id field must be a valid UUID"))
		}
	} else if !strings.HasPrefix(ps.GetId(), permissionSetIDPrefix) {
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

	if err := ValidateAccessScopeID(scope); err != nil {
		multiErr = multierror.Append(multiErr, err)
	}
	if scope.GetName() == "" {
		multiErr = multierror.Append(multiErr, errors.New("name field must be set"))
	}

	err := ValidateSimpleAccessScopeRules(scope.GetRules())
	if err != nil {
		multiErr = multierror.Append(multiErr, err)
	}

	return multiErr
}

// ValidateSimpleAccessScopeRules checks whether the supplied protobuf message
// represents valid simple access scope rules.
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
		err := validateSelectorRequirement(labelSelector)
		if err != nil {
			multiErr = multierror.Append(multiErr, err)
		}
	}
	for _, labelSelector := range scopeRules.GetNamespaceLabelSelectors() {
		err := validateSelectorRequirement(labelSelector)
		if err != nil {
			multiErr = multierror.Append(multiErr, err)
		}
	}

	return multiErr
}

func validateSelectorRequirement(labelSelector *storage.SetBasedLabelSelector) error {
	var multiErr error
	for _, requirement := range labelSelector.GetRequirements() {
		op := effectiveaccessscope.ConvertLabelSelectorOperatorToSelectionOperator(requirement.GetOp())
		_, err := labels.NewRequirement(requirement.GetKey(), op, requirement.Values)
		if err != nil {
			multiErr = multierror.Append(multiErr, err)
		}
	}
	return multiErr
}
