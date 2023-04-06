package role

import (
	"github.com/hashicorp/go-multierror"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"k8s.io/apimachinery/pkg/labels"
)

// ValidateSimpleAccessScopeRules checks whether the supplied protobuf message
// represents valid simple access scope rules.
func ValidateSimpleAccessScopeRules(scopeRules *storage.SimpleAccessScope_Rules) error {
	var multiErr error

	for _, ns := range scopeRules.GetIncludedNamespaces() {
		if ns.GetClusterName() == "" || ns.GetNamespaceName() == "" {
			multiErr = multierror.Append(multiErr, errox.InvalidArgs.Newf(
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
