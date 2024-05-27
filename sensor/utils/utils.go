package utils

import (
	"k8s.io/client-go/kubernetes"
)

const (
	KubernetesLabelManagedBy = "app.kubernetes.io/managed-by"
	KubernetesLabelPartOf    = "app.kubernetes.io/created-by-deployment-id"
	KubernetesLabelCreatedBy = "app.kubernetes.io/created-by"
	KubernetesLabelName      = "app.kubernetes.io/name"
)

// GetSensorKubernetesLabels returns the default labels for resources created by the sensor.
func GetSensorKubernetesLabels() map[string]string {
	return map[string]string{
		KubernetesLabelManagedBy: "sensor",
		KubernetesLabelCreatedBy: "sensor",
		KubernetesLabelName:      "stackrox",
	}
}

// GetSensorKubernetesAnnotations returns the default annotations for resources created by the sensor.
func GetSensorKubernetesAnnotations() map[string]string {
	return map[string]string{
		"owner": "stackrox",
	}
}

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
