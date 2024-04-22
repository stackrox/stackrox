package utils

import (
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	log = logging.LoggerForModule()
)

// ServerResourcesForGroup retrieves the APIResourceList of the given group.
func ServerResourcesForGroup(client client.Interface, group string) (*metav1.APIResourceList, error) {
	resourceList, err := client.Kubernetes().Discovery().ServerResourcesForGroupVersion(group)
	return resourceList, err
}

// ResourceExists returns true if resource exists in list.  Use with output from
// `ServerResourcesForGroup` to verify a resource exists prior to starting an
// Informer to prevent client-go from spamming the k8s API and logs.
func ResourceExists(list *metav1.APIResourceList, resource string, group string) bool {
	for _, apiResource := range list.APIResources {
		if apiResource.Name == resource {
			return true
		}
	}

	log.Warnf("Resource %q does not exist in the group %s", resource, group)
	return false
}
