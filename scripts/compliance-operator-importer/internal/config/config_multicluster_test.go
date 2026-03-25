package config

import (
	"testing"
)

// TestIMP_CLI_003_KubeconfigRepeatable verifies that --kubeconfig can be
// repeated multiple times for multi-cluster mode.
func TestIMP_CLI_003_KubeconfigRepeatable(t *testing.T) {
	setenv(t, "ROX_API_TOKEN", "tok")

	cfg, err := ParseAndValidate([]string{
		"--endpoint", "https://central.example.com",
		"--co-namespace", "openshift-compliance",
		"--kubeconfig", "/path/to/kube1.yaml",
		"--kubeconfig", "/path/to/kube2.yaml",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Kubeconfigs) != 2 {
		t.Errorf("expected 2 kubeconfigs, got %d", len(cfg.Kubeconfigs))
	}
	if cfg.Kubeconfigs[0] != "/path/to/kube1.yaml" {
		t.Errorf("expected first kubeconfig path, got %q", cfg.Kubeconfigs[0])
	}
	if cfg.Kubeconfigs[1] != "/path/to/kube2.yaml" {
		t.Errorf("expected second kubeconfig path, got %q", cfg.Kubeconfigs[1])
	}
}

// TestIMP_CLI_003_KubecontextRepeatable verifies that --kubecontext can be
// repeated multiple times for multi-cluster mode.
func TestIMP_CLI_003_KubecontextRepeatable(t *testing.T) {
	setenv(t, "ROX_API_TOKEN", "tok")

	cfg, err := ParseAndValidate([]string{
		"--endpoint", "https://central.example.com",
		"--co-namespace", "openshift-compliance",
		"--kubecontext", "ctx1",
		"--kubecontext", "ctx2",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Kubecontexts) != 2 {
		t.Errorf("expected 2 kubecontexts, got %d", len(cfg.Kubecontexts))
	}
	if cfg.Kubecontexts[0] != "ctx1" {
		t.Errorf("expected first context, got %q", cfg.Kubecontexts[0])
	}
	if cfg.Kubecontexts[1] != "ctx2" {
		t.Errorf("expected second context, got %q", cfg.Kubecontexts[1])
	}
}

// TestIMP_CLI_003_KubecontextAll verifies that --kubecontext all signals
// iteration of all contexts in the active kubeconfig.
func TestIMP_CLI_003_KubecontextAll(t *testing.T) {
	setenv(t, "ROX_API_TOKEN", "tok")

	cfg, err := ParseAndValidate([]string{
		"--endpoint", "https://central.example.com",
		"--co-namespace", "openshift-compliance",
		"--kubecontext", "all",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Kubecontexts) != 1 || cfg.Kubecontexts[0] != "all" {
		t.Errorf("expected kubecontext 'all', got %v", cfg.Kubecontexts)
	}
}

// TestIMP_CLI_003_ClusterOverrideRepeatable verifies that --cluster ctx=value
// can be repeated for manual cluster name mappings in multi-cluster mode.
func TestIMP_CLI_003_ClusterOverrideRepeatable(t *testing.T) {
	setenv(t, "ROX_API_TOKEN", "tok")

	cfg, err := ParseAndValidate([]string{
		"--endpoint", "https://central.example.com",
		"--co-namespace", "openshift-compliance",
		"--kubecontext", "ctx1",
		"--cluster", "ctx1=acs-cluster-1",
		"--cluster", "ctx2=acs-cluster-2",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.ClusterOverrides) != 2 {
		t.Errorf("expected 2 cluster overrides, got %d", len(cfg.ClusterOverrides))
	}
	if cfg.ClusterOverrides[0] != "ctx1=acs-cluster-1" {
		t.Errorf("expected first override, got %q", cfg.ClusterOverrides[0])
	}
}

// TestIMP_CLI_003_KubeconfigContextMutuallyExclusive verifies that
// --kubeconfig and --kubecontext cannot be used together.
func TestIMP_CLI_003_KubeconfigContextMutuallyExclusive(t *testing.T) {
	setenv(t, "ROX_API_TOKEN", "tok")

	_, err := ParseAndValidate([]string{
		"--endpoint", "https://central.example.com",
		"--co-namespace", "openshift-compliance",
		"--kubeconfig", "/path/to/kube1.yaml",
		"--kubecontext", "ctx1",
	})
	if err == nil {
		t.Fatal("expected error for both --kubeconfig and --kubecontext, got nil")
	}
}

// TestIMP_CLI_003_DefaultSingleClusterMode verifies that when no multi-cluster
// flags are provided, the importer uses the current context.
func TestIMP_CLI_003_DefaultSingleClusterMode(t *testing.T) {
	setenv(t, "ROX_API_TOKEN", "tok")

	cfg, err := ParseAndValidate([]string{
		"--endpoint", "https://central.example.com",
		"--co-namespace", "openshift-compliance",
		"--cluster", "65640fbb-ac7c-42a8-9e65-883c3f35f23b",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Kubeconfigs) != 0 {
		t.Errorf("expected no kubeconfigs in single-cluster mode, got %d", len(cfg.Kubeconfigs))
	}
	if len(cfg.Kubecontexts) != 0 {
		t.Errorf("expected no kubecontexts in single-cluster mode, got %d", len(cfg.Kubecontexts))
	}
	if cfg.ACSClusterID != "65640fbb-ac7c-42a8-9e65-883c3f35f23b" {
		t.Errorf("expected ACSClusterID, got %q", cfg.ACSClusterID)
	}
}

// TestSingleClusterAutoDiscoveryWhenNoCluster verifies that omitting
// --cluster in single-cluster mode enables auto-discovery.
func TestSingleClusterAutoDiscoveryWhenNoCluster(t *testing.T) {
	setenv(t, "ROX_API_TOKEN", "tok")

	cfg, err := ParseAndValidate([]string{
		"--endpoint", "https://central.example.com",
		"--co-namespace", "openshift-compliance",
		// No --cluster and no multi-cluster flags
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !cfg.AutoDiscoverClusterID {
		t.Fatal("expected AutoDiscoverClusterID to be true")
	}
}
