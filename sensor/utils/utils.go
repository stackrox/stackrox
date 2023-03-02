package utils

import (
	"k8s.io/client-go/kubernetes"
)

// HasAPI checks whether the kubernetes server supports the groupVersion API for the specified kind
func HasAPI(client kubernetes.Interface, groupVersion, kind string) (bool, error) {
	apiResourceList, err := client.Discovery().ServerResourcesForGroupVersion(groupVersion)
	if err != nil {
		return false, err
	}
	for _, apiResource := range apiResourceList.APIResources {
		if apiResource.Kind == kind {
			return true, nil
		}
	}
	return false, nil
}
