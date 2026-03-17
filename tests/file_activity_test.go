//go:build test_e2e

package tests

import (
	"context"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	testImage    = "docker.io/library/nginx:latest"
	nsenterImage = "docker.io/library/busybox:latest"
)

// TestFileActivity is a sanity check that the file activity detection pipeline
// works end-to-end: Fact (Collector) -> Sensor -> Central -> Alert.
// Detailed operation/path/policy logic is covered by unit tests.
func TestFileActivity(t *testing.T) {
	skipIfNoFact(t)

	k8sClient := createK8sClient(t)
	conn := centralgrpc.GRPCConnectionToCentral(t)
	policyService := v1.NewPolicyServiceClient(conn)
	alertService := v1.NewAlertServiceClient(conn)

	restoreFactPaths := patchFactPaths(t, k8sClient)
	t.Cleanup(restoreFactPaths)

	t.Run("DeploymentLevel", func(t *testing.T) {
		deploymentName := "fa-test-" + uuid.NewV4().String()[:8]
		setupDeploymentInNamespace(t, testImage, deploymentName, "default")
		t.Cleanup(func() { teardownDeploymentWithoutCheck(t, deploymentName, "default") })

		podName := waitForRunningPod(t, k8sClient, deploymentName)

		path := uniquePath("deploy")
		policyName := "FA-E2E-deploy-" + uuid.NewV4().String()[:8]

		policy := createFileActivityPolicy(policyName, path,
			storage.EventSource_DEPLOYMENT_EVENT, "CREATE")
		_, cleanup := importAndCleanupPolicy(t, policy, policyService)
		t.Cleanup(cleanup)

		execInPod(t, k8sClient, "default", podName, deploymentName, []string{"touch", path})

		req := buildDeploymentAlertQuery(deploymentName, policyName)
		waitForAlert(t, alertService, req, 1)

		// Sanity check: verify the alert contains meaningful violation details.
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		resp, err := alertService.ListAlerts(ctx, req)
		require.NoError(t, err)
		require.Len(t, resp.GetAlerts(), 1)

		alert, err := alertService.GetAlert(ctx, &v1.ResourceByID{Id: resp.GetAlerts()[0].GetId()})
		require.NoError(t, err)
		require.NotEmpty(t, alert.GetViolations())

		v := alert.GetViolations()[0]
		assert.Equal(t, storage.Alert_Violation_FILE_ACCESS, v.GetType(),
			"violation type should be FILE_ACCESS")
		assert.Contains(t, v.GetMessage(), path,
			"violation message should contain the file path")
	})

	t.Run("NodeLevel", func(t *testing.T) {
		hostPodName := "fa-host-" + uuid.NewV4().String()[:8]
		createHostExecPod(t, k8sClient, hostPodName)
		t.Cleanup(func() { deleteHostExecPod(t, k8sClient, hostPodName) })

		path := uniquePath("node")
		policyName := "FA-E2E-node-" + uuid.NewV4().String()[:8]

		policy := createFileActivityPolicy(policyName, path,
			storage.EventSource_NODE_EVENT, "CREATE")
		_, cleanup := importAndCleanupPolicy(t, policy, policyService)
		t.Cleanup(cleanup)

		execInPod(t, k8sClient, "default", hostPodName, "nsenter",
			[]string{"chroot", "/host", "sudo", "touch", path})
		t.Cleanup(func() {
			execInPod(t, k8sClient, "default", hostPodName, "nsenter",
				[]string{"chroot", "/host", "sudo", "rm", "-f", path})
		})

		req := buildNodeAlertQuery(policyName)
		waitForAlert(t, alertService, req, 1)
	})
}

// waitForRunningPod finds a running pod for the given deployment.
func waitForRunningPod(t *testing.T, client kubernetes.Interface, deploymentName string) string {
	t.Helper()

	var podName string
	waitForCondition(t, func() bool {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		pods, err := client.CoreV1().Pods("default").List(ctx, metaV1.ListOptions{
			LabelSelector: "app=" + deploymentName,
		})
		if err != nil {
			return false
		}
		for _, pod := range pods.Items {
			if pod.Status.Phase == coreV1.PodRunning {
				podName = pod.Name
				return true
			}
		}
		return false
	}, "pod running for "+deploymentName, 2*time.Minute, 5*time.Second)

	return podName
}

// createHostExecPod creates a privileged pod with hostPID for nsenter commands.
func createHostExecPod(t *testing.T, client kubernetes.Interface, name string) {
	t.Helper()

	privileged := true
	hostPID := true
	pod := &coreV1.Pod{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: coreV1.PodSpec{
			HostPID: hostPID,
			Containers: []coreV1.Container{
				{
					Name:    "nsenter",
					Image:   nsenterImage,
					Command: []string{"sh", "-c", "sleep 3600"},
					SecurityContext: &coreV1.SecurityContext{
						Privileged: &privileged,
					},
					VolumeMounts: []coreV1.VolumeMount{
						{Name: "host-root", MountPath: "/host"},
					},
				},
			},
			Volumes: []coreV1.Volume{
				{
					Name: "host-root",
					VolumeSource: coreV1.VolumeSource{
						HostPath: &coreV1.HostPathVolumeSource{Path: "/"},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := client.CoreV1().Pods("default").Create(ctx, pod, metaV1.CreateOptions{})
	require.NoError(t, err, "creating host exec pod %s", name)

	waitForCondition(t, func() bool {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		p, err := client.CoreV1().Pods("default").Get(ctx, name, metaV1.GetOptions{})
		if err != nil {
			return false
		}
		return p.Status.Phase == coreV1.PodRunning
	}, "host exec pod running", 2*time.Minute, 5*time.Second)
}

// deleteHostExecPod removes the privileged pod.
func deleteHostExecPod(t *testing.T, client kubernetes.Interface, name string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := client.CoreV1().Pods("default").Delete(ctx, name, metaV1.DeleteOptions{})
	if err != nil {
		t.Logf("Warning: failed to delete host exec pod %s: %v", name, err)
	}
}
