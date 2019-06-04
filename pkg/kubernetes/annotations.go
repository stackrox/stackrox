package kubernetes

import (
	"github.com/stackrox/rox/pkg/stringutils"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	kubectlAppliedAnnotationKey = "kubectl.kubernetes.io/last-applied-configuration"

	maxValueLen = 256
)

// RemoveAppliedAnnotation removes the kubectl apply annotation
func RemoveAppliedAnnotation(object v1.Object) {
	annotations := object.GetAnnotations()
	delete(annotations, kubectlAppliedAnnotationKey)
	for k, v := range annotations {
		annotations[k] = stringutils.Truncate(v, maxValueLen, stringutils.WordOriented{})
	}
}
