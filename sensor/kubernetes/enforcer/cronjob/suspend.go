package cronjob

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/dynamic"
)

const (
	batchV1      = "batch/v1"
	batchV1beta1 = "batch/v1beta1"
	fieldManager = "StackRox"
)

func makePatch(deploymentInfo *central.DeploymentEnforcement, apiVersion string) ([]byte, metav1.PatchOptions, error) {
	forcePatch := true

	patch := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name": deploymentInfo.GetDeploymentName(),
		},
		"kind":       deploymentInfo.GetDeploymentType(),
		"apiVersion": apiVersion,
		"spec": map[string]interface{}{
			"suspend": true,
		},
	}
	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return nil, metav1.PatchOptions{}, errors.Wrap(err, "marshalling patch for suspending cronjob")
	}

	options := metav1.PatchOptions{
		TypeMeta: metav1.TypeMeta{
			Kind:       deploymentInfo.GetDeploymentType(),
			APIVersion: apiVersion,
		},
		FieldManager: fieldManager,
		Force:        &forcePatch,
	}

	return patchBytes, options, nil
}

// Suspend suspends the cron job
func Suspend(ctx context.Context, dynClient dynamic.Interface, deploymentInfo *central.DeploymentEnforcement) (err error) {
	// Try batch/v1 first (available in k8s 1.21+), fall back to v1beta1.
	patchBytes, patchOptions, err := makePatch(deploymentInfo, batchV1)
	if err != nil {
		return err
	}
	_, err = dynClient.Resource(client.CronJobGVR).Namespace(deploymentInfo.GetNamespace()).Patch(ctx, deploymentInfo.GetDeploymentName(),
		types.ApplyPatchType,
		patchBytes,
		patchOptions)
	if err == nil {
		return nil
	}

	// Fall back to v1beta1
	patchBytes, patchOptions, err = makePatch(deploymentInfo, batchV1beta1)
	if err != nil {
		return err
	}
	_, err = dynClient.Resource(client.CronJobBetaGVR).Namespace(deploymentInfo.GetNamespace()).Patch(ctx, deploymentInfo.GetDeploymentName(), types.ApplyPatchType,
		patchBytes,
		patchOptions)
	if err != nil {
		return retry.MakeRetryable(err)
	}
	return nil
}
