package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// writeTestFile persists test fixture data and returns its path.
func writeTestFile(t *testing.T, path, data string) string {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(data), 0o600))
	return path
}

// e2eTargets lists the e2e suites enrolled in the selection policy; it
// mirrors the list in internal/resolver's tests, which pin the policy
// itself.
func e2eTargets() []string {
	return []string{
		"e2e-byodb-tests", "e2e-db-backup-restore-tests-gke", "e2e-db-backup-restore-tests-gke-konflux",
		"e2e-gke-upgrade-tests", "e2e-nongroovy-tests-gke", "e2e-nongroovy-tests-gke-konflux",
		"e2e-qa-tests-gke", "e2e-qa-tests-gke-konflux", "e2e-qa-tests-ocp-fips-konflux",
		"e2e-ui-tests-gke", "e2e-ui-tests-gke-konflux",
	}
}
