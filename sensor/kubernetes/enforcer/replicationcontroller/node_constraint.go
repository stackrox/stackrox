package replicationcontroller

import (
	"context"

	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/pkg/retry"
	"github.com/stackrox/stackrox/sensor/kubernetes/enforcer/common"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// EnforceNodeConstraint reschedules the ReplicationController with unsatisfiable constraints.
func EnforceNodeConstraint(ctx context.Context, client kubernetes.Interface, deploymentInfo *central.DeploymentEnforcement) (err error) {
	// Load the current ReplicationController for the deployment.
	var rc *coreV1.ReplicationController
	rc, err = client.CoreV1().ReplicationControllers(deploymentInfo.GetNamespace()).Get(ctx, deploymentInfo.GetDeploymentName(), metav1.GetOptions{})
	if err != nil {
		return retry.MakeRetryable(err)
	}

	// Apply the constraint modification.
	err = common.ApplyNodeConstraintToObj(rc, deploymentInfo.GetAlertId())
	if err != nil {
		return
	}

	// Post the new ReplicationController data.
	_, err = client.CoreV1().ReplicationControllers(deploymentInfo.GetNamespace()).Update(ctx, rc, metav1.UpdateOptions{})
	if err != nil {
		return retry.MakeRetryable(err)
	}
	return
}
