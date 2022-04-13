package common

import (
	"github.com/stackrox/stackrox/pkg/k8sutil/k8sobjects"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// SharedObjects are objects shared with other resource bundles (i.e., central). Not creating these objects is
	// okay - Central takes precedence here.
	SharedObjects = []k8sobjects.ObjectRef{
		{
			GVK: schema.GroupVersionKind{
				Version: "v1",
				Kind:    "Secret",
			},
			Namespace: Namespace,
			Name:      "monitoring-client",
		},
		{
			GVK: schema.GroupVersionKind{
				Version: "v1",
				Kind:    "ConfigMap",
			},
			Namespace: Namespace,
			Name:      "telegraf",
		},
	}
)

// IsSharedObject checks if the given object is a shared object.
func IsSharedObject(objRef k8sobjects.ObjectRef) bool {
	for _, sharedObjRef := range SharedObjects {
		if objRef == sharedObjRef {
			return true
		}
	}
	return false
}
