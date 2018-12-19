package deployment

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/sensor/kubernetes/enforcer/common"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// EnforceNodeConstraint reschedules the Deployment with unsatisfiable constraints.
func EnforceNodeConstraint(client *kubernetes.Clientset, deploymentInfo *central.DeploymentEnforcement) (err error) {
	// Load the current Deployment.
	var d *v1beta1.Deployment
	d, err = client.ExtensionsV1beta1().Deployments(deploymentInfo.GetNamespace()).Get(deploymentInfo.GetDeploymentName(), metav1.GetOptions{})
	if err != nil {
		return retry.MakeRetryable(err)
	}

	// Apply the constraint modification.
	common.ApplyNodeConstraintToObj(d, deploymentInfo.GetAlertId())

	// Post the new Deployment data.
	_, err = client.ExtensionsV1beta1().Deployments(deploymentInfo.GetNamespace()).Update(d)
	if err != nil {
		return retry.MakeRetryable(err)
	}
	return
}
