package status

import (
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/pkg/reflectutils"
)

// SkipStatusControllerUpdates filters events triggered by status controller updates to prevent unnecessary reconciliations.
// It skips reconciliation when status controller owned conditions (Available, Progressing) have changed.
//
// This can be instantiated with either:
//   - A specific CR type (e.g., SkipStatusControllerUpdates[*Central]{}) for typed usage in controller.go.
//   - ctrlClient.Object (e.g., SkipStatusControllerUpdates[ctrlClient.Object]{}) for untyped usage as predicate.Predicate in reconciler.go.
type SkipStatusControllerUpdates[T ctrlClient.Object] struct {
	predicate.TypedFuncs[T]
}

// Update implements predicate.TypedPredicate.
// Returns true to allow the update event to trigger reconciliation, false to skip it.
func (p SkipStatusControllerUpdates[T]) Update(e event.TypedUpdateEvent[T]) bool {
	// Check for nil using reflection to handle the interface nil gotcha.
	if reflectutils.IsNil(e.ObjectOld) || reflectutils.IsNil(e.ObjectNew) {
		// One of the objects is nil, allow reconciliation.
		return true
	}

	// Type assert to ObjectForStatusController to access GetCondition method.
	// This allows the same struct to work with both specific types (for controller.go)
	// and ctrlClient.Object (for reconciler.go as predicate.Predicate).
	objOldT, ok := any(e.ObjectOld).(platform.ObjectForStatusController)
	if !ok {
		// Not an ObjectForStatusController, allow reconciliation.
		return true
	}
	objNewT, ok := any(e.ObjectNew).(platform.ObjectForStatusController)
	if !ok {
		// Not an ObjectForStatusController, allow reconciliation.
		return true
	}

	statusControllerConditionsChanged :=
		conditionsChanged(objOldT, objNewT, platform.ConditionProgressing) ||
			conditionsChanged(objOldT, objNewT, platform.ConditionAvailable)

	return !statusControllerConditionsChanged
}

// conditionsChanged returns true if the conditions are different.
func conditionsChanged[T platform.ObjectForStatusController](oldObj, newObj T, condType platform.ConditionType) bool {
	old := oldObj.GetCondition(condType)
	new := newObj.GetCondition(condType)

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
