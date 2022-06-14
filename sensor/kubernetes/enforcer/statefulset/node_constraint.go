package statefulset

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/sensor/kubernetes/enforcer/common"
	appsV1 "k8s.io/api/apps/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// EnforceNodeConstraint reschedules the StatefulSet with unsatisfiable constraints.
func EnforceNodeConstraint(ctx context.Context, client kubernetes.Interface, deploymentInfo *central.DeploymentEnforcement) (err error) {
	// Load the current StatefulSet for the deployment.
	var ds *appsV1.StatefulSet
	ds, err = client.AppsV1beta1().StatefulSets(deploymentInfo.GetNamespace()).Get(ctx, deploymentInfo.GetDeploymentName(), metav1.GetOptions{})
	if err != nil {
		return retry.MakeRetryable(err)
	}

	// Apply the constraint modification.
	err = common.ApplyNodeConstraintToObj(ds, deploymentInfo.GetAlertId())
	if err != nil {
		return
	}

	// Post the new StatefulSet data.
	_, err = client.AppsV1beta1().StatefulSets(deploymentInfo.GetNamespace()).Update(ctx, ds, metav1.UpdateOptions{})
	if err != nil {
		return retry.MakeRetryable(err)
	}
	return
}
