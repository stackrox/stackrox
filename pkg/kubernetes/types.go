package kubernetes

import (
	"github.com/stackrox/rox/pkg/set"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Supported kubernetes resource types.
var (
	Deployment            = newDeploymentResource(`Deployment`)
	DaemonSet             = newDeploymentResource(`DaemonSet`)
	Pod                   = newDeploymentResource(`Pod`)
	ReplicationController = newDeploymentResource(`ReplicationController`)
	ReplicaSet            = newDeploymentResource(`ReplicaSet`)
	StatefulSet           = newDeploymentResource(`StatefulSet`)
	CronJob               = newDeploymentResource(`CronJob`)
	Job                   = newDeploymentResource(`Job`)

	Service = `Service`

	// OpenShift specific
	DeploymentConfig = newDeploymentResource(`DeploymentConfig`)
)

// Kubernetes delete options that ensure that dependent objects (e.g. pods) are deleted when
// the owning resource is deleted.
var (
	DeletePolicy = metav1.DeletePropagationForeground
	DeleteOption = &metav1.DeleteOptions{PropagationPolicy: &DeletePolicy}

	ScaleToZeroSpec = v1beta1.ScaleSpec{
		Replicas: 0,
	}

	deploymentResourceSet = set.NewStringSet()
)

func newDeploymentResource(s string) string {
	deploymentResourceSet.Add(s)
	return s
}

// IsDeploymentResource will return true if the passed string is a type we can convert to our concept of Deployment
func IsDeploymentResource(s string) bool {
	return deploymentResourceSet.Contains(s)
}
