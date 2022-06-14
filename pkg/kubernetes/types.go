package kubernetes

import (
	"github.com/stackrox/stackrox/pkg/set"
	autoscalingV1 "k8s.io/api/autoscaling/v1"
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
	DeletePolicyForeground = metav1.DeletePropagationForeground
	DeletePolicyBackground = metav1.DeletePropagationBackground
	DeleteOption           = metav1.DeleteOptions{PropagationPolicy: &DeletePolicyForeground}
	DeleteBackgroundOption = metav1.DeleteOptions{PropagationPolicy: &DeletePolicyBackground}

	ScaleToZeroSpec = autoscalingV1.ScaleSpec{
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
