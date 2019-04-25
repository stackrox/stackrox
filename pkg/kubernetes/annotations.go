package kubernetes

const (
	kubectlAppliedAnnotationKey = "kubectl.kubernetes.io/last-applied-configuration"
)

// RemoveAppliedAnnotation removes the kubectl apply annotation
func RemoveAppliedAnnotation(annotations map[string]string) {
	delete(annotations, kubectlAppliedAnnotationKey)
}
