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
// narrowed via cone-mode sparse-checkout to sparsePath, and returns the
// path to sparsePath inside the clone plus a cleanup func that removes the
// temporary directory.
func cloneSparse(url, branch, sparsePath string) (dir string, cleanup func(), err error) {
	tmpDir, err := os.MkdirTemp("", "job-matrix-")
	if err != nil {
		return "", nil, fmt.Errorf("creating temp dir: %w", err)
	}
	cleanup = func() { _ = os.RemoveAll(tmpDir) }

	clone := exec.Command("git", "clone", "--quiet", "--depth=1", "--branch", branch, "--filter=blob:none", "--sparse", url, tmpDir)
	clone.Stderr = os.Stderr
	if err := clone.Run(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("cloning %s (branch %s): %w", url, branch, err)
	}

	sparse := exec.Command("git", "-C", tmpDir, "sparse-checkout", "set", sparsePath)
	sparse.Stderr = os.Stderr
	if err := sparse.Run(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("setting sparse-checkout to %s: %w", sparsePath, err)
	}

	return filepath.Join(tmpDir, sparsePath), cleanup, nil
}

// fetchWorkflowsDir fetches the .github directory of repo ("owner/name") at
// branch and returns the path to its workflows subdirectory.
func fetchWorkflowsDir(repo, branch string) (string, func(), error) {
	dir, cleanup, err := cloneSparse(githubCloneURL(repo), branch, ".github")
	if err != nil {
		return "", nil, err
	}
	return filepath.Join(dir, "workflows"), cleanup, nil
}
