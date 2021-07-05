package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RemoveOwnerRef removes an owner ref of the given owner object from the given object.
func RemoveOwnerRef(obj metav1.Object, owner metav1.Object) {
	r := obj.GetOwnerReferences()[:0]
	for _, v := range obj.GetOwnerReferences() {
		if v.UID == owner.GetUID() {
			continue
		}
		r = append(r, v)
	}
	obj.SetOwnerReferences(r)
}
