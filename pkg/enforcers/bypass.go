package enforcers

const (
	// EnforcementBypassAnnotationKey is the key used to bypass enforcement in case of critical deployment
	EnforcementBypassAnnotationKey = "admission.stackrox.io/break-glass"

	// KubeEventEnforcementBypassAnnotationKey is the key used to bypass enforcement on kubernetes events.
	KubeEventEnforcementBypassAnnotationKey = "admission.stackrox.io/bypass-kube-event-enforcement"
)

// ShouldEnforce takes in annotations and returns a bool whether or not enforcement should be taken
func ShouldEnforce(annotations map[string]string) bool {
	_, ok := annotations[EnforcementBypassAnnotationKey]
	return !ok
}

// ShouldEnforceOnKubeEvent takes in annotations and returns a bool whether or not fail kube event enforcement should be performed.
func ShouldEnforceOnKubeEvent(annotations map[string]string) bool {
	_, ok := annotations[KubeEventEnforcementBypassAnnotationKey]
	return !ok
}
