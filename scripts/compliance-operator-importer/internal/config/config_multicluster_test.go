package config

import (
	"testing"
)

// TestIMP_CLI_003_ContextRepeatable verifies that --context can be
// repeated to filter which kubeconfig contexts are processed.
func TestIMP_CLI_003_ContextRepeatable(t *testing.T) {
	setenv(t, "ROX_API_TOKEN", "tok")

	cfg, err := ParseAndValidate([]string{
		"--endpoint", "https://central.example.com",
		"--context", "ctx-a",
		"--context", "ctx-b",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Contexts) != 2 {
		t.Errorf("expected 2 contexts, got %d", len(cfg.Contexts))
	}
	if cfg.Contexts[0] != "ctx-a" {
		t.Errorf("expected first context 'ctx-a', got %q", cfg.Contexts[0])
	}
	if cfg.Contexts[1] != "ctx-b" {
		t.Errorf("expected second context 'ctx-b', got %q", cfg.Contexts[1])
	}
}

// TestIMP_CLI_003_NoContextMeansAll verifies that omitting --context
// results in an empty Contexts slice (meaning "all contexts").
func TestIMP_CLI_003_NoContextMeansAll(t *testing.T) {
	setenv(t, "ROX_API_TOKEN", "tok")

	cfg, err := ParseAndValidate([]string{
		"--endpoint", "https://central.example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Contexts) != 0 {
		t.Errorf("expected empty contexts (all), got %v", cfg.Contexts)
	}
}

// TestIMP_CLI_003_RemovedFlagsRejected verifies that removed multi-cluster
// flags are not accepted.
func TestIMP_CLI_003_RemovedFlagsRejected(t *testing.T) {
	setenv(t, "ROX_API_TOKEN", "tok")

	for _, flag := range []string{"--kubeconfig", "--kubecontext", "--cluster"} {
		t.Run(flag, func(t *testing.T) {
			_, err := ParseAndValidate([]string{
				"--endpoint", "https://central.example.com",
				flag, "some-value",
			})
			if err == nil {
				t.Errorf("expected error for %s, got nil", flag)
			}
		})
	}
}
