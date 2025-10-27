package status

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	platform "github.com/stackrox/rox/operator/api/v1alpha1"
)

// CentralStatusPredicate filters events for the status controller to prevent unnecessary reconciliations.
//
// Logic:
// - If any of our owned status fields changed → block reconciliation (the update is coming from this controller).
// - If none of our owned status fields changed → allow reconciliation (the update is not coming from this controller).
type CentralStatusPredicate struct {
	predicate.Funcs
}

// Update implements predicate.Predicate.
func (CentralStatusPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil || e.ObjectNew == nil {
		return false
	}

	oldCentral, okOld := e.ObjectOld.(*platform.Central)
	newCentral, okNew := e.ObjectNew.(*platform.Central)
	if !okOld || !okNew || oldCentral == nil || newCentral == nil {
		// Not a Central CR or nil pointer wrapped in interface - block it.
		return false
	}

	// Conditions owned by this status controller.
	ownedConditions := []platform.ConditionType{
		"Ready",
		"Progressing",
	}

	// Check if ANY of our owned conditions changed.
	for _, condType := range ownedConditions {
		oldCond := getCondition(oldCentral.Status.Conditions, condType)
		newCond := getCondition(newCentral.Status.Conditions, condType)

		if conditionsChanged(oldCond, newCond) {
			// One of our owned conditions changed → this is our own update → block.
			return false
		}
	}

	// All our owned conditions are unchanged → something else changed → allow.
	return true
}

// getCondition finds a condition by type, returns nil if not found.
func getCondition(conditions []platform.StackRoxCondition, condType platform.ConditionType) *platform.StackRoxCondition {
	for i := range conditions {
		if conditions[i].Type == condType {
			return &conditions[i]
		}
	}
	return nil
}

// conditionsChanged returns true if the conditions are different.
func conditionsChanged(old, new *platform.StackRoxCondition) bool {
	// One exists, one doesn't.
	if (old == nil) != (new == nil) {
		return true
	}

	// Both nil.
	if old == nil {
		return false
	}

	// Check if meaningful fields changed.
	return old.Status != new.Status ||
		old.Reason != new.Reason ||
		old.Message != new.Message
}
