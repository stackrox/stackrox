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

func TestExportStatus_SuccessCount(t *testing.T) {
	status := ExportStatus{
		Updaters: []UpdaterStatus{
			{Name: "alpine", Status: StatusSuccess},
			{Name: "nvd", Status: StatusSuccess},
			{Name: "photon", Status: StatusFailed, Error: "404"},
			{Name: "debian", Status: StatusSuccess},
		},
	}

	assert.Equal(t, 3, status.SuccessCount())
}

func TestExportStatus_FailureCount(t *testing.T) {
	status := ExportStatus{
		Updaters: []UpdaterStatus{
			{Name: "alpine", Status: StatusSuccess},
			{Name: "nvd", Status: StatusSuccess},
			{Name: "photon", Status: StatusFailed, Error: "404"},
			{Name: "oracle", Status: StatusFailed, Error: "timeout"},
		},
	}

	assert.Equal(t, 2, status.FailureCount())
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

func TestExport_PartialFailure(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	// Create a mock exporter that fails for specific bundles.
	callCount := 0
	exporter := &testBundleExporter{
		exportFunc: func(ctx context.Context, w io.Writer, opts []updates.ManagerOption) error {
			callCount++
			// Write some data to simulate bundle content.
			if _, err := w.Write([]byte(`{"test":"data"}`)); err != nil {
				return err
			}

			// Simulate failure for the third bundle (arbitrary choice for testing).
			if callCount == 3 {
				return errors.New("simulated bundle export failure")
			}
			return nil
		},
	}

	// Run export with a minimal manual URL (will be one of the bundles).
	opts := &ExportOptions{
		ManualVulnURL: "", // Empty URL should still work for testing structure.
	}

	status, err := Export(ctx, tmpDir, opts, exporter)

	// Should not return error since not ALL bundles failed.
	require.NoError(t, err)
	require.NotNil(t, status)

	// Verify status was recorded.
	assert.Greater(t, len(status.Updaters), 0, "should have recorded updater statuses")

	// Verify we have both successes and failures.
	assert.Greater(t, status.SuccessCount(), 0, "should have at least one success")
	assert.Greater(t, status.FailureCount(), 0, "should have at least one failure")
	assert.True(t, status.HasFailures(), "should report having failures")

	// Verify status.json was created.
	statusPath := filepath.Join(tmpDir, "status.json")
	assert.FileExists(t, statusPath)

	// Verify status.json content.
	statusData, err := os.ReadFile(statusPath)
	require.NoError(t, err)

	var readStatus ExportStatus
	err = json.Unmarshal(statusData, &readStatus)
	require.NoError(t, err)
	assert.Equal(t, len(status.Updaters), len(readStatus.Updaters))

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

func TestExport_AllFailed(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	// Create a mock exporter that always fails.
	exporter := &testBundleExporter{
		exportFunc: func(ctx context.Context, w io.Writer, opts []updates.ManagerOption) error {
			return errors.New("all bundles fail")
		},
	}

	opts := &ExportOptions{
		ManualVulnURL: "",
	}

	status, err := Export(ctx, tmpDir, opts, exporter)

	// Should return error when ALL bundles fail.
	require.Error(t, err)
	assert.Contains(t, err.Error(), "all")
	require.NotNil(t, status)

	// Verify all updaters failed.
	assert.Equal(t, 0, status.SuccessCount())
	assert.Greater(t, status.FailureCount(), 0)

	// Verify status.json was still created.
	statusPath := filepath.Join(tmpDir, "status.json")
	assert.FileExists(t, statusPath)

	// Verify no bundle files remain.
	files, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	for _, f := range files {
		if filepath.Ext(f.Name()) == ".zst" {
			t.Errorf("bundle file should not exist: %s", f.Name())
		}
	}
}
