package replicaset

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/sensor/kubernetes/enforcer/common"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// EnforceNodeConstraint reschedules the ReplicaSet with unsatisfiable constraints.
func EnforceNodeConstraint(client *kubernetes.Clientset, deploymentInfo *central.DeploymentEnforcement) (err error) {
	// Load the current ReplicaSet for the deployment.
	var rs *v1beta1.ReplicaSet
	rs, err = client.ExtensionsV1beta1().ReplicaSets(deploymentInfo.GetNamespace()).Get(deploymentInfo.GetDeploymentName(), metav1.GetOptions{})
	if err != nil {
		return retry.MakeRetryable(err)
	}

	// Apply the constraint modification.
	common.ApplyNodeConstraintToObj(rs, deploymentInfo.GetAlertId())

	// Post the new ReplicaSet data.
	_, err = client.ExtensionsV1beta1().ReplicaSets(deploymentInfo.GetNamespace()).Update(rs)
	if err != nil {
		return retry.MakeRetryable(err)
	}
	return
}
