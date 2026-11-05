package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_prowConfigRelPath(t *testing.T) {
	require.Equal(t, "ci-operator/config/stackrox/stackrox", prowConfigRelPath("stackrox/stackrox"))
	require.Equal(t, "ci-operator/config/foo/bar", prowConfigRelPath("foo/bar"))
}

func Test_fetchProwConfigDir(t *testing.T) {
	const targetRepo = "foo/bar"

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

	configDir := filepath.Join(dir, prowConfigRelPath(targetRepo))
	require.NoError(t, os.MkdirAll(configDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "foo-bar-master.yaml"), []byte("tests: []\n"), 0o644))

	git("add", ".")
	git("commit", "-qm", "initial commit")

	// This replicates fetchProwConfigDir's own logic against a local file://
	// repo (rather than calling it directly, which always targets a real
	// https://github.com/... URL) so the test stays hermetic.
	cloneDir, cleanup, err := cloneSparse("file://"+dir, "main", prowConfigRelPath(targetRepo))
	require.NoError(t, err)
	defer cleanup()

	_, err = os.Stat(filepath.Join(cloneDir, prowConfigRelPath(targetRepo), "foo-bar-master.yaml"))
	require.NoError(t, err)
}

func Test_resolveProwConfigDir(t *testing.T) {
	for name, tc := range map[string]struct {
		prowDirSet  bool
		repoSet     bool
		branchSet   bool
		wantErr     string
		wantFetched bool
	}{
		"prow-dir only": {
			prowDirSet: true,
		},
		"neither set falls back to fetch defaults": {
			wantFetched: true,
		},
		"prow-repo set only": {
			repoSet:     true,
			wantFetched: true,
		},
		"prow-branch set only": {
			branchSet:   true,
			wantFetched: true,
		},
		"prow-dir and prow-repo both set": {
			prowDirSet: true,
			repoSet:    true,
			wantErr:    "mutually exclusive",
		},
		"prow-dir and prow-branch both set": {
			prowDirSet: true,
			branchSet:  true,
			wantErr:    "mutually exclusive",
		},
		"prow-dir, prow-repo, and prow-branch all set": {
			prowDirSet: true,
			repoSet:    true,
			branchSet:  true,
			wantErr:    "mutually exclusive",
		},
	} {
		t.Run(name, func(t *testing.T) {
			var fetchCalled bool
			var gotRepo, gotBranch string
			fetch := func(repo, branch string) (string, func(), error) {
				fetchCalled = true
				gotRepo, gotBranch = repo, branch
				return "fetched-dir", func() {}, nil
			}

			dir, cleanup, err := resolveProwConfigDir("some-dir", tc.prowDirSet, "some/repo", "some-branch", tc.repoSet, tc.branchSet, fetch)

			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				require.False(t, fetchCalled)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, cleanup)
			if tc.wantFetched {
				require.True(t, fetchCalled)
				require.Equal(t, "fetched-dir", dir)
				require.Equal(t, "some/repo", gotRepo)
				require.Equal(t, "some-branch", gotBranch)
			} else {
				require.False(t, fetchCalled)
				require.Equal(t, "some-dir", dir)
			}
		})
	}
}
