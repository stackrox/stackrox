package replicaset

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"github.com/stackrox/rox/sensor/kubernetes/enforcer/common"
	appsV1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
)

// EnforceNodeConstraint reschedules the ReplicaSet with unsatisfiable constraints.
func EnforceNodeConstraint(ctx context.Context, dynClient dynamic.Interface, deploymentInfo *central.DeploymentEnforcement) (err error) {
	// Load the current ReplicaSet for the deployment.
	unstructuredObj, err := dynClient.Resource(client.ReplicaSetGVR).Namespace(deploymentInfo.GetNamespace()).Get(ctx, deploymentInfo.GetDeploymentName(), metav1.GetOptions{})
	if err != nil {
		return retry.MakeRetryable(err)
	}

	var rs appsV1.ReplicaSet
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.Object, &rs); err != nil {
		return err
	}

	// Apply the constraint modification.
	if err := common.ApplyNodeConstraintToObj(&rs, deploymentInfo.GetAlertId()); err != nil {
		return err
	}

	// Convert back and update.
	updated, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&rs)
	if err != nil {
		return err
	}
	unstructuredObj.Object = updated

	_, err = dynClient.Resource(client.ReplicaSetGVR).Namespace(deploymentInfo.GetNamespace()).Update(ctx, unstructuredObj, metav1.UpdateOptions{})
	if err != nil {
		return retry.MakeRetryable(err)
	}
	return nil
}
