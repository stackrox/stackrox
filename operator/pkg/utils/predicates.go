package utils

import (
	"github.com/cloudflare/cfssl/log"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
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

// PauseReconcileAnnotationPredicate is a Predicate which stops reconcilliation for update
// events, if the object has a specific annotation set.
type PauseReconcileAnnotationPredicate struct{}

var _ predicate.Predicate = PauseReconcileAnnotationPredicate{}

func hasPauseReconcileAnnotation(obj ctrlClient.Object) bool {
	if v, ok := obj.GetAnnotations()["stackrox.io/pause-reconcile"]; ok {
		if v == "true" {
			log.Info("Object has 'pause-reconcile' annotation")
			return true
		}
	}

	return false
}

// Create returns true if no pause reconcile annotation is found.
func (c PauseReconcileAnnotationPredicate) Create(e event.CreateEvent) bool {
	if e.Object == nil {
		log.Error(nil, "Update event has no object to create", "event", e)
		return false
	}
	return !hasPauseReconcileAnnotation(e.Object)
}

// Delete returns true if no pause reconcile annotation is found.
func (c PauseReconcileAnnotationPredicate) Delete(e event.DeleteEvent) bool {
	if e.Object == nil {
		log.Error(nil, "Update event has no object to delete", "event", e)
		return false
	}
	return !hasPauseReconcileAnnotation(e.Object)
}

// Update returns true if no pause reconcile annotation is found.
func (c PauseReconcileAnnotationPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil {
		log.Error(nil, "Update event has no old object to update", "event", e)
		return false
	}

	return !hasPauseReconcileAnnotation(e.ObjectOld)
}

// Generic returns true if no pause reconcile annotation is found.
func (c PauseReconcileAnnotationPredicate) Generic(e event.GenericEvent) bool {
	if e.Object == nil {
		log.Error(nil, "Generic event has no object", "event", e)
		return false
	}
	return !hasPauseReconcileAnnotation(e.Object)
}
