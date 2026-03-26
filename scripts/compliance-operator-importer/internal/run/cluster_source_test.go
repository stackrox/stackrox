package run

import (
	"os"
	"path/filepath"
	"testing"

	"k8s.io/client-go/tools/clientcmd"
)

// writeMinimalKubeconfig writes a kubeconfig with the given contexts to a file.
// Each context gets a unique cluster and user entry within the file.
func writeMinimalKubeconfig(t *testing.T, dir, filename string, contextNames []string) string {
	t.Helper()

	var clusters, contexts, users string
	for _, name := range contextNames {
		clusterName := "cluster-" + name
		userName := "user-" + name
		clusters += `
- cluster:
    server: https://` + name + `.example.com:6443
  name: ` + clusterName
		contexts += `
- context:
    cluster: ` + clusterName + `
    user: ` + userName + `
  name: ` + name
		users += `
- name: ` + userName + `
  user:
    token: token-` + filename + `-` + name
	}

	content := `apiVersion: v1
kind: Config
clusters:` + clusters + `
contexts:` + contexts + `
current-context: ` + contextNames[0] + `
users:` + users + `
`
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write kubeconfig %s: %v", path, err)
	}
	return path
}

// TestIMP_CLI_003_SingleFileAllContexts verifies that all contexts from a
// single kubeconfig file are discovered.
func TestIMP_CLI_003_SingleFileAllContexts(t *testing.T) {
	dir := t.TempDir()
	path := writeMinimalKubeconfig(t, dir, "config", []string{"ctx-a", "ctx-b"})
	t.Setenv(clientcmd.RecommendedConfigPathEnvVar, path)

	refs, err := listContextRefs()
	if err != nil {
		t.Fatalf("listContextRefs: %v", err)
	}
	if len(refs) != 2 {
		t.Fatalf("expected 2 refs, got %d", len(refs))
	}
	names := contextNames(refs)
	for _, want := range []string{"ctx-a", "ctx-b"} {
		found := false
		for _, n := range names {
			if n == want {
				found = true
			}
		}
		if !found {
			t.Errorf("expected context %q in %v", want, names)
		}
	}
}

// TestIMP_CLI_003_MultiFileUniqueContexts verifies that contexts from multiple
// kubeconfig files are all discovered when names are unique.
func TestIMP_CLI_003_MultiFileUniqueContexts(t *testing.T) {
	dir := t.TempDir()
	path1 := writeMinimalKubeconfig(t, dir, "config-a", []string{"ctx-a"})
	path2 := writeMinimalKubeconfig(t, dir, "config-b", []string{"ctx-b"})
	t.Setenv(clientcmd.RecommendedConfigPathEnvVar, path1+string(os.PathListSeparator)+path2)

	refs, err := listContextRefs()
	if err != nil {
		t.Fatalf("listContextRefs: %v", err)
	}
	if len(refs) != 2 {
		t.Fatalf("expected 2 refs, got %d", len(refs))
	}
}

// TestIMP_CLI_003_MultiFileDuplicateContextsBothProcessed verifies that when
// the same context name appears in multiple files, both are returned.
func TestIMP_CLI_003_MultiFileDuplicateContextsBothProcessed(t *testing.T) {
	dir := t.TempDir()
	path1 := writeMinimalKubeconfig(t, dir, "config", []string{"admin"})
	path2 := writeMinimalKubeconfig(t, dir, "config-secured-cluster", []string{"admin"})
	t.Setenv(clientcmd.RecommendedConfigPathEnvVar, path1+string(os.PathListSeparator)+path2)

	refs, err := listContextRefs()
	if err != nil {
		t.Fatalf("listContextRefs: %v", err)
	}
	if len(refs) != 2 {
		t.Fatalf("expected 2 refs (one per file), got %d", len(refs))
	}
	if refs[0].KubeconfigFile == refs[1].KubeconfigFile {
		t.Error("expected refs from different files")
	}
}

// TestIMP_CLI_003_PerFileIsolation verifies that each file is loaded
// independently: a user named "user-admin" in file A gets its own credentials,
// not file B's.
func TestIMP_CLI_003_PerFileIsolation(t *testing.T) {
	dir := t.TempDir()
	path1 := writeMinimalKubeconfig(t, dir, "config", []string{"admin"})
	path2 := writeMinimalKubeconfig(t, dir, "config-cluster-2", []string{"admin"})
	t.Setenv(clientcmd.RecommendedConfigPathEnvVar, path1+string(os.PathListSeparator)+path2)

	refs, err := listContextRefs()
	if err != nil {
		t.Fatalf("listContextRefs: %v", err)
	}

	// Build rest.Config for each ref and verify they use their own file's token.
	for _, ref := range refs {
		cfg, err := restConfigForRef(ref)
		if err != nil {
			t.Fatalf("restConfigForRef(%s from %s): %v", ref.Context, ref.KubeconfigFile, err)
		}
		expectedToken := "token-" + filepath.Base(ref.KubeconfigFile) + "-admin"
		if cfg.BearerToken != expectedToken {
			t.Errorf("ref from %s: expected token %q, got %q (credential isolation broken)",
				filepath.Base(ref.KubeconfigFile), expectedToken, cfg.BearerToken)
		}
	}
}

// TestIMP_CLI_003_FilterByContextName verifies that --context filtering matches
// context names across all files.
func TestIMP_CLI_003_FilterByContextName(t *testing.T) {
	dir := t.TempDir()
	path1 := writeMinimalKubeconfig(t, dir, "config", []string{"admin", "staging"})
	path2 := writeMinimalKubeconfig(t, dir, "config-cluster-2", []string{"admin"})
	t.Setenv(clientcmd.RecommendedConfigPathEnvVar, path1+string(os.PathListSeparator)+path2)

	refs, err := listContextRefs()
	if err != nil {
		t.Fatalf("listContextRefs: %v", err)
	}

	// Filter by "admin" — should match both files.
	filtered := filterRefs(refs, []string{"admin"})
	if len(filtered) != 2 {
		t.Errorf("filter by 'admin': expected 2 matches, got %d", len(filtered))
	}

	// Filter by "staging" — should match one.
	filtered = filterRefs(refs, []string{"staging"})
	if len(filtered) != 1 {
		t.Errorf("filter by 'staging': expected 1 match, got %d", len(filtered))
	}

	// Filter by nonexistent — should match none.
	filtered = filterRefs(refs, []string{"nonexistent"})
	if len(filtered) != 0 {
		t.Errorf("filter by 'nonexistent': expected 0 matches, got %d", len(filtered))
	}
}

// TestIMP_CLI_003_NoKubeconfigFiles verifies clear error when no files exist.
func TestIMP_CLI_003_NoKubeconfigFiles(t *testing.T) {
	t.Setenv(clientcmd.RecommendedConfigPathEnvVar, "/nonexistent/path")
	t.Setenv("HOME", t.TempDir())

	_, err := listContextRefs()
	if err == nil {
		t.Fatal("expected error when no kubeconfig files exist")
	}
}
