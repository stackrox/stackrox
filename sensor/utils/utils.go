package utils

import (
	"github.com/pkg/errors"
	commonLabels "github.com/stackrox/rox/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

const (
	KubernetesLabelManagedBy  = "app.kubernetes.io/managed-by"
	KubernetesLabelCreatedBy  = "app.kubernetes.io/created-by"
	KubernetesLabelName       = "app.kubernetes.io/name"
	KubernetesOwnerAnnotation = "owner"
)

// GetSensorKubernetesLabels returns the default labels for resources created by the sensor.
func GetSensorKubernetesLabels() map[string]string {
	return map[string]string{
		KubernetesLabelManagedBy: "sensor",
		KubernetesLabelCreatedBy: "sensor",
		KubernetesLabelName:      "stackrox",
	}
}

func GetTLSSecretLabels() map[string]string {
	labels := GetSensorKubernetesLabels()
	labels["rhacs.redhat.com/tls"] = "true"
	// Add the StackRox managed-by label so Operator can watch these secrets for CA rotation
	labels[commonLabels.ManagedByLabelKey] = commonLabels.ManagedBySensor
	return labels
}

// GetSensorKubernetesAnnotations returns the default annotations for resources created by the sensor.
func GetSensorKubernetesAnnotations() map[string]string {
	return map[string]string{
		KubernetesOwnerAnnotation: "stackrox",
	}
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
