package utils

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/sensor/common/annotations"
	"k8s.io/client-go/kubernetes"
)

// GetSensorKubernetesLabels returns the default labels for resources created by the sensor.
func GetSensorKubernetesLabels() map[string]string {
	return annotations.SensorK8sLabels
}

func GetTLSSecretLabels() map[string]string {
	labels := GetSensorKubernetesLabels()
	labels["rhacs.redhat.com/tls"] = "true"
	return labels
}

// GetSensorKubernetesAnnotations returns the default annotations for resources created by the sensor.
func GetSensorKubernetesAnnotations() map[string]string {
	return annotations.SensorK8sAnnotations
}

// HasAPI checks whether the kubernetes server supports the groupVersion API for the specified kind
func HasAPI(client kubernetes.Interface, groupVersion, kind string) (bool, error) {
	apiResourceList, err := client.Discovery().ServerResourcesForGroupVersion(groupVersion)
	if err != nil {
		return false, errors.Wrap(err, "checking API support for groupVersion "+groupVersion)
	}
	for _, apiResource := range apiResourceList.APIResources {
		if apiResource.Kind == kind {
			return true, nil
		}
	}
	return false, nil
}
