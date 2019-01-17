package enforcers

const (
	enforcementBypassAnnotationKey = "admission.stackrox.io/break-glass"
)

// ShouldEnforce takes in an annotationGetter and returns a bool whether or not enforcement should be taken
func ShouldEnforce(annotations map[string]string) bool {
	_, ok := annotations[enforcementBypassAnnotationKey]
	return !ok
}
