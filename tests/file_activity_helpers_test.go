//go:build test_e2e

package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

const factContainerName = "fact"

// skipIfNoFact skips the test if the Fact container is not running in the Collector DaemonSet.
func skipIfNoFact(t *testing.T) {
	skipIfNoCollection(t)

	client := createK8sClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pods, err := client.CoreV1().Pods(namespaces.StackRox).List(ctx, metaV1.ListOptions{
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

// patchFactPaths appends /tmp/**/* to the FACT_PATHS env var on the Fact container
// in the collector DaemonSet. Returns a restore function that reverts the change.
func patchFactPaths(t *testing.T, client kubernetes.Interface) func() {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ds, err := client.AppsV1().DaemonSets(namespaces.StackRox).Get(ctx, "collector", metaV1.GetOptions{})
	require.NoError(t, err, "getting collector DaemonSet")

	factIdx := -1
	for i, c := range ds.Spec.Template.Spec.Containers {
		if c.Name == factContainerName {
			factIdx = i
			break
		}
	}
	require.NotEqual(t, -1, factIdx, "Fact container not found in collector DaemonSet")

	// Find current FACT_PATHS value.
	envIdx := -1
	originalValue := ""
	for i, env := range ds.Spec.Template.Spec.Containers[factIdx].Env {
		if env.Name == "FACT_PATHS" {
			envIdx = i
			originalValue = env.Value
			break
		}
	}

	const tmpGlob = "/tmp/**/*"

	if strings.Contains(originalValue, tmpGlob) {
		t.Log("FACT_PATHS already contains /tmp/**/*")
		return func() {}
	}

	newValue := originalValue
	if newValue == "" {
		newValue = tmpGlob
	} else {
		newValue = newValue + ":" + tmpGlob
	}

	var patch []map[string]interface{}
	if envIdx >= 0 {
		patch = []map[string]interface{}{
			{
				"op":    "replace",
				"path":  fmt.Sprintf("/spec/template/spec/containers/%d/env/%d/value", factIdx, envIdx),
				"value": newValue,
			},
		}
	} else {
		patch = []map[string]interface{}{
			{
				"op":   "add",
				"path": fmt.Sprintf("/spec/template/spec/containers/%d/env/-", factIdx),
				"value": map[string]interface{}{
					"name":  "FACT_PATHS",
					"value": newValue,
				},
			},
		}
	}

	patchBytes, err := json.Marshal(patch)
	require.NoError(t, err, "marshalling patch")

	_, err = client.AppsV1().DaemonSets(namespaces.StackRox).Patch(
		ctx, "collector", types.JSONPatchType, patchBytes, metaV1.PatchOptions{})
	require.NoError(t, err, "patching collector DaemonSet FACT_PATHS")
	t.Logf("Patched FACT_PATHS to %q", newValue)

	waitForCollectorReady(t, client)

	restore := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		var restorePatch []map[string]interface{}
		if envIdx >= 0 {
			restorePatch = []map[string]interface{}{
				{
					"op":    "replace",
					"path":  fmt.Sprintf("/spec/template/spec/containers/%d/env/%d/value", factIdx, envIdx),
					"value": originalValue,
				},
			}
		} else {
			// Remove the env var we added (it was last).
			ds, err := client.AppsV1().DaemonSets(namespaces.StackRox).Get(ctx, "collector", metaV1.GetOptions{})
			if err != nil {
				t.Logf("Warning: failed to get DaemonSet for FACT_PATHS restore: %v", err)
				return
			}
			newEnvIdx := -1
			for i, env := range ds.Spec.Template.Spec.Containers[factIdx].Env {
				if env.Name == "FACT_PATHS" {
					newEnvIdx = i
					break
				}
			}
			if newEnvIdx < 0 {
				return
			}
			restorePatch = []map[string]interface{}{
				{
					"op":   "remove",
					"path": fmt.Sprintf("/spec/template/spec/containers/%d/env/%d", factIdx, newEnvIdx),
				},
			}
		}

		restoreBytes, err := json.Marshal(restorePatch)
		if err != nil {
			t.Logf("Warning: failed to marshal restore patch: %v", err)
			return
		}
		_, err = client.AppsV1().DaemonSets(namespaces.StackRox).Patch(
			ctx, "collector", types.JSONPatchType, restoreBytes, metaV1.PatchOptions{})
		if err != nil {
			t.Logf("Warning: failed to restore FACT_PATHS: %v", err)
		} else {
			t.Log("Restored FACT_PATHS")
			waitForCollectorReady(t, client)
		}
	}

	return restore
}

// createFileActivityPolicy builds a storage.Policy for file activity detection.
func createFileActivityPolicy(name, path string, eventSource storage.EventSource, operations ...string) *storage.Policy {
	groups := []*storage.PolicyGroup{
		{
			FieldName: fieldnames.FilePath,
			Values:    []*storage.PolicyValue{{Value: path}},
		},
	}

	if len(operations) > 0 {
		values := make([]*storage.PolicyValue, len(operations))
		for i, op := range operations {
			values[i] = &storage.PolicyValue{Value: op}
		}
		groups = append(groups, &storage.PolicyGroup{
			FieldName: fieldnames.FileOperation,
			Values:    values,
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

// uniquePath returns a unique file path under /tmp for a subtest.
func uniquePath(prefix string) string {
	return fmt.Sprintf("/tmp/%s-%s", prefix, uuid.NewV4().String()[:8])
}

// buildDeploymentAlertQuery builds a ListAlertsRequest for deployment-level alerts.
func buildDeploymentAlertQuery(deploymentName, policyName string) *v1.ListAlertsRequest {
	qb := search.NewQueryBuilder().
		AddStrings(search.DeploymentName, deploymentName).
		AddStrings(search.PolicyName, policyName).
		AddStrings(search.EntityType, storage.Alert_DEPLOYMENT.String()).
		AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String())
	return &v1.ListAlertsRequest{Query: qb.Query()}
}

// buildNodeAlertQuery builds a ListAlertsRequest for node-level alerts.
func buildNodeAlertQuery(policyName string) *v1.ListAlertsRequest {
	qb := search.NewQueryBuilder().
		AddStrings(search.PolicyName, policyName).
		AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String()).
		AddStrings(search.EntityType, storage.Alert_NODE.String())
	return &v1.ListAlertsRequest{Query: qb.Query()}
}
