package kubernetes

import (
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Supported kubernetes resource types.
const (
	Deployment            = `Deployment`
	DaemonSet             = `DaemonSet`
	Pod                   = `Pod`
	ReplicationController = `ReplicationController`
	ReplicaSet            = `ReplicaSet`
	StatefulSet           = `StatefulSet`

	Service = `Service`

	// OpenShift specific
	DeploymentConfig = `DeploymentConfig`
)

// Kubernetes delete options that ensure that dependent objects (e.g. pods) are deleted when
// the owning resource is deleted.
var (
	DeletePolicy = metav1.DeletePropagationForeground
	DeleteOption = &metav1.DeleteOptions{PropagationPolicy: &DeletePolicy}

	ScaleToZeroSpec = v1beta1.ScaleSpec{
		Replicas: 0,
	}
)
