package env

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetProcessFilterModeConfig(t *testing.T) {
	tests := []struct {
		name           string
		mode           string
		expectedConfig *ProcessFilterModeConfig
		expectNil      bool
	}{
		{
			name: "Aggressive mode",
			mode: "aggressive",
			expectedConfig: &ProcessFilterModeConfig{
				MaxExactPathMatches: 1,
				FanOutLevels:        []int{},
				MaxProcessPaths:     1000,
			},
		},
		{
			name: "Default mode",
			mode: "default",
			expectedConfig: &ProcessFilterModeConfig{
				MaxExactPathMatches: 5,
				FanOutLevels:        []int{8, 6, 4, 2},
				MaxProcessPaths:     5000,
			},
		},
		{
			name: "Minimal mode",
			mode: "minimal",
			expectedConfig: &ProcessFilterModeConfig{
				MaxExactPathMatches: 100,
				FanOutLevels:        []int{20, 15, 10, 5},
				MaxProcessPaths:     20000,
			},
		},
		{
			name: "Invalid mode defaults to default config",
			mode: "invalid",
			expectedConfig: &ProcessFilterModeConfig{
				MaxExactPathMatches: 5,
				FanOutLevels:        []int{8, 6, 4, 2},
				MaxProcessPaths:     5000,
			},
		},
		{
			name: "Empty mode defaults to default",
			mode: "",
			expectedConfig: &ProcessFilterModeConfig{
				MaxExactPathMatches: 5,
				FanOutLevels:        []int{8, 6, 4, 2},
				MaxProcessPaths:     5000,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Always set the mode (including empty string)
			err := os.Setenv("ROX_PROCESS_FILTER_MODE", tc.mode)
			require.NoError(t, err)
			defer func() {
				_ = os.Unsetenv("ROX_PROCESS_FILTER_MODE")
			}()

			config, _ := GetProcessFilterModeConfig()

			if tc.expectNil {
				assert.Nil(t, config)
			} else {
				require.NotNil(t, config)
				assert.Equal(t, tc.expectedConfig.MaxExactPathMatches, config.MaxExactPathMatches)
				assert.Equal(t, tc.expectedConfig.FanOutLevels, config.FanOutLevels)
				assert.Equal(t, tc.expectedConfig.MaxProcessPaths, config.MaxProcessPaths)
			}
		})
	}
}

func TestGetEffectiveProcessFilterConfig(t *testing.T) {
	tests := []struct {
		name                    string
		mode                    string
		maxExactPathMatchesEnv  string
		fanOutLevelsEnv         string
		maxProcessPathsEnv      string
		expectedMaxExactMatches int
		expectedFanOutLevels    []int
		expectedMaxProcessPaths int
	}{
		{
			name:                    "Aggressive mode with no overrides",
			mode:                    "aggressive",
			expectedMaxExactMatches: 1,
			expectedFanOutLevels:    []int{},
			expectedMaxProcessPaths: 1000,
		},
		{
			name:                    "Default mode with no overrides",
			mode:                    "default",
			expectedMaxExactMatches: 5,
			expectedFanOutLevels:    []int{8, 6, 4, 2},
			expectedMaxProcessPaths: 5000,
		},
		{
			name:                    "Minimal mode with no overrides",
			mode:                    "minimal",
			expectedMaxExactMatches: 100,
			expectedFanOutLevels:    []int{20, 15, 10, 5},
			expectedMaxProcessPaths: 20000,
		},
		{
			name:                    "Aggressive mode with maxExactPathMatches override",
			mode:                    "aggressive",
			maxExactPathMatchesEnv:  "10",
			expectedMaxExactMatches: 10,
			expectedFanOutLevels:    []int{},
			expectedMaxProcessPaths: 1000,
		},
		{
			name:                    "Aggressive mode with fanOutLevels override",
			mode:                    "aggressive",
			fanOutLevelsEnv:         "[5,3]",
			expectedMaxExactMatches: 1,
			expectedFanOutLevels:    []int{5, 3},
			expectedMaxProcessPaths: 1000,
		},
		{
			name:                    "Aggressive mode with maxProcessPaths override",
			mode:                    "aggressive",
			maxProcessPathsEnv:      "2000",
			expectedMaxExactMatches: 1,
			expectedFanOutLevels:    []int{},
			expectedMaxProcessPaths: 2000,
		},
		{
			name:                    "Aggressive mode with all overrides",
			mode:                    "aggressive",
			maxExactPathMatchesEnv:  "20",
			fanOutLevelsEnv:         "[10,8,6]",
			maxProcessPathsEnv:      "3000",
			expectedMaxExactMatches: 20,
			expectedFanOutLevels:    []int{10, 8, 6},
			expectedMaxProcessPaths: 3000,
		},
		{
			name:                    "No mode set uses defaults from individual settings",
			mode:                    "",
			expectedMaxExactMatches: 5,
			expectedFanOutLevels:    []int{8, 6, 4, 2},
			expectedMaxProcessPaths: 5000,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Clean up any existing env vars
			_ = os.Unsetenv("ROX_PROCESS_FILTER_MODE")
			_ = os.Unsetenv("ROX_PROCESS_FILTER_MAX_EXACT_PATH_MATCHES")
			_ = os.Unsetenv("ROX_PROCESS_FILTER_FAN_OUT_LEVELS")
			_ = os.Unsetenv("ROX_PROCESS_FILTER_MAX_PROCESS_PATHS")

			// Set the mode if provided
			if tc.mode != "" {
				err := os.Setenv("ROX_PROCESS_FILTER_MODE", tc.mode)
				require.NoError(t, err)
			}

			// Set individual overrides if provided
			if tc.maxExactPathMatchesEnv != "" {
				err := os.Setenv("ROX_PROCESS_FILTER_MAX_EXACT_PATH_MATCHES", tc.maxExactPathMatchesEnv)
				require.NoError(t, err)
			}
			if tc.fanOutLevelsEnv != "" {
				err := os.Setenv("ROX_PROCESS_FILTER_FAN_OUT_LEVELS", tc.fanOutLevelsEnv)
				require.NoError(t, err)
			}
			if tc.maxProcessPathsEnv != "" {
				err := os.Setenv("ROX_PROCESS_FILTER_MAX_PROCESS_PATHS", tc.maxProcessPathsEnv)
				require.NoError(t, err)
			}

			config, _ := GetEffectiveProcessFilterConfig()

			assert.Equal(t, tc.expectedMaxExactMatches, config.MaxExactPathMatches, "MaxExactPathMatches mismatch")
			assert.Equal(t, tc.expectedFanOutLevels, config.FanOutLevels, "FanOutLevels mismatch")
			assert.Equal(t, tc.expectedMaxProcessPaths, config.MaxProcessPaths, "MaxProcessPaths mismatch")

			// Clean up
			_ = os.Unsetenv("ROX_PROCESS_FILTER_MODE")
			_ = os.Unsetenv("ROX_PROCESS_FILTER_MAX_EXACT_PATH_MATCHES")
			_ = os.Unsetenv("ROX_PROCESS_FILTER_FAN_OUT_LEVELS")
			_ = os.Unsetenv("ROX_PROCESS_FILTER_MAX_PROCESS_PATHS")
		})
	}
}

func TestIsEnvVarExplicitlySet(t *testing.T) {
	testVar := "TEST_ENV_VAR_FOR_TESTING"

	// Clean up before and after
	_ = os.Unsetenv(testVar)
	defer func() {
		_ = os.Unsetenv(testVar)
	}()

	// Test when not set
	_, ok := os.LookupEnv(testVar)
	assert.False(t, ok)

	// Test when set to non-empty value
	err := os.Setenv(testVar, "value")
	require.NoError(t, err)
	_, ok = os.LookupEnv(testVar)
	assert.True(t, ok)

	// Test when set to empty value (should still be considered "set")
	err = os.Setenv(testVar, "")
	require.NoError(t, err)
	_, ok = os.LookupEnv(testVar)
	assert.True(t, ok)
}
