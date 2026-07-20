package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stackrox/rox/compliance/virtualmachines/roxagent/internal/hostprobe"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// indexReportWith builds a minimal *v4.IndexReport with numRepos repositories
// and numDists distributions, keyed "id-0", "id-1", ... so that
// logIndexReportDiagnostics's truncation logic can be exercised deterministically.
func indexReportWith(numRepos, numDists int) *v4.IndexReport {
	repos := make(map[string]*v4.Repository, numRepos)
	for i := range numRepos {
		id := fmt.Sprintf("repo-%02d", i)
		repos[id] = &v4.Repository{Name: id, Key: "key", Cpe: "cpe"}
	}
	dists := make(map[string]*v4.Distribution, numDists)
	for i := range numDists {
		id := fmt.Sprintf("dist-%02d", i)
		dists[id] = &v4.Distribution{Name: id, Version: "1", Cpe: "cpe", Did: "did"}
	}
	return &v4.IndexReport{
		Contents: &v4.Contents{
			Repositories:  repos,
			Distributions: dists,
		},
	}
}

// TestLogIndexReportDiagnostics exercises every branch of the truncation and
// zero-repositories logic. Assertions are limited to "does not panic": the
// function only produces log output, so branch/statement coverage is the
// goal here, not asserting exact log content.
func TestLogIndexReportDiagnostics(t *testing.T) {
	tests := map[string]*v4.IndexReport{
		"nil report":                      nil,
		"report with nil contents":        {},
		"empty contents":                  indexReportWith(0, 0),
		"fewer than truncation threshold": indexReportWith(3, 2),
		"exactly at truncation threshold": indexReportWith(10, 10),
		"over truncation threshold":       indexReportWith(15, 12),
	}

	for name, report := range tests {
		t.Run(name, func(t *testing.T) {
			assert.NotPanics(t, func() { logIndexReportDiagnostics(report) })
		})
	}
}

// TestLogRepoError exercises every branch of logRepoError's errors.Is switch.
func TestLogRepoError(t *testing.T) {
	tests := map[string]error{
		"not exist":  fmt.Errorf("wrapped: %w", fs.ErrNotExist),
		"permission": fmt.Errorf("wrapped: %w", fs.ErrPermission),
		"other":      errors.New("some other failure"),
	}

	for name, err := range tests {
		t.Run(name, func(t *testing.T) {
			assert.NotPanics(t, func() { logRepoError(err) })
		})
	}
}

// TestLogFilesystemDiagnostics exercises logFilesystemDiagnostics against
// synthetic hostPath layouts covering DNF4/DNF5 detection, entitlement
// presence/absence, and repo-file presence/absence. As with the other
// diagnostics tests, only "does not panic" is asserted.
func TestLogFilesystemDiagnostics(t *testing.T) {
	t.Run("dnf4 host with entitlement and repo files", func(t *testing.T) {
		hostPath := t.TempDir()
		writeFile(t, hostprobe.HostPathFor(hostPath, hostprobe.DNF4HistoryDBPath), "db")
		writeFile(t, filepath.Join(hostprobe.HostPathFor(hostPath, hostprobe.EntitlementDirPath), "1-key.pem"), "k")
		writeFile(t, filepath.Join(hostprobe.HostPathFor(hostPath, hostprobe.EntitlementDirPath), "1.pem"), "c")
		writeFile(t, filepath.Join(hostprobe.HostPathFor(hostPath, hostprobe.YumReposDirPath), "redhat.repo"), "[baseos]")

		assert.NotPanics(t, func() { logFilesystemDiagnostics(hostPath) })
	})

	t.Run("dnf5 host with no entitlement and no repo files", func(t *testing.T) {
		hostPath := t.TempDir()
		writeFile(t, hostprobe.HostPathFor(hostPath, hostprobe.DNF5HistoryDBPath), "db")
		require.NoError(t, os.MkdirAll(hostprobe.HostPathFor(hostPath, hostprobe.YumReposDirPath), 0o755))

		assert.NotPanics(t, func() { logFilesystemDiagnostics(hostPath) })
	})

	t.Run("bare host with nothing present", func(t *testing.T) {
		hostPath := t.TempDir()
		assert.NotPanics(t, func() { logFilesystemDiagnostics(hostPath) })
	})
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
