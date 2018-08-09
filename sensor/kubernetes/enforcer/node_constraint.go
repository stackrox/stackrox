package enforcer

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/enforcers"
	pkgKubernetes "github.com/stackrox/rox/pkg/kubernetes"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (e *enforcer) unsatisfiableNodeConstraint(enforcement *enforcers.DeploymentEnforcement) (err error) {
	const maxRetries = 5
	obj := enforcement.OriginalSpec

	for i := 0; i < maxRetries; i++ {
		applyNodeConstraintToObj(obj, enforcement.AlertID)
		if err = e.updateResourceWithConstraint(enforcement.Deployment, obj); err == nil {
			return nil
		}

		logger.Errorf("unable to update k8s resource: %s. Retrying...", err)

		time.Sleep(time.Second)

		if newObj, err := e.getLatestResource(enforcement.Deployment); err == nil {
			obj = newObj
		} else {
			logger.Errorf("unable to get latest k8s resource object: %s", err)
		}
	}

	return
}

func applyNodeConstraintToObj(obj interface{}, alertID string) (err error) {
	objValue := reflect.Indirect(reflect.ValueOf(obj))
	specValue := objValue.FieldByName("Spec")
	templateValue := reflect.Indirect(specValue.FieldByName("Template"))
	podSpecValue := templateValue.FieldByName("Spec")
	nodeSelector := podSpecValue.FieldByName("NodeSelector")

	if nodeSelector.Kind() != reflect.Map {
		return errors.New("unable to extract node selector from kubernetes object")
	}

	if nodeSelector.IsNil() {
		nodeSelector.Set(reflect.MakeMap(nodeSelector.Type()))
	}
	nodeSelector.SetMapIndex(reflect.ValueOf(enforcers.UnsatisfiableNodeConstraintKey), reflect.ValueOf(alertID))

	return
}

func (e *enforcer) updateResourceWithConstraint(deployment *v1.Deployment, obj interface{}) (err error) {
	var ok bool

	switch deployment.GetType() {
	case pkgKubernetes.Deployment:
		var d *v1beta1.Deployment
		if d, ok = obj.(*v1beta1.Deployment); !ok {
			return fmt.Errorf("object is not of deployment type: %+v", obj)
		}
		_, err = e.client.ExtensionsV1beta1().Deployments(deployment.GetNamespace()).Update(d)
	case pkgKubernetes.DaemonSet:
		var d *v1beta1.DaemonSet
		if d, ok = obj.(*v1beta1.DaemonSet); !ok {
			return fmt.Errorf("object is not of daemonset type: %+v", obj)
		}
		_, err = e.client.ExtensionsV1beta1().DaemonSets(deployment.GetNamespace()).Update(d)
	case pkgKubernetes.ReplicaSet:
		var r *v1beta1.ReplicaSet
		if r, ok = obj.(*v1beta1.ReplicaSet); !ok {
			return fmt.Errorf("object is not of replicaset type: %+v", obj)
		}
		_, err = e.client.ExtensionsV1beta1().ReplicaSets(deployment.GetNamespace()).Update(r)
	case pkgKubernetes.ReplicationController:
		var r *corev1.ReplicationController
		if r, ok = obj.(*corev1.ReplicationController); !ok {
			return fmt.Errorf("object is not of replicationcontroller type: %+v", obj)
		}
		_, err = e.client.CoreV1().ReplicationControllers(deployment.GetNamespace()).Update(r)
	case pkgKubernetes.StatefulSet:
		var ss *appsv1beta1.StatefulSet
		if ss, ok = obj.(*appsv1beta1.StatefulSet); !ok {
			return fmt.Errorf("object is not of statefulset type: %+v", obj)
		}
		_, err = e.client.AppsV1beta1().StatefulSets(deployment.GetNamespace()).Update(ss)
	default:
		return fmt.Errorf("unknown type %s", deployment.GetType())
	}

	return
}

func (e *enforcer) getLatestResource(deployment *v1.Deployment) (obj interface{}, err error) {
	switch deployment.GetType() {
	case pkgKubernetes.Deployment:
		obj, err = e.client.ExtensionsV1beta1().Deployments(deployment.GetNamespace()).Get(deployment.GetName(), metav1.GetOptions{})
	case pkgKubernetes.DaemonSet:
		obj, err = e.client.ExtensionsV1beta1().DaemonSets(deployment.GetNamespace()).Get(deployment.GetName(), metav1.GetOptions{})
	case pkgKubernetes.ReplicaSet:
		obj, err = e.client.ExtensionsV1beta1().ReplicaSets(deployment.GetNamespace()).Get(deployment.GetName(), metav1.GetOptions{})
	case pkgKubernetes.ReplicationController:
		obj, err = e.client.CoreV1().ReplicationControllers(deployment.GetNamespace()).Get(deployment.GetName(), metav1.GetOptions{})
	case pkgKubernetes.StatefulSet:
		obj, err = e.client.AppsV1beta1().StatefulSets(deployment.GetNamespace()).Get(deployment.GetName(), metav1.GetOptions{})
	}

	return
}
