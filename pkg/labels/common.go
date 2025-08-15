package labels

const (
	// ManagedByLabelKey is the StackRox-specific managed-by label key.
	// This is separate from the Kubernetes standard app.kubernetes.io/managed-by
	// and is used for operator caching and resource management.
	ManagedByLabelKey = "app.stackrox.io/managed-by"

	// ManagedByOperator indicates a resource is managed by the Operator.
	ManagedByOperator = "operator"

	// ManagedBySensor indicates a resource is managed by the Sensor.
	ManagedBySensor = "sensor"
)
