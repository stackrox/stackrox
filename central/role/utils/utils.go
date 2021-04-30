package utils

import (
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
)

const (
	// AccessScopeIDPrefix should be prepended to every human-hostile ID of an
	// access scope for readability, e.g.,
	//     "acs.authz.accessscope.94ac7bfe-f9b2-402e-b4f2-bfda480e1a13".
	AccessScopeIDPrefix = "acs.authz.accessscope."
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

// ValidateSimpleAccessScope checks whether the supplied protobuf message is a
// valid simple access scope.
func ValidateSimpleAccessScope(scope *storage.SimpleAccessScope) error {
	var multiErr error

	if !strings.HasPrefix(scope.GetId(), AccessScopeIDPrefix) {
		multiErr = multierror.Append(multiErr, errors.Errorf("id field must be in '%s*' format", AccessScopeIDPrefix))
	}
	if scope.GetName() == "" {
		multiErr = multierror.Append(multiErr, errors.New("name field must be set"))
	}
	for _, ns := range scope.GetRules().GetIncludedNamespaces() {
		if ns.GetClusterName() == "" || ns.GetNamespaceName() == "" {
			multiErr = multierror.Append(multiErr, errors.Errorf(
				"both cluster_name and namespace_name fields must be set in namespace rule <%s, %s>",
				ns.GetClusterName(), ns.GetNamespaceName()))
		}
	}
	for _, labelSelector := range scope.GetRules().GetClusterLabelSelectors() {
		if len(labelSelector.GetRequirements()) == 0 {
			multiErr = multierror.Append(multiErr, errors.New(
				"requirements field must be set in every cluster label selector"))
			break
		}
	}
	for _, labelSelector := range scope.GetRules().GetNamespaceLabelSelectors() {
		if len(labelSelector.GetRequirements()) == 0 {
			multiErr = multierror.Append(multiErr, errors.New(
				"requirements field must be set in every namespace label selector"))
			break
		}
	}

	return multiErr
}
