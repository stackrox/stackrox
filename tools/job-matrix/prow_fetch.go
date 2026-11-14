package main

import (
	"path"
	"path/filepath"
)

// prowConfigRelPath locates repo's ci-operator config directory within an
// openshift/release checkout, e.g. "ci-operator/config/stackrox/stackrox"
// for repo "stackrox/stackrox".
func prowConfigRelPath(repo string) string {
	return path.Join("ci-operator/config", repo)
}

// fetchProwConfigDir fetches targetRepo's ("owner/name") ci-operator config
// directory from prowRepo ("owner/name") at branch, returning its path on
// disk plus a cleanup func.
func fetchProwConfigDir(prowRepo, branch, targetRepo string) (string, func(), error) {
	dir, cleanup, err := cloneSparse(githubCloneURL(prowRepo), branch, prowConfigRelPath(targetRepo))
	if err != nil {
		return "", nil, err
	}
	return filepath.Join(dir, prowConfigRelPath(targetRepo)), cleanup, nil
}
