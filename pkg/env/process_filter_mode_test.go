package env

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetEffectiveProcessFilterConfig(t *testing.T) {
	tests := []struct {
		name                    string
		setMode                 bool // whether to set the mode env var
		mode                    string
		maxExactPathMatchesEnv  string
		fanOutLevelsEnv         string
		maxProcessPathsEnv      string
		expectedMaxExactMatches int
		expectedFanOutLevels    []int
		expectedMaxProcessPaths int
		expectedMode            string
		expectError             bool
	}{
		{
			name:                    "Aggressive mode with no overrides",
			setMode:                 true,
			mode:                    "aggressive",
			expectedMaxExactMatches: 1,
			expectedFanOutLevels:    []int{},
			expectedMaxProcessPaths: 1000,
			expectedMode:            "aggressive",
			expectError:             false,
		},
		{
			name:                    "Default mode with no overrides",
			setMode:                 true,
			mode:                    "default",
			expectedMaxExactMatches: 5,
			expectedFanOutLevels:    []int{8, 6, 4, 2},
			expectedMaxProcessPaths: 5000,
			expectedMode:            "default",
			expectError:             false,
		},
		{
			name:                    "Minimal mode with no overrides",
			setMode:                 true,
			mode:                    "minimal",
			expectedMaxExactMatches: 100,
			expectedFanOutLevels:    []int{20, 15, 10, 5},
			expectedMaxProcessPaths: 20000,
			expectedMode:            "minimal",
			expectError:             false,
		},
		{
			name:                    "Invalid mode falls back to default",
			setMode:                 true,
			mode:                    "invalid",
			expectedMaxExactMatches: 5,
			expectedFanOutLevels:    []int{8, 6, 4, 2},
			expectedMaxProcessPaths: 5000,
			expectedMode:            "default",
			expectError:             true,
		},
		{
			name:                    "Empty mode falls back to default",
			setMode:                 true,
			mode:                    "",
			expectedMaxExactMatches: 5,
			expectedFanOutLevels:    []int{8, 6, 4, 2},
			expectedMaxProcessPaths: 5000,
			expectedMode:            "default",
			expectError:             true,
		},
		{
			name:                    "Aggressive mode with maxExactPathMatches override",
			setMode:                 true,
			mode:                    "aggressive",
			maxExactPathMatchesEnv:  "10",
			expectedMaxExactMatches: 10,
			expectedFanOutLevels:    []int{},
			expectedMaxProcessPaths: 1000,
			expectedMode:            "aggressive",
			expectError:             false,
		},
		{
			name:                    "Aggressive mode with fanOutLevels override",
			setMode:                 true,
			mode:                    "aggressive",
			fanOutLevelsEnv:         "[5,3]",
			expectedMaxExactMatches: 1,
			expectedFanOutLevels:    []int{5, 3},
			expectedMaxProcessPaths: 1000,
			expectedMode:            "aggressive",
			expectError:             false,
		},
		{
			name:                    "Aggressive mode with maxProcessPaths override",
			setMode:                 true,
			mode:                    "aggressive",
			maxProcessPathsEnv:      "2000",
			expectedMaxExactMatches: 1,
			expectedFanOutLevels:    []int{},
			expectedMaxProcessPaths: 2000,
			expectedMode:            "aggressive",
			expectError:             false,
		},
		{
			name:                    "Aggressive mode with all overrides",
			setMode:                 true,
			mode:                    "aggressive",
			maxExactPathMatchesEnv:  "20",
			fanOutLevelsEnv:         "[10,8,6]",
			maxProcessPathsEnv:      "3000",
			expectedMaxExactMatches: 20,
			expectedFanOutLevels:    []int{10, 8, 6},
			expectedMaxProcessPaths: 3000,
			expectedMode:            "aggressive",
			expectError:             false,
		},
		{
			name:                    "No mode set uses defaults from individual settings",
			setMode:                 false,
			expectedMaxExactMatches: 5,
			expectedFanOutLevels:    []int{8, 6, 4, 2},
			expectedMaxProcessPaths: 5000,
			expectedMode:            "",
			expectError:             false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set the mode if requested
			if tc.setMode {
				t.Setenv("ROX_PROCESS_FILTER_MODE", tc.mode)
			}

			// Set individual overrides if provided
			if tc.maxExactPathMatchesEnv != "" {
				t.Setenv("ROX_PROCESS_FILTER_MAX_EXACT_PATH_MATCHES", tc.maxExactPathMatchesEnv)
			}
			if tc.fanOutLevelsEnv != "" {
				t.Setenv("ROX_PROCESS_FILTER_FAN_OUT_LEVELS", tc.fanOutLevelsEnv)
			}
			if tc.maxProcessPathsEnv != "" {
				t.Setenv("ROX_PROCESS_FILTER_MAX_PROCESS_PATHS", tc.maxProcessPathsEnv)
			}

			config, mode, err := GetEffectiveProcessFilterConfig()

			assert.Equal(t, tc.expectedMaxExactMatches, config.MaxExactPathMatches, "MaxExactPathMatches mismatch")
			assert.Equal(t, tc.expectedFanOutLevels, config.FanOutLevels, "FanOutLevels mismatch")
			assert.Equal(t, tc.expectedMaxProcessPaths, config.MaxProcessPaths, "MaxProcessPaths mismatch")
			assert.Equal(t, tc.expectedMode, mode, "Mode mismatch")

			if tc.expectError {
				assert.Error(t, err, "Expected an error")
			} else {
				assert.NoError(t, err, "Expected no error")
			}
		})
	}
}
