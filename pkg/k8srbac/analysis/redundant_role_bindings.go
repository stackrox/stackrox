package analysis

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/k8srbac"
)

// MatchingRoleBindings holds RoleBindings that are the same or a subset.
type MatchingRoleBindings struct {
	// RoleBindings that map the same Subjects to the same role.
	Equivalent []*storage.K8SRoleBinding
	// RoleBindings that map a subset of the same Subjects to the same role.
	Shadows []*storage.K8SRoleBinding
}

// getRedundantRoleBindings returns a map from role binding to a list of structs, containing equivalent and occluded
// role bindings.
func getRedundantRoleBindings(roleBindings []*storage.K8SRoleBinding) map[*storage.K8SRoleBinding]*MatchingRoleBindings {
	// Build a subject set for each of the role bindings.
	bindingsToSubjects := make(map[*storage.K8SRoleBinding]k8srbac.SubjectSet, len(roleBindings))
	for _, binding := range roleBindings {
		if !k8srbac.IsDefaultRoleBinding(binding) {
			bindingsToSubjects[binding] = k8srbac.NewSubjectSet(binding.GetSubjects()...)
		}
	}

	// Find all matching role bindings.
	redundants := make(map[*storage.K8SRoleBinding]*MatchingRoleBindings)
	for sourceIndex, source := range roleBindings {
		sourceSet := bindingsToSubjects[source]
		if sourceSet == nil || k8srbac.IsDefaultRoleBinding(source) {
			continue
		}

		for _, target := range roleBindings[sourceIndex+1:] {
			targetSet := bindingsToSubjects[target]
			if targetSet == nil || k8srbac.IsDefaultRoleBinding(target) || target.GetRoleId() != source.GetRoleId() {
				continue
			}

			// Check whether either subject list is the same or a subset of the other.
			sourceContainsTarget := sourceSet.ContainsSet(targetSet)
			if _, hasMatch := redundants[source]; !hasMatch && sourceContainsTarget {
				redundants[source] = &MatchingRoleBindings{}
			}
			targetContainsSource := targetSet.ContainsSet(sourceSet)
			if _, hasMatch := redundants[target]; !hasMatch && targetContainsSource {
				redundants[target] = &MatchingRoleBindings{}
			}

			// If either is a subset, or the are equal, add an entry into the matching role bindings.
			if sourceContainsTarget && targetContainsSource { // Equal sets
				redundants[source].Equivalent = append(redundants[source].Equivalent, target)
				redundants[target].Equivalent = append(redundants[target].Equivalent, source)
			} else if sourceContainsTarget { // Target is a subset.
				redundants[source].Shadows = append(redundants[source].Shadows, target)
			} else if targetContainsSource { // Source is a subset.
				redundants[target].Shadows = append(redundants[target].Shadows, source)
			}
		}
	}
	return redundants
}
