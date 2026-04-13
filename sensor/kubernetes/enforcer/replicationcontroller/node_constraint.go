package replicationcontroller

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"github.com/stackrox/rox/sensor/kubernetes/enforcer/common"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
)

// EnforceNodeConstraint reschedules the ReplicationController with unsatisfiable constraints.
func EnforceNodeConstraint(ctx context.Context, dynClient dynamic.Interface, deploymentInfo *central.DeploymentEnforcement) (err error) {
	// Load the current ReplicationController for the deployment.
	unstructuredObj, err := dynClient.Resource(client.ReplicationControllerGVR).Namespace(deploymentInfo.GetNamespace()).Get(ctx, deploymentInfo.GetDeploymentName(), metav1.GetOptions{})
	if err != nil {
		return retry.MakeRetryable(err)
	}

	var rc coreV1.ReplicationController
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.Object, &rc); err != nil {
		return err
	}

	// Apply the constraint modification.
	if err := common.ApplyNodeConstraintToObj(&rc, deploymentInfo.GetAlertId()); err != nil {
		return err
	}

	// Convert back and update.
	updated, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&rc)
	if err != nil {
		return err
	}
	unstructuredObj.Object = updated

	_, err = dynClient.Resource(client.ReplicationControllerGVR).Namespace(deploymentInfo.GetNamespace()).Update(ctx, unstructuredObj, metav1.UpdateOptions{})
	if err != nil {
		return retry.MakeRetryable(err)
	}
	return nil
}
