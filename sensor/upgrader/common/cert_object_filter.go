package common

import (
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/k8sutil/k8sobjects"
)

// FilterToOnlyCertObjects takes the given slice, and returns a filtered slice containing only
// the objects that correspond to certs.
func FilterToOnlyCertObjects(objects []k8sutil.Object) []k8sutil.Object {
	var filtered []k8sutil.Object
	for _, obj := range objects {
		if _, exists := image.SensorCertObjectRefs[k8sobjects.RefOf(obj)]; exists {
			filtered = append(filtered, obj)
		}
	}
	return filtered
}
