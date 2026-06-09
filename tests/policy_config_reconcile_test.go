//go:build test_e2e

package tests

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	reconcileTestScope = "e2e-test-scope"
)

func TestPolicyConfigReconcile(t *testing.T) {
	conn := centralgrpc.GRPCConnectionToCentral(t)
	policySvc := v1.NewPolicyServiceClient(conn)

	dir := t.TempDir()
	defer cleanupReconcileTestPolicies(t, policySvc)

	t.Run("CreatePolicies", func(t *testing.T) {
		writePolicyFile(t, filepath.Join(dir, "policy1.yaml"), `
policyName: "E2E Test Policy One"
description: "First test policy for reconcile e2e"
categories:
  - "DevOps Best Practices"
lifecycleStages:
  - DEPLOY
severity: HIGH_SEVERITY
policySections:
  - sectionName: "test"
    policyGroups:
      - fieldName: "Image Tag"
        values:
          - value: "latest"
`)

		writePolicyFile(t, filepath.Join(dir, "policy2.yaml"), `
policyName: "E2E Test Policy Two"
description: "Second test policy for reconcile e2e"
categories:
  - "DevOps Best Practices"
lifecycleStages:
  - BUILD
severity: LOW_SEVERITY
policySections:
  - sectionName: "test"
    policyGroups:
      - fieldName: "Image Tag"
        values:
          - value: "latest"
`)

		runReconcile(t, dir, reconcileTestScope)

		policies := listManagedPolicies(t, policySvc, reconcileTestScope)
		require.Len(t, policies, 2, "expected 2 policies after initial reconcile")

		names := policyNames(policies)
		assert.Contains(t, names, "E2E Test Policy One")
		assert.Contains(t, names, "E2E Test Policy Two")

		for _, p := range policies {
			assert.Equal(t, storage.PolicySource_DECLARATIVE, p.GetSource())
			assert.Equal(t, reconcileTestScope, p.GetConfigScope())
		}
	})

	t.Run("UpdatePolicy", func(t *testing.T) {
		writePolicyFile(t, filepath.Join(dir, "policy1.yaml"), `
policyName: "E2E Test Policy One"
description: "Updated description"
categories:
  - "DevOps Best Practices"
lifecycleStages:
  - DEPLOY
severity: CRITICAL_SEVERITY
policySections:
  - sectionName: "test"
    policyGroups:
      - fieldName: "Image Tag"
        values:
          - value: "latest"
`)

		runReconcile(t, dir, reconcileTestScope)

		policies := listManagedPolicies(t, policySvc, reconcileTestScope)
		require.Len(t, policies, 2)

		for _, p := range policies {
			if p.GetName() == "E2E Test Policy One" {
				assert.Equal(t, "Updated description", p.GetDescription())
				assert.Equal(t, storage.Severity_CRITICAL_SEVERITY, p.GetSeverity())
			}
		}
	})

	t.Run("DeleteOrphan", func(t *testing.T) {
		require.NoError(t, os.Remove(filepath.Join(dir, "policy2.yaml")))

		runReconcile(t, dir, reconcileTestScope)

		policies := listManagedPolicies(t, policySvc, reconcileTestScope)
		require.Len(t, policies, 1, "expected 1 policy after removing policy2.yaml")
		assert.Equal(t, "E2E Test Policy One", policies[0].GetName())
	})

	t.Run("ScopeIsolation", func(t *testing.T) {
		otherDir := t.TempDir()
		writePolicyFile(t, filepath.Join(otherDir, "other.yaml"), `
policyName: "Other Scope Policy"
categories:
  - "DevOps Best Practices"
lifecycleStages:
  - DEPLOY
severity: LOW_SEVERITY
policySections:
  - sectionName: "test"
    policyGroups:
      - fieldName: "Image Tag"
        values:
          - value: "latest"
`)

		runReconcile(t, otherDir, "other-scope")
		defer cleanupReconcileScope(t, policySvc, "other-scope")

		originalPolicies := listManagedPolicies(t, policySvc, reconcileTestScope)
		require.Len(t, originalPolicies, 1, "original scope policies should be untouched")

		otherPolicies := listManagedPolicies(t, policySvc, "other-scope")
		require.Len(t, otherPolicies, 1)
		assert.Equal(t, "Other Scope Policy", otherPolicies[0].GetName())
	})

	t.Run("DryRun", func(t *testing.T) {
		emptyDir := t.TempDir()

		runReconcileDryRun(t, emptyDir, reconcileTestScope)

		policies := listManagedPolicies(t, policySvc, reconcileTestScope)
		require.Len(t, policies, 1, "dry-run should not have deleted anything")
	})

	t.Run("Idempotent", func(t *testing.T) {
		runReconcile(t, dir, reconcileTestScope)
		runReconcile(t, dir, reconcileTestScope)

		policies := listManagedPolicies(t, policySvc, reconcileTestScope)
		require.Len(t, policies, 1, "idempotent reconcile should maintain same state")
	})
}

func runReconcile(t *testing.T, dir, scope string) {
	t.Helper()
	cmd := roxctlCmd(t, "policy-config", "reconcile",
		"--dir", dir,
		"--config-scope", scope,
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "roxctl policy-config reconcile failed: %s", string(out))
	t.Logf("reconcile output: %s", string(out))
}

func runReconcileDryRun(t *testing.T, dir, scope string) {
	t.Helper()
	cmd := roxctlCmd(t, "policy-config", "reconcile",
		"--dir", dir,
		"--config-scope", scope,
		"--dry-run",
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "roxctl policy-config reconcile --dry-run failed: %s", string(out))
	t.Logf("dry-run output: %s", string(out))
}

func roxctlCmd(t *testing.T, args ...string) *exec.Cmd {
	t.Helper()
	endpoint := centralgrpc.RoxAPIEndpoint(t)
	password := centralgrpc.RoxPassword(t)

	allArgs := append([]string{
		"--insecure-skip-tls-verify",
		"--insecure",
		"-e", endpoint,
		"--password", password,
	}, args...)

	return exec.Command("roxctl", allArgs...)
}

func listManagedPolicies(t *testing.T, svc v1.PolicyServiceClient, scope string) []*storage.Policy {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	listResp, err := svc.ListPolicies(ctx, &v1.RawQuery{
		Query: "Config Scope:" + scope,
	})
	require.NoError(t, err)

	var policies []*storage.Policy
	for _, lp := range listResp.GetPolicies() {
		if lp.GetSource() != storage.PolicySource_DECLARATIVE {
			continue
		}
		ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
		p, err := svc.GetPolicy(ctx2, &v1.ResourceByID{Id: lp.GetId()})
		cancel2()
		require.NoError(t, err)
		policies = append(policies, p)
	}
	return policies
}

func cleanupReconcileTestPolicies(t *testing.T, svc v1.PolicyServiceClient) {
	t.Helper()
	cleanupReconcileScope(t, svc, reconcileTestScope)
}

func cleanupReconcileScope(t *testing.T, svc v1.PolicyServiceClient, scope string) {
	t.Helper()
	policies := listManagedPolicies(t, svc, scope)
	for _, p := range policies {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, err := svc.DeletePolicy(ctx, &v1.ResourceByID{Id: p.GetId()})
		cancel()
		if err != nil {
			t.Logf("warning: failed to clean up policy %q: %v", p.GetName(), err)
		}
	}
}

func policyNames(policies []*storage.Policy) []string {
	names := make([]string, 0, len(policies))
	for _, p := range policies {
		names = append(names, p.GetName())
	}
	return names
}

func writePolicyFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
}
