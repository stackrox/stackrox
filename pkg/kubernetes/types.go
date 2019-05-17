package kubernetes

import (
	"github.com/stackrox/rox/pkg/set"
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
	CronJob               = `CronJob`
	Job                   = `Job`

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

	deploymentResourceSet = set.NewFrozenStringSet(
		Deployment,
		DaemonSet,
		Pod,
		ReplicaSet,
		ReplicationController,
		StatefulSet,
		CronJob,
		Job,
		DeploymentConfig,
	)
)

// IsDeploymentResource will return true if the passed string is a type we can convert to our concept of Deployment
func IsDeploymentResource(s string) bool {
	return deploymentResourceSet.Contains(s)
}
