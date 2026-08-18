package fake

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeFileActivityParams(t *testing.T) {
	tests := map[string]struct {
		input                    FileActivityWorkload
		expectedNumPaths         int
		expectedBatchSize        int
		expectedNodeEventPercent int
		expectedActivityInterval time.Duration
	}{
		"all valid parameters should remain unchanged": {
			input: FileActivityWorkload{
				NumPaths:         100,
				BatchSize:        10,
				NodeEventPercent: 25,
				ActivityInterval: 100 * time.Millisecond,
			},
			expectedNumPaths:         100,
			expectedBatchSize:        10,
			expectedNodeEventPercent: 25,
			expectedActivityInterval: 100 * time.Millisecond,
		},
		"zero NumPaths should default to 50": {
			input: FileActivityWorkload{
				NumPaths:         0,
				BatchSize:        5,
				NodeEventPercent: 50,
				ActivityInterval: 100 * time.Millisecond,
			},
			expectedNumPaths:         50,
			expectedBatchSize:        5,
			expectedNodeEventPercent: 50,
			expectedActivityInterval: 100 * time.Millisecond,
		},
		"negative NumPaths should default to 50": {
			input: FileActivityWorkload{
				NumPaths:         -10,
				BatchSize:        5,
				NodeEventPercent: 50,
				ActivityInterval: 100 * time.Millisecond,
			},
			expectedNumPaths:         50,
			expectedBatchSize:        5,
			expectedNodeEventPercent: 50,
			expectedActivityInterval: 100 * time.Millisecond,
		},
		"zero BatchSize should default to 1": {
			input: FileActivityWorkload{
				NumPaths:         100,
				BatchSize:        0,
				NodeEventPercent: 50,
				ActivityInterval: 100 * time.Millisecond,
			},
			expectedNumPaths:         100,
			expectedBatchSize:        1,
			expectedNodeEventPercent: 50,
			expectedActivityInterval: 100 * time.Millisecond,
		},
		"negative BatchSize should default to 1": {
			input: FileActivityWorkload{
				NumPaths:         100,
				BatchSize:        -5,
				NodeEventPercent: 50,
				ActivityInterval: 100 * time.Millisecond,
			},
			expectedNumPaths:         100,
			expectedBatchSize:        1,
			expectedNodeEventPercent: 50,
			expectedActivityInterval: 100 * time.Millisecond,
		},
		"negative NodeEventPercent should default to 50": {
			input: FileActivityWorkload{
				NumPaths:         100,
				BatchSize:        5,
				NodeEventPercent: -10,
				ActivityInterval: 100 * time.Millisecond,
			},
			expectedNumPaths:         100,
			expectedBatchSize:        5,
			expectedNodeEventPercent: 50,
			expectedActivityInterval: 100 * time.Millisecond,
		},
		"NodeEventPercent above 100 should default to 50": {
			input: FileActivityWorkload{
				NumPaths:         100,
				BatchSize:        5,
				NodeEventPercent: 150,
				ActivityInterval: 100 * time.Millisecond,
			},
			expectedNumPaths:         100,
			expectedBatchSize:        5,
			expectedNodeEventPercent: 50,
			expectedActivityInterval: 100 * time.Millisecond,
		},
		"NodeEventPercent of 0 (all container events) should remain 0": {
			input: FileActivityWorkload{
				NumPaths:         100,
				BatchSize:        5,
				NodeEventPercent: 0,
				ActivityInterval: 100 * time.Millisecond,
			},
			expectedNumPaths:         100,
			expectedBatchSize:        5,
			expectedNodeEventPercent: 0,
			expectedActivityInterval: 100 * time.Millisecond,
		},
		"NodeEventPercent of 100 (all node events) should remain 100": {
			input: FileActivityWorkload{
				NumPaths:         100,
				BatchSize:        5,
				NodeEventPercent: 100,
				ActivityInterval: 100 * time.Millisecond,
			},
			expectedNumPaths:         100,
			expectedBatchSize:        5,
			expectedNodeEventPercent: 100,
			expectedActivityInterval: 100 * time.Millisecond,
		},
		"zero ActivityInterval should default to 50ms": {
			input: FileActivityWorkload{
				NumPaths:         100,
				BatchSize:        5,
				NodeEventPercent: 50,
				ActivityInterval: 0,
			},
			expectedNumPaths:         100,
			expectedBatchSize:        5,
			expectedNodeEventPercent: 50,
			expectedActivityInterval: 50 * time.Millisecond,
		},
		"negative ActivityInterval should default to 50ms": {
			input: FileActivityWorkload{
				NumPaths:         100,
				BatchSize:        5,
				NodeEventPercent: 50,
				ActivityInterval: -100 * time.Millisecond,
			},
			expectedNumPaths:         100,
			expectedBatchSize:        5,
			expectedNodeEventPercent: 50,
			expectedActivityInterval: 50 * time.Millisecond,
		},
		"all invalid parameters should all be defaulted": {
			input: FileActivityWorkload{
				NumPaths:         -5,
				BatchSize:        -10,
				NodeEventPercent: 200,
				ActivityInterval: -50 * time.Millisecond,
			},
			expectedNumPaths:         50,
			expectedBatchSize:        1,
			expectedNodeEventPercent: 50,
			expectedActivityInterval: 50 * time.Millisecond,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create a minimal WorkloadManager with the test workload
			wm := &WorkloadManager{
				workload: &Workload{
					FileActivityWorkload: tc.input,
				},
			}

			// Run sanitization
			wm.sanitizeFileActivityParams()

			// Verify results
			assert.Equal(t, tc.expectedNumPaths, wm.workload.FileActivityWorkload.NumPaths,
				"NumPaths mismatch")
			assert.Equal(t, tc.expectedBatchSize, wm.workload.FileActivityWorkload.BatchSize,
				"BatchSize mismatch")
			assert.Equal(t, tc.expectedNodeEventPercent, wm.workload.FileActivityWorkload.NodeEventPercent,
				"NodeEventPercent mismatch")
			assert.Equal(t, tc.expectedActivityInterval, wm.workload.FileActivityWorkload.ActivityInterval,
				"ActivityInterval mismatch")
		})
	}
}

func TestGenerateFileActivityPaths(t *testing.T) {
	tests := map[string]struct {
		numPaths int
	}{
		"generate 10 paths": {
			numPaths: 10,
		},
		"generate 50 paths": {
			numPaths: 50,
		},
		"generate 100 paths": {
			numPaths: 100,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			paths := generateFileActivityPaths(tc.numPaths)

			// Verify correct number of paths
			assert.Len(t, paths, tc.numPaths, "Should generate exactly the requested number of paths")

			// Verify all paths are non-empty and start with a known directory
			for _, path := range paths {
				assert.NotEmpty(t, path, "Path should not be empty")

				// Verify path starts with one of the known directories
				foundPrefix := false
				for _, dir := range fileActivityDirs {
					if len(path) > len(dir) && path[:len(dir)] == dir {
						foundPrefix = true
						break
					}
				}
				assert.True(t, foundPrefix, "Path should start with a known directory prefix")
			}
		})
	}
}

func TestGenerateFileActivity(t *testing.T) {
	tests := map[string]struct {
		nodeEventPercent     int
		expectNodeEvent      bool // if true, we expect container ID to be empty (node event)
		expectContainerEvent bool // if true, we expect container ID to be non-empty (container event)
	}{
		"NodeEventPercent 100 should always generate node events": {
			nodeEventPercent:     100,
			expectNodeEvent:      true,
			expectContainerEvent: false,
		},
		"NodeEventPercent 0 should always generate container events": {
			nodeEventPercent:     0,
			expectNodeEvent:      false,
			expectContainerEvent: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			wm := &WorkloadManager{
				workload: &Workload{
					FileActivityWorkload: FileActivityWorkload{
						NodeEventPercent: tc.nodeEventPercent,
					},
				},
				containerPool: newPool(),
				processPool:   newProcessPool(),
			}

			// Add test containers to the pool for container event testing
			if tc.expectContainerEvent {
				wm.containerPool.add("test-container-1")
				wm.containerPool.add("test-container-2")
			}

			paths := generateFileActivityPaths(10)
			hostname := "test-node"

			// Generate multiple activities to test randomness
			for i := 0; i < 50; i++ {
				activity := wm.generateFileActivity(paths, hostname)

				assert.NotNil(t, activity, "Activity should not be nil")
				assert.NotNil(t, activity.Process, "Process should not be nil")
				assert.Equal(t, hostname, activity.Hostname, "Hostname should match")
				assert.NotNil(t, activity.Timestamp, "Timestamp should not be nil")
				assert.NotNil(t, activity.File, "File activity type should be set")

				// Verify process signal
				process := activity.Process
				assert.NotEmpty(t, process.Id, "Process ID should not be empty")
				assert.NotEmpty(t, process.Name, "Process name should not be empty")
				assert.NotEmpty(t, process.ExecFilePath, "Exec file path should not be empty")

				if tc.expectNodeEvent {
					assert.Empty(t, process.ContainerId, "Container ID should be empty for node events")
				}
				if tc.expectContainerEvent {
					assert.NotEmpty(t, process.ContainerId, "Container ID should not be empty for container events")
				}
			}
		})
	}
}
