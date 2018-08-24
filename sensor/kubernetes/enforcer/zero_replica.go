package enforcer

import (
	"fmt"
	"time"

	"github.com/stackrox/rox/pkg/enforcers"
	pkgKubernetes "github.com/stackrox/rox/pkg/kubernetes"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (e *enforcerImpl) scaleToZero(enforcement *enforcers.DeploymentEnforcement) (err error) {
	d := enforcement.Deployment
	scaleRequest := &v1beta1.Scale{
		Spec: pkgKubernetes.ScaleToZeroSpec,
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.GetName(),
			Namespace: d.GetNamespace(),
		},
	}

	switch d.GetType() {
	case pkgKubernetes.Deployment:
		_, err = e.client.ExtensionsV1beta1().Deployments(d.GetNamespace()).UpdateScale(d.GetName(), scaleRequest)
	case pkgKubernetes.DaemonSet:
		return fmt.Errorf("scaling to 0 is not supported for %s", pkgKubernetes.DaemonSet)
	case pkgKubernetes.ReplicaSet:
		_, err = e.client.ExtensionsV1beta1().ReplicaSets(d.GetNamespace()).UpdateScale(d.GetName(), scaleRequest)
	case pkgKubernetes.ReplicationController:
		_, err = e.client.CoreV1().ReplicationControllers(d.GetNamespace()).UpdateScale(d.GetName(), scaleRequest)
	case pkgKubernetes.StatefulSet:
		var ss *appsv1beta1.StatefulSet
		var ok bool
		if ss, ok = enforcement.OriginalSpec.(*appsv1beta1.StatefulSet); !ok {
			return fmt.Errorf("original object is not of statefulset type: %+v", enforcement.OriginalSpec)
		}

		const maxRetries = 5

		for i := 0; i < maxRetries; i++ {
			if err = e.scaleStatefulSetToZero(ss); err == nil {
				return nil
			}
			time.Sleep(time.Second)
		}
	default:
		return fmt.Errorf("unknown type %s", enforcement.Deployment.GetType())
	}

	return
}

func (e *enforcerImpl) scaleStatefulSetToZero(ss *appsv1beta1.StatefulSet) (err error) {
	ss.Spec.Replicas = &[]int32{0}[0]
	_, err = e.client.AppsV1beta1().StatefulSets(ss.GetNamespace()).Update(ss)
	return
}
