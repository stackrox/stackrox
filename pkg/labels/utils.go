package labels

import "github.com/stackrox/rox/generated/storage"

// LabelSelectorRequirement creates *storage.SetBasedLabelSelector_Requirement.
func LabelSelectorRequirement(key string, op storage.SetBasedLabelSelector_Operator, values []string) *storage.SetBasedLabelSelector_Requirement {
	return &storage.SetBasedLabelSelector_Requirement{
		Key:    key,
		Op:     op,
		Values: values,
	}
}

// LabelSelector creates *storage.SetBasedLabelSelector.
func LabelSelector(key string, op storage.SetBasedLabelSelector_Operator, values []string) *storage.SetBasedLabelSelector {
	return &storage.SetBasedLabelSelector{
		Requirements: []*storage.SetBasedLabelSelector_Requirement{
			LabelSelectorRequirement(key, op, values),
		},
	}
}

// LabelSelectors creates []*storage.SetBasedLabelSelector.
func LabelSelectors(key string, op storage.SetBasedLabelSelector_Operator, values []string) []*storage.SetBasedLabelSelector {
	return []*storage.SetBasedLabelSelector{
		LabelSelector(key, op, values),
	}
}
