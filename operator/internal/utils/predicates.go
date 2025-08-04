package utils

import (
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// CreateAndDeleteOnlyPredicate is a Predicate which triggers reconciliations only on creation and deletion events.
// We define our own type to avoid the default-true behaviour of the Funcs predicate in case
// new methods are added to the Predicate interface in the future.
type CreateAndDeleteOnlyPredicate[T ctrlClient.Object] struct{}

var _ predicate.Predicate = CreateAndDeleteOnlyPredicate[ctrlClient.Object]{}

// Create returns true.
func (c CreateAndDeleteOnlyPredicate[T]) Create(_ event.TypedCreateEvent[T]) bool {
	return true
}

// Delete returns true.
func (c CreateAndDeleteOnlyPredicate[T]) Delete(_ event.TypedDeleteEvent[T]) bool {
	return true
}

// Update returns false.
func (c CreateAndDeleteOnlyPredicate[T]) Update(_ event.TypedUpdateEvent[T]) bool {
	return false
}

// Generic returns false.
func (c CreateAndDeleteOnlyPredicate[T]) Generic(_ event.TypedGenericEvent[T]) bool {
	return false
}

// ResourceWithNamePredicate triggers reconciliation on Create, Update, and Delete events
// for an object that has a specific name.
type ResourceWithNamePredicate[T ctrlClient.Object] struct {
	Name string
}

var _ predicate.Predicate = (*ResourceWithNamePredicate[ctrlClient.Object])(nil)

func (p *ResourceWithNamePredicate[T]) Create(e event.TypedCreateEvent[T]) bool {
	return e.Object.GetName() == p.Name
}

func (p *ResourceWithNamePredicate[T]) Update(e event.TypedUpdateEvent[T]) bool {
	if e.ObjectNew.GetName() != p.Name {
		return false
	}

	return e.ObjectOld.GetResourceVersion() != e.ObjectNew.GetResourceVersion()
}

func (p *ResourceWithNamePredicate[T]) Delete(e event.TypedDeleteEvent[T]) bool {
	return e.Object.GetName() == p.Name
}

func (p *ResourceWithNamePredicate[T]) Generic(_ event.TypedGenericEvent[T]) bool {
	return false
}
