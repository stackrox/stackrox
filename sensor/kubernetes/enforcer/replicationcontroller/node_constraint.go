package replicationcontroller

import (
	roxV1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/sensor/kubernetes/enforcer/common"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// EnforceNodeConstraint reschedules the ReplicationController with unsatisfiable constraints.
func EnforceNodeConstraint(client *kubernetes.Clientset, deploymentInfo *roxV1.DeploymentEnforcement) (err error) {
	// Load the current ReplicationController for the deployment.
	var rc *coreV1.ReplicationController
	rc, err = client.CoreV1().ReplicationControllers(deploymentInfo.GetNamespace()).Get(deploymentInfo.GetDeploymentName(), metav1.GetOptions{})
	if err != nil {
		return retry.MakeRetryable(err)
	}

	// Apply the constraint modification.
	common.ApplyNodeConstraintToObj(rc, deploymentInfo.GetAlertId())

	// Post the new ReplicationController data.
	_, err = client.CoreV1().ReplicationControllers(deploymentInfo.GetNamespace()).Update(rc)
	if err != nil {
		return retry.MakeRetryable(err)
	}
	return
}
