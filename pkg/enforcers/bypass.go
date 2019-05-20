package enforcers

const (
	// EnforcementBypassAnnotationKey is the key used to bypass enforcement in case of critical deployment
	EnforcementBypassAnnotationKey = "admission.stackrox.io/break-glass"
)

// ShouldEnforce takes in an annotationGetter and returns a bool whether or not enforcement should be taken
func ShouldEnforce(annotations map[string]string) bool {
	_, ok := annotations[EnforcementBypassAnnotationKey]
	return !ok
}
