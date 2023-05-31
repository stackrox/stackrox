package accessscope

import (
	"github.com/hashicorp/go-multierror"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"k8s.io/apimachinery/pkg/labels"
)

// ValidateSimpleAccessScopeProto checks whether the supplied protobuf message
// represents valid simple access scope.
func ValidateSimpleAccessScopeProto(scope *storage.SimpleAccessScope) error {
	var validationErrs *multierror.Error

	if scope.GetName() == "" {
		validationErrs = multierror.Append(validationErrs, errox.InvalidArgs.New("name field must be set"))
	}

	if err := ValidateSimpleAccessScopeRules(scope.GetRules()); err != nil {
		validationErrs = multierror.Append(validationErrs, err)
	}

	return validationErrs.ErrorOrNil()
}

// ValidateSimpleAccessScopeRules checks whether the supplied protobuf message
// represents valid simple access scope rule.
func ValidateSimpleAccessScopeRules(rules *storage.SimpleAccessScope_Rules) error {
	var validationErrs *multierror.Error
	if rules == nil {
		validationErrs = multierror.Append(validationErrs, errox.InvalidArgs.New("rules field must be set"))
	}
	for _, ns := range rules.GetIncludedNamespaces() {
		if ns.GetClusterName() == "" || ns.GetNamespaceName() == "" {
			validationErrs = multierror.Append(validationErrs, errox.InvalidArgs.Newf(
				"both cluster_name and namespace_name fields must be set in namespace rule <%s, %s>",
				ns.GetClusterName(), ns.GetNamespaceName()))
		}
	}
	for _, labelSelector := range rules.GetClusterLabelSelectors() {
		err := validateSelectorRequirement(labelSelector)
		if err != nil {
			validationErrs = multierror.Append(validationErrs, err)
		}
	}
	for _, labelSelector := range rules.GetNamespaceLabelSelectors() {
		err := validateSelectorRequirement(labelSelector)
		if err != nil {
			validationErrs = multierror.Append(validationErrs, err)
		}
	}
	return validationErrs.ErrorOrNil()
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
