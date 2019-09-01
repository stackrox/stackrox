package kubernetes

import (
	"github.com/stackrox/rox/pkg/stringutils"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	maxValueLen = 256
)

var (
	annotationKeys = []string{
		"kubectl.kubernetes.io/last-applied-configuration",
		"deployment.kubernetes.io/revision",
	}
)

// TrimAnnotations removes the kubectl apply annotation
func TrimAnnotations(object v1.Object) {
	annotations := object.GetAnnotations()
	for _, key := range annotationKeys {
		delete(annotations, key)
	}
	for k, v := range annotations {
		annotations[k] = stringutils.Truncate(v, maxValueLen, stringutils.WordOriented{})
	}
}
