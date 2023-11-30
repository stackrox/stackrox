package kubernetes

import (
	"github.com/stackrox/rox/pkg/set"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	maxValueLen = 256
)

var (
	annotationKeys = set.NewFrozenStringSet(
		"kubectl.kubernetes.io/last-applied-configuration",
		"deployment.kubernetes.io/revision",
		"k8s.ovn.org/pod-networks",
		"k8s.v1.cni.cncf.io/network-status",
		"k8s.v1.cni.cncf.io/networks-status",
		"operator.tekton.dev/last-applied-hash",
	)
)

// TrimAnnotations removes the kubectl apply annotation.
// Note: this function is intentionally written in an idempotent fashion, in the sense that there are no writes
// whatsoever to the map upon subsequent invocations (at least if the map is not modified between invocations).
// This is required since Kubernetes reuses the actual object when resyncing, hence this function might be invoked
// while the annotations are being read concurrently.
func TrimAnnotations(object v1.Object) {
	//annotations := object.GetAnnotations()
	//for k, v := range annotations {
	//	if annotationKeys.Contains(k) {
	//		delete(annotations, k)
	//		continue
	//	}
	//	if len(v) > maxValueLen {
	//		annotations[k] = stringutils.Truncate(v, maxValueLen, stringutils.WordOriented{})
	//	}
	//}
}
