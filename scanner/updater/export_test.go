package updater

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/quay/claircore/libvuln/updates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportStatus_HasFailures(t *testing.T) {
	tests := []struct {
		name     string
		status   ExportStatus
		expected bool
	}{
		{
			name:     "empty status",
			status:   ExportStatus{},
			expected: false,
		},
		{
			name: "all success",
			status: ExportStatus{
				Updaters: []UpdaterStatus{
					{Name: "alpine", Status: StatusSuccess},
					{Name: "nvd", Status: StatusSuccess},
				},
			},
			expected: false,
		},
		{
			name: "one failure",
			status: ExportStatus{
				Updaters: []UpdaterStatus{
					{Name: "alpine", Status: StatusSuccess},
					{Name: "photon", Status: StatusFailed, Error: "404 not found"},
				},
			},
			expected: true,
		},
		{
			name: "all failures",
			status: ExportStatus{
				Updaters: []UpdaterStatus{
					{Name: "alpine", Status: StatusFailed, Error: "timeout"},
					{Name: "photon", Status: StatusFailed, Error: "404 not found"},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.HasFailures())
		})
	}
}

func TestExportStatus_Counts(t *testing.T) {
	status := ExportStatus{
		Updaters: []UpdaterStatus{
			{Name: "alpine", Status: StatusSuccess},
			{Name: "nvd", Status: StatusSuccess},
			{Name: "photon", Status: StatusFailed, Error: "404"},
			{Name: "debian", Status: StatusSuccess},
			{Name: "oracle", Status: StatusFailed, Error: "timeout"},
		},
	}

	success, failure := status.Counts()
	assert.Equal(t, 3, success)
	assert.Equal(t, 2, failure)
}

func TestWriteStatusFile(t *testing.T) {
	tmpDir := t.TempDir()

	now := time.Now()
	status := &ExportStatus{
		Updaters: []UpdaterStatus{
			{Name: "alpine", Status: StatusSuccess, LastAttempt: now},
			{Name: "photon", Status: StatusFailed, Error: "404 not found", LastAttempt: now},
		},
	}

	err := writeStatusFile(tmpDir, status)
	require.NoError(t, err)

	// Read back the file
	statusPath := filepath.Join(tmpDir, "status.json")
	data, err := os.ReadFile(statusPath)
	require.NoError(t, err)

	// Parse and verify
	var readStatus ExportStatus
	err = json.Unmarshal(data, &readStatus)
	require.NoError(t, err)

	assert.Len(t, readStatus.Updaters, 2)

	// Find alpine status
	var alpineStatus, photonStatus *UpdaterStatus
	for i := range readStatus.Updaters {
		switch readStatus.Updaters[i].Name {
		case "alpine":
			alpineStatus = &readStatus.Updaters[i]
		case "photon":
			photonStatus = &readStatus.Updaters[i]
		}
	}

	require.NotNil(t, alpineStatus)
	assert.Equal(t, StatusSuccess, alpineStatus.Status)
	assert.Empty(t, alpineStatus.Error)

	require.NotNil(t, photonStatus)
	assert.Equal(t, StatusFailed, photonStatus.Status)
	assert.Equal(t, "404 not found", photonStatus.Error)
	assert.WithinDuration(t, now, photonStatus.LastAttempt, time.Second)
}

func TestUpdaterStatus_JSONSerialization(t *testing.T) {
	now := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)

	// Test success case - error should be omitted
	successStatus := UpdaterStatus{
		Name:        "alpine",
		Status:      "success",
		LastAttempt: now,
	}
	data, err := json.Marshal(successStatus)
	require.NoError(t, err)

	// Error field should not appear in JSON for success
	assert.NotContains(t, string(data), "error")

	// Test failure case - error should be present
	failureStatus := UpdaterStatus{
		Name:        "photon",
		Status:      "failed",
		Error:       "connection timeout",
		LastAttempt: now,
	}
	data, err = json.Marshal(failureStatus)
	require.NoError(t, err)

	assert.Contains(t, string(data), `"error":"connection timeout"`)
}

// testBundleExporter is a simple test double for BundleExporter.
type testBundleExporter struct {
	exportFunc func(ctx context.Context, w io.Writer, opts []updates.ManagerOption) error
}

func (t *testBundleExporter) ExportBundle(ctx context.Context, w io.Writer, opts []updates.ManagerOption) error {
	return t.exportFunc(ctx, w, opts)
}

func readStatusFile(t *testing.T, dir string) ExportStatus {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "status.json"))
	require.NoError(t, err)
	var status ExportStatus
	require.NoError(t, json.Unmarshal(data, &status))
	return status
}

func TestExport_PartialFailure(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	// Create a mock exporter that fails for specific bundles.
	callCount := 0
	exporter := &testBundleExporter{
		exportFunc: func(ctx context.Context, w io.Writer, opts []updates.ManagerOption) error {
			callCount++
			if _, err := w.Write([]byte(`{"test":"data"}`)); err != nil {
				return err
			}
			if callCount == 3 {
				return errors.New("simulated bundle export failure")
			}
			return nil
		},
	}

	opts := &ExportOptions{ManualVulnURL: ""}

	err := Export(ctx, tmpDir, opts, exporter)

	// Should not return error since not ALL bundles failed.
	require.NoError(t, err)

	// Verify status.json was created with expected content.
	status := readStatusFile(t, tmpDir)
	assert.Greater(t, len(status.Updaters), 0, "should have recorded updater statuses")

	sc, fc := status.Counts()
	assert.Greater(t, sc, 0, "should have at least one success")
	assert.Greater(t, fc, 0, "should have at least one failure")
	assert.True(t, status.HasFailures(), "should report having failures")

	// Verify failed bundle file was deleted and successful ones remain.
	for _, u := range status.Updaters {
		bundlePath := filepath.Join(tmpDir, u.Name+".json.zst")
		if u.Status == StatusFailed {
			assert.NoFileExists(t, bundlePath, "failed bundle file should be deleted: %s", u.Name)
		} else {
			assert.FileExists(t, bundlePath, "successful bundle file should exist: %s", u.Name)
		}
	}
}

func TestExport_AllSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	exporter := &testBundleExporter{
		exportFunc: func(ctx context.Context, w io.Writer, opts []updates.ManagerOption) error {
			_, err := w.Write([]byte(`{"test":"data"}`))
			return err
		},
	}

	opts := &ExportOptions{ManualVulnURL: ""}

	err := Export(ctx, tmpDir, opts, exporter)
	require.NoError(t, err)

	status := readStatusFile(t, tmpDir)
	sc, fc := status.Counts()
	assert.Equal(t, 13, sc)
	assert.Equal(t, 0, fc)
	assert.False(t, status.HasFailures())
}

func TestWriteStatusFile_InvalidPath(t *testing.T) {
	status := &ExportStatus{
		Updaters: []UpdaterStatus{
			{Name: "alpine", Status: StatusSuccess},
		},
	}

	err := writeStatusFile("/nonexistent/path", status)
	require.Error(t, err)
}

func TestExport_AllFailed(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	exporter := &testBundleExporter{
		exportFunc: func(ctx context.Context, w io.Writer, opts []updates.ManagerOption) error {
			return errors.New("all bundles fail")
		},
	}

	opts := &ExportOptions{ManualVulnURL: ""}

	err := Export(ctx, tmpDir, opts, exporter)

	// Should return error when ALL bundles fail.
	require.Error(t, err)
	assert.Contains(t, err.Error(), "all")

	// Verify status.json was still created with all failures.
	status := readStatusFile(t, tmpDir)
	sc, fc := status.Counts()
	assert.Equal(t, 0, sc)
	assert.Greater(t, fc, 0)

	// Verify no bundle files remain.
	files, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	for _, f := range files {
		if filepath.Ext(f.Name()) == ".zst" {
			t.Errorf("bundle file should not exist: %s", f.Name())
		}
	}
}
