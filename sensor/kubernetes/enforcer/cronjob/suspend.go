package cronjob

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/central"
	kubernetesPkg "github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/sensor/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/kubernetes"
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
		return nil, metav1.PatchOptions{}, err
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
func Suspend(ctx context.Context, client kubernetes.Interface, deploymentInfo *central.DeploymentEnforcement) (err error) {
	ok, apiErr := utils.HasAPI(client, batchV1, kubernetesPkg.CronJob)
	if apiErr != nil {
		return retry.MakeRetryable(apiErr)
	}

	if ok {
		patchBytes, patchOptions, err := makePatch(deploymentInfo, batchV1)
		if err != nil {
			return err
		}
		_, err = client.BatchV1().CronJobs(deploymentInfo.GetNamespace()).Patch(ctx, deploymentInfo.GetDeploymentName(),
			types.ApplyPatchType,
			patchBytes,
			patchOptions)
		if err != nil {
			return retry.MakeRetryable(err)
		}
	} else {
		patchBytes, patchOptions, err := makePatch(deploymentInfo, batchV1beta1)
		if err != nil {
			return err
		}
		_, err = client.BatchV1beta1().CronJobs(deploymentInfo.GetNamespace()).Patch(ctx, deploymentInfo.GetDeploymentName(), types.ApplyPatchType,
			patchBytes,
			patchOptions)
		if err != nil {
			return retry.MakeRetryable(err)
		}
	}
	return nil
}
