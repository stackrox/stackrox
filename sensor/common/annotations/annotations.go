package annotations

const (
	KubernetesLabelManagedBy  = "app.kubernetes.io/managed-by"
	KubernetesLabelCreatedBy  = "app.kubernetes.io/created-by"
	KubernetesLabelName       = "app.kubernetes.io/name"
	KubernetesOwnerAnnotation = "owner"

	// UpgradeResourceLabelSensorKey is a label key attached to all auto-upgradeable resources.
	// The delete-sensor script relies on this label when cleaning up.
	UpgradeResourceLabelSensorKey = `auto-upgrade.stackrox.io/component`
	// UpgradeResourceLabelSensorValue is the label value for the above key that identifies resources from the sensor bundle.
	UpgradeResourceLabelSensorValue = `sensor`
)

var (
	SensorK8sLabels = map[string]string{
		KubernetesLabelManagedBy: "sensor",
		KubernetesLabelCreatedBy: "sensor",
		KubernetesLabelName:      "stackrox",
	}
	SensorK8sAnnotations = map[string]string{
		KubernetesOwnerAnnotation: "stackrox",
	}
)
