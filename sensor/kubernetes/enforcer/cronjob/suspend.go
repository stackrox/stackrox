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

// Suspend suspends the cron job
func Suspend(ctx context.Context, client kubernetes.Interface, deploymentInfo *central.DeploymentEnforcement) (err error) {
	forcePatch := true

	if ok, apiErr := utils.HasAPI(client, batchV1, kubernetesPkg.CronJob); ok && apiErr == nil {
		patch := map[string]interface{}{
			"metadata": map[string]interface{}{
				"name": deploymentInfo.GetDeploymentName(),
			},
			"kind":       deploymentInfo.GetDeploymentType(),
			"apiVersion": batchV1,
			"spec": map[string]interface{}{
				"suspend": true,
			},
		}
		patchBytes, err := json.Marshal(patch)
		if err != nil {
			return err
		}

		_, err = client.BatchV1().CronJobs(deploymentInfo.GetNamespace()).Patch(ctx, deploymentInfo.GetDeploymentName(),
			types.ApplyPatchType,
			patchBytes,
			metav1.PatchOptions{
				TypeMeta: metav1.TypeMeta{
					Kind:       deploymentInfo.GetDeploymentType(),
					APIVersion: batchV1,
				},
				FieldManager: fieldManager,
				Force:        &forcePatch,
			})
		if err != nil {
			return retry.MakeRetryable(err)
		}
	} else {
		patch := map[string]interface{}{
			"metadata": map[string]interface{}{
				"name": deploymentInfo.GetDeploymentName(),
			},
			"kind":       deploymentInfo.GetDeploymentType(),
			"apiVersion": batchV1beta1,
			"spec": map[string]interface{}{
				"suspend": true,
			},
		}
		patchBytes, err := json.Marshal(patch)
		if err != nil {
			return err
		}

		_, err = client.BatchV1beta1().CronJobs(deploymentInfo.GetNamespace()).Patch(ctx, deploymentInfo.GetDeploymentName(), types.ApplyPatchType,
			patchBytes,
			metav1.PatchOptions{
				TypeMeta: metav1.TypeMeta{
					Kind:       deploymentInfo.GetDeploymentType(),
					APIVersion: batchV1beta1,
				},
				FieldManager: fieldManager,
				Force:        &forcePatch,
			})
		if err != nil {
			return retry.MakeRetryable(err)
		}
	}
	return nil
}
