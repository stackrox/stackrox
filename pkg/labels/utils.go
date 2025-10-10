package labels

import "github.com/stackrox/rox/generated/storage"

// LabelSelectorRequirement creates *storage.SetBasedLabelSelector_Requirement.
func LabelSelectorRequirement(key string, op storage.SetBasedLabelSelector_Operator, values []string) *storage.SetBasedLabelSelector_Requirement {
	return storage.SetBasedLabelSelector_Requirement_builder{
		Key:    &key,
		Op:     &op,
		Values: values,
	}.Build()
}

// LabelSelector creates *storage.SetBasedLabelSelector.
func LabelSelector(key string, op storage.SetBasedLabelSelector_Operator, values []string) *storage.SetBasedLabelSelector {
	return storage.SetBasedLabelSelector_builder{
		Requirements: []*storage.SetBasedLabelSelector_Requirement{
			LabelSelectorRequirement(key, op, values),
		},
	}.Build()
}

// LabelSelectors creates []*storage.SetBasedLabelSelector.
func LabelSelectors(key string, op storage.SetBasedLabelSelector_Operator, values []string) []*storage.SetBasedLabelSelector {
	return []*storage.SetBasedLabelSelector{
		LabelSelector(key, op, values),
	}
}
