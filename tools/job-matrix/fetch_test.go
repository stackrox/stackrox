package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// repository creates a throwaway git repository in a temp dir with a
// top-level file, a .github/workflows file, and an unrelated nested
// directory, committed on branch "main". It returns the repo's path.
func repository(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "%s", output)
	}
	git("init", "-q", "--initial-branch=main")
	git("config", "user.email", "test@example.com")
	git("config", "user.name", "Test")
	git("config", "commit.gpgsign", "false")

	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".github", "workflows"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".github", "workflows", "foo.yaml"), []byte("name: foo\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "other"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "other", "bar.txt"), []byte("bar\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "unrelated"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "unrelated", "baz.txt"), []byte("baz\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("readme\n"), 0o644))

	git("add", ".")
	git("commit", "-qm", "initial commit")
	return dir
}

func Test_cloneSparse(t *testing.T) {
	repoDir := repository(t)

	dir, cleanup, err := cloneSparse("file://"+repoDir, "main", ".github")
	require.NoError(t, err)
	defer cleanup()

	content, err := os.ReadFile(filepath.Join(dir, ".github", "workflows", "foo.yaml"))
	require.NoError(t, err)
	require.Equal(t, "name: foo\n", string(content))

	// Sparse-checkout should have narrowed the clone to .github, excluding
	// unrelated top-level directories.
	_, err = os.Stat(filepath.Join(dir, "other", "bar.txt"))
	require.True(t, os.IsNotExist(err))
}

func Test_cloneSparse_multiplePaths(t *testing.T) {
	repoDir := repository(t)

	dir, cleanup, err := cloneSparse("file://"+repoDir, "main", ".github", "other")
	require.NoError(t, err)
	defer cleanup()

	content, err := os.ReadFile(filepath.Join(dir, ".github", "workflows", "foo.yaml"))
	require.NoError(t, err)
	require.Equal(t, "name: foo\n", string(content))

	content, err = os.ReadFile(filepath.Join(dir, "other", "bar.txt"))
	require.NoError(t, err)
	require.Equal(t, "bar\n", string(content))

	// The unrelated top-level directory should still be excluded.
	_, err = os.Stat(filepath.Join(dir, "unrelated", "baz.txt"))
	require.True(t, os.IsNotExist(err))
}

func Test_cloneSparse_badBranch(t *testing.T) {
	repoDir := repository(t)

	_, _, err := cloneSparse("file://"+repoDir, "does-not-exist", ".github")
	require.Error(t, err)
}

func Test_githubCloneURL(t *testing.T) {
	require.Equal(t, "https://github.com/stackrox/stackrox.git", githubCloneURL("stackrox/stackrox"))
}
