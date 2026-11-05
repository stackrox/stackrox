package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// githubCloneURL returns the HTTPS clone URL for a GitHub repo given in
// "owner/name" form.
func githubCloneURL(repo string) string {
	return fmt.Sprintf("https://github.com/%s.git", repo)
}

// cloneSparse shallow-clones url at branch into a new temporary directory,
// narrowed via cone-mode sparse-checkout to sparsePaths, and returns the
// clone's root directory plus a cleanup func that removes it.
func cloneSparse(url, branch string, sparsePaths ...string) (repoDir string, cleanup func(), err error) {
	tmpDir, err := os.MkdirTemp("", "job-matrix-")
	if err != nil {
		return "", nil, fmt.Errorf("creating temp dir: %w", err)
	}
	cleanup = func() { _ = os.RemoveAll(tmpDir) }

	clone := exec.Command("git", "clone", "--quiet", "--depth=1", "--branch", branch, "--filter=blob:none", "--sparse", url, tmpDir)
	if output, err := clone.CombinedOutput(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("cloning %s (branch %s): %w: %s", url, branch, err, output)
	}

	args := append([]string{"-C", tmpDir, "sparse-checkout", "set"}, sparsePaths...)
	sparse := exec.Command("git", args...)
	if output, err := sparse.CombinedOutput(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("setting sparse-checkout to %v: %w: %s", sparsePaths, err, output)
	}

	return tmpDir, cleanup, nil
}

// fetchWorkflowsDir fetches the .github directory of repo ("owner/name") at
// branch and returns the path to its workflows subdirectory.
func fetchWorkflowsDir(repo, branch string) (string, func(), error) {
	dir, cleanup, err := cloneSparse(githubCloneURL(repo), branch, ".github")
	if err != nil {
		return "", nil, err
	}
	return filepath.Join(dir, ".github", "workflows"), cleanup, nil
}
