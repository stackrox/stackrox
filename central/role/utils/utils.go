package utils

import (
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
)

const (
	// accessScopeIDPrefix should be prepended to every human-hostile ID of an
	// access scope for readability, e.g.,
	//     "io.stackrox.authz.accessscope.94ac7bfe-f9b2-402e-b4f2-bfda480e1a13".
	accessScopeIDPrefix = "io.stackrox.authz.accessscope."
)

// FillAccessList fills in the access list if the role uses the GlobalAccess field.
func FillAccessList(role *storage.Role) {
	if role.GetGlobalAccess() == storage.Access_NO_ACCESS {
		return
	}
	// If the role has global access, fill in the full list of resources with the max of the role's current access and global access.
	if role.GetResourceToAccess() == nil {
		role.ResourceToAccess = make(map[string]storage.Access)
	}
	for _, resource := range resources.ListAll() {
		if role.ResourceToAccess[string(resource)] < role.GetGlobalAccess() {
			role.ResourceToAccess[string(resource)] = role.GetGlobalAccess()
		}
	}
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
