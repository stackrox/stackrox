//go:build test_e2e

package tests

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	busyboxImage       = "quay.io/rhacs-eng/qa-multi-arch:busybox"
	busyboxContainer   = "busybox"
	fileActivityTestNS = "default"
)

func TestFileActivity(t *testing.T) {
	skipIfNoFact(t)

	setup := newFileActivityTestSetup(t)

	// Configure Fact to monitor /tmp paths.
	ensureFactConfigMount(t)
	t.Cleanup(func() { cleanupFactConfig(t) })
	configureFactPaths(t, []string{"/tmp/**/*"})

	// Create shared busybox deployment.
	deploymentName := "fa-test-" + uuid.NewV4().String()[:8]
	setupDeploymentInNamespace(t, busyboxImage, deploymentName, fileActivityTestNS)
	t.Cleanup(func() { teardownDeploymentWithoutCheck(t, deploymentName, fileActivityTestNS) })

	podName := getDeploymentPodName(t, setup, deploymentName)

	t.Run("Group1_BasicOperations", func(t *testing.T) {
		testCases := []struct {
			name      string
			operation string
			setup     func(path string)
			command   func(path string) []string
		}{
			{
				name:      "CREATE",
				operation: "CREATE",
				command:   func(path string) []string { return []string{"touch", path} },
			},
			{
				name:      "OPEN",
				operation: "OPEN",
				setup: func(path string) {
					execInPod(t, setup.k8sClient, fileActivityTestNS, podName, busyboxContainer, []string{"touch", path})
				},
				command: func(path string) []string { return []string{"sh", "-c", "echo data >> " + path} },
			},
			{
				name:      "UNLINK",
				operation: "UNLINK",
				setup: func(path string) {
					execInPod(t, setup.k8sClient, fileActivityTestNS, podName, busyboxContainer, []string{"touch", path})
				},
				command: func(path string) []string { return []string{"rm", path} },
			},
			{
				name:      "RENAME",
				operation: "RENAME",
				setup: func(path string) {
					execInPod(t, setup.k8sClient, fileActivityTestNS, podName, busyboxContainer, []string{"touch", path})
				},
				command: func(path string) []string { return []string{"mv", path, path + ".renamed"} },
			},
			{
				name:      "PERMISSION_CHANGE",
				operation: "PERMISSION_CHANGE",
				setup: func(path string) {
					execInPod(t, setup.k8sClient, fileActivityTestNS, podName, busyboxContainer, []string{"touch", path})
				},
				command: func(path string) []string { return []string{"chmod", "777", path} },
			},
			{
				name:      "OWNERSHIP_CHANGE",
				operation: "OWNERSHIP_CHANGE",
				setup: func(path string) {
					execInPod(t, setup.k8sClient, fileActivityTestNS, podName, busyboxContainer, []string{"touch", path})
				},
				command: func(path string) []string { return []string{"chown", "nobody", path} },
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				path := uniquePath("basic-" + tc.name)
				policyName := "FA-E2E-" + tc.name + "-" + path[len("/tmp/"):]

				policy := createFileActivityPolicy(policyName, path, tc.operation,
					storage.EventSource_DEPLOYMENT_EVENT, false)
				_, cleanup := importAndCleanupPolicy(t, policy, setup.policyService)
				t.Cleanup(cleanup)

				if tc.setup != nil {
					tc.setup(path)
				}

				execInPod(t, setup.k8sClient, fileActivityTestNS, podName, busyboxContainer, tc.command(path))

				req := buildAlertQuery(deploymentName, policyName)
				waitForAlert(t, setup.alertService, req, 1)
			})
		}
	})

	t.Run("Group2_PathMatching", func(t *testing.T) {
		t.Run("ExactPath", func(t *testing.T) {
			path := uniquePath("exact")
			policyName := "FA-E2E-exact-" + path[len("/tmp/"):]

			policy := createFileActivityPolicy(policyName, path, "CREATE",
				storage.EventSource_DEPLOYMENT_EVENT, false)
			_, cleanup := importAndCleanupPolicy(t, policy, setup.policyService)
			t.Cleanup(cleanup)

			execInPod(t, setup.k8sClient, fileActivityTestNS, podName, busyboxContainer, []string{"touch", path})

			req := buildAlertQuery(deploymentName, policyName)
			waitForAlert(t, setup.alertService, req, 1)
		})

		t.Run("Wildcard", func(t *testing.T) {
			prefix := uniquePath("wild")
			policyPath := prefix + "-*"
			touchPath := prefix + "-test"
			policyName := "FA-E2E-wild-" + prefix[len("/tmp/"):]

			policy := createFileActivityPolicy(policyName, policyPath, "CREATE",
				storage.EventSource_DEPLOYMENT_EVENT, false)
			_, cleanup := importAndCleanupPolicy(t, policy, setup.policyService)
			t.Cleanup(cleanup)

			execInPod(t, setup.k8sClient, fileActivityTestNS, podName, busyboxContainer, []string{"touch", touchPath})

			req := buildAlertQuery(deploymentName, policyName)
			waitForAlert(t, setup.alertService, req, 1)
		})

		t.Run("Globstar", func(t *testing.T) {
			dirPrefix := uniquePath("glob")
			policyPath := dirPrefix + "/**/deep-file"
			touchPath := dirPrefix + "/a/b/deep-file"
			policyName := "FA-E2E-glob-" + dirPrefix[len("/tmp/"):]

			policy := createFileActivityPolicy(policyName, policyPath, "CREATE",
				storage.EventSource_DEPLOYMENT_EVENT, false)
			_, cleanup := importAndCleanupPolicy(t, policy, setup.policyService)
			t.Cleanup(cleanup)

			execInPod(t, setup.k8sClient, fileActivityTestNS, podName, busyboxContainer,
				[]string{"sh", "-c", fmt.Sprintf("mkdir -p %s/a/b && touch %s", dirPrefix, touchPath)})

			req := buildAlertQuery(deploymentName, policyName)
			waitForAlert(t, setup.alertService, req, 1)
		})

		t.Run("NonMatching", func(t *testing.T) {
			prefix := uniquePath("monitored")
			policyPath := prefix + "-*"
			unmonitoredPath := uniquePath("unmonitored")
			policyName := "FA-E2E-nomatch-" + prefix[len("/tmp/"):]

			policy := createFileActivityPolicy(policyName, policyPath, "CREATE",
				storage.EventSource_DEPLOYMENT_EVENT, false)
			_, cleanup := importAndCleanupPolicy(t, policy, setup.policyService)
			t.Cleanup(cleanup)

			execInPod(t, setup.k8sClient, fileActivityTestNS, podName, busyboxContainer, []string{"touch", unmonitoredPath})

			req := buildAlertQuery(deploymentName, policyName)
			assertNoAlert(t, setup.alertService, req)
		})
	})

	t.Run("Group3_PolicyLogic", func(t *testing.T) {
		t.Run("MultipleOperationsOR", func(t *testing.T) {
			path := uniquePath("multi-op")
			policyName := "FA-E2E-multi-" + path[len("/tmp/"):]

			policy := createFileActivityPolicyMultiOps(policyName, path,
				[]string{"CREATE", "UNLINK"}, storage.EventSource_DEPLOYMENT_EVENT)
			_, cleanup := importAndCleanupPolicy(t, policy, setup.policyService)
			t.Cleanup(cleanup)

			// Touch (CREATE) then rm (UNLINK).
			execInPod(t, setup.k8sClient, fileActivityTestNS, podName, busyboxContainer, []string{"touch", path})
			execInPod(t, setup.k8sClient, fileActivityTestNS, podName, busyboxContainer, []string{"rm", path})

			req := buildAlertQuery(deploymentName, policyName)
			waitForAlert(t, setup.alertService, req, 1)

			// Verify the alert has at least 2 violations (CREATE + UNLINK).
			alerts := findAlerts(t, setup.alertService, deploymentName, policyName)
			require.Len(t, alerts, 1)
			alert := getAlertWithViolations(t, setup.alertService, alerts[0].GetId())
			assert.GreaterOrEqual(t, len(alert.GetViolations()), 2,
				"expected at least 2 violations (CREATE + UNLINK)")
		})

		t.Run("NegatedOperation", func(t *testing.T) {
			path := uniquePath("negate")
			policyName := "FA-E2E-negate-" + path[len("/tmp/"):]

			// Policy: NOT OPEN — should match CREATE but not OPEN.
			policy := createFileActivityPolicy(policyName, path, "OPEN",
				storage.EventSource_DEPLOYMENT_EVENT, true)
			_, cleanup := importAndCleanupPolicy(t, policy, setup.policyService)
			t.Cleanup(cleanup)

			// Touch creates the file (CREATE — should trigger).
			execInPod(t, setup.k8sClient, fileActivityTestNS, podName, busyboxContainer, []string{"touch", path})

			req := buildAlertQuery(deploymentName, policyName)
			waitForAlert(t, setup.alertService, req, 1)

			// Open the file for writing (OPEN — should NOT add new violation).
			execInPod(t, setup.k8sClient, fileActivityTestNS, podName, busyboxContainer,
				[]string{"sh", "-c", "echo data >> " + path})

			// Short wait, then verify still exactly 1 alert.
			time.Sleep(5 * time.Second)
			alerts := findAlerts(t, setup.alertService, deploymentName, policyName)
			require.Len(t, alerts, 1)
		})

		t.Run("PathOnlyPolicy", func(t *testing.T) {
			path := uniquePath("pathonly")
			policyName := "FA-E2E-pathonly-" + path[len("/tmp/"):]

			// No operation criterion — matches any operation.
			policy := createFileActivityPolicy(policyName, path, "",
				storage.EventSource_DEPLOYMENT_EVENT, false)
			_, cleanup := importAndCleanupPolicy(t, policy, setup.policyService)
			t.Cleanup(cleanup)

			execInPod(t, setup.k8sClient, fileActivityTestNS, podName, busyboxContainer, []string{"touch", path})

			req := buildAlertQuery(deploymentName, policyName)
			waitForAlert(t, setup.alertService, req, 1)
		})
	})

	t.Run("Group4_ViolationDetails", func(t *testing.T) {
		path := uniquePath("details")
		policyName := "FA-E2E-details-" + path[len("/tmp/"):]

		policy := createFileActivityPolicy(policyName, path, "CREATE",
			storage.EventSource_DEPLOYMENT_EVENT, false)
		_, cleanup := importAndCleanupPolicy(t, policy, setup.policyService)
		t.Cleanup(cleanup)

		execInPod(t, setup.k8sClient, fileActivityTestNS, podName, busyboxContainer, []string{"touch", path})

		req := buildAlertQuery(deploymentName, policyName)
		waitForAlert(t, setup.alertService, req, 1)

		alerts := findAlerts(t, setup.alertService, deploymentName, policyName)
		require.Len(t, alerts, 1)

		alert := getAlertWithViolations(t, setup.alertService, alerts[0].GetId())
		require.NotEmpty(t, alert.GetViolations(), "expected at least one violation")

		v := alert.GetViolations()[0]
		assert.Equal(t, storage.Alert_Violation_FILE_ACCESS, v.GetType(),
			"violation type should be FILE_ACCESS")
		assert.Contains(t, v.GetMessage(), path,
			"violation message should contain the file path")
		assert.Contains(t, strings.ToLower(v.GetMessage()), "created",
			"violation message should mention 'created'")

		fa := v.GetFileAccess()
		require.NotNil(t, fa, "FileAccess should be non-nil")
		assert.Equal(t, storage.FileAccess_CREATE, fa.GetOperation(),
			"operation should be CREATE")
		assert.Contains(t, fa.GetFile().GetEffectivePath(), path,
			"effective path should contain the test path")
	})

	t.Run("Group5_NegativeCases", func(t *testing.T) {
		t.Run("ReadOnlyOpen", func(t *testing.T) {
			path := uniquePath("readonly")
			policyName := "FA-E2E-readonly-" + path[len("/tmp/"):]

			policy := createFileActivityPolicy(policyName, path, "OPEN",
				storage.EventSource_DEPLOYMENT_EVENT, false)
			_, cleanup := importAndCleanupPolicy(t, policy, setup.policyService)
			t.Cleanup(cleanup)

			// Create the file, then read it (read-only open should not trigger OPEN).
			execInPod(t, setup.k8sClient, fileActivityTestNS, podName, busyboxContainer, []string{"touch", path})
			// Wait for any CREATE alert to settle before testing read-only.
			time.Sleep(5 * time.Second)
			execInPod(t, setup.k8sClient, fileActivityTestNS, podName, busyboxContainer, []string{"cat", path})

			req := buildAlertQuery(deploymentName, policyName)
			assertNoAlert(t, setup.alertService, req)
		})

		t.Run("DisabledPolicy", func(t *testing.T) {
			path := uniquePath("disabled")
			policyName := "FA-E2E-disabled-" + path[len("/tmp/"):]

			policy := createFileActivityPolicy(policyName, path, "CREATE",
				storage.EventSource_DEPLOYMENT_EVENT, false)
			policy.Disabled = true
			_, cleanup := importAndCleanupPolicy(t, policy, setup.policyService)
			t.Cleanup(cleanup)

			execInPod(t, setup.k8sClient, fileActivityTestNS, podName, busyboxContainer, []string{"touch", path})

			req := buildAlertQuery(deploymentName, policyName)
			assertNoAlert(t, setup.alertService, req)
		})
	})

	t.Run("Group6_NodeLevel", func(t *testing.T) {
		// Create privileged pod for nsenter-based host commands.
		hostPodName := "fa-host-exec-" + uuid.NewV4().String()[:8]
		createHostExecPod(t, setup, hostPodName)
		t.Cleanup(func() { deleteHostExecPod(t, setup, hostPodName) })

		nsenter := func(cmd string) {
			execInPod(t, setup.k8sClient, fileActivityTestNS, hostPodName, "nsenter",
				[]string{"nsenter", "--target", "1", "--mount", "--", "sh", "-c", cmd})
		}

		t.Run("NodeCREATE", func(t *testing.T) {
			path := uniquePath("node-create")
			policyName := "FA-E2E-node-create-" + path[len("/tmp/"):]

			policy := createFileActivityPolicy(policyName, path, "CREATE",
				storage.EventSource_NODE_EVENT, false)
			_, cleanup := importAndCleanupPolicy(t, policy, setup.policyService)
			t.Cleanup(cleanup)
			t.Cleanup(func() { nsenter("rm -f " + path) })

			nsenter("touch " + path)

			req := buildNodeAlertQuery(policyName)
			waitForAlert(t, setup.alertService, req, 1)
		})

		t.Run("NodeUNLINK", func(t *testing.T) {
			path := uniquePath("node-unlink")
			policyName := "FA-E2E-node-unlink-" + path[len("/tmp/"):]

			policy := createFileActivityPolicy(policyName, path, "UNLINK",
				storage.EventSource_NODE_EVENT, false)
			_, cleanup := importAndCleanupPolicy(t, policy, setup.policyService)
			t.Cleanup(cleanup)

			nsenter("touch " + path)
			nsenter("rm " + path)

			req := buildNodeAlertQuery(policyName)
			waitForAlert(t, setup.alertService, req, 1)
		})

		t.Run("NodeVsDeploymentDistinction", func(t *testing.T) {
			prefix := uniquePath("distinct")
			policyPath := prefix + "-*"
			nodePolicyName := "FA-E2E-node-distinct-" + prefix[len("/tmp/"):]
			deployPolicyName := "FA-E2E-deploy-distinct-" + prefix[len("/tmp/"):]

			nodePolicy := createFileActivityPolicy(nodePolicyName, policyPath, "CREATE",
				storage.EventSource_NODE_EVENT, false)
			_, cleanupNode := importAndCleanupPolicy(t, nodePolicy, setup.policyService)
			t.Cleanup(cleanupNode)

			deployPolicy := createFileActivityPolicy(deployPolicyName, policyPath, "CREATE",
				storage.EventSource_DEPLOYMENT_EVENT, false)
			_, cleanupDeploy := importAndCleanupPolicy(t, deployPolicy, setup.policyService)
			t.Cleanup(cleanupDeploy)

			// Node-level touch should only trigger the node policy.
			nodeFile := prefix + "-node"
			t.Cleanup(func() { nsenter("rm -f " + nodeFile) })
			nsenter("touch " + nodeFile)

			nodeReq := buildNodeAlertQuery(nodePolicyName)
			waitForAlert(t, setup.alertService, nodeReq, 1)

			// Deployment-level touch should only trigger the deployment policy.
			deployFile := prefix + "-deploy"
			execInPod(t, setup.k8sClient, fileActivityTestNS, podName, busyboxContainer, []string{"touch", deployFile})

			deployReq := buildAlertQuery(deploymentName, deployPolicyName)
			waitForAlert(t, setup.alertService, deployReq, 1)

			// Verify no cross-contamination: node policy should not get deployment alerts.
			deployNodeReq := buildAlertQuery(deploymentName, nodePolicyName)
			assertNoAlert(t, setup.alertService, deployNodeReq)
		})
	})
}

// getDeploymentPodName finds a running pod for the given deployment.
func getDeploymentPodName(t *testing.T, setup *fileActivityTestSetup, deploymentName string) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pods, err := setup.k8sClient.CoreV1().Pods(fileActivityTestNS).List(ctx, metaV1.ListOptions{
		LabelSelector: "app=" + deploymentName,
	})
	require.NoError(t, err, "listing pods for deployment %s", deploymentName)
	require.NotEmpty(t, pods.Items, "no pods found for deployment %s", deploymentName)

	for _, pod := range pods.Items {
		if pod.Status.Phase == coreV1.PodRunning {
			return pod.Name
		}
	}

	t.Fatalf("no running pod found for deployment %s", deploymentName)
	return ""
}

// createHostExecPod creates a privileged pod with hostPID for nsenter commands.
func createHostExecPod(t *testing.T, setup *fileActivityTestSetup, name string) {
	t.Helper()

	privileged := true
	hostPID := true
	pod := &coreV1.Pod{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: fileActivityTestNS,
		},
		Spec: coreV1.PodSpec{
			HostPID: hostPID,
			Containers: []coreV1.Container{
				{
					Name:    "nsenter",
					Image:   busyboxImage,
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

	_, err := setup.k8sClient.CoreV1().Pods(fileActivityTestNS).Create(ctx, pod, metaV1.CreateOptions{})
	require.NoError(t, err, "creating host exec pod %s", name)

	// Wait for pod to be running.
	waitForCondition(t, func() bool {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		p, err := setup.k8sClient.CoreV1().Pods(fileActivityTestNS).Get(ctx, name, metaV1.GetOptions{})
		if err != nil {
			return false
		}
		return p.Status.Phase == coreV1.PodRunning
	}, "host exec pod running", 2*time.Minute, 5*time.Second)
}

// deleteHostExecPod removes the privileged pod.
func deleteHostExecPod(t *testing.T, setup *fileActivityTestSetup, name string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := setup.k8sClient.CoreV1().Pods(fileActivityTestNS).Delete(ctx, name, metaV1.DeleteOptions{})
	if err != nil {
		t.Logf("Warning: failed to delete host exec pod %s: %v", name, err)
	}
}

// buildNodeAlertQuery builds a ListAlertsRequest for node-level alerts by policy name.
func buildNodeAlertQuery(policyName string) *v1.ListAlertsRequest {
	qb := search.NewQueryBuilder().
		AddStrings(search.PolicyName, policyName).
		AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String()).
		AddStrings(search.EntityType, storage.Alert_NODE.String())
	return &v1.ListAlertsRequest{Query: qb.Query()}
}
