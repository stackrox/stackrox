package common

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	platform "github.com/stackrox/rox/operator/api/v1alpha1"
)

// CentralStatusPredicate filters events triggered by status controller updates to prevent unnecessary reconciliations.
//
// Logic:
// - If any of the externally managed status conditions changed → block reconciliation (the update is coming from status controller).
// - If none of the externally managed status conditions changed → allow reconciliation (the update is not coming from the status controller).
type TypedCentralStatusPredicate struct {
	predicate.TypedFuncs[*platform.Central]
}

type CentralStatusPredicate struct {
	predicate.Funcs
	typedPredicate TypedCentralStatusPredicate
}

var (
	// Conditions owned by the status controller.
	statusControllerOwnedConditions = []platform.ConditionType{
		"Available",
		"Progressing",
	}
)

func (p CentralStatusPredicate) Update(e event.UpdateEvent) bool {
	oldCentral, ok := e.ObjectOld.(*platform.Central)
	if !ok {
		// Unable to cast old object → allow reconciliation.
		return true
	}
	newCentral, ok := e.ObjectNew.(*platform.Central)
	if !ok {
		// Unable to cast new object → allow reconciliation.
		return true
	}
	typedEvent := event.TypedUpdateEvent[*platform.Central]{
		ObjectOld: oldCentral,
		ObjectNew: newCentral,
	}
	return p.typedPredicate.Update(typedEvent)
}

// Update implements predicate.TypedPredicate.
// Returns true to allow the update event to trigger reconciliation, false to block it.
func (TypedCentralStatusPredicate) Update(e event.TypedUpdateEvent[*platform.Central]) bool {
	oldCentral := e.ObjectOld
	newCentral := e.ObjectNew

	if oldCentral == nil || newCentral == nil {
		return true
	}

	// Check if ANY of our owned conditions changed.
	for _, condType := range statusControllerOwnedConditions {
		oldCond := GetCondition(oldCentral.Status.Conditions, condType)
		newCond := GetCondition(newCentral.Status.Conditions, condType)

		if conditionsChanged(oldCond, newCond) {
			// One of the status controller owned conditions changed → block.
			return false
		}
	}

	// All of the status controller owned conditions unchanged → something else changed → allow.
	return true
}

// getCondition finds a condition by type, returns nil if not found.
func GetCondition(conditions []platform.StackRoxCondition, condType platform.ConditionType) *platform.StackRoxCondition {
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
