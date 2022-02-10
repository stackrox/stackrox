package common

import (
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/k8sutil/k8sobjects"
)

// K8sObjectPredicateFunc represents a predicate function which takes in a k8s object.
type K8sObjectPredicateFunc func(object k8sutil.Object) bool

// Filter modifies the given slice to remove any elements which do not pass the predicate.
func Filter(objects *[]k8sutil.Object, predicate K8sObjectPredicateFunc) {
	if objects == nil {
		return
	}

	filtered := (*objects)[:0]
	for _, obj := range *objects {
		if predicate(obj) {
			filtered = append(filtered, obj)
		}
	}
	*objects = filtered
}

// Not returns a predicate which negates the output of the given predicate.
func Not(predicate K8sObjectPredicateFunc) K8sObjectPredicateFunc {
	return func(object k8sutil.Object) bool {
		return !predicate(object)
	}
}

// All returns a predicate which returns true if all given predicates return
// true.
func All(predicates ...K8sObjectPredicateFunc) K8sObjectPredicateFunc {
	return func(object k8sutil.Object) bool {
		for _, predicate := range predicates {
			if !predicate(object) {
				return false
			}
		}
		return true
	}
}

// CertObjectPredicate takes the given obj, and returns `true` if the object corresponds to a cert.
func CertObjectPredicate(obj k8sutil.Object) bool {
	_, exists := image.SensorCertObjectRefs[k8sobjects.RefOf(obj)]
	return exists
}

// AdditionalCASecretPredicate takes the given obj, and returns `true`
// if the object corresponds to an additional ca secret.
func AdditionalCASecretPredicate(obj k8sutil.Object) bool {
	return k8sobjects.RefOf(obj) == image.AdditionalCASensorSecretRef
}

// InjectedCABundleConfigMapPredicate takes the given obj, and returns `true`
// if the object corresponds to an additional ca secret.
func InjectedCABundleConfigMapPredicate(obj k8sutil.Object) bool {
	return k8sobjects.RefOf(obj) == image.InjectedCABundleConfigMapRef
}
