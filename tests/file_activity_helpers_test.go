//go:build test_e2e

package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

const (
	stackroxNamespace = "stackrox"
	factContainerName = "fact"
)

// skipIfNoFact skips the test if the Fact container is not running in the Collector DaemonSet.
func skipIfNoFact(t *testing.T) {
	skipIfNoCollection(t)

	client := createK8sClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pods, err := client.CoreV1().Pods(stackroxNamespace).List(ctx, metaV1.ListOptions{
		LabelSelector: "app=collector",
	})
	require.NoError(t, err, "listing collector pods")

	if len(pods.Items) == 0 {
		t.Skip("No collector pods found, skipping file activity test")
	}

	for _, pod := range pods.Items {
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.Name == factContainerName && cs.State.Running != nil {
				return
			}
		}
	}

	t.Skip("Fact container not found or not running in collector pods, skipping file activity test")
}

// waitForFactHealthy polls until all Fact containers in collector pods are running
// and the DaemonSet has the expected number of ready pods.
func waitForFactHealthy(t *testing.T) {
	client := createK8sClient(t)

	waitForCondition(t, func() bool {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		ds, err := client.AppsV1().DaemonSets(stackroxNamespace).Get(ctx, "collector", metaV1.GetOptions{})
		if err != nil {
			t.Logf("waiting for collector DaemonSet: %v", err)
			return false
		}

		if ds.Status.DesiredNumberScheduled == 0 {
			t.Log("collector DaemonSet has 0 desired pods")
			return false
		}

		if ds.Status.NumberReady != ds.Status.DesiredNumberScheduled {
			t.Logf("collector DaemonSet: %d/%d ready", ds.Status.NumberReady, ds.Status.DesiredNumberScheduled)
			return false
		}

		pods, err := client.CoreV1().Pods(stackroxNamespace).List(ctx, metaV1.ListOptions{
			LabelSelector: "app=collector",
		})
		if err != nil {
			t.Logf("listing collector pods: %v", err)
			return false
		}

		for _, pod := range pods.Items {
			found := false
			for _, cs := range pod.Status.ContainerStatuses {
				if cs.Name == factContainerName {
					if cs.State.Running == nil {
						t.Logf("Fact container in pod %s not running", pod.Name)
						return false
					}
					found = true
					break
				}
			}
			if !found {
				t.Logf("Fact container not found in pod %s", pod.Name)
				return false
			}
		}

		return true
	}, "Fact containers healthy", 5*time.Minute, 5*time.Second)
}

const factConfigMapName = "fact-config"

// ensureFactConfigMount creates the fact-config ConfigMap and patches the collector
// DaemonSet to mount it into the Fact container at /etc/stackrox/.
// This triggers a rolling restart; waits for all pods to be healthy afterwards.
func ensureFactConfigMount(t *testing.T) {
	client := createK8sClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create the ConfigMap (or update if it exists).
	cm := &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      factConfigMapName,
			Namespace: stackroxNamespace,
		},
		Data: map[string]string{
			"fact.yaml": "paths: []",
		},
	}

	existing, err := client.CoreV1().ConfigMaps(stackroxNamespace).Get(ctx, factConfigMapName, metaV1.GetOptions{})
	if err != nil {
		_, err = client.CoreV1().ConfigMaps(stackroxNamespace).Create(ctx, cm, metaV1.CreateOptions{})
		require.NoError(t, err, "creating fact-config ConfigMap")
		t.Log("Created fact-config ConfigMap")
	} else {
		existing.Data = cm.Data
		_, err = client.CoreV1().ConfigMaps(stackroxNamespace).Update(ctx, existing, metaV1.UpdateOptions{})
		require.NoError(t, err, "updating fact-config ConfigMap")
		t.Log("Updated fact-config ConfigMap")
	}

	// Patch the collector DaemonSet to add the volume and volumeMount.
	// We find the Fact container index first, then apply a strategic merge patch.
	ds, err := client.AppsV1().DaemonSets(stackroxNamespace).Get(ctx, "collector", metaV1.GetOptions{})
	require.NoError(t, err, "getting collector DaemonSet")

	factIdx := -1
	for i, c := range ds.Spec.Template.Spec.Containers {
		if c.Name == factContainerName {
			factIdx = i
			break
		}
	}
	require.NotEqual(t, -1, factIdx, "Fact container not found in collector DaemonSet")

	// Check if already mounted.
	for _, vm := range ds.Spec.Template.Spec.Containers[factIdx].VolumeMounts {
		if vm.Name == factConfigMapName {
			t.Log("fact-config volume mount already present, skipping patch")
			waitForFactHealthy(t)
			return
		}
	}

	// JSON patch to add volume and volumeMount.
	patch := []map[string]interface{}{
		{
			"op":   "add",
			"path": "/spec/template/spec/volumes/-",
			"value": map[string]interface{}{
				"name": factConfigMapName,
				"configMap": map[string]interface{}{
					"name": factConfigMapName,
				},
			},
		},
		{
			"op":   "add",
			"path": fmt.Sprintf("/spec/template/spec/containers/%d/volumeMounts/-", factIdx),
			"value": map[string]interface{}{
				"name":      factConfigMapName,
				"mountPath": "/etc/stackrox",
				"readOnly":  true,
			},
		},
	}

	patchBytes, err := json.Marshal(patch)
	require.NoError(t, err, "marshalling patch")

	_, err = client.AppsV1().DaemonSets(stackroxNamespace).Patch(
		ctx, "collector", types.JSONPatchType, patchBytes, metaV1.PatchOptions{})
	require.NoError(t, err, "patching collector DaemonSet with fact-config mount")
	t.Log("Patched collector DaemonSet with fact-config volume mount")

	waitForFactHealthy(t)
}

// configureFactPaths updates the fact-config ConfigMap with the given paths.
// Fact's hotreloader picks up changes within 10 seconds; waits 15s for safety.
func configureFactPaths(t *testing.T, paths []string) {
	client := createK8sClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Build YAML content: paths: [/tmp/**/*, /var/**/*]
	quotedPaths := make([]string, len(paths))
	for i, p := range paths {
		quotedPaths[i] = fmt.Sprintf("%q", p)
	}
	yamlContent := fmt.Sprintf("paths: [%s]", strings.Join(quotedPaths, ", "))

	cm, err := client.CoreV1().ConfigMaps(stackroxNamespace).Get(ctx, factConfigMapName, metaV1.GetOptions{})
	require.NoError(t, err, "getting fact-config ConfigMap")

	cm.Data["fact.yaml"] = yamlContent
	_, err = client.CoreV1().ConfigMaps(stackroxNamespace).Update(ctx, cm, metaV1.UpdateOptions{})
	require.NoError(t, err, "updating fact-config ConfigMap with paths")
	t.Logf("Configured Fact paths: %s", yamlContent)

	// Wait for Fact hotreloader to pick up changes (10s poll interval + margin).
	time.Sleep(15 * time.Second)
}

// cleanupFactConfig removes the fact-config volumeMount from the collector DaemonSet
// and deletes the ConfigMap.
func cleanupFactConfig(t *testing.T) {
	client := createK8sClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ds, err := client.AppsV1().DaemonSets(stackroxNamespace).Get(ctx, "collector", metaV1.GetOptions{})
	if err != nil {
		t.Logf("Warning: failed to get collector DaemonSet for cleanup: %v", err)
		return
	}

	// Find Fact container and remove the volume mount.
	needsPatch := false
	for i, c := range ds.Spec.Template.Spec.Containers {
		if c.Name == factContainerName {
			var filtered []coreV1.VolumeMount
			for _, vm := range c.VolumeMounts {
				if vm.Name != factConfigMapName {
					filtered = append(filtered, vm)
				} else {
					needsPatch = true
				}
			}
			ds.Spec.Template.Spec.Containers[i].VolumeMounts = filtered
			break
		}
	}

	// Remove the volume.
	var filteredVolumes []coreV1.Volume
	for _, v := range ds.Spec.Template.Spec.Volumes {
		if v.Name != factConfigMapName {
			filteredVolumes = append(filteredVolumes, v)
		} else {
			needsPatch = true
		}
	}
	ds.Spec.Template.Spec.Volumes = filteredVolumes

	if needsPatch {
		_, err = client.AppsV1().DaemonSets(stackroxNamespace).Update(ctx, ds, metaV1.UpdateOptions{})
		if err != nil {
			t.Logf("Warning: failed to remove fact-config from DaemonSet: %v", err)
		} else {
			t.Log("Removed fact-config volume mount from collector DaemonSet")
			waitForFactHealthy(t)
		}
	}

	// Delete the ConfigMap.
	err = client.CoreV1().ConfigMaps(stackroxNamespace).Delete(ctx, factConfigMapName, metaV1.DeleteOptions{})
	if err != nil {
		t.Logf("Warning: failed to delete fact-config ConfigMap: %v", err)
	} else {
		t.Log("Deleted fact-config ConfigMap")
	}
}

// execInPod runs a command inside a pod container and returns stdout and stderr.
func execInPod(t *testing.T, k8sClient kubernetes.Interface, namespace, podName, containerName string, command []string) (string, string) {
	t.Helper()

	config := getConfig(t)

	req := k8sClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&coreV1.PodExecOptions{
			Container: containerName,
			Command:   command,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	require.NoError(t, err, "creating SPDY executor")

	var stdout, stderr bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		t.Logf("execInPod command %v failed: %v\nstdout: %s\nstderr: %s", command, err, stdout.String(), stderr.String())
	}
	require.NoError(t, err, "executing command in pod %s/%s", namespace, podName)

	return stdout.String(), stderr.String()
}

// createFileActivityPolicy builds a storage.Policy for file activity detection.
// If operation is empty, no operation criterion is added (matches any operation).
// If negateOp is true, the operation criterion is negated.
func createFileActivityPolicy(name, path, operation string, eventSource storage.EventSource, negateOp bool) *storage.Policy {
	groups := []*storage.PolicyGroup{
		{
			FieldName: fieldnames.FilePath,
			Values:    []*storage.PolicyValue{{Value: path}},
		},
	}

	if operation != "" {
		groups = append(groups, &storage.PolicyGroup{
			FieldName: fieldnames.FileOperation,
			Negate:    negateOp,
			Values:    []*storage.PolicyValue{{Value: operation}},
		})
	}

	return &storage.Policy{
		Name:            name,
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		EventSource:     eventSource,
		Severity:        storage.Severity_HIGH_SEVERITY,
		Categories:      []string{"File Activity Monitoring"},
		PolicySections: []*storage.PolicySection{{
			SectionName:  "file-access",
			PolicyGroups: groups,
		}},
	}
}

// createFileActivityPolicyMultiOps builds a policy with multiple operations (OR logic).
func createFileActivityPolicyMultiOps(name, path string, operations []string, eventSource storage.EventSource) *storage.Policy {
	values := make([]*storage.PolicyValue, len(operations))
	for i, op := range operations {
		values[i] = &storage.PolicyValue{Value: op}
	}

	return &storage.Policy{
		Name:            name,
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		EventSource:     eventSource,
		Severity:        storage.Severity_HIGH_SEVERITY,
		Categories:      []string{"File Activity Monitoring"},
		PolicySections: []*storage.PolicySection{{
			SectionName: "file-access",
			PolicyGroups: []*storage.PolicyGroup{
				{
					FieldName: fieldnames.FilePath,
					Values:    []*storage.PolicyValue{{Value: path}},
				},
				{
					FieldName: fieldnames.FileOperation,
					Values:    values,
				},
			},
		}},
	}
}

// importAndCleanupPolicy creates a policy via the API and returns its ID and a cleanup function.
func importAndCleanupPolicy(t *testing.T, policy *storage.Policy, policyService v1.PolicyServiceClient) (string, func()) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := policyService.PostPolicy(ctx, &v1.PostPolicyRequest{
		Policy:                 policy,
		EnableStrictValidation: false,
	})
	require.NoError(t, err, "creating policy %q", policy.GetName())

	policyID := resp.GetId()
	t.Logf("Created policy %q with ID %s", policy.GetName(), policyID)

	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, err := policyService.DeletePolicy(ctx, &v1.ResourceByID{Id: policyID})
		if err != nil {
			t.Logf("Warning: failed to delete policy %s: %v", policyID, err)
		}
	}

	return policyID, cleanup
}

// assertNoAlert verifies that no alerts exist matching the given query.
// Waits a settling period, then polls to confirm no alert appears.
func assertNoAlert(t *testing.T, alertService v1.AlertServiceClient, req *v1.ListAlertsRequest) {
	t.Helper()

	// Wait for events to propagate through the pipeline
	time.Sleep(10 * time.Second)

	// Verify no alert exists
	testutils.Retry(t, 10, 2*time.Second, func(retryT testutils.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		resp, err := alertService.ListAlerts(ctx, req)
		require.NoError(retryT, err)
		assert.Empty(retryT, resp.GetAlerts(),
			"expected no alerts but found %d for query %q", len(resp.GetAlerts()), req.GetQuery())
	})
}

// getAlertWithViolations fetches the full alert by ID and returns it.
func getAlertWithViolations(t *testing.T, alertService v1.AlertServiceClient, alertID string) *storage.Alert {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	alert, err := alertService.GetAlert(ctx, &v1.ResourceByID{Id: alertID})
	require.NoError(t, err, "fetching alert %s", alertID)

	return alert
}

// findAlerts returns all alerts matching the given deployment and policy name.
func findAlerts(t *testing.T, alertService v1.AlertServiceClient, deploymentName, policyName string) []*storage.ListAlert {
	t.Helper()

	qb := search.NewQueryBuilder().
		AddStrings(search.DeploymentName, deploymentName).
		AddStrings(search.PolicyName, policyName).
		AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String())

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	resp, err := alertService.ListAlerts(ctx, &v1.ListAlertsRequest{Query: qb.Query()})
	require.NoError(t, err)

	return resp.GetAlerts()
}

// fileActivityTestSetup contains shared test infrastructure.
type fileActivityTestSetup struct {
	conn          *grpc.ClientConn
	policyService v1.PolicyServiceClient
	alertService  v1.AlertServiceClient
	k8sClient     kubernetes.Interface
}

// newFileActivityTestSetup creates the shared test setup.
func newFileActivityTestSetup(t *testing.T) *fileActivityTestSetup {
	conn := centralgrpc.GRPCConnectionToCentral(t)
	return &fileActivityTestSetup{
		conn:          conn,
		policyService: v1.NewPolicyServiceClient(conn),
		alertService:  v1.NewAlertServiceClient(conn),
		k8sClient:     createK8sClient(t),
	}
}

// uniquePath returns a unique file path for a subtest to avoid cross-contamination.
func uniquePath(prefix string) string {
	return fmt.Sprintf("/tmp/%s-%s", prefix, uuid.NewV4().String()[:8])
}

// buildAlertQuery builds a ListAlertsRequest for a deployment and policy name.
func buildAlertQuery(deploymentName, policyName string) *v1.ListAlertsRequest {
	qb := search.NewQueryBuilder().
		AddStrings(search.DeploymentName, deploymentName).
		AddStrings(search.PolicyName, policyName).
		AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String())
	return &v1.ListAlertsRequest{Query: qb.Query()}
}
