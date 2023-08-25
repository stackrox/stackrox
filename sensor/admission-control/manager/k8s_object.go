package manager

import (
	openshift_appsv1 "github.com/openshift/api/apps/v1"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/kubernetes"
	apps "k8s.io/api/apps/v1"
	batchV1 "k8s.io/api/batch/v1"
	batchV1beta1 "k8s.io/api/batch/v1beta1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
	universalDeserializer = serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()
)

func unmarshalK8sObject(gvk metav1.GroupVersionKind, raw []byte) (k8sutil.Object, error) {
	var obj k8sutil.Object
	switch gvk.Kind {
	case kubernetes.Pod:
		obj = &core.Pod{}
	case kubernetes.Deployment:
		obj = &apps.Deployment{}
	case kubernetes.StatefulSet:
		obj = &apps.StatefulSet{}
	case kubernetes.DaemonSet:
		obj = &apps.DaemonSet{}
	case kubernetes.ReplicationController:
		obj = &core.ReplicationController{}
	case kubernetes.ReplicaSet:
		obj = &apps.ReplicaSet{}
	case kubernetes.CronJob:
		if gvk.Version == "v1beta1" {
			obj = &batchV1beta1.CronJob{}
		} else {
			obj = &batchV1.CronJob{}
		}
	case kubernetes.Job:
		obj = &batchV1.Job{}
	case kubernetes.DeploymentConfig:
		obj = &openshift_appsv1.DeploymentConfig{}
	default:
		return nil, errors.Errorf("currently do not recognize kind %q in admission controller", gvk.Kind)
	}

	if _, _, err := universalDeserializer.Decode(raw, nil, obj); err != nil {
		return nil, err
	}

	return obj, nil
}
