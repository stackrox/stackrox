//go:build test_e2e || sql_integration || compliance || destructive || externalbackups || test_compatibility

package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/docker/config"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/retryablehttp"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

//lint:file-ignore U1000 since common.go is included in several different go:build tags but not every function is used
// within every tag, hence the linter incorrectly shows most functions in this file as unused.

const (
	nginxDeploymentName     = `nginx`
	expectedLatestTagPolicy = `Latest tag`

	sensorDeployment = "sensor"
	sensorContainer  = "sensor"

	waitTimeout = 5 * time.Minute
	timeout     = 30 * time.Second
)

var (
	sensorPodLabels = map[string]string{"app": "sensor"}
)

// testutilsLogger adapts testutils.T to retryablehttp.Logger interface.
//
// WHY THIS EXISTS (and why it's unfortunate):
// The retryablehttp library requires a logger implementing its Logger interface with Printf method.
// Go's *testing.T has Logf but NOT Printf (different method names, same functionality).
// We cannot add Printf to testutils.T because that would break compatibility with *testing.T,
// which is used throughout the codebase (e.g., centralgrpc.GRPCConnectionToCentral(t testutils.T)).
//
// CLEANER ALTERNATIVE:
// Accept *testing.T directly in helper functions and create testutils.T wrappers locally where needed
// (specifically for the retry mechanism). This would avoid the interface constraint propagating everywhere.
// However, this would be a larger refactor affecting many test helper functions.
//
// CURRENT COMPROMISE:
// Use this tiny adapter ONLY where retryablehttp requires it. Everywhere else uses testutils.T naturally.
type testutilsLogger struct{ testutils.T }

func (l testutilsLogger) Printf(format string, v ...interface{}) { l.Logf(format, v...) }

// logf logs using the testing logger, prefixing a high-resolution timestamp.
// Using testing.T.Logf means that the output is hidden unless the test fails or verbose logging is enabled with -v.
func logf(t *testing.T, format string, args ...any) {
	t.Logf(time.Now().Format(time.StampMilli)+" "+format, args...)
}

// testContexts returns a couple of contexts for the given test: with a timeout for the testing logic and another
// with an additional longer timeout for debug artifact gathering and cleanup.
func testContexts(t *testing.T, name string, timeout time.Duration) (testCtx context.Context, overallCtx context.Context, cancel func()) {
	var (
		overallCancel func()
		testCancel    func()
	)
	cleanupTimeout := 10 * time.Minute
	logf(t, "Running %s with a timeout of %s plus %s for cleanup", name, timeout, cleanupTimeout)
	overallTimeout := timeout + cleanupTimeout
	overallErr := fmt.Errorf("overall %s test+cleanup timeout of %s reached", name, overallTimeout)
	testErr := fmt.Errorf("%s test timeout of %s reached", name, timeout)
	overallCtx, overallCancel = context.WithTimeoutCause(context.Background(), overallTimeout, overallErr)
	testCtx, testCancel = context.WithTimeoutCause(overallCtx, timeout, testErr)
	cancel = func() {
		testCancel()
		overallCancel()
	}
	return
}

// mustGetEnv calls os.GetEnv and fails the test if result is empty.
func mustGetEnv(t *testing.T, varName string) string {
	val := os.Getenv(varName)
	require.NotEmptyf(t, val, "Environment variable %q must be set.", varName)
	return val
}

func retrieveDeployment(service v1.DeploymentServiceClient, deploymentID string) (*storage.Deployment, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return service.GetDeployment(ctx, &v1.ResourceByID{Id: deploymentID})
}

func retrieveDeployments(service v1.DeploymentServiceClient, deps []*storage.ListDeployment) ([]*storage.Deployment, error) {
	deployments := make([]*storage.Deployment, 0, len(deps))
	for _, d := range deps {
		deployment, err := retrieveDeployment(service, d.GetId())
		if err != nil {
			return nil, err
		}
		deployments = append(deployments, deployment)
	}
	return deployments, nil
}

func waitForDeploymentCountInCentral(t testutils.T, query string, count int) {
	conn := centralgrpc.GRPCConnectionToCentral(t)

	service := v1.NewDeploymentServiceClient(conn)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	timer := time.NewTimer(waitTimeout)
	defer timer.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			deploymentCount, err := service.CountDeployments(ctx, &v1.RawQuery{Query: query})
			cancel()
			if err != nil {
				t.Logf("Error listing deployments: %s", err)
				continue
			}
			if deploymentCount.GetCount() == int32(count) {
				return
			}

		case <-timer.C:
			t.Fatalf("Timed out waiting for deployments %q", query)
		}
	}

}

func waitForDeploymentReadyInK8s(t testutils.T, deploymentName, namespace string) {
	client := createK8sClient(t)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timer := time.NewTimer(waitTimeout)
	defer timer.Stop()

	ctx := context.Background()
	t.Logf("Waiting for deployment %q in namespace %q to be ready in Kubernetes", deploymentName, namespace)

	for {
		select {
		case <-ticker.C:
			deploy, err := client.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metaV1.GetOptions{})
			if err != nil {
				if apiErrors.IsNotFound(err) {
					t.Logf("Deployment %q in namespace %q not found yet, waiting...", deploymentName, namespace)
					continue
				}
				t.Logf("Error getting deployment %q from namespace %q: %v", deploymentName, namespace, err)
				continue
			}

			// Check if generation matches observed generation.
			// Practical Example: generation: 5, observedGeneration: 4
			// This tells you: "Someone just updated the deployment (5th change),
			// but the controller is still working on the 4th revision—be patient, reconciliation is in progress."
			// Without these numbers, you'd just see repeated "NOT ready" messages without understanding the root cause.
			if deploy.GetGeneration() != deploy.Status.ObservedGeneration {
				t.Logf("Deployment %q in namespace %q NOT ready: generation %d != observed generation %d",
					deploymentName, namespace, deploy.GetGeneration(), deploy.Status.ObservedGeneration)
				continue
			}

			// Check if all replicas are ready
			if deploy.Status.Replicas == 0 || deploy.Status.Replicas != deploy.Status.ReadyReplicas {
				t.Logf("Deployment %q in namespace %q NOT ready: %d/%d ready replicas",
					deploymentName, namespace, deploy.Status.ReadyReplicas, deploy.Status.Replicas)
				continue
			}

			// Ensure all pods are from the current generation (no old pods during rollout)
			if deploy.Status.UpdatedReplicas > 0 && deploy.Status.UpdatedReplicas != deploy.Status.Replicas {
				t.Logf("Deployment %q in namespace %q NOT ready: rollout incomplete (%d/%d updated replicas)",
					deploymentName, namespace, deploy.Status.UpdatedReplicas, deploy.Status.Replicas)
				continue
			}

			// Check conditions for additional insights
			printDeploymentConditions(t, deploy.Status.Conditions, deploymentName, namespace)

			t.Logf("Deployment %q in namespace %q READY in Kubernetes (%d/%d ready replicas)",
				deploymentName, namespace, deploy.Status.ReadyReplicas, deploy.Status.Replicas)
			return
		case <-timer.C:
			// Get final status for error message
			deploy, err := client.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metaV1.GetOptions{})
			if err != nil {
				require.NoError(t, err, "Timed out waiting for deployment %q in namespace %q. Failed to get final status: %v", deploymentName, namespace, err)
			}
			require.Failf(t, "Timed out waiting for deployment in Kubernetes",
				"Deployment %q in namespace %q did not become ready within %v.\nStatus: Replicas=%d, ReadyReplicas=%d, UpdatedReplicas=%d, AvailableReplicas=%d, UnavailableReplicas=%d\nGeneration=%d, ObservedGeneration=%d",
				deploymentName, namespace, waitTimeout,
				deploy.Status.Replicas, deploy.Status.ReadyReplicas, deploy.Status.UpdatedReplicas,
				deploy.Status.AvailableReplicas, deploy.Status.UnavailableReplicas,
				deploy.GetGeneration(), deploy.Status.ObservedGeneration)
		}
	}
}

func printDeploymentConditions(t testutils.T, conditions []appsV1.DeploymentCondition, deploymentName, namespace string) {
	for _, cond := range conditions {
		if cond.Type == appsV1.DeploymentAvailable && cond.Status != coreV1.ConditionTrue {
			t.Logf("Deployment %q in namespace %q NOT ready: Available condition is %s: %s",
				deploymentName, namespace, cond.Status, cond.Message)
		}
		if cond.Type == appsV1.DeploymentProgressing && cond.Status != coreV1.ConditionTrue {
			t.Logf("Deployment %q in namespace %q NOT ready: Progressing condition is %s: %s",
				deploymentName, namespace, cond.Status, cond.Message)
		}
	}
}

func waitForDeploymentInCentral(t testutils.T, deploymentName string) {
	conn := centralgrpc.GRPCConnectionToCentral(t)

	service := v1.NewDeploymentServiceClient(conn)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	timer := time.NewTimer(waitTimeout)
	defer timer.Stop()

	qb := search.NewQueryBuilder().AddExactMatches(search.DeploymentName, deploymentName)

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			listDeployments, err := service.ListDeployments(ctx, &v1.RawQuery{
				Query: qb.Query(),
			},
			)
			cancel()
			if err != nil {
				t.Logf("Error listing deployments: %s", err)
				continue
			}

			deployments, err := retrieveDeployments(service, listDeployments.GetDeployments())
			if err != nil {
				t.Logf("Error retrieving deployments: %s", err)
				continue
			}

			if len(deployments) > 0 {
				d := deployments[0]

				if len(d.GetContainers()) > 0 && d.GetContainers()[0].GetImage().GetId() != "" {
					return
				}
			}
		case <-timer.C:
			t.Fatalf("Timed out waiting for deployment %s", deploymentName)
		}
	}
}

func waitForTermination(t testutils.T, deploymentName string) {
	conn := centralgrpc.GRPCConnectionToCentral(t)

	service := v1.NewDeploymentServiceClient(conn)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	timer := time.NewTimer(waitTimeout)
	defer timer.Stop()

	query := search.NewQueryBuilder().AddStrings(search.DeploymentName, deploymentName).Query()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			listDeployments, err := service.ListDeployments(ctx, &v1.RawQuery{
				Query: query,
			})
			cancel()
			if err != nil {
				t.Logf("Error listing deployments: %v", err)
				continue
			}

			if len(listDeployments.GetDeployments()) == 0 {
				return
			}
		case <-timer.C:
			t.Fatalf("Timed out waiting for deployment %s to stop", deploymentName)
		}
	}
}

func getPodFromFile(t testutils.T, path string) *coreV1.Pod {
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewBuffer(data), len(data))
	var pod coreV1.Pod
	err = decoder.Decode(&pod)
	require.NoError(t, err)
	return &pod
}

func setupDeploymentInNamespace(t *testing.T, image, deploymentName, namespace string) {
	setupDeploymentWithReplicasInNamespace(t, image, deploymentName, 1, namespace)
}

func setupDeploymentWithReplicas(t *testing.T, image, deploymentName string, replicas int) {
	setupDeploymentWithReplicasInNamespace(t, image, deploymentName, replicas, "default")
}

func setupDeploymentWithReplicasInNamespace(t *testing.T, image, deploymentName string, replicas int, namespace string) {
	setupDeploymentNoWaitInNamespace(t, image, deploymentName, replicas, namespace)
	waitForDeploymentReadyInK8s(t, deploymentName, namespace)
	waitForDeploymentInCentral(t, deploymentName)
}

func setupDeploymentNoWait(t *testing.T, image, deploymentName string, replicas int) {
	setupDeploymentNoWaitInNamespace(t, image, deploymentName, replicas, "default")
}

func setupDeploymentNoWaitInNamespace(t *testing.T, image, deploymentName string, replicas int, namespace string) {
	require.NoError(t, createDeploymentViaAPI(t, image, deploymentName, replicas, namespace))
}

// createDeploymentViaAPI creates a Kubernetes deployment using the K8s API client.
// Mirrors qa-tests-backend/src/main/groovy/orchestratormanager/Kubernetes.groovy:2316-2318
// to support IMAGE_PULL_POLICY_FOR_QUAY_IO for prefetched images.
func createDeploymentViaAPI(t *testing.T, image, deploymentName string, replicas int, namespace string) error {
	client := createK8sClient(t)

	t.Logf("Creating deployment %q in namespace %q with image %q and %d replicas", deploymentName, namespace, image, replicas)

	// Determine imagePullPolicy - allow override ONLY for actual quay.io/ images.
	// NOTE: This intentionally does NOT apply to mirrored images (e.g., icsp.invalid, idms.invalid)
	// as those are used to test mirroring functionality and should use their own pull behavior.
	pullPolicy := coreV1.PullIfNotPresent
	if policy := os.Getenv("IMAGE_PULL_POLICY_FOR_QUAY_IO"); policy != "" && strings.HasPrefix(image, "quay.io/") {
		pullPolicy = coreV1.PullPolicy(policy)
		t.Logf("Setting imagePullPolicy=%s for quay.io image (IMAGE_PULL_POLICY_FOR_QUAY_IO)", policy)
	}

	// Build deployment object
	deployment := &appsV1.Deployment{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      deploymentName,
			Namespace: namespace,
			Labels: map[string]string{
				"app": deploymentName,
			},
		},
		Spec: appsV1.DeploymentSpec{
			Replicas: pointers.Int32(int32(replicas)),
			Selector: &metaV1.LabelSelector{
				MatchLabels: map[string]string{"app": deploymentName},
			},
			Template: coreV1.PodTemplateSpec{
				ObjectMeta: metaV1.ObjectMeta{
					Labels: map[string]string{"app": deploymentName},
				},
				Spec: coreV1.PodSpec{
					Containers: []coreV1.Container{{
						Name:            deploymentName,
						Image:           image,
						ImagePullPolicy: pullPolicy,
						Resources:       coreV1.ResourceRequirements{}, // Match kubectl behavior
					}},
				},
			},
		},
	}

	t.Logf("Deployment object created: name=%s, namespace=%s, replicas=%d, image=%s, labels=%v",
		deployment.Name, deployment.Namespace, *deployment.Spec.Replicas, image, deployment.Labels)

	// Create the deployment with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Logf("Calling K8s API to create deployment %q in namespace %q...", deploymentName, namespace)
	createdDeployment, err := client.AppsV1().Deployments(namespace).Create(ctx, deployment, metaV1.CreateOptions{})

	if err != nil {
		// Detailed error logging
		if apiErrors.IsAlreadyExists(err) {
			t.Logf("ERROR: Deployment %q already exists in namespace %q: %v", deploymentName, namespace, err)
		} else if apiErrors.IsInvalid(err) {
			t.Logf("ERROR: Deployment %q spec is invalid: %v", deploymentName, err)
		} else if apiErrors.IsForbidden(err) {
			t.Logf("ERROR: Permission denied creating deployment %q in namespace %q: %v", deploymentName, namespace, err)
		} else if apiErrors.IsTimeout(err) {
			t.Logf("ERROR: Timeout creating deployment %q in namespace %q after 30s: %v", deploymentName, namespace, err)
		} else if apiErrors.IsServerTimeout(err) {
			t.Logf("ERROR: Server timeout creating deployment %q in namespace %q: %v", deploymentName, namespace, err)
		} else if apiErrors.IsServiceUnavailable(err) {
			t.Logf("ERROR: K8s API service unavailable when creating deployment %q: %v", deploymentName, err)
		} else {
			t.Logf("ERROR: Unexpected error creating deployment %q in namespace %q: %v (type: %T)", deploymentName, namespace, err, err)
		}
		// Log deployment conditions only if deployment was partially created (useful for debugging failures)
		if createdDeployment != nil && len(createdDeployment.Status.Conditions) > 0 {
			t.Logf("Deployment %q has %d status conditions:", deploymentName, len(createdDeployment.Status.Conditions))
			for i, cond := range createdDeployment.Status.Conditions {
				t.Logf("  Condition[%d]: Type=%s, Status=%s, Reason=%s, Message=%q, LastUpdateTime=%v",
					i, cond.Type, cond.Status, cond.Reason, cond.Message, cond.LastUpdateTime)
			}
		}
		return fmt.Errorf("failed to create deployment %q: %w", deploymentName, err)
	}

	t.Logf("Deployment %q successfully created in namespace %q", deploymentName, namespace)
	t.Logf("Deployment UID: %s, ResourceVersion: %s, Generation: %d",
		createdDeployment.UID, createdDeployment.ResourceVersion, createdDeployment.Generation)
	t.Logf("Deployment status: Replicas=%d, UpdatedReplicas=%d, ReadyReplicas=%d, AvailableReplicas=%d, UnavailableReplicas=%d",
		createdDeployment.Status.Replicas,
		createdDeployment.Status.UpdatedReplicas,
		createdDeployment.Status.ReadyReplicas,
		createdDeployment.Status.AvailableReplicas,
		createdDeployment.Status.UnavailableReplicas)

	t.Logf("Deployment %q creation completed successfully", deploymentName)
	return nil
}

func setImage(t *testing.T, deploymentName string, deploymentID string, containerName string, image string) {
	cmd := exec.Command(`kubectl`, `set`, `image`, fmt.Sprintf("deployment/%s", deploymentName), fmt.Sprintf("%s=%s", containerName, image))
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
	conn := centralgrpc.GRPCConnectionToCentral(t)
	service := v1.NewDeploymentServiceClient(conn)

	waitForCondition(t, func() bool {
		deployment, err := retrieveDeployment(service, deploymentID)
		if err != nil {
			t.Logf("Error retrieving deployment: %v", err)
			return false
		}
		containers := deployment.GetContainers()
		for _, container := range containers {
			if container.GetImage().GetName().GetFullName() != image {
				return false
			}
		}
		t.Logf("Image set to %s for deployment %s(%s) container %s", image, deploymentName, deploymentID, containerName)
		return true
	}, "image updated", time.Minute, 5*time.Second)
}

// ensurePodExists creates a pod in Kubernetes. If the pod already exists, this is a no-op.
// This makes the function idempotent and safe to retry.
func ensurePodExists(t testutils.T, client kubernetes.Interface, pod *coreV1.Pod) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Logf("Ensuring pod %s %s exists", pod.GetNamespace(), pod.GetName())
	_, err := client.CoreV1().Pods(pod.GetNamespace()).Create(ctx, pod, metaV1.CreateOptions{})
	if err != nil && !apiErrors.IsAlreadyExists(err) {
		require.NoError(t, err)
	}
	if apiErrors.IsAlreadyExists(err) {
		t.Logf("Pod %s already exists, continuing", pod.GetName())
	}
}

// waitForPodRunning waits for a Kubernetes pod to be in Running phase with all containers ready.
// It polls the pod status with retries and provides detailed error messages about pod and container states.
// Timeout is set to 3 minutes to handle slow CI environments (image pull, scheduling, etc.).
func waitForPodRunning(t testutils.T, client kubernetes.Interface, podNamespace, podName string) *coreV1.Pod {
	// Increased timeout to 3 minutes to handle slow CI environments (image pull, scheduling, etc.)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	var k8sPod *coreV1.Pod
	// Increased from 30×2s (60s) to 60×3s (180s) to account for slower pod startup in CI
	testutils.Retry(t, 60, 3*time.Second, func(waitT testutils.T) {
		var err error
		k8sPod, err = client.CoreV1().Pods(podNamespace).Get(ctx, podName, metaV1.GetOptions{})
		require.NoError(waitT, err, "failed to get pod %s", podName)

		// Log pod and container status for debugging
		// Note: ImagePullBackOff, ErrImagePull, etc. appear in container status, not pod status
		logMsg := fmt.Sprintf("Pod phase: %s, Reason: %q, Message: %q",
			k8sPod.Status.Phase, k8sPod.Status.Reason, k8sPod.Status.Message)
		var containerInfo strings.Builder
		for _, status := range k8sPod.Status.ContainerStatuses {
			// Build log message for non-ready containers
			if !status.Ready {
				if status.State.Waiting != nil {
					logMsg += fmt.Sprintf(", Container %q: %q", status.Name, status.State.Waiting.Reason)
				} else if status.State.Terminated != nil {
					logMsg += fmt.Sprintf(", Container %q: Terminated (%q)", status.Name, status.State.Terminated.Reason)
				}
			}
			// Build detailed info for error message (always, in case pod is not running)
			containerInfo.WriteString(fmt.Sprintf("\n  - %s: ready=%v, started=%v",
				status.Name, status.Ready, status.Started != nil && *status.Started))
			if status.State.Waiting != nil {
				containerInfo.WriteString(fmt.Sprintf(", waiting: %s - %s",
					status.State.Waiting.Reason, status.State.Waiting.Message))
			}
		}
		waitT.Logf(logMsg)

		// Provide detailed error message if pod is not running
		if k8sPod.Status.Phase != coreV1.PodRunning {
			require.Failf(waitT, "pod not in Running phase",
				"Pod %s is in %s phase (expected Running)\nContainers:%s\nPod Reason: %s\nPod Message: %s",
				podName, k8sPod.Status.Phase, containerInfo.String(),
				k8sPod.Status.Reason, k8sPod.Status.Message)
		}

		// Ensure all containers are ready before checking for process events
		for _, status := range k8sPod.Status.ContainerStatuses {
			require.True(waitT, status.Ready, "container %s not ready (state: %+v)",
				status.Name, status.State)
		}
	})

	t.Logf("Pod %s is running with all containers ready in Kubernetes", k8sPod.Name)
	return k8sPod
}

func teardownPod(t testutils.T, client kubernetes.Interface, pod *coreV1.Pod) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := client.CoreV1().Pods(pod.GetNamespace()).Delete(ctx, pod.GetName(), metaV1.DeleteOptions{GracePeriodSeconds: pointers.Int64(0)})
	require.NoError(t, err)

	waitForTermination(t, pod.GetName())
}

// teardownDeploymentInternal handles deployment deletion with configurable verification.
// When waitForCompletion is true, it retries deletion and waits for both K8s and Central cleanup.
// When false, it only issues the delete command without waiting or failing on errors.
func teardownDeploymentInternal(t *testing.T, deploymentName string, namespace string, waitForCompletion bool) {
	client := createK8sClient(t)
	deletePolicy := metaV1.DeletePropagationForeground
	gracePeriod := int64(1)

	deleteFunc := func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		logf(t, "Deleting deployment %q in namespace %q via API with propagation policy %s", deploymentName, namespace, deletePolicy)
		err := client.AppsV1().Deployments(namespace).Delete(ctx, deploymentName, metaV1.DeleteOptions{
			GracePeriodSeconds: &gracePeriod,
			PropagationPolicy:  &deletePolicy,
		})

		if err != nil {
			if apiErrors.IsNotFound(err) {
				logf(t, "Deployment %q not found in namespace %q (already deleted)", deploymentName, namespace)
				return nil
			}
			return fmt.Errorf("failed to delete deployment %q in namespace %q: %w", deploymentName, namespace, err)
		}
		logf(t, "Successfully initiated deletion of deployment %q in namespace %q", deploymentName, namespace)
		return nil
	}

	if waitForCompletion {
		// Retry the delete + wait sequence if the deployment doesn't delete within 15 seconds
		err := retry.WithRetry(
			func() error {
				if err := deleteFunc(); err != nil {
					return retry.MakeRetryable(err)
				}
				// Verify deployment is actually deleted from Kubernetes
				return waitForK8sDeploymentDeletion(t, client, deploymentName, namespace, 15*time.Second)
			},
			retry.Tries(4),              // Try up to 4 times (1 initial + 3 retries)
			retry.OnlyRetryableErrors(), // Only retry on retriable errors
			retry.BetweenAttempts(func(int) {
				logf(t, "Retrying deployment %q deletion in namespace %q", deploymentName, namespace)
			}),
		)
		require.NoError(t, err, "Failed to delete deployment %q in namespace %q after retries", deploymentName, namespace)

		// Wait for Central to recognize the deletion
		waitForTermination(t, deploymentName)
	} else {
		// Fire and forget - don't fail the test if deletion fails
		if err := deleteFunc(); err != nil {
			logf(t, "Deployment %q deletion in namespace %q failed (non-fatal): %v", deploymentName, namespace, err)
		}
	}
}

func teardownDeployment(t *testing.T, deploymentName string, namespace string) {
	teardownDeploymentInternal(t, deploymentName, namespace, true)
}

// waitForK8sDeploymentDeletion polls Kubernetes to verify the deployment is actually gone.
// Returns an error if the deployment still exists after the timeout, which triggers a retry
// of the entire delete operation.
func waitForK8sDeploymentDeletion(t *testing.T, client kubernetes.Interface, deploymentName string, namespace string, timeout time.Duration) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	attempt := 0
	for {
		select {
		case <-ticker.C:
			attempt++
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_, err := client.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metaV1.GetOptions{})
			cancel()

			if apiErrors.IsNotFound(err) {
				logf(t, "Verified: deployment %q deleted from namespace %q after %d attempt(s)", deploymentName, namespace, attempt)
				return nil
			}
			if err != nil {
				logf(t, "Error checking deployment %q status in namespace %q (attempt %d): %v", deploymentName, namespace, attempt, err)
				continue
			}
			logf(t, "Deployment %q still exists in namespace %q (attempt %d)", deploymentName, namespace, attempt)
		case <-timer.C:
			return retry.MakeRetryable(fmt.Errorf("deployment %s in namespace %s still exists in Kubernetes after %v", deploymentName, namespace, timeout))
		}
	}
}

func teardownDeploymentWithoutCheck(t *testing.T, deploymentName string, namespace string) {
	// In cases where deployment will not impact other tests,
	// we can trigger deletion and assume that it will be deleted eventually.
	teardownDeploymentInternal(t, deploymentName, namespace, false)
}

func getConfig(t testutils.T) *rest.Config {
	config, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	require.NoError(t, err, "could not load default Kubernetes client config")

	restCfg, err := clientcmd.NewDefaultClientConfig(*config, &clientcmd.ConfigOverrides{}).ClientConfig()
	require.NoError(t, err, "could not get REST client config from kubernetes config")

	configureRetryableTransport(t, restCfg)

	return restCfg
}

// configureRetryableTransport configures a rest.Config to use retryable HTTP client
// for network resilience. This adds automatic retry logic for transient network errors.
func configureRetryableTransport(t testutils.T, restCfg *rest.Config) {
	if restCfg.Timeout == 0 {
		restCfg.Timeout = 30 * time.Second
	}
	retryablehttp.ConfigureRESTConfig(restCfg,
		retryablehttp.WithLogger(&testutilsLogger{t}),
	)
}

func createK8sClient(t testutils.T) kubernetes.Interface {
	return createK8sClientWithConfig(t, getConfig(t))
}

func createK8sClientWithConfig(t testutils.T, restCfg *rest.Config) kubernetes.Interface {
	k8sClient, err := kubernetes.NewForConfig(restCfg)
	require.NoError(t, err, "creating Kubernetes client from REST config")

	return k8sClient
}

func waitForCondition(t testutils.T, condition func() bool, desc string, timeout time.Duration, frequency time.Duration) {
	ticker := time.NewTicker(frequency)
	defer ticker.Stop()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case <-ticker.C:
			if condition() {
				return
			}
		case <-timer.C:
			t.Fatalf("Timed out waiting for condition %s to stop", desc)
		}
	}
}

type KubernetesSuite struct {
	suite.Suite
	k8s kubernetes.Interface
}

func (ks *KubernetesSuite) SetupSuite() {
	ks.k8s = createK8sClient(ks.T())
}

func (ks *KubernetesSuite) logf(format string, args ...any) {
	logf(ks.T(), format, args...)
}

type logMatcher interface {
	Match(reader io.ReadSeeker) (bool, error)
	fmt.Stringer
}

// waitUntilLog waits until ctx expires or logs of container in all pods matching podLabels satisfy all logMatchers.
func (ks *KubernetesSuite) waitUntilLog(ctx context.Context, namespace string, podLabels map[string]string, container string, description string, logMatchers ...logMatcher) {
	ls := labels.SelectorFromSet(podLabels).String()
	checkLogs := ks.checkLogsClosure(ctx, namespace, ls, container, logMatchers...)
	ks.logf("Waiting until %q pods logs %s: %s", ls, description, logMatchers)
	mustEventually(ks.T(), ctx, checkLogs, 10*time.Second, fmt.Sprintf("Not all %q pods logs %s", ls, description))
}

// checkLogsMatch will check if the logs of container in all pods matching pod labels satisfy all log matchers.
func (ks *KubernetesSuite) checkLogsMatch(ctx context.Context, namespace string, podLabels map[string]string, container string, description string, logMatchers ...logMatcher) {
	ls := labels.SelectorFromSet(podLabels).String()
	ks.logf("Checking %q pods logs %s: %s", ls, description, logMatchers)
	err := ks.checkLogsClosure(ctx, namespace, ls, container, logMatchers...)()
	require.NoError(ks.T(), err, fmt.Sprintf("%q was untrue", description))
}

// checkLogsClosure returns a function that checks if the logs of container in all pods matching pod labels satisfy all the log matchers.
func (ks *KubernetesSuite) checkLogsClosure(ctx context.Context, namespace, labelSelector, container string, logMatchers ...logMatcher) func() error {
	return func() error {
		podList, err := ks.k8s.CoreV1().Pods(namespace).List(ctx, metaV1.ListOptions{LabelSelector: labelSelector})
		if err != nil {
			return fmt.Errorf("could not list pods matching %q in namespace %q: %w", labelSelector, namespace, err)
		}
		if len(podList.Items) == 0 {
			if ok, err := allMatch(strings.NewReader(""), logMatchers...); ok {
				return nil
			} else if err != nil {
				return fmt.Errorf("empty list of pods caused failure: %w", err)
			}
			return fmt.Errorf("empty list of pods does not satisfy the condition")
		}
		for _, pod := range podList.Items {
			resp := ks.k8s.CoreV1().Pods(namespace).GetLogs(pod.GetName(), &coreV1.PodLogOptions{Container: container}).Do(ctx)
			log, err := resp.Raw()
			if err != nil {
				return fmt.Errorf("retrieving logs of pod %q in namespace %q failed: %w", pod.GetName(), namespace, err)
			}
			if ok, err := allMatch(bytes.NewReader(log), logMatchers...); ok {
				continue
			} else if err != nil {
				return fmt.Errorf("log of pod %q in namespace %q caused failure: %w", pod.GetName(), namespace, err)
			}
			return fmt.Errorf("log of pod %q in namespace %q does not satisfy the condition", pod.GetName(), namespace)
		}
		return nil
	}
}

// getLastLogBytePos will return the position of the last log byte for a particular pod.  This is intended to be used with
// the log matchers to ensure a log from previous tests isn't matched against.
func (ks *KubernetesSuite) getLastLogBytePos(ctx context.Context, namespace, pod, container string) (int64, error) {
	resp := ks.k8s.CoreV1().Pods(namespace).GetLogs(pod, &coreV1.PodLogOptions{Container: container}).Do(ctx)

	logB, err := resp.Raw()
	if err != nil {
		return 0, fmt.Errorf("retrieving logs of pod %q in namespace %q failed: %w", pod, namespace, err)
	}

	return int64(len(logB)), nil
}

// getSensorPod is a convenience method to get details of the Sensor pod.
func (ks *KubernetesSuite) getSensorPod(ctx context.Context, namespace string) (*coreV1.Pod, error) {
	ls := labels.SelectorFromSet(sensorPodLabels).String()
	podList, err := ks.k8s.CoreV1().Pods(namespace).List(ctx, metaV1.ListOptions{LabelSelector: ls})
	if err != nil {
		return nil, fmt.Errorf("could not list pods matching %q in namespace %q: %w", sensorPodLabels, namespace, err)
	}
	if len(podList.Items) == 0 {
		return nil, fmt.Errorf("empty list of pods does not satisfy the condition")
	}
	if len(podList.Items) > 1 {
		return nil, fmt.Errorf("more than one sensor pod running")
	}

	return &podList.Items[0], nil
}

// waitUntilK8sDeploymentReady waits until k8s reports all pods for a deployment are ready or context done.
func (ks *KubernetesSuite) waitUntilK8sDeploymentReady(ctx context.Context, namespace string, deploymentName string) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timer := time.NewTimer(waitTimeout)

	ks.logf("Waiting for deployment %q in namespace %q to be ready", deploymentName, namespace)
	for {
		select {
		case <-ctx.Done():
			require.NoError(ks.T(), ctx.Err())
		case <-ticker.C:
			deploy, err := ks.k8s.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metaV1.GetOptions{})
			require.NoError(ks.T(), err, "getting deployment %q from namespace %q", deploymentName, namespace)

			if deploy.GetGeneration() != deploy.Status.ObservedGeneration {
				ks.logf("deployment %q in namespace %q NOT ready, generation %d, observed generation %d", deploymentName, namespace, deploy.GetGeneration(), deploy.Status.ObservedGeneration)
				continue
			}

			if deploy.Status.Replicas == 0 || deploy.Status.Replicas != deploy.Status.ReadyReplicas {
				ks.logf("deployment %q in namespace %q NOT ready (%d/%d ready replicas)", deploymentName, namespace, deploy.Status.ReadyReplicas, deploy.Status.Replicas)
				continue
			}

			// Ensure all pods are from the current generation (no old pods during rollout).
			if deploy.Status.UpdatedReplicas > 0 && deploy.Status.UpdatedReplicas != deploy.Status.Replicas {
				ks.logf("deployment %q in namespace %q NOT ready, rollout incomplete (%d/%d updated replicas)", deploymentName, namespace, deploy.Status.UpdatedReplicas, deploy.Status.Replicas)
				continue
			}

			ks.logf("deployment %q in namespace %q READY (%d/%d ready replicas)", deploymentName, namespace, deploy.Status.ReadyReplicas, deploy.Status.Replicas)
			return
		case <-timer.C:
			ks.T().Fatalf("Timed out waiting for deployment %s", deploymentName)
		}
	}
}

// waitUntilK8sDeploymentGenerationReady waits until a deployment's specified generation is fully rolled out and ready.
func (ks *KubernetesSuite) waitUntilK8sDeploymentGenerationReady(ctx context.Context, namespace string, deploymentName string, targetGeneration int64) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timer := time.NewTimer(waitTimeout)
	defer timer.Stop()

	ks.logf("Waiting for deployment %q in namespace %q to reach generation %d", deploymentName, namespace, targetGeneration)
	for {
		select {
		case <-ctx.Done():
			require.NoError(ks.T(), ctx.Err())
		case <-ticker.C:
			deploy, err := ks.k8s.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metaV1.GetOptions{})
			require.NoError(ks.T(), err, "getting deployment %q from namespace %q", deploymentName, namespace)

			currentGen := deploy.GetGeneration()
			if currentGen >= targetGeneration {
				ks.logf("deployment %q in namespace %q reached generation %d", deploymentName, namespace, currentGen)
				ks.waitUntilK8sDeploymentReady(ctx, namespace, deploymentName)
				return
			}
			ks.logf("deployment %q in namespace %q waiting for generation update (current: %d, target: %d)", deploymentName, namespace, currentGen, targetGeneration)
		case <-timer.C:
			ks.T().Fatalf("Timed out waiting for deployment %s to reach generation %d", deploymentName, targetGeneration)
		}
	}
}

// mustEventually retries every pauseInterval until ctx expires or f succeeds.
func mustEventually(t *testing.T, ctx context.Context, f func() error, pauseInterval time.Duration, failureMsgPrefix string) {
	require.NoError(t, retry.WithRetry(f,
		retry.Tries(math.MaxInt),
		retry.WithContext(ctx),
		retry.BetweenAttempts(func(_ int) {
			time.Sleep(pauseInterval)
		}),
		retry.OnFailedAttempts(func(err error) { logf(t, failureMsgPrefix+": %s", err) })))
}

// createService creates a k8s Service object.
func (ks *KubernetesSuite) createService(ctx context.Context, namespace string, name string, labels map[string]string, ports map[int32]int32) {
	svc := &coreV1.Service{
		ObjectMeta: metaV1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Spec: coreV1.ServiceSpec{
			Selector: labels,
			Type:     coreV1.ServiceTypeClusterIP,
		},
	}
	for portNum, targetPortNum := range ports {
		svc.Spec.Ports = append(svc.Spec.Ports, coreV1.ServicePort{
			Name:       fmt.Sprintf("%d", portNum),
			Protocol:   coreV1.ProtocolTCP,
			Port:       portNum,
			TargetPort: intstr.IntOrString{IntVal: targetPortNum},
		})
	}
	_, err := ks.k8s.CoreV1().Services(namespace).Create(ctx, svc, metaV1.CreateOptions{})
	ks.Require().NoError(err, "cannot create service %q in namespace %q", name, namespace)
}

// ensureSecretExists creates a k8s Secret object. If one exists, it makes sure the type and data matches.
func (ks *KubernetesSuite) ensureSecretExists(ctx context.Context, namespace string, name string, secretType coreV1.SecretType, data map[string][]byte) {
	secret := &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name: name,
		},
		Type: secretType,
		Data: data,
	}
	if _, err := ks.k8s.CoreV1().Secrets(namespace).Create(ctx, secret, metaV1.CreateOptions{}); err != nil {
		if apiErrors.IsAlreadyExists(err) {
			actualSecret, err := ks.k8s.CoreV1().Secrets(namespace).Get(ctx, name, metaV1.GetOptions{})
			ks.Require().NoError(err, "secret %q in namespace %q already exists but cannot retrieve it for verification", name, namespace)
			ks.Equal(secretType, actualSecret.Type, "secret %q in namespace %q already exists but its type is not as expected", name, namespace)
			ks.Equal(data, actualSecret.Data, "secret %q in namespace %q already exists but its data is not as expected", name, namespace)
			ks.Require().False(ks.T().Failed(), "secrets do not match")
		}
		ks.Require().NoError(err, "cannot create secret %q in namespace %q", name, namespace)
	}
}

// ensureQuayImagePullSecretExists creates an image pull secret for quay.io using credentials from
// REGISTRY_USERNAME and REGISTRY_PASSWORD environment variables. This is a common pattern across e2e tests.
func (ks *KubernetesSuite) ensureQuayImagePullSecretExists(ctx context.Context, namespace string, secretName string) {
	configBytes, err := json.Marshal(config.DockerConfigJSON{
		Auths: map[string]config.DockerConfigEntry{
			"https://quay.io": {
				Username: mustGetEnv(ks.T(), "REGISTRY_USERNAME"),
				Password: mustGetEnv(ks.T(), "REGISTRY_PASSWORD"),
			},
		},
	})
	ks.Require().NoError(err, "cannot serialize docker config for image pull secret %q in namespace %q", secretName, namespace)
	ks.ensureSecretExists(ctx, namespace, secretName, coreV1.SecretTypeDockerConfigJson, map[string][]byte{coreV1.DockerConfigJsonKey: configBytes})
}

// ensureConfigMapExists creates a k8s ConfigMap object. If one exists, it makes sure the data matches.
func (ks *KubernetesSuite) ensureConfigMapExists(ctx context.Context, namespace string, name string, data map[string]string) {
	cm := &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name: name,
		},
		Data: data,
	}
	if _, err := ks.k8s.CoreV1().ConfigMaps(namespace).Create(ctx, cm, metaV1.CreateOptions{}); err != nil {
		if apiErrors.IsAlreadyExists(err) {
			actualCM, err := ks.k8s.CoreV1().ConfigMaps(namespace).Get(ctx, name, metaV1.GetOptions{})
			ks.Require().NoError(err, "configMap %q in namespace %q already exists but cannot retrieve it for verification", name, namespace)
			ks.Require().Equal(data, actualCM.Data, "configMap %q in namespace %q already exists but its data is not as expected", name, namespace)
		}
		ks.Require().NoError(err, "cannot create configMap %q in namespace %q", name, namespace)
	}
}

// waitUntilCentralSensorConnectionIs makes sure there is only one cluster defined on central, and its status is eventually
// one of the specified statuses.
func waitUntilCentralSensorConnectionIs(t *testing.T, ctx context.Context, statuses ...storage.ClusterHealthStatus_HealthStatusLabel) {
	logf(t, "Waiting until central-sensor connection is in state(s) %s...", statuses)
	conn := centralgrpc.GRPCConnectionToCentral(t)
	checkCentralSensorConnection := func() error {
		cluster, err := getCluster(ctx, conn)
		if err != nil {
			return err
		}

		clusterStatus := cluster.GetHealthStatus().GetSensorHealthStatus()
		for _, status := range statuses {
			if clusterStatus == status {
				logf(t, "Central-sensor connection now in state %q.", clusterStatus)
				return nil
			}
		}
		return fmt.Errorf("encountered cluster status %q", clusterStatus.String())
	}
	// TODO: Sensor reports status every 30 seconds, consider sleeping here to ensure we have the most up to date status.
	mustEventually(t, ctx, checkCentralSensorConnection, 10*time.Second, "Central-sensor connection not in desired state (yet)")
}

// mustSetDeploymentEnvVal sets the specified env variable on a container in a deployment using strategic merge patch, or fails the test.
func (ks *KubernetesSuite) mustSetDeploymentEnvVal(ctx context.Context, namespace string, deployment string, container string, envVar string, value string) *appsV1.Deployment {
	patch := []byte(fmt.Sprintf(`{"spec":{"template":{"spec":{"containers":[{"name":%q,"env":[{"name":%q,"value":%q}]}]}}}}`,
		container, envVar, value))
	whatVar := fmt.Sprintf("variable %q on deployment %q in namespace %q to %q", envVar, deployment, namespace, value)
	ks.logf("Setting %s", whatVar)
	var patchedDeploy *appsV1.Deployment
	mustEventually(ks.T(), ctx, func() error {
		var err error
		patchedDeploy, err = ks.k8s.AppsV1().Deployments(namespace).Patch(ctx, deployment, types.StrategicMergePatchType, patch, metaV1.PatchOptions{})
		return err
	}, 5*time.Second, fmt.Sprintf("cannot set %s", whatVar))
	return patchedDeploy
}

// mustGetDeploymentEnvVal retrieves the value of environment variable in a deployment, or fails the test.
func (ks *KubernetesSuite) mustGetDeploymentEnvVal(ctx context.Context, namespace string, deployment string, container string, envVar string) string {
	val, err := ks.getDeploymentEnvVal(ctx, namespace, deployment, container, envVar)
	ks.Require().NoError(err, "cannot find envVar %q in container %q in deployment %q in namespace %q", envVar, container, deployment, namespace)
	return val
}

// getDeploymentEnvVal returns the value of an environment variable or the empty string if not found.
// Fails the test if deployment or container are missing, or API call fails repeatedly.
// Please use mustGetDeploymentEnvVal instead, unless you must tolerate a missing env var definition in the container.
func (ks *KubernetesSuite) getDeploymentEnvVal(ctx context.Context, namespace string, deployment string, container string, envVar string) (string, error) {
	var d *appsV1.Deployment
	mustEventually(ks.T(), ctx, func() error {
		var err error
		d, err = ks.k8s.AppsV1().Deployments(namespace).Get(ctx, deployment, metaV1.GetOptions{})
		return err
	}, 5*time.Second, fmt.Sprintf("cannot retrieve deployment %q in namespace %q", deployment, namespace))
	c, err := getContainer(d, container)
	ks.Require().NoError(err, "cannot find container %q in deployment %q in namespace %q", container, deployment, namespace)
	return getEnvVal(c, envVar)
}

// mustDeleteDeploymentEnvVar deletes an env var from all containers of a deployment, if any errors
// are encountered the suite will fail. This is a no-op if the env var is not found.
func (ks *KubernetesSuite) mustDeleteDeploymentEnvVar(ctx context.Context, namespace, deployment, envVar string) {
	d, err := ks.k8s.AppsV1().Deployments(namespace).Get(ctx, deployment, metaV1.GetOptions{})
	ks.Require().NoError(err, "cannot retrieve deployment %q in namespace %q", deployment, namespace)

	sb := strings.Builder{}
	for i, c := range d.Spec.Template.Spec.Containers {
		for j, e := range c.Env {
			if e.Name == envVar {
				if sb.Len() > 0 {
					sb.WriteString(",")
				}
				sb.WriteString(fmt.Sprintf(`{"op":"remove", "path":"/spec/template/spec/containers/%d/env/%d"}`, i, j))
				break
			}
		}
	}

	patch := fmt.Sprintf("[%s]", sb.String())
	_, err = ks.k8s.AppsV1().Deployments(namespace).Patch(ctx, deployment, types.JSONPatchType, []byte(patch), metaV1.PatchOptions{})
	ks.Require().NoError(err, "failed to remove env var %q from deployment %q namespace %q", envVar, deployment, namespace)
	ks.logf("Removed env variable %q from deployment %q namespace %q", envVar, deployment, namespace)
}

// listK8sAPIResources will return a list of all custom resources the k8s (or OCP) API supports.
func (ks *KubernetesSuite) listK8sAPIResources() []*metaV1.APIResourceList {
	t := ks.T()

	apiResourceList, err := ks.k8s.Discovery().ServerPreferredResources()
	require.NoError(t, err)

	return apiResourceList
}

// mustCreateAPIToken will create an API token. Returns the token ID and value.
func mustCreateAPIToken(t *testing.T, ctx context.Context, name string, roles []string) (string, string) {
	require := require.New(t)
	conn := centralgrpc.GRPCConnectionToCentral(t)
	service := v1.NewAPITokenServiceClient(conn)

	tokResp, err := service.GenerateToken(ctx, &v1.GenerateTokenRequest{
		Name:  name,
		Roles: roles,
	})
	require.NoError(err)

	logf(t, "Created API Token %q (%v): %v", name, tokResp.GetMetadata().GetId(), roles)
	return tokResp.GetMetadata().GetId(), tokResp.GetToken()
}

// revokeAPIToken will revoke an API token.
func revokeAPIToken(t *testing.T, ctx context.Context, idOrName string) {
	conn := centralgrpc.GRPCConnectionToCentral(t)
	service := v1.NewAPITokenServiceClient(conn)

	toksResp, err := service.GetAPITokens(ctx, &v1.GetAPITokensRequest{
		RevokedOneof: &v1.GetAPITokensRequest_Revoked{Revoked: false},
	})
	require.NoError(t, err)

	for _, tok := range toksResp.GetTokens() {
		if tok.GetId() == idOrName || tok.GetName() == idOrName {
			_, err = service.RevokeToken(ctx, &v1.ResourceByID{Id: tok.GetId()})
			require.NoError(t, err)
			logf(t, "Revoked API token %q (%s)", tok.GetName(), tok.GetId())
			return
		}
	}
}

// mustCreatePermissionSet will create a permission set and return the associated ID.
func mustCreatePermissionSet(t *testing.T, ctx context.Context, permissionSet *storage.PermissionSet) string {
	conn := centralgrpc.GRPCConnectionToCentral(t)
	service := v1.NewRoleServiceClient(conn)

	ps, err := service.PostPermissionSet(ctx, permissionSet)
	require.NoError(t, err)

	logf(t, "Created permission set: %v", ps)
	return ps.GetId()
}

// deletePermissionSet will delete the specified permission set.
func deletePermissionSet(t *testing.T, ctx context.Context, idOrName string) {
	conn := centralgrpc.GRPCConnectionToCentral(t)
	service := v1.NewRoleServiceClient(conn)

	resp, err := service.ListPermissionSets(ctx, &v1.Empty{})
	require.NoError(t, err)

	for _, ps := range resp.GetPermissionSets() {
		if ps.GetId() == idOrName || ps.GetName() == idOrName {
			_, err = service.DeletePermissionSet(ctx, &v1.ResourceByID{Id: ps.GetId()})
			require.NoError(t, err)
			logf(t, "Deleted permission set %q (%s)", ps.GetName(), ps.GetId())
			return
		}
	}
}

// mustCreateRole will create a role.
func mustCreateRole(t *testing.T, ctx context.Context, role *storage.Role) {
	conn := centralgrpc.GRPCConnectionToCentral(t)
	service := v1.NewRoleServiceClient(conn)

	_, err := service.CreateRole(ctx, &v1.CreateRoleRequest{
		Name: role.GetName(),
		Role: role,
	})
	require.NoError(t, err)

	logf(t, "Created role: %v", role)
}

// deleteRole will delete the specified role.
func deleteRole(t *testing.T, ctx context.Context, name string) {
	conn := centralgrpc.GRPCConnectionToCentral(t)
	service := v1.NewRoleServiceClient(conn)

	resp, err := service.GetRoles(ctx, &v1.Empty{})
	require.NoError(t, err)

	for _, role := range resp.GetRoles() {
		if role.GetName() == name {
			_, err = service.DeleteRole(ctx, &v1.ResourceByID{Id: name})
			require.NoError(t, err)
			logf(t, "Deleted role %q", name)
			return
		}
	}
}

type EnvVarNotFound []string

func (e EnvVarNotFound) Error() string {
	return fmt.Sprintf("actual vars are: %q", []string(e))
}

func requireNoErrorOrEnvVarNotFound(t require.TestingT, err error) {
	if err == nil {
		return
	}
	require.ErrorAs(t, err, &EnvVarNotFound{})
}

// getEnvVal returns the value of envVar from a given container or returns a helpful error.
func getEnvVal(c *coreV1.Container, envVar string) (string, error) {
	var vars []string
	for _, v := range c.Env {
		if v.Name == envVar {
			return v.Value, nil
		}
		vars = append(vars, v.Name)
	}
	return "", EnvVarNotFound(vars)
}

// getContainer returns the given container from a deployment or returns a helpful error.
func getContainer(deployment *appsV1.Deployment, container string) (*coreV1.Container, error) {
	var containers []string
	for _, c := range deployment.Spec.Template.Spec.Containers {
		if c.Name == container {
			return &c, nil
		}
		containers = append(containers, c.Name)
	}
	return nil, fmt.Errorf("actual containers are %q", containers)
}

// collectLogs collects service logs from a given namespace into a subdirectory that will be gathered by the CI scripts.
func collectLogs(t *testing.T, ns string, dir string) {
	subDir := "e2e-test-logs/" + dir
	start := time.Now()
	err := exec.Command("../scripts/ci/collect-service-logs.sh", ns, "/tmp/"+subDir).Run()
	if err != nil {
		logf(t, "Collecting %q logs returned error %s", ns, err)
	}
	logf(t, "Collected logs from namespace %q to directory %q, took: %s", ns, subDir, time.Since(start))
}

// isOpenshift returns true when the test env is a flavor of OCP, false otherwise.
func isOpenshift() bool {
	return os.Getenv("ORCHESTRATOR_FLAVOR") == "openshift"
}

// mustGetCluster returns the details of the one and only known secured cluster.
func mustGetCluster(t *testing.T, ctx context.Context) *storage.Cluster {
	conn := centralgrpc.GRPCConnectionToCentral(t)

	cluster, err := getCluster(ctx, conn)
	require.NoError(t, err)

	return cluster
}

// getCluster returns the details of the one and only known secured cluster.
func getCluster(ctx context.Context, conn *grpc.ClientConn) (*storage.Cluster, error) {
	clustersSvc := v1.NewClustersServiceClient(conn)

	clusters, err := clustersSvc.GetClusters(ctx, &v1.GetClustersRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve clusters from central: %w", err)
	}

	if len(clusters.GetClusters()) != 1 {
		var clusterNames []string
		for _, cluster := range clusters.GetClusters() {
			clusterNames = append(clusterNames, cluster.GetName())
		}
		return nil, fmt.Errorf("expected one cluster, found %d: %+v", len(clusters.GetClusters()), clusterNames)
	}

	return clusters.GetClusters()[0], nil
}

type collectT struct {
	t *testing.T
	c *assert.CollectT
}

func (c *collectT) Fatalf(format string, args ...interface{}) {
	if c.t != nil {
		c.t.Fatalf(format, args...)
	}
}

func (c *collectT) Errorf(format string, args ...interface{}) {
	if c.c != nil {
		c.c.Errorf(format, args...)
	}
}

func (c *collectT) FailNow() {
	if c.c != nil {
		c.c.FailNow()
	}
}

func (c *collectT) Logf(format string, values ...interface{}) {
	if c.t != nil {
		c.t.Logf(format, values...)
	}
}

func wrapCollectT(t *testing.T, c *assert.CollectT) *collectT {
	return &collectT{
		t: t,
		c: c,
	}
}
