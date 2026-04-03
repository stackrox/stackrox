package updater

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

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
