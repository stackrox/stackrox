package labels

import "github.com/stackrox/rox/generated/storage"

// LabelSelectorRequirement creates *storage.SetBasedLabelSelector_Requirement.
func LabelSelectorRequirement(key string, op storage.SetBasedLabelSelector_Operator, values []string) *storage.SetBasedLabelSelector_Requirement {
	sr := &storage.SetBasedLabelSelector_Requirement{}
	sr.SetKey(key)
	sr.SetOp(op)
	sr.SetValues(values)
	return sr
}

// LabelSelector creates *storage.SetBasedLabelSelector.
func LabelSelector(key string, op storage.SetBasedLabelSelector_Operator, values []string) *storage.SetBasedLabelSelector {
	sbls := &storage.SetBasedLabelSelector{}
	sbls.SetRequirements([]*storage.SetBasedLabelSelector_Requirement{
		LabelSelectorRequirement(key, op, values),
	})
	return sbls
}

// LabelSelectors creates []*storage.SetBasedLabelSelector.
func LabelSelectors(key string, op storage.SetBasedLabelSelector_Operator, values []string) []*storage.SetBasedLabelSelector {
	return []*storage.SetBasedLabelSelector{
		LabelSelector(key, op, values),
	}
}
