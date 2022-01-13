package utils

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// CreateAndDeleteOnlyPredicate is a Predicate which triggers reconciliations only on creation and deletion events.
// We define our own type to avoid the default-true behaviour of the Funcs predicate in case
// new methods are added to the Predicate interface in the future.
type CreateAndDeleteOnlyPredicate struct{}

var _ predicate.Predicate = CreateAndDeleteOnlyPredicate{}

// Create returns true.
func (c CreateAndDeleteOnlyPredicate) Create(_ event.CreateEvent) bool {
	return true
}

// Delete returns true.
func (c CreateAndDeleteOnlyPredicate) Delete(_ event.DeleteEvent) bool {
	return true
}

// Update returns false.
func (c CreateAndDeleteOnlyPredicate) Update(_ event.UpdateEvent) bool {
	return false
}

// Generic returns false.
func (c CreateAndDeleteOnlyPredicate) Generic(_ event.GenericEvent) bool {
	return false
}
