package common

import (
	"github.com/stackrox/rox/pkg/k8sutil/k8sobjects"
	"github.com/stackrox/rox/pkg/pods"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	sensorNamespace = pods.GetPodNamespace()
	// SharedObjects are objects shared with other resource bundles (i.e., central). Not creating these objects is
	// okay - Central takes precedence here.
	SharedObjects = []k8sobjects.ObjectRef{
		{
			GVK: schema.GroupVersionKind{
				Version: "v1",
				Kind:    "Secret",
			},
			Namespace: sensorNamespace,
			Name:      "monitoring-client",
		},
		{
			GVK: schema.GroupVersionKind{
				Version: "v1",
				Kind:    "ConfigMap",
			},
			Namespace: sensorNamespace,
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
